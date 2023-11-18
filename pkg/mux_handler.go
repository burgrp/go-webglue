package webglue

import (
	"embed"
	"net/http"
)

type ApiFunction func(request []byte) ([]byte, error)

type Module struct {
	Name      string
	Resources embed.FS
	Api       interface{}
}

type Options struct {
	Modules   []Module
	IndexHtml string
}

func NewHandler(options Options) (http.Handler, error) {

	// apiDispatcher, err := newApiDispatcher(options)
	// if err != nil {
	// 	return nil, err
	// }

	staticHandler, err := newStaticHandler(options)
	if err != nil {
		return nil, err
	}

	messageHandler, err := newMessageHandler(options)
	if err != nil {
		return nil, err
	}

	mux := http.ServeMux{}

	mux.Handle("/", staticHandler)
	mux.Handle("/msg", messageHandler)

	return &mux, nil

}
