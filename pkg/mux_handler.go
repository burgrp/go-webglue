package webglue

import (
	"context"
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

type SessionFactory func(id string) any

type Options struct {
	Modules        []Module
	IndexHtml      string
	SessionFactory SessionFactory
	Context        context.Context
}

func NewHandler(options Options) (http.Handler, error) {

	var ctx context.Context
	if options.Context != nil {
		ctx = options.Context
	} else {
		ctx = context.Background()
	}

	allModules := append([]Module{
		newCoreModule(&options),
	}, options.Modules...)

	staticHandler, err := newStaticHandler(ctx, &options, allModules)
	if err != nil {
		return nil, err
	}

	apiHandler, err := newApiHandler(ctx, &options, allModules)
	if err != nil {
		return nil, err
	}

	eventHandler, err := newEventHandler(ctx, &options, allModules)
	if err != nil {
		return nil, err
	}

	mux := http.ServeMux{}

	mux.Handle("/", staticHandler)
	mux.Handle("/api/", apiHandler)
	mux.Handle("/events", eventHandler)

	return &mux, nil
}
