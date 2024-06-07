package webglue

import (
	"embed"
	"net/http"
)

type ApiFunction func(request []byte) ([]byte, error)

type Module struct {
	Name      string
	Resources *embed.FS
	Events    []*Event
	Api       any
}

type Options struct {
	Modules   []*Module
	IndexHtml string
}

func NewHandler(options Options) (*http.ServeMux, error) {

	allModules := append([]*Module{
		newCoreModule(&options),
	}, options.Modules...)

	staticHandler, err := newStaticHandler(allModules, options.IndexHtml)
	if err != nil {
		return nil, err
	}

	apiHandler, err := newApiHandler(allModules)
	if err != nil {
		return nil, err
	}

	eventHandler, err := newEventHandler(allModules)
	if err != nil {
		return nil, err
	}

	mux := http.ServeMux{}

	mux.Handle("/", staticHandler)
	mux.Handle("/api/", apiHandler)
	mux.Handle("/events", eventHandler)

	return &mux, nil
}
