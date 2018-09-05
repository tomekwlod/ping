package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Router
type router struct {
	*httprouter.Router
}

func (r *router) Get(path string, handler http.Handler) {
	r.GET(path, wrapHandler(handler))
}

func (r *router) Post(path string, handler http.Handler) {
	r.POST(path, wrapHandler(handler))
}

func (r *router) Put(path string, handler http.Handler) {
	r.PUT(path, wrapHandler(handler))
}

func (r *router) Delete(path string, handler http.Handler) {
	r.DELETE(path, wrapHandler(handler))
}

func (r *router) Options(path string, handler http.Handler) {
	r.OPTIONS(path, wrapHandler(handler))
}

func NewRouter() *router {
	return &router{httprouter.New()}
}
