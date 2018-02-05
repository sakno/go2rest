package raml

import (
	"io"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
	"log"
	"regexp"
	"math"
	"strconv"
	"errors"
	"fmt"
	"reflect"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/sakno/go2rest/cmdexec"
	"github.com/sakno/go2rest/rest"
	"net/http"
	"net/url"
)

const (
	//RAML primitive types
	tString  = "string"
	tInteger = "integer"
	tNumber  = "number"
	tBoolean = "boolean"
	tFile    = "file"
	tArray   = "array"
	tAny     = "any"
	//RAML fields
	fType      = "type"
	fRequired  = "required"
	fDefault   = "default"
	fPattern   = "pattern"
	fMinimum   = "minimum"
	fMaximum   = "maximum"
	fMinLength = "minLength"
	fMaxLength = "maxLength"
	fMinItems  = "minItems"
	fMaxItems  = "maxItems"
	fItems     = "items"
	fHeaders   = "headers"
	fBaseUri = "baseUri"
	fTitle = "title"
	fQueryParameters = "queryParameters"
	fBody 	   = "body"
	fResponses = "responses"
	fExitCode  = "(exitCode)"
	fCommandPattern = "(commandPattern)"
)

func mapSliceToMap(tree yaml.MapSlice) map[string]interface{} {
	//convert parameter fields into map
	fields := make(map[string]interface{}, len(tree))
	for _, item := range tree {
		if name, ok := item.Key.(string); ok {
			fields[name] = item.Value
		}
	}
	return fields
}

type Parameter struct {
	hasDefaultValue bool
	required bool
}

func (self *Parameter) HasOption(name string) bool {
	return false
}

func parseBaseParameter(description map[string]interface{}) Parameter {
	result := Parameter{}
	//parse 'required' field
	if required, ok := description[fRequired]; ok {
		switch required {
		case "false", false:
			result.required = false
		default:
			result.required = true
		}
	} else {
		result.required = true
	}
	//parse 'default' field
	_, hasDefault := description[fDefault]
	result.hasDefaultValue = hasDefault
	return result
}

func (self *Parameter) Required() bool {
	return self.required
}

func (self *Parameter) HasDefaultValue() bool {
	return self.hasDefaultValue
}

type AnyParameter struct {
	Parameter
}

func (self *AnyParameter) ReadValue(value io.Reader, format rest.ParameterValueFormat) (interface{}, error) {
	switch format {
	case rest.FormatText:
		if value, err := ioutil.ReadAll(value); err == nil {
			return string(value), nil
		} else {
			return nil, err
		}
	case rest.FormatBinary:
		if file, err := cmdexec.NewTempFile(); err == nil {
			io.Copy(file, value)
			file.Seek(0, io.SeekStart)
			return file, nil
		} else {
			return nil, err
		}
	case rest.FormatJSON:
		var result interface{}
		err := json.NewDecoder(value).Decode(&result)
		return result, err
	default:
		return nil, errors.New("unsupported value format")
	}
}

func (self *AnyParameter) parse(description map[string]interface{}) {
	self.Parameter = parseBaseParameter(description)

}

func (self *AnyParameter) Validate(value interface{}) bool {
	switch value.(type) {
	case bytes.Buffer, os.File, string, float64, bool, nil:
		return true
	default:
		return false
	}
}

//Represents parameter of type 'file' restored from RAML model
type FileParameter struct {
	Parameter
}

func (self *FileParameter) ReadValue(value io.Reader, format rest.ParameterValueFormat) (interface{}, error) {
	if file, err := cmdexec.NewTempFile(); err == nil {
		if _, err := io.Copy(file, value); err != nil {
			return nil, err
		} else if _, err := file.Seek(0, io.SeekStart); err == nil {//return file content with correct position in the reader
			return file, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (self *FileParameter) parse(description map[string]interface{}){
	self.Parameter = parseBaseParameter(description)
	self.hasDefaultValue = false
}

func (self *FileParameter) Validate(value interface{}) bool {
	switch value.(type) {
	case []byte, io.Reader, bytes.Buffer, os.File:
		return true
	default:
		return false
	}
}

func (self *FileParameter) Persistent() bool {
	return true
}

//Represents parameter of type 'number' restored from RAML model
type NumberParameter struct {
	Parameter
	defaultValue float64
	minimum float64
	maximum float64
}

func (self *NumberParameter) readValue(value []byte, format rest.ParameterValueFormat) (interface{}, error) {
	switch format {
	case rest.FormatBinary:
		return math.Float64frombits(binary.LittleEndian.Uint64(value)), nil
	case rest.FormatText:
		return strconv.ParseFloat(bytes.NewBuffer(value).String(), 64)
	case rest.FormatJSON:
		result := new(float64)
		err := json.Unmarshal(value, result)
		return *result, err
	default:
		return nil, new(rest.UnsupportedParameterValueFormat)
	}
}

func (self *NumberParameter) ReadValue(value io.Reader, format rest.ParameterValueFormat) (interface{}, error) {
	if value, err := ioutil.ReadAll(value); err == nil {
		return self.readValue(value, format)
	} else {
		return nil, err
	}
}

func toFloat64(value interface{}) (float64, error) {
	switch defaultValue := value.(type) {
	case int:
		return float64(defaultValue), nil
	case int8:
		return float64(defaultValue), nil
	case uint8:
		return float64(defaultValue), nil
	case int16:
		return float64(defaultValue), nil
	case uint16:
		return float64(defaultValue), nil
	case int32:
		return float64(defaultValue), nil
	case uint32:
		return float64(defaultValue), nil
	case int64:
		return float64(defaultValue), nil
	case uint64:
		return float64(defaultValue), nil
	case float32:
		return float64(defaultValue), nil
	case float64:
		return defaultValue, nil
	case string:
		if defaultValue, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			return defaultValue, nil
		} else {
			return 0, err
		}
	default:
		return 0, errors.New(fmt.Sprintf("Failed to parse numeric constant: %v", defaultValue))
	}
}

func (self *NumberParameter) init(){
	//defaults
	self.maximum = math.MaxFloat64
	self.minimum = math.SmallestNonzeroFloat64
}

func (self *NumberParameter) parse(description map[string]interface{}){
	self.init()
	self.Parameter = parseBaseParameter(description)

	//parse default value
	if self.hasDefaultValue {
		if defaultValue, err := toFloat64(description[fDefault]); err == nil {
			self.defaultValue = defaultValue
		} else {
			log.Fatal(err.Error())
		}
	}
	//parse minimum
	if minimum, err := toFloat64(description[fMinimum]); err == nil {
		self.minimum = minimum
	}
	//parse maximum
	if maximum, err := toFloat64(description[fMaximum]); err == nil {
		self.maximum = maximum
	}
}

func (self *NumberParameter) DefaultValue() float64 {
	return self.defaultValue
}

func (self *NumberParameter) validate(value float64) bool{
	return value >= self.minimum && value <= self.maximum
}

func (self *NumberParameter) Validate(value interface{}) bool {
	switch typed := value.(type) {
	case int8:
		return self.validate(float64(typed))
	case uint8:
		return self.validate(float64(typed))
	case int16:
		return self.validate(float64(typed))
	case uint16:
		return self.validate(float64(typed))
	case int32:
		return self.validate(float64(typed))
	case int64:
		return self.validate(float64(typed))
	case uint64:
		return self.validate(float64(typed))
	case float32:
		return self.validate(float64(typed))
	case float64:
		return self.validate(typed)
	case int:
		return self.validate(float64(typed))
	default:
		return false
	}
}

//Represents parameter of type 'integer' restored from RAML model
type IntegerParameter struct {
	Parameter
	defaultValue int64
	minimum int64
	maximum int64
}

func (self *IntegerParameter) readValue(value []byte, format rest.ParameterValueFormat) (interface{}, error) {
	switch format {
	case rest.FormatBinary:
		return int64(binary.LittleEndian.Uint64(value)), nil
	case rest.FormatText:
		return strconv.ParseInt(bytes.NewBuffer(value).String(), 0, 64)
	case rest.FormatJSON:
		result := new(int64)
		err := json.Unmarshal(value, result)
		return *result, err
	default:
		return nil, new(rest.UnsupportedParameterValueFormat)
	}
}

func (self *IntegerParameter) ReadValue(value io.Reader, format rest.ParameterValueFormat) (interface{}, error) {
	if value, err := ioutil.ReadAll(value); err == nil {
		return self.readValue(value, format)
	} else {
		return nil, err
	}
}

func toUInt32(value interface{}) (uint32, error) {
	switch defaultValue := value.(type) {
	case int:
		return uint32(defaultValue), nil
	case int8:
		return uint32(defaultValue), nil
	case uint8:
		return uint32(defaultValue), nil
	case int16:
		return uint32(defaultValue), nil
	case uint16:
		return uint32(defaultValue), nil
	case int32:
		return uint32(defaultValue), nil
	case uint32:
		return defaultValue, nil
	case string:
		if defaultValue, err := strconv.ParseUint(defaultValue, 0, 32); err == nil {
			return uint32(defaultValue), nil
		} else {
			return 0, err
		}
	default:
		return 0, errors.New(fmt.Sprintf("Failed to parse integer constant: %v", defaultValue))
	}
}

func toInt64(value interface{}) (int64, error) {
	switch defaultValue := value.(type) {
	case int:
		return int64(defaultValue), nil
	case int8:
		return int64(defaultValue), nil
	case uint8:
		return int64(defaultValue), nil
	case int16:
		return int64(defaultValue), nil
	case uint16:
		return int64(defaultValue), nil
	case int32:
		return int64(defaultValue), nil
	case uint32:
		return int64(defaultValue), nil
	case int64:
		return defaultValue, nil
	case string:
		if defaultValue, err := strconv.ParseInt(defaultValue, 0, 64); err == nil {
			return defaultValue, nil
		} else {
			return 0, err
		}
	default:
		return 0, errors.New(fmt.Sprintf("Failed to parse integer constant: %v", defaultValue))
	}
}

func (self *IntegerParameter) init() {
	//defaults
	self.maximum = math.MaxInt64
	self.minimum = math.MinInt64
}

func (self *IntegerParameter) parse(description map[string]interface{}) {
	self.init()
	self.Parameter = parseBaseParameter(description)

	//parse default value
	if self.hasDefaultValue {
		if defaultValue, err := toInt64(description[fDefault]); err == nil {
			self.defaultValue = defaultValue
		} else {
			log.Fatal(err.Error())
		}
	}
	//parse minimum
	if minimum, err := toInt64(description[fMinimum]); err == nil {
		self.minimum = minimum
	}
	//parse maximum
	if maximum, err := toInt64(description[fMaximum]); err == nil {
		self.maximum = maximum
	}
}

func (self *IntegerParameter) validate(value int64) bool {
	return value >= self.minimum && value <= self.maximum
}

func (self *IntegerParameter) Validate(value interface{}) bool {
	switch typed := value.(type) {
	case int8:
		return self.validate(int64(typed))
	case uint8:
		return self.validate(int64(typed))
	case int16:
		return self.validate(int64(typed))
	case uint16:
		return self.validate(int64(typed))
	case int32:
		return self.validate(int64(typed))
	case uint32:
		return self.validate(int64(typed))
	case int64:
		return self.validate(typed)
	case uint:
		return self.validate(int64(typed))
	case int:
		return self.validate(int64(typed))
	case float64:
		return self.validate(int64(typed))
	default:
		return false
	}
}

func (self *IntegerParameter) DefaultValue() int64 {
	return self.defaultValue
}

//Represents parameter of type 'boolean' restored from RAML model
type BooleanParameter struct {
	Parameter
	defaultValue bool
}

func (self *BooleanParameter) readValue(value []byte, format rest.ParameterValueFormat) (interface{}, error) {
	switch format {
	case rest.FormatBinary:
		if len(value) > 0{
			return value[0] != 0, nil
		} else {
			return false, nil
		}
	case rest.FormatText:
		return strconv.ParseBool(bytes.NewBuffer(value).String())
	case rest.FormatJSON:
		result := new(bool)
		err := json.Unmarshal(value, result);
		return *result, err
	default:
		return nil, new(rest.UnsupportedParameterValueFormat)
	}
}

func (self *BooleanParameter) ReadValue(value io.Reader, format rest.ParameterValueFormat) (interface{}, error) {
	if value, err := ioutil.ReadAll(value); err == nil {
		return self.readValue(value, format)
	} else {
		return nil, err
	}
}

func (self *BooleanParameter) parse(description map[string]interface{}) {
	self.Parameter = parseBaseParameter(description)
	//parse default value
	if self.hasDefaultValue {
		switch defaultValue := description[fDefault].(type) {
		case bool:
			self.defaultValue = defaultValue
		case string:
			if defaultValue, err := strconv.ParseBool(defaultValue); err == nil {
				self.defaultValue = defaultValue
			} else {
				log.Fatalf("Failed to parse boolean constant: %s", err.Error())
			}
		default:
			log.Fatalf("Failed to parse boolean constant: %v", defaultValue)
		}
	}
}

func (self *BooleanParameter) DefaultValue() bool {
	return self.defaultValue
}

func (self *BooleanParameter) Validate(value interface{}) bool {
	_, ok := value.(bool)
	return ok
}

//Represents parameter of type 'string' restored from RAML model
type StringParameter struct {
	Parameter
	defaultValue string
	minLength uint32
	maxLength uint32
	pattern *regexp.Regexp
}

func (self *StringParameter) readValue(value []byte, format rest.ParameterValueFormat) (interface{}, error) {
	switch format {
	case rest.FormatBinary, rest.FormatText:
		return bytes.NewBuffer(value).String(), nil
	case rest.FormatJSON:
		result := new(string)
		err := json.Unmarshal(value, result)
		return *result, err
	default:
		return nil, new(rest.UnsupportedParameterValueFormat)
	}
}

func (self *StringParameter) ReadValue(value io.Reader, format rest.ParameterValueFormat) (interface{}, error) {
	if value, err := ioutil.ReadAll(value); err == nil {
		return self.readValue(value, format)
	} else {
		return nil, err
	}
}

func (self *StringParameter) init() {
	//defaults
	self.minLength = 0
	self.maxLength = math.MaxInt32
	self.pattern = nil
}

func (self *StringParameter) parse(description map[string]interface{}) {
	//defaults
	self.init()

	self.Parameter = parseBaseParameter(description)
	//parse default value
	if self.hasDefaultValue {
		self.defaultValue = description[fDefault].(string)
	}
	//parse pattern
	if pattern, ok := description[fPattern].(string); ok {
		self.pattern = regexp.MustCompile(pattern)
	} else {
		self.pattern = nil
	}
	//parse min length
	if minLength, ok := description[fMinLength]; ok {
		if minLength, err := toUInt32(minLength); err == nil {
			self.minLength = minLength
		} else {
			log.Fatal(err.Error())
		}
	}
	//parse max length
	if maxLength, ok := description[fMaxLength]; ok {
		if maxLength, err := toUInt32(maxLength); err == nil {
			self.maxLength = maxLength
		} else {
			log.Fatal(err.Error())
		}
	}
}

func (self *StringParameter) DefaultValue() string {
	return self.defaultValue
}

func (self *StringParameter) validate(value string) bool {
	if length := len(value); length < int(self.minLength) || length > int(self.maxLength) {
		return false
	} else if self.pattern == nil {
		return true
	} else {
		return self.pattern.MatchString(value)
	}
}

func (self *StringParameter) Validate(value interface{}) bool {
	if value == nil {
		return true
	} else if typed, ok := value.(string); ok {
		return self.validate(typed)
	} else {
		return false
	}
}

//Represents RAML parameter of type '[]' restored from RAML model
type ArrayParameter struct {
	Parameter
	minItems uint32
	maxItems uint32
	elementType rest.Parameter
}

func (self *ArrayParameter) readValue(value []byte, format rest.ParameterValueFormat) (interface{}, error) {
	switch format {
	case rest.FormatJSON, rest.FormatText:
		result := new([]interface{})
		err := json.Unmarshal(value, result)
		return *result, err
	default:
		return nil, new(rest.UnsupportedParameterValueFormat)
	}
}

func (self *ArrayParameter) ReadValue(value io.Reader, format rest.ParameterValueFormat) (interface{}, error) {
	if value, err := ioutil.ReadAll(value); err == nil {
		return self.readValue(value, format)
	} else {
		return nil, err
	}
}

func (self *ArrayParameter) init(){
	//defaults
	self.minItems = 0
	self.maxItems = math.MaxInt32
}

func (self *ArrayParameter) parse(description map[string]interface{}) {
	self.init()
	self.Parameter = parseBaseParameter(description)
	self.hasDefaultValue = false
	//parse min items
	if minItems, err := toUInt32(description[fMinItems]); err == nil {
		self.minItems = minItems
	}
	//parse max items
	if maxItems, err := toUInt32(description[fMaxItems]); err == nil {
		self.maxItems = maxItems
	}
	//parse element type
	switch items := description[fItems].(type) {
	case string: //items contains name of type
		self.elementType = parseParameterType(items, make(map[string]interface{}, 0))
	case yaml.MapSlice:
		self.elementType = parseParameter(items)
	case rest.Parameter:
		self.elementType = items
	default:
		if self.elementType == nil {
			log.Fatalf("Unsupported array type %+v", items)
		}
	}
}

func (self *ArrayParameter) Validate(value interface{}) bool {
	if self.elementType == nil {
		return false
	}
	reflectedValue := reflect.ValueOf(value)
	switch reflectedValue.Kind() {
	case reflect.Slice:
		//validate each element of array
		length := reflectedValue.Len()
		if uint32(length) <= self.minItems || uint32(length) >= self.maxItems {
			return false
		}
		for i := 0; i < length; i++ {
			item := reflectedValue.Index(i)
			if item.CanInterface() && self.elementType.Validate(item.Interface()) {
				continue
			} else {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (self *ArrayParameter) ElementType() rest.Parameter {
	return self.elementType
}

type MethodDescriptor struct {
	queryParameters rest.ParameterList
	reqHeaders rest.ParameterList
	request rest.ParameterList
	responses map[int]rest.ResponseDescriptor
	executor cmdexec.CommandExecutor
}

func (self *MethodDescriptor) HasOption(name string) bool {
	return false
}

func (self *MethodDescriptor) Executor() cmdexec.CommandExecutor {
	return self.executor
}

func (self *MethodDescriptor) parse(description interface{}) {
	if tree, ok := description.(yaml.MapSlice); ok {
		tree := mapSliceToMap(tree)
		//parse headers
		self.reqHeaders = make(rest.ParameterList)
		if reqHeaders, ok := tree[fHeaders]; ok {
			parseParameterList(reqHeaders, self.reqHeaders)
		}
		//parse query parameters
		self.queryParameters = make(rest.ParameterList)
		if queryParameters, ok := tree[fQueryParameters]; ok {
			parseParameterList(queryParameters, self.queryParameters)
		}
		//parse body
		self.request = make(rest.ParameterList)
		if request, ok := tree[fBody]; ok {
			parseParameterList(request, self.request)
		}
		//parse command pattern
		if commandPattern, ok := tree[fCommandPattern]; ok {
			if commandPattern, ok := commandPattern.(string); ok {
				if renderer, err := cmdexec.NewAutoNamedRenderer(commandPattern); err == nil {
					self.executor = cmdexec.NewCommandExecutor(renderer)
				} else {
					log.Fatalf("Failed to parse command pattern %s. Error %s", commandPattern, err.Error())
				}
			} else {
				log.Fatal("Invalid format of command pattern")
			}
		} else {
			log.Fatal("Command pattern is not specified")
		}
		//parse responses
		self.responses = make(map[int]rest.ResponseDescriptor)
		if responses, ok := tree[fResponses]; ok {//move to 'responses'
			if responses, ok := responses.(yaml.MapSlice); ok {
				for _, response := range responses {	//each response is STATUS CODE: RESPONSE
					if statusCode, ok := response.Key.(int); ok {	//parse status code
						if response, ok := response.Value.(yaml.MapSlice); ok {	//parse response
							response := mapSliceToMap(response)
							//extract exit code
							if exitCode, ok := response[fExitCode]; ok {
								if exitCode, ok := exitCode.(int); ok {
									if body, ok := response[fBody]; ok {
										responses := make(rest.ParameterList)
										parseParameterList(body, responses)
										for mimeType, body := range responses {
											self.responses[exitCode] = rest.ResponseDescriptor{StatusCode: statusCode, Body: body, MimeType: mimeType}
										}
									} else {
										log.Fatalf("Response body is not specified for status code %v", statusCode)
									}
								} else {
									log.Fatalf("Exit code for status code %s has invalid value", statusCode)
								}
							} else {
								log.Fatalf("Exit code for status code %s is not specified", statusCode)
							}
						} else {
							log.Fatalf("Description of body for status code %s is invalid", statusCode)
						}
					} else {
						log.Fatalf("Incorrect HTTP status code: %v", response.Key)
					}
				}
			} else {
				log.Fatalf("Description of endpoint responses is not valid: %+v", responses)
			}
		} else {
			parameter := new(StringParameter)
			parameter.init()
			self.responses[0] = rest.ResponseDescriptor{StatusCode: 200, MimeType: "text/plain", Body: parameter}
		}
	} else {
		log.Fatalf("Unrecognized description of HTTP method: %+v", description)
	}
}

func (self *MethodDescriptor) QueryParameters() rest.ParameterList {
	return self.queryParameters
}

func (self *MethodDescriptor) RequestHeaders() rest.ParameterList {
	return self.reqHeaders
}

func (self *MethodDescriptor) Request() rest.ParameterList {
	return self.request
}

func (self *MethodDescriptor) Response() map[int]rest.ResponseDescriptor {
	return self.responses
}

//Represents endpoint described in RAML format
type Endpoint struct {
	uriParameters rest.ParameterList
	methods map[string]*MethodDescriptor
}

func (self *Endpoint) HasOption(name string) bool {
	return false
}

func parseParameterType(parameterType interface{}, fields map[string]interface{}) rest.Parameter{
	switch parameterType {
	case tString:
		result := new(StringParameter)
		result.parse(fields)
		return result
	case tBoolean:
		result := new(BooleanParameter)
		result.parse(fields)
		return result
	case tInteger:
		result := new(IntegerParameter)
		result.parse(fields)
		return result
	case tNumber:
		result := new(NumberParameter)
		result.parse(fields)
		return result
	case tFile:
		result := new(FileParameter)
		result.parse(fields)
		return result
	case tAny:
		result := new(AnyParameter)
		result.parse(fields)
		return result
	case tString + "[]":
		fields[fItems] = tString
		return parseParameterType(tArray, fields)
	case tInteger + "[]":
		fields[fItems] = tInteger
		return parseParameterType(tArray, fields)
	case tNumber + "[]":
		fields[fItems] = tNumber
		return parseParameterType(tArray, fields)
	case tBoolean + "[]":
		fields[fItems] = tBoolean
		return parseParameterType(tArray, fields)
	case tArray:
		result := new(ArrayParameter)
		result.parse(fields)
		return result
	default:
		log.Fatalf("Unsupported parameter type %s", parameterType)
		return nil //never happens
	}
}

func parseParameter(description interface{}) rest.Parameter {
	if tree, ok := description.(yaml.MapSlice); ok {
		//convert parameter fields into map
		fields := mapSliceToMap(tree)
		//parse parameter fields
		if parameterType, ok := fields[fType]; ok {
			return parseParameterType(parameterType, fields)
		} else {
			return parseParameterType(tAny, fields)
		}
	} else {
		log.Printf("Parameter has incorrect declaration: %+v", description)
		return &StringParameter{
			pattern:      nil,
			defaultValue: "",
			Parameter:    Parameter{hasDefaultValue: true, required: false},
		}
	}
}

func parseParameterList(input interface{}, output rest.ParameterList) {
	if tree, ok := input.(yaml.MapSlice); ok {
		for _, item := range tree { //iterate over parameters
			if name, ok := item.Key.(string); ok {
				log.Printf("Start parsing parameter %s", name)
				output[name] = parseParameter(item.Value)	//parse parameter
			}
		}
	} else {
		log.Printf("Unexpected tree type inside of parameter list: %+v", input)
	}
}

func (self *Endpoint) parseMethod(method string, description interface{}){
	m := new(MethodDescriptor)
	m.parse(description)
	self.methods[method] = m
}

func (self *Endpoint) parse(tree interface{}) {
	if t, ok := tree.(yaml.MapSlice); ok {
		for _, item := range t {
			switch item.Key {
			case "uriParameters":
				parseParameterList(item.Value, self.uriParameters)
			case "get":
				self.parseMethod(http.MethodGet, item.Value)
			case "post":
				self.parseMethod(http.MethodPost, item.Value)
			case "put":
				self.parseMethod(http.MethodPut, item.Value)
			case "delete":
				self.parseMethod(http.MethodDelete, item.Value)
			case "patch":
				self.parseMethod(http.MethodPatch, item.Value)
			case "head":
				self.parseMethod(http.MethodHead, item.Value)
			}
		}
	} else {
		log.Printf("Unexpected tree type inside of endpoint: %+v", tree)
	}
}

func (self *Endpoint) PathParameters() rest.ParameterList {
	return self.uriParameters
}

func (self *Endpoint) GetMethodDescriptor(method string) rest.HttpMethodDescriptor {
	return self.methods[method]
}

//Represents REST model restored from RAML markup
type Model struct {
	title string
	baseUri *url.URL
	endpoints map[string]rest.Endpoint
}

func (self *Model) newEndpoint(name string) *Endpoint {
	endpoint := &Endpoint{uriParameters: make(rest.ParameterList), methods: make(map[string]*MethodDescriptor)}
	self.endpoints[name] = endpoint
	return endpoint
}

func (self *Model) Endpoints() map[string]rest.Endpoint {
	return self.endpoints
}

func (self *Model) Name() string {
	return self.title
}

func (self *Model) parse(model yaml.MapSlice) {
	self.endpoints = make(map[string]rest.Endpoint)
	for _, item := range model {
		if field, ok := item.Key.(string); ok {
			switch field {
			case fTitle:
				self.title = item.Value.(string)
			case fBaseUri:
				if baseUri, err := url.Parse(item.Value.(string)); err == nil {
					self.baseUri = baseUri
				} else {
					log.Printf("Failed to parse base URI: %s", err.Error())
				}
			default:
				if strings.Index(field, "/") == 0 { //endpoint detected
					log.Printf("Start parsing endpoint %s", field)
					self.newEndpoint(field).parse(item.Value)
				}
			}
		}
	}
}

//Read RAML model
func (self *Model) ReadModel(input io.Reader) error {
	if content, err := ioutil.ReadAll(input); err == nil {
		parsedYAML := &yaml.MapSlice{}
		if err := yaml.Unmarshal(content, parsedYAML); err == nil {
			self.parse(*parsedYAML)
			return nil
		} else {
			return err
		}
	} else {
		return err
	}
}

//Read RAML model from file
func (self *Model) ReadModelFromFile(fileName string) error {
	if file, err := os.Open(fileName); err == nil {
		defer file.Close()
		return self.ReadModel(file)
	} else {
		return err
	}
}

func (self *Model) BaseUrl() *url.URL {
	return self.baseUri
}
