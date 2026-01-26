package webglue

import (
	"encoding/json"

	sse "github.com/r3labs/sse/v2"
)

const (
	// EventStreamName is the SSE stream name used for all webglue events.
	EventStreamName = "webglue"
)

// Event represents a server-to-client event that can be emitted to all connected clients.
// Events use Server-Sent Events (SSE) for real-time communication.
type Event struct {
	Module  string          // Module name (auto-populated by framework)
	Name    string          // Event name
	servers []*sse.Server   // SSE servers broadcasting this event
}

// NewEvent creates a new event with the given name.
// The event must be registered with a module to be functional.
func NewEvent(name string) *Event {
	return &Event{
		Name:    name,
		servers: make([]*sse.Server, 0),
	}
}

// Emit broadcasts the event with the given parameters to all connected clients.
// Parameters are marshaled to JSON and sent via Server-Sent Events.
// If marshaling fails, this method panics.
func (event *Event) Emit(params ...any) {
	data, err := event.marshall(params...)
	if err != nil {
		panic(err)
	}

	// Broadcast to all connected SSE servers
	for _, server := range event.servers {
		server.TryPublish(EventStreamName, &sse.Event{
			Data: data,
		})
	}
}

// marshall converts the event and its parameters to JSON format.
// The resulting JSON includes the module name, event name, and parameters array.
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

// newEventHandler creates and configures an SSE server for event streaming.
// It registers all events from all modules with the SSE server.
func newEventHandler(modules []*Module) (*sse.Server, error) {
	eventHandler := sse.New()
	eventHandler.AutoReplay = false
	eventHandler.CreateStream(EventStreamName)

	// Register all events with the SSE server
	for _, module := range modules {
		for _, event := range module.Events {
			event.servers = append(event.servers, eventHandler)
			event.Module = module.Name
		}
	}

	return eventHandler, nil
}
