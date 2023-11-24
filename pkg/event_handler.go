package webglue

import (
	"context"
	"encoding/json"
	"errors"

	sse "github.com/r3labs/sse/v2"
)

const (
	EventStreamName = "events"
)

type Event struct {
	Name   string
	server *sse.Server
}

func (event *Event) Emit(params any) (bool, error) {
	data, err := json.Marshal(struct {
		Event  string `json:"event"`
		Params any    `json:"params"`
	}{
		Event:  event.Name,
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
		}
	}

	return eventHandler, nil
}
