package webglue

import (
	"context"
	"encoding/json"
	"errors"

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

func (event *Event) Emit(params ...any) (bool, error) {
	data, err := json.Marshal(struct {
		Module string `json:"module"`
		Name   string `json:"name"`
		Params any    `json:"params"`
	}{
		Module: event.Module,
		Name:   event.Name,
		Params: params,
	})
	if err != nil {
		return false, err
	}

	if event.server == nil {
		return false, errors.New("event not bound to server")
	}

	return event.server.TryPublish(EventStreamName, &sse.Event{
		Data: data,
	}), nil

}

func NewEvent(name string) *Event {
	return &Event{
		Name: name,
	}
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
