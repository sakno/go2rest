package rest

import (
	"net/http"
	"errors"
	"context"
	"github.com/gorilla/mux"
	"log"
	"../cmdexec"
	"fmt"
	"strings"
	"mime"
	"io"
	"os"
	"net/textproto"
	"strconv"
)
const (
	headerContentType = "Content-Type"
	headerContentLength = "Content-Length"
	TemplateParamBody = "body"
)
var wellKnownMethods = []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodOptions, http.MethodHead, http.MethodPatch}

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

type parametersExtractor func(*http.Request) map[string]string

func (self ParameterList) parseArguments(input *http.Request, output cmdexec.Arguments, varResolver parametersExtractor) error {
	vars := varResolver(input)
	for name, parameter := range self {
		if value, exists := vars[name]; exists { //parameter exists in the request
			if value, err := parameter.ReadValue(strings.NewReader(value), FormatText); err == nil {
				if parameter.Validate(value) {
					output[name] = value
				} else {
					return &textproto.Error{Msg: fmt.Sprintf("Argument %s has invalid value %v", name, value), Code:http.StatusBadRequest}
				}
			} else {
				return err
			}
		} else if parameter.HasDefaultValue() { //parameter doesn't exist and not required
			setDefaultValue(name, parameter, output)
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

func (self ParameterList) parseRequestBody(requestType string, request io.ReadCloser, output cmdexec.Arguments) error {
	defer request.Close()
	if body, exists := self[requestType]; exists { //body for this MIME type is specified
		if body, err := body.ReadValue(request, GetFormatByMIME(requestType)); err == nil {
			switch body := body.(type) {
			case os.File:
				//for file we need to return file name only.
				defer body.Close()
				output[TemplateParamBody] = body.Name()
				return nil
			default:
				output[TemplateParamBody] = body
				return nil
			}
		} else if err == io.EOF { //no body is present
			return &textproto.Error{Msg: "Request body is empty", Code: http.StatusBadRequest}
		} else { //failed to read body
			return err
		}
	} else if len(self) == 0 { //model has no definition of the body. It's ok and just return without any error
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
func parseRequest(descriptor HttpMethodDescriptor, request *http.Request, response http.ResponseWriter, executionArgs cmdexec.Arguments) bool {
	//check media type and define default media type if necessary
	contentType := request.Header.Get(headerContentType)
	if len(contentType) == 0 {
		contentType = "text/plain"
	}
	if contentType, _, err := mime.ParseMediaType(contentType); err == nil {
		//parse query parameters
		if err := descriptor.QueryParameters().parseArguments(request, executionArgs, extractQueryParameters); err != nil {
			convertToHttpError(err, response)
			return false
		} else if err := descriptor.RequestHeaders().parseArguments(request, executionArgs, extractHeaders); err != nil {
			convertToHttpError(err, response)
			return false
		} else if err := descriptor.Request().parseRequestBody(contentType, request.Body, executionArgs); err != nil {
			convertToHttpError(err, response)
			return false
		}
	} else {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

//creates HTTP handler for the specified endpoint
func CreateEndpointHandler(endpoint Endpoint) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		executionArguments := cmdexec.NewArguments()
		//parse path arguments
		if err := endpoint.PathParameters().parseArguments(request, executionArguments, mux.Vars); err != nil {
			http.Error(response, fmt.Sprintf("Incorrect path arguments. Error: %s", err.Error()), http.StatusBadRequest)
		}
		method := endpoint.GetMethodDescriptor(request.Method)
		if method == nil {
			http.Error(response, fmt.Sprintf("Method %s is not supported", request.Method), http.StatusMethodNotAllowed)
		} else if parseRequest(method, request, response, executionArguments) {//prepare execution arguments
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
				defer receiver.Close() //ensure that response buffer will be closed
				//now execute command
				if err := method.Executor()(executionArguments, receiver); err == nil {
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