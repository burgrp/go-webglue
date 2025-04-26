package webglue

import (
	"encoding/json"

	sse "github.com/r3labs/sse/v2"
)

const (
	EventStreamName = "webglue"
)

type Event struct {
	Module  string
	Name    string
	servers []*sse.Server
}

func NewEvent(name string) *Event {
	return &Event{
		Name:    name,
		servers: make([]*sse.Server, 0),
	}
}

func (event *Event) Emit(params ...any) {
	data, err := event.marshall(params...)
	if err != nil {
		panic(err)
	}

	for _, server := range event.servers {
		server.TryPublish(EventStreamName, &sse.Event{
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

func newEventHandler(modules []*Module) (*sse.Server, error) {
	eventHandler := sse.New()
	eventHandler.AutoReplay = false
	eventHandler.CreateStream(EventStreamName)

	for _, module := range modules {
		for _, event := range module.Events {
			event.servers = append(event.servers, eventHandler)
			event.Module = module.Name
		}
	}

	return eventHandler, nil
}
