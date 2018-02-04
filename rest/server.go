package rest

import (
	"net/http"
	"errors"
	"context"
	"github.com/gorilla/mux"
	"log"
	"github.com/sakno/go2rest/cmdexec"
	"fmt"
	"strings"
	"mime"
	"io"
	"os"
	"net/textproto"
	"strconv"
	"github.com/sakno/go2rest/core"
)
const (
	headerContentType = "Content-Type"
	headerContentLength = "Content-Length"
	TemplateParamBody = "body"
)
var wellKnownMethods = []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodOptions, http.MethodHead, http.MethodPatch}

//context of service request
type requestContext struct {
	*http.Request
	args cmdexec.Arguments
	deferredActions []core.DeferredAction
}

func (self *requestContext) finalize() {
	for _, action := range self.deferredActions {
		action()
	}
}

func (self *requestContext) Defer(action core.DeferredAction) {
	self.deferredActions = append(self.deferredActions, action)
}

func (self *requestContext) deferClose(closer io.Closer) {
	cl := closer.Close
	self.Defer(func() { cl() })
}

type StandaloneServer struct {
	http.Server
	CertFile, KeyFile string
	Model Model
}

func setDefaultValue(name string, input Parameter, output cmdexec.Arguments) bool{
	switch parameter := input.(type) {
	case IntegerParameter:
		output[name] = parameter.DefaultValue()
		return true
	case StringParameter:
		output[name] = parameter.DefaultValue()
		return true
	case BoolParameter:
		output[name] = parameter.DefaultValue()
		return true
	case NumberParameter:
		output[name] = parameter.DefaultValue()
		return true
	case ArrayParameter:
		output[name] = make([]interface{}, 0, 0)
		return true
	default:
		return false
	}
}

type parametersExtractor func(r *http.Request) map[string]string

func (self *requestContext) parseArguments(parameters ParameterList, varResolver parametersExtractor) error {
	vars := varResolver(self.Request)
	for name, parameter := range parameters {
		//check parameter type
		if _, ok := parameter.(FileParameter); ok {
			return errors.New(fmt.Sprintf("parameter %s can't have FILE type", name))
		} else if value, exists := vars[name]; exists { //parameter exists in the request
			if value, err := parameter.ReadValue(strings.NewReader(value), FormatText); err == nil {
				if parameter.Validate(value) {
					self.args[name] = value
				} else {
					return &textproto.Error{Msg: fmt.Sprintf("Argument %s has invalid value %v", name, value), Code: http.StatusBadRequest}
				}
			} else {
				return err
			}
		} else if parameter.HasDefaultValue() { //parameter doesn't exist and not required
			setDefaultValue(name, parameter, self.args)
		} else if parameter.Required() {
			return &textproto.Error{
				Msg:  fmt.Sprintf("Parameter %s is required but not specified in actual request", name),
				Code: http.StatusBadRequest,
			}
		}
	}
	return nil
}

func extractQueryParameters(request *http.Request) map[string]string {
	return ExtractQueryParameters(request.URL, ",")
}

func extractHeaders(request *http.Request) map[string]string {
	result := make(map[string]string, len(request.Header))
	for header := range request.Header {
		result[header] = request.Header.Get(header)
	}
	return result
}

func (self *requestContext) parseRequestBody(bodyDefinition ParameterList, requestType string) error {
	defer self.Body.Close()
	if body, exists := bodyDefinition[requestType]; exists { //body for this MIME type is specified
		if body, err := body.ReadValue(self.Body, GetFormatByMIME(requestType)); err == nil {
			switch body := body.(type) {
			case os.File:
				//for file we need to return file name only.
				defer body.Close()
				fileName := body.Name()
				self.args[TemplateParamBody] = fileName
				self.Defer(func() { os.Remove(fileName) })	//ensure that temporary file with request body will be deleted
				return nil
			default:
				self.args[TemplateParamBody] = body
				return nil
			}
		} else if err == io.EOF { //no body is present
			return &textproto.Error{Msg: "Request body is empty", Code: http.StatusBadRequest}
		} else { //failed to read body
			return err
		}
	} else if len(bodyDefinition) == 0 { //model has no definition of the body. It's ok and just return without any error
		return nil
	} else { //media type is not configured in model
		return &textproto.Error{Msg: fmt.Sprintf("Unsupported media type: %s", requestType), Code: http.StatusUnsupportedMediaType}
	}
}

func convertToHttpError(err error, response http.ResponseWriter) {
	switch err := err.(type) {
	case *textproto.Error:
		http.Error(response, err.Msg, err.Code)
	default:
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

//handles HTTP request according with model specification
func (self *requestContext) parseRequest(descriptor HttpMethodDescriptor, response http.ResponseWriter) bool {
	//check media type and define default media type if necessary
	contentType := self.Header.Get(headerContentType)
	if len(contentType) == 0 {
		contentType = "text/plain"
	}
	if contentType, _, err := mime.ParseMediaType(contentType); err == nil {
		if err := self.parseArguments(descriptor.QueryParameters(), extractQueryParameters); err != nil {//parse query parameters
			convertToHttpError(err, response)
			return false
		} else if err := self.parseArguments(descriptor.RequestHeaders(), extractHeaders); err != nil {//parse headers
			convertToHttpError(err, response)
			return false
		} else if err := self.parseRequestBody(descriptor.Request(), contentType); err != nil {//parse request body
			convertToHttpError(err, response)
			return false
		}
	} else {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func (self *requestContext) handleRequest(endpoint Endpoint, response http.ResponseWriter) {
	//parse path arguments
	if err := self.parseArguments(endpoint.PathParameters(), mux.Vars); err != nil {
		http.Error(response, fmt.Sprintf("Incorrect path arguments. Error: %s", err.Error()), http.StatusBadRequest)
	}
	method := endpoint.GetMethodDescriptor(self.Method)
	if method == nil {
		http.Error(response, fmt.Sprintf("Method %s is not supported", self.Method), http.StatusMethodNotAllowed)
	} else if self.parseRequest(method, response) {//prepare execution arguments
		//execute command-line tool
		if successResponse, ok := method.Response()[0]; ok { //success response always associated with zero exit code
			response.Header().Set(headerContentType, successResponse.MimeType)
			var receiver cmdexec.ExecutionResultRecorder
			//select buffer for output according with response type
			switch successResponse.Body.(type) {
			case FileParameter: //for file response result should be saved into temporary file, not in memory
				if tempFile, err := cmdexec.NewTempFileRecorder(true); err == nil {
					receiver = tempFile
				} else {
					convertToHttpError(err, response)
					return
				}
			default: //for non-file response, content can be saved into in-memory buffer
				receiver = cmdexec.NewTextRecorder()
			}
			self.deferClose(receiver) //ensure that response buffer will be closed
			//now execute command
			if err := method.Executor()(self.args, receiver); err == nil {
				//extract content length from execution result
				response.Header().Set(headerContentLength, strconv.Itoa(receiver.Len()))
				response.WriteHeader(successResponse.StatusCode)
				//copy buffer into HTTP response
				if _, err := receiver.WriteTo(response); err != nil {
					convertToHttpError(err, response)
				}
			} else {
				switch err := err.(type) {
				case *cmdexec.ExecutionError:
					if resp, exists := method.Response()[err.ProcessExitCode]; exists {
						response.Header().Set(headerContentType, resp.MimeType)
						http.Error(response, err.Error(), resp.StatusCode)
					} else {
						http.Error(response, err.Error(), http.StatusInternalServerError)
					}
				default:
					convertToHttpError(err, response)
				}
			}
		} else {
			http.Error(response, "There is no status code associated with process exit code 0", http.StatusInternalServerError)
		}
	}
}

//creates HTTP handler for the specified endpoint
func CreateEndpointHandler(endpoint Endpoint) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		//initialize logical operation context
		ctx := &requestContext{
			deferredActions: make([]core.DeferredAction, 0, 3),
			Request: request,
			args: cmdexec.NewArguments(),
			}
		defer ctx.finalize()	//ensure that context will be closed
		ctx.handleRequest(endpoint, response)
	}
}

func getAllowedMethods(endpoint Endpoint) []string {
	methods := make([]string, 0, len(wellKnownMethods))
	for _, method := range wellKnownMethods {
		if endpoint.GetMethodDescriptor(method) != nil {
			methods = append(methods, method)
		}
	}
	return methods
}

func prepareRouter(router *mux.Router, model Model) {
	log.Printf("Starting REST service %s", model.Name())
	for path, endpoint := range model.Endpoints() {
		router.NewRoute().
			Path(path).
			Methods(getAllowedMethods(endpoint)...).
			HandlerFunc(CreateEndpointHandler(endpoint))
	}
}

func (self *StandaloneServer) run() error {
	//start HTTP server
	if self.CertFile != "" && self.KeyFile != "" {
		return self.ListenAndServeTLS(self.CertFile, self.KeyFile)
	} else {
		return self.ListenAndServe()
	}
}

func (self *StandaloneServer) Run(async bool) error {
	//setup model
	if self.Model == nil {
		return errors.New("REST model is not defined")
	} else {
		router := mux.NewRouter()
		prepareRouter(router, self.Model)
		self.Handler = router
	}
	if async {
		go self.run()
		return nil
	} else {
		return self.run()
	}
}

func (self *StandaloneServer) Close() error {
	return self.Shutdown(context.Background())
}