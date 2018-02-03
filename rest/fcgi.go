package rest

import (
	"net/http/fcgi"
	"github.com/gorilla/mux"
	"errors"
)

type FastCGI struct {
	Model Model
}

func (self *FastCGI) Close() error {
	return nil
}

func (self *FastCGI) Run(async bool) error {
	router := mux.NewRouter()
	prepareRouter(router, self.Model)
	if async {
		return errors.New("asynchronous launch is not supported")
	} else {
		return fcgi.Serve(nil, router)
	}
}



