package hosting

import "io"

//Represents hosting server
type Server interface {
	io.Closer
	//Starts server in synchronous mode
	Run(async bool) error
}