package raml

import (
	"testing"
	"strings"
	"math"
	"reflect"
	"github.com/sakno/go2rest/rest"
	"net/http"
)

func testFormatParameter(endpoint rest.Endpoint, t *testing.T){
	if parameter, ok := endpoint.PathParameters()["format"]; ok {
		//validate value
		if !parameter.Validate("abc") {
			t.Fatal("String validation test failed")
		}
		//check parameter type
		if _, ok := parameter.(rest.StringParameter); !ok {
			t.Fatal("Incorrect type of 'format' parameter")
		}
		//check 'required'
		if !parameter.Required(){
			t.Fatal("String parameter is required parameter")
		}
	} else {
		t.Fatal("'format' parameter missing")
	}
}

func testFlagParameter(endpoint rest.Endpoint, t *testing.T) {
	if parameter, ok := endpoint.PathParameters()["flag"]; ok {
		//validate value
		if !parameter.Validate(false) {
			t.Fatal("Boolean validation test failed")
		}
		//check parameter type
		if _, ok := parameter.(rest.BoolParameter); !ok {
			t.Fatal("Incorrect type of 'flag' parameter")
		}
		//check 'required'
		if parameter.Required(){
			t.Fatal("Bool parameter is optional parameter")
		}
	} else {
		t.Fatal("'flag' parameter missing")
	}
}

func testIntParameter(endpoint rest.Endpoint, t *testing.T) {
	if parameter, ok := endpoint.PathParameters()["int"]; ok {
		//validate value
		if !parameter.Validate(10) {
			t.Fatal("Integer validation test failed")
		}
		if parameter.Validate(201) {
			t.Fatal("Range validation failed")
		}
		//check parameter type
		if parameter, ok := parameter.(rest.IntegerParameter); ok {
			if !(parameter.HasDefaultValue() && parameter.DefaultValue() == 100) {
				t.Fatal("Default value test failed")
			}
		} else {
			t.Fatal("Incorrect type of 'int' parameter")
		}
		//check 'required'
		if !parameter.Required(){
			t.Fatal("Integer parameter is required parameter")
		}
	} else {
		t.Fatal("'int' parameter missing")
	}
}

func testNumParameter(endpoint rest.Endpoint, t *testing.T) {
	if parameter, ok := endpoint.PathParameters()["num"]; ok {
		//validate value
		if !parameter.Validate(41.0) {
			t.Fatal("Number validation test failed")
		}
		if parameter.Validate(43) {
			t.Fatal("Range validation failed")
		}
		//check parameter type
		if _, ok := parameter.(rest.NumberParameter); !ok {
			t.Fatal("Incorrect type of 'num' parameter")
		}
		//check 'required'
		if parameter.Required(){
			t.Fatal("Integer parameter is optional parameter")
		}
	} else {
		t.Fatal("'num' parameter missing")
	}
}

func testArray1(endpoint rest.Endpoint, t *testing.T) {
	if parameter, ok := endpoint.PathParameters()["array1"]; ok {
		//validate value
		if !parameter.Validate([]bool{true, false}) {
			t.Fatal("Array validation test failed")
		}
		//check parameter type
		if parameter, ok := parameter.(rest.ArrayParameter); ok {
			if _, ok := parameter.ElementType().(rest.BoolParameter); !ok {
				t.Fatal("Incorrect element type of 'array1' parameter")
			}
		} else {
			t.Fatal("Incorrect type of 'array1' parameter")
		}
	} else {
		t.Fatal("'array1' parameter missing")
	}
}

func testArray2(endpoint rest.Endpoint, t *testing.T) {
	if parameter, ok := endpoint.PathParameters()["array2"]; ok {
		//validate value
		if !parameter.Validate([]string{"ab", "cd"}) {
			t.Fatal("Array validation test failed")
		}
		//check parameter type
		if parameter, ok := parameter.(rest.ArrayParameter); ok {
			if _, ok := parameter.ElementType().(rest.StringParameter); !ok {
				t.Fatal("Incorrect element type of 'array2' parameter")
			}
		} else {
			t.Fatal("Incorrect type of 'array2' parameter")
		}
	} else {
		t.Fatal("'array2' parameter missing")
	}
}

func testArray3(endpoint rest.Endpoint, t *testing.T) {
	if parameter, ok := endpoint.PathParameters()["array3"]; ok {
		//validate value
		if !parameter.Validate([]int{2, 3}) {
			t.Fatal("Array validation test failed")
		}
		//check parameter type
		if parameter, ok := parameter.(rest.ArrayParameter); ok {
			if _, ok := parameter.ElementType().(rest.IntegerParameter); !ok {
				t.Fatal("Incorrect element type of 'array3' parameter")
			}
		} else {
			t.Fatal("Incorrect type of 'array3' parameter")
		}
	} else {
		t.Fatal("Missing 'array3' parameter ")
	}
}

func testGetMethod(endpoint rest.Endpoint, t *testing.T) {
	if method := endpoint.GetMethodDescriptor(http.MethodGet); method != nil {
		//test headers
		if xDeptHeader, ok := method.RequestHeaders()["X-Dept"]; ok {
			if xDeptHeader, ok := xDeptHeader.(rest.IntegerParameter); ok {
				if xDeptHeader.Required() {
					t.Fatalf("X-Dept should not be required header")
				}
			} else {
				t.Fatalf("X-Dept has incorrect type")
			}
		} else {
			t.Fatalf("Missing X-Dept header")
		}
		//test request body
		if body, ok := method.Request()["application/octet-stream"]; ok {
			if _, ok := body.(rest.FileParameter); !ok {
				t.Fatalf("Body has incorrect type")
			}
		}
		//test response body
		for exitCode, response := range method.Response() {
			switch exitCode {
			case 0:
				if response.StatusCode != 200{
					t.Fatalf("Incorrect status code %v", response.StatusCode)
				}
				if response.MimeType != "application/json" {
					t.Fatalf("Incorrect MIME type %v", response.MimeType)
				}
				if _, ok := response.Body.(rest.NumberParameter); !ok {
					t.Fatalf("Incorrect type of parameter")
				}
			case -1:
				if response.StatusCode != 404{
					t.Fatalf("Incorrect status code %v", response.StatusCode)
				}
				if response.MimeType != "text/plain" {
					t.Fatalf("Incorrect MIME type %v", response.MimeType)
				}
				if _, ok := response.Body.(rest.StringParameter); !ok {
					t.Fatalf("Incorrect type of parameter")
				}
			default:
				t.Fatalf("Unexpected exit code %v", exitCode)
			}
		}
	} else {
		t.Fatalf("GET handler is not presented")
	}
}

func TestReadRamlModelFromFile(t *testing.T) {
	model := new(Model)
	err := model.ReadModelFromFile("test-raml-model.raml")
	if err != nil {
		t.Fatal(err)
	}
	if endpoint, ok := model.Endpoints()["/freemem/{format}"]; ok {
		testFormatParameter(endpoint, t)
		testFlagParameter(endpoint, t)
		testIntParameter(endpoint, t)
		testNumParameter(endpoint, t)
		testArray1(endpoint, t)
		testArray2(endpoint, t)
		testArray3(endpoint, t)
		testGetMethod(endpoint, t)
	} else {
		t.Fatal("Invalid number of endpoints")
	}
}

func TestIntegerDeserialization(t *testing.T) {
	parameter := new(IntegerParameter)
	parameter.init()
	if value, err := parameter.ReadValue(strings.NewReader("42"), rest.FormatText); err != nil || value.(int64) != 42 {
		t.Fatal("Incorrect deserialization of string")
	}
	if value, err := parameter.ReadValue(strings.NewReader("42"), rest.FormatJSON); err != nil || value.(int64) != 42 {
		t.Fatal("Incorrect deserialization of string")
	}
}

func TestStringDeserialization(t *testing.T) {
	parameter := new(StringParameter)
	parameter.init()
	if value, err := parameter.ReadValue(strings.NewReader("Hello, world!"), rest.FormatText); err != nil || value.(string) != "Hello, world!" {
		t.Fatal("Incorrect deserialization of string")
	}
	if value, err := parameter.ReadValue(strings.NewReader("\"Hello, world!\""), rest.FormatJSON); err != nil || value.(string) != "Hello, world!" {
		t.Fatal("Incorrect deserialization of string")
	}
}

func TestNumberDeserialization(t *testing.T) {
	parameter := new(NumberParameter)
	parameter.init()
	if value, err := parameter.ReadValue(strings.NewReader("42.4"), rest.FormatText); err != nil || value.(float64) != 42.4 {
		t.Fatal("Incorrect deserialization of string")
	}
	if value, err := parameter.ReadValue(strings.NewReader("42.4"), rest.FormatJSON); err != nil || value.(float64) != 42.4 {
		t.Fatal("Incorrect deserialization of string")
	}
}

func TestBoolDeserialization(t *testing.T) {
	parameter := new(BooleanParameter)
	if value, err := parameter.ReadValue(strings.NewReader("true"), rest.FormatText); err != nil || value.(bool) != true {
		t.Fatal("Incorrect deserialization of string")
	}
	if value, err := parameter.ReadValue(strings.NewReader("true"), rest.FormatJSON); err != nil || value.(bool) != true {
		t.Fatal("Incorrect deserialization of string")
	}
}

func TestArrayDeserialization(t *testing.T) {
	parameter := new(ArrayParameter)
	parameter.init()
	parameter.elementType = &IntegerParameter{ minimum: 0, maximum: math.MaxInt64 }
	if value, err := parameter.ReadValue(strings.NewReader("[23, 45]"), rest.FormatJSON); err == nil {
		for _, element := range value.([]interface{}) {
			if !parameter.elementType.Validate(element) {
				t.Fatal("Incorrent array element ", reflect.ValueOf(element).Kind())
			}
		}
	} else {
		t.Fatal("Incorrect deserialization of string: ", err.Error())
	}
}




