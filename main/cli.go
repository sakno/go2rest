package main

import (
	"flag"
	"os"
	"log"
	"path"
	"../rest"
	"../rest/raml"
	"github.com/sakno/go2rest/hosting"
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
		rest.Addr = address
		rest.KeyFile = keyFile
		rest.CertFile = certFile
		server = rest
		log.Printf("Starting standalone server at %s", address)
	}
	server.Run(false)
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
	var address, certFile, keyFile string
	flags.StringVar(&address, "address", ":http", "TCP address to listen on")
	flags.StringVar(&certFile, "certFile", "", "Absolute path to certificate file")
	flags.StringVar(&keyFile, "keyFile", "", "Absolute path to key file")
	flags.Parse(os.Args[1:])
	run(flag.Arg(0), address, certFile, keyFile)
}