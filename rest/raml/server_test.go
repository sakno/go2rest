package raml

import (
	"testing"
	"../../rest"
	"net/http"
	"io/ioutil"
	_ "time"
)

const(
	serverPort = ":9535"
	serverAddress = "http://localhost" + serverPort
)

func TestServer(test *testing.T) {
	model := new(Model)
	if err := model.ReadModelFromFile("server-api.raml"); err != nil {
		test.Fatal(err)
	}
	server := rest.StandaloneServer{Model: model}
	server.Addr = ":9535"
	server.Run(true)
	defer server.Close()
	client := new(http.Client)
	if response, err := client.Get(serverAddress + "/echo1/bla_bla"); err == nil {
		defer response.Body.Close()
		if message, err := ioutil.ReadAll(response.Body); err == nil {
			message := string(message)
			if message != "bla_bla\n" {
				test.Fatalf("Unexpected result: %s", message)
			}
		}
	} else {
		test.Fatalf("Failed to GET. Error: %s", err.Error())
	}
	//time.Sleep(time.Hour)
	server.Shutdown(nil)
}