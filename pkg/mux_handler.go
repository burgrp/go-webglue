package webglue

import (
	"embed"
	"net/http"
)

type ApiFunction func(request []byte) ([]byte, error)

type Module struct {
	Name      string
	Resources embed.FS
}

type SessionFactory func(id string) any

type Options struct {
	Modules        []Module
	IndexHtml      string
	SessionFactory SessionFactory
}

func NewHandler(options Options) (http.Handler, error) {

	// apiMarshaler, err := newApiMarshaler(options)
	// if err != nil {
	// 	return nil, err
	// }

	staticHandler, err := newStaticHandler(options)
	if err != nil {
		return nil, err
	}

	apiHandler, err := newMessageHandler(options)
	if err != nil {
		return nil, err
	}

	mux := http.ServeMux{}

	mux.Handle("/", staticHandler)
	mux.Handle("/api", apiHandler)

	return &mux, nil

}
