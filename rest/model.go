package rest

import (
	"io"
	"github.com/sakno/go2rest/cmdexec"
	"strings"
	"net/url"
)

type ParameterValueFormat int8

//Formats
const (
	FormatText   = ParameterValueFormat(0)
	FormatJSON   = ParameterValueFormat(1)
	FormatBinary = ParameterValueFormat(2)
	FormatXML    = ParameterValueFormat(3)
)

type UnsupportedParameterValueFormat struct {

}

func (UnsupportedParameterValueFormat) Error() string {
	return "Unsupported parameter value format"
}

func ExtractQueryParameters(url *url.URL, separator string) map[string]string {
	result := make(map[string]string, len(url.Query()))
	for name, values := range url.Query() {
		result[name] = strings.Join(values, separator)
	}
	return result
}

func GetFormatByMIME(mediaType string) ParameterValueFormat {
	//handle well-known types
	switch mediaType {
	case "application/xml", "text/xml":
		return FormatXML
	case "application/json":
		return FormatJSON
	case "application/octet-stream":
		return FormatBinary
	case "text/plain", "application/javascript", "text/javascript", "application/rtf", "application/sql":
		return FormatText
	}
	//more generic algorithm
	switch parts := strings.FieldsFunc(mediaType, func(r rune) bool { return r == '/' }); len(parts) {
	case 2:
		switch parts[0] {
		case "text":
			return FormatText
		case "audio", "video", "image":
			return FormatBinary
		}
		fallthrough
	default:
		return FormatBinary
	}
}

//Represents model element
type ModelElement interface {
	//Indicates that custom option is defined for model element
	HasOption(directive string) bool
}

//Represents parameter in model
//this interface can be used to represent variant parameter type
type Parameter interface{
	ModelElement
	Required() bool	//parameter is required to be presented
	HasDefaultValue() bool	//parameter has default value
	Validate(value interface{}) bool	//validate value of parameter
	ReadValue(value io.Reader, format ParameterValueFormat) (interface{}, error)
}

//Represents list of parameters
type ParameterList map[string]Parameter

type IntegerParameter interface{
	Parameter
	DefaultValue() int64	//default value of parameter
}

type NumberParameter interface {
	Parameter
	DefaultValue() float64
}

//Represents parameter of type String
type StringParameter interface {
	Parameter
	DefaultValue() string
}

//Represents parameter of type Boolean
type BoolParameter interface {
	Parameter
	DefaultValue() bool
}

//Represents file content as a parameter
// value of this parameter can be represented as BASE64 string in case of plain/text format
// value of this parameter can be represented as quoted BASE64 string in case of application/json format
// value of this parameter can be represented as raw set of bytes in case of application/octet-stream format
type FileParameter interface {
	Parameter
	//Indicates that file should be saved into temporary file. Otherwise, it will be recorded into memory
	Persistent() bool
}

type ArrayParameter interface {
	Parameter
	ElementType() Parameter
}

type ObjectParameter interface {
	Parameter
	Fields() ParameterList
}

//Describes response associated with exit code
type ResponseDescriptor struct {
	Body Parameter	//response parameter description
	StatusCode int	//HTTP status code
	MimeType string	//MIME type
}

//Describes HTTP method
type HttpMethodDescriptor interface {
	ModelElement
	Executor() cmdexec.CommandExecutor
	QueryParameters() ParameterList
	RequestHeaders() ParameterList
	Request() ParameterList               //definition of body bounded to MIME types
	Response() map[int]ResponseDescriptor //mapping between exit code of process and response
}

//Describes single endpoint
type Endpoint interface {
	ModelElement
	PathParameters() ParameterList	//path parameters
	GetMethodDescriptor(method string) HttpMethodDescriptor
}

//REST model describes command execution
type Model interface {
	Endpoints() map[string]Endpoint	//set of endpoints declared in the model
	Name() string	//name of model
	//address of service, may be nil if model doesn't support definition of preferred address
	BaseUrl() *url.URL
}

