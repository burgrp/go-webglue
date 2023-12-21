package webglue

import (
	"context"
	"encoding/json"

	sse "github.com/r3labs/sse/v2"
)

const (
	EventStreamName = "webglue"
)

type Event struct {
	Module string
	Name   string
	server *sse.Server
}

func NewEvent(name string) *Event {
	return &Event{
		Name: name,
	}
}

func (event *Event) MustEmit(params ...any) {
	data, err := event.marshall(params...)
	if err != nil {
		panic(err)
	}

	if event.server == nil {
		panic("event not bound to server")
	}

	event.server.Publish(EventStreamName, &sse.Event{
		Data: data,
	})
}

func (event *Event) TryEmit(params ...any) {
	data, err := event.marshall(params...)
	if err == nil && event.server != nil {
		event.server.TryPublish(EventStreamName, &sse.Event{
			Data: data,
		})
	}
}

func (event *Event) marshall(params ...any) ([]byte, error) {
	return json.Marshal(struct {
		Module string `json:"module"`
		Name   string `json:"name"`
		Params any    `json:"params"`
	}{
		Module: event.Module,
		Name:   event.Name,
		Params: params,
	})
}

func newEventHandler(ctx context.Context, options *Options, allModules []Module) (*sse.Server, error) {
	eventHandler := sse.New()
	eventHandler.AutoReplay = false
	eventHandler.CreateStream(EventStreamName)

	for _, module := range options.Modules {
		for _, event := range module.Events {
			event.server = eventHandler
			event.Module = module.Name
		}
	}

	return eventHandler, nil
}
