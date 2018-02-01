package main

import (
	"flag"
	"os"
	"log"
	"path"
	"../rest"
	"../rest/raml"
)

func startRestService(model rest.Model, address string) {

}

func run(fileName, address string) {
	switch extension := path.Ext(fileName); extension {
	case ".raml":
		model := new(raml.Model)
		if err := model.ReadModelFromFile(fileName); err == nil {
			startRestService(model, address)
		} else {
			log.Fatalf("Failed to read RAML file %s. Error: %s", fileName, err.Error())
		}
	default:
		log.Fatalf("Unsupported API description format: %s", extension)
	}
}

func main() {
	flags := flag.NewFlagSet("rest2go", flag.ExitOnError)
	var address string
	flags.StringVar(&address, "address", ":http", "TCP address to listen on")
	flags.Parse(os.Args[1:])
	run(flag.Arg(0), address)
}