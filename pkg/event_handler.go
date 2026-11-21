package webglue

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	// EventStreamName is the SSE stream name used for all webglue events.
	EventStreamName   = "webglue"
	eventQueueSize    = 64
	eventWriteTimeout = 5 * time.Second
)

// Event represents a server-to-client event that can be emitted to all connected clients.
// Events use Server-Sent Events (SSE) for real-time communication.
type Event struct {
	Module  string // Module name (auto-populated by framework)
	Name    string // Event name
	mu      sync.RWMutex
	servers []*eventHandler
}

// NewEvent creates a new event with the given name.
// The event must be registered with a module to be functional.
func NewEvent(name string) *Event {
	return &Event{
		Name:    name,
		servers: make([]*eventHandler, 0),
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

	event.mu.RLock()
	defer event.mu.RUnlock()
	for _, server := range event.servers {
		server.publish(data)
	}
}

// marshall converts the event and its parameters to JSON format.
// The resulting JSON includes the module name, event name, and parameters array.
func (event *Event) marshall(params ...any) ([]byte, error) {
	event.mu.RLock()
	message := struct {
		Module string `json:"module"`
		Name   string `json:"name"`
		Params any    `json:"params"`
	}{
		Module: event.Module,
		Name:   event.Name,
		Params: params,
	}
	event.mu.RUnlock()
	return json.Marshal(message)
}

// newEventHandler creates and configures an SSE server for event streaming.
// It registers all events from all modules with the SSE server.
func newEventHandler(modules []*Module) (*eventHandler, error) {
	handler := &eventHandler{clients: make(map[*eventClient]struct{})}

	// Register all events with the SSE server
	for _, module := range modules {
		for _, event := range module.Events {
			event.mu.Lock()
			event.servers = append(event.servers, handler)
			event.Module = module.Name
			event.mu.Unlock()
		}
	}

	return handler, nil
}

type eventClient struct {
	events   chan []byte
	done     chan struct{}
	overflow bool
}

type eventHandler struct {
	mu      sync.Mutex
	clients map[*eventClient]struct{}
	closed  bool
}

func (handler *eventHandler) subscribe() *eventClient {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if handler.closed {
		return nil
	}
	client := &eventClient{events: make(chan []byte, eventQueueSize), done: make(chan struct{})}
	handler.clients[client] = struct{}{}
	return client
}

// The caller holds handler.mu.
func (handler *eventHandler) removeClient(client *eventClient) {
	if _, exists := handler.clients[client]; exists {
		delete(handler.clients, client)
		close(client.done)
	}
}

func (handler *eventHandler) unsubscribe(client *eventClient) {
	handler.mu.Lock()
	handler.removeClient(client)
	overflow := client.overflow
	handler.mu.Unlock()
	if overflow {
		log.Print("webglue: SSE client disconnected because its event queue is full")
	}
}

func (handler *eventHandler) publish(data []byte) {
	frame := append([]byte("id: \ndata: "), data...)
	frame = append(frame, '\n', '\n')
	handler.mu.Lock()
	defer handler.mu.Unlock()
	for client := range handler.clients {
		select {
		case client.events <- frame:
		default:
			client.overflow = true
			handler.removeClient(client)
		}
	}
}

func (handler *eventHandler) Close() {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	handler.closed = true
	for client := range handler.clients {
		handler.removeClient(client)
	}
}

func (handler *eventHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	stream := request.URL.Query().Get("stream")
	if stream == "" {
		http.Error(writer, "Please specify a stream!", http.StatusBadRequest)
		return
	}
	if stream != EventStreamName {
		http.Error(writer, "Stream not found!", http.StatusBadRequest)
		return
	}
	if id := request.Header.Get("Last-Event-ID"); id != "" {
		if _, err := strconv.Atoi(id); err != nil {
			http.Error(writer, "Last-Event-ID must be a number!", http.StatusBadRequest)
			return
		}
	}
	if _, ok := writer.(http.Flusher); !ok {
		http.Error(writer, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	controller := http.NewResponseController(writer)
	deadlines := true
	if err := controller.SetWriteDeadline(time.Time{}); err != nil {
		if !errors.Is(err, http.ErrNotSupported) {
			log.Printf("webglue: SSE write deadline failed: %v", err)
			http.Error(writer, "Cannot configure SSE write deadline.", http.StatusInternalServerError)
			return
		}
		deadlines = false
		log.Print("webglue: SSE writer has no write-deadline support; response wrappers should expose Unwrap")
	}
	client := handler.subscribe()
	if client == nil {
		http.Error(writer, "Event stream is closed.", http.StatusServiceUnavailable)
		return
	}
	defer handler.unsubscribe(client)
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	write := func(frame []byte) error {
		if deadlines {
			if err := controller.SetWriteDeadline(time.Now().Add(eventWriteTimeout)); err != nil {
				return err
			}
		}
		if len(frame) != 0 {
			written, err := writer.Write(frame)
			if err != nil {
				return err
			}
			if written != len(frame) {
				return io.ErrShortWrite
			}
		}
		if err := controller.Flush(); err != nil {
			return err
		}
		if deadlines {
			return controller.SetWriteDeadline(time.Time{})
		}
		return nil
	}
	writer.WriteHeader(http.StatusOK)
	if err := write(nil); err != nil {
		log.Printf("webglue: SSE connection flush failed: %v", err)
		return
	}
	for {
		select {
		case <-request.Context().Done():
			return
		case <-client.done:
			return
		case frame := <-client.events:
			select {
			case <-client.done:
				return
			default:
			}
			if err := write(frame); err != nil {
				if request.Context().Err() == nil {
					log.Printf("webglue: SSE write failed: %v", err)
				}
				return
			}
		}
	}
}
