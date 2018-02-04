package main

import (
	"flag"
	"os"
	"log"
	"path"
	"github.com/sakno/go2rest/rest"
	"github.com/sakno/go2rest/rest/raml"
	"github.com/sakno/go2rest/hosting"
	"fmt"
)

func startRestService(model rest.Model, address, certFile, keyFile string) {
	var server hosting.Server
	if len(address) == 0 {
		fcgi := new(rest.FastCGI)
		fcgi.Model = model
		server = fcgi
		log.Printf("Starting FastCGI process")
	} else {
		rest := new(rest.StandaloneServer)
		rest.Addr = ":" + address
		rest.KeyFile = keyFile
		rest.CertFile = certFile
		rest.Model = model
		server = rest
		log.Printf("Starting standalone server at %s", address)
	}
	err := server.Run(false)
	log.Printf("Unable to run server. Reason: %s", err.Error())
}

func run(fileName, address, certFile, keyFile string) {
	switch extension := path.Ext(fileName); extension {
	case ".raml":
		model := new(raml.Model)
		if err := model.ReadModelFromFile(fileName); err == nil {
			startRestService(model, address, certFile, keyFile)
		} else {
			log.Fatalf("Failed to read RAML file %s. Error: %s", fileName, err.Error())
		}
	default:
		log.Fatalf("Unsupported API description format: %s", extension)
	}
}

func main() {
	flags := flag.NewFlagSet("rest2go", flag.ExitOnError)
	flags.SetOutput(os.Stdout)
	var port, certFile, keyFile string
	flags.StringVar(&port, "port", "http", "TCP port to listen on")
	flags.StringVar(&certFile, "cert", "", "Absolute path to certificate file")
	flags.StringVar(&keyFile, "key", "", "Absolute path to key file")
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stdout, "go2rest [-port port-number] [-cert path/to/x509/cert] [-key path/to/cert/key] <path/to/model>")
		flags.PrintDefaults()
	} else {
		flags.Parse(os.Args[1:])
		run(flags.Arg(0), port, certFile, keyFile)
	}
}