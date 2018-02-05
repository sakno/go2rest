**go2rest** exposes any existing command-line program as REST service.

# Features
* Cross-platform. Can run on every platform supported by [Go](https://golang.org/) language compiler
* Lightweight. Just one executable file. All dependencies are statically linked
* RAML compatible. You can describe REST API for any command-line program using [RAML 1.0](https://github.com/raml-org/raml-spec/blob/master/versions/raml-10/raml-10.md) markup language based on YAML. The same RAML file you can use to generate documentation for your REST API using [ReadTheDocs](https://solidity.readthedocs.io) or any other toolchain for documentation
* Supports for file transfer through REST API that can be used as input argument for command-line program
* Mapping between process exit statuses and HTTP statuses
* Supported JSON types: number, string, boolean, array
* FastCGI support

# How to build
1. Install [Go](https://golang.org/dl/) compiler according to your Operating System
1. Run `go get github.com/sakno/go2rest`
1. Run `go build github.com/sakno/go2rest`
1. Grab compiled executable from your current directory

# How to use
1. Describe REST API in the form of RAML file with `.raml` extension. Read [Wiki](https://github.com/sakno/go2rest/wiki) for detailed guide of how to write correct RAML file; or look at [RAML file](https://github.com/sakno/go2rest/blob/master/rest/raml/test-raml-model.raml) used for tests.
1. Run `go2rest [--port <port>] <path/to/file.raml>`. Now REST service is hosted on specified port

If you want to run service in FastCGI mode then omit port number like this: `go2rest <path/to/file.raml>`

RAML file should have `.raml` extension. **go2rest** uses file extension to determine correct model parser because, in future, the program may support 
another model formats such as OpenAPI.

# Room for improvements
Internal representation of REST model does not rely on RAML directly. It is possible to implement any descriptive model of API. For example, [OpenAPI Spec](https://www.openapis.org/) used by [Swagger](https://swagger.io/) toolchain.

JSON Object (dictionary) data type is not supported at this moment but can be implemented easily.

Some of RAML features are not supported:
1. Multipart form data
1. Includes, Libraries, Overlays, and Extensions
