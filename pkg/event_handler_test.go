package webglue

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	event := NewEvent("testEvent")

	if event.Name != "testEvent" {
		t.Errorf("Expected event name 'testEvent', got '%s'", event.Name)
	}

	if event.servers == nil {
		t.Error("Expected servers slice to be initialized")
	}

	if len(event.servers) != 0 {
		t.Errorf("Expected empty servers slice, got length %d", len(event.servers))
	}
}

func TestEventMarshall(t *testing.T) {
	event := &Event{
		Module: "test",
		Name:   "dataUpdated",
	}

	data, err := event.marshall("param1", 42, true)
	if err != nil {
		t.Fatalf("Failed to marshall event: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result["module"] != "test" {
		t.Errorf("Expected module 'test', got '%v'", result["module"])
	}

	if result["name"] != "dataUpdated" {
		t.Errorf("Expected name 'dataUpdated', got '%v'", result["name"])
	}

	params, ok := result["params"].([]any)
	if !ok {
		t.Fatalf("Expected params to be array, got %T", result["params"])
	}

	if len(params) != 3 {
		t.Errorf("Expected 3 params, got %d", len(params))
	}

	if params[0] != "param1" || params[1].(float64) != 42 || params[2] != true {
		t.Errorf("Unexpected params: %v", params)
	}
}

func TestEventEmit(t *testing.T) {
	event1 := NewEvent("event1")
	event2 := NewEvent("event2")

	modules := []*Module{
		{
			Name:   "module1",
			Events: []*Event{event1},
		},
		{
			Name:   "module2",
			Events: []*Event{event2},
		},
	}

	handler, err := newEventHandler(modules)
	if err != nil {
		t.Fatalf("Failed to create event handler: %v", err)
	}

	// Check that events are registered with handler
	if event1.Module != "module1" {
		t.Errorf("Expected event1 module 'module1', got '%s'", event1.Module)
	}

	if event2.Module != "module2" {
		t.Errorf("Expected event2 module 'module2', got '%s'", event2.Module)
	}

	if len(event1.servers) != 1 {
		t.Errorf("Expected event1 to have 1 server, got %d", len(event1.servers))
	}

	// Test SSE connection
	req := httptest.NewRequest("GET", "/events?stream=webglue", nil)
	w := httptest.NewRecorder()

	// Start serving in goroutine
	done := make(chan bool)
	go func() {
		handler.ServeHTTP(w, req)
		done <- true
	}()

	// Give it time to connect
	time.Sleep(50 * time.Millisecond)

	// Emit event
	event1.Emit("test", 123)

	// Give it time to receive
	time.Sleep(50 * time.Millisecond)

	// Close connection
	handler.Close()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("Handler did not complete")
	}

	body := w.Body.String()

	// Check for SSE format
	if !strings.Contains(body, "data:") {
		t.Error("Expected SSE format with 'data:' prefix")
	}
}

func TestEventEmitMultipleParams(t *testing.T) {
	event := NewEvent("multiParam")

	data, err := event.marshall(1, "two", 3.0, true, nil)
	if err != nil {
		t.Fatalf("Failed to marshall: %v", err)
	}

	var result map[string]any
	json.Unmarshal(data, &result)

	params := result["params"].([]any)
	if len(params) != 5 {
		t.Errorf("Expected 5 params, got %d", len(params))
	}
}

func TestEventEmitNoParams(t *testing.T) {
	event := NewEvent("noParam")
	event.Module = "test"

	data, err := event.marshall()
	if err != nil {
		t.Fatalf("Failed to marshall: %v", err)
	}

	var result map[string]any
	json.Unmarshal(data, &result)

	// When no params, it's an empty slice in JSON (marshals as [])
	params, ok := result["params"].([]any)
	if !ok {
		// If it's null, that's also acceptable
		if result["params"] != nil {
			t.Errorf("Expected params to be array or nil, got %T", result["params"])
		}
	} else if len(params) != 0 {
		t.Errorf("Expected 0 params, got %d", len(params))
	}
}

func TestEventHandlerStreamName(t *testing.T) {
	if EventStreamName != "webglue" {
		t.Errorf("Expected stream name 'webglue', got '%s'", EventStreamName)
	}
}

func TestEventHandlerMultipleModules(t *testing.T) {
	event1 := NewEvent("event1")
	event2 := NewEvent("event2")
	event3 := NewEvent("event3")

	modules := []*Module{
		{
			Name:   "module1",
			Events: []*Event{event1, event2},
		},
		{
			Name:   "module2",
			Events: []*Event{event3},
		},
	}

	handler, err := newEventHandler(modules)
	if err != nil {
		t.Fatalf("Failed to create event handler: %v", err)
	}

	// All events should have the same server
	if event1.servers[0] != event2.servers[0] || event2.servers[0] != event3.servers[0] {
		t.Error("Expected all events to share the same server")
	}

	// Check module names are set correctly
	if event1.Module != "module1" || event2.Module != "module1" {
		t.Error("Expected events to have correct module name")
	}

	if event3.Module != "module2" {
		t.Error("Expected event3 to have module name 'module2'")
	}

	if handler.AutoReplay {
		t.Error("Expected AutoReplay to be false")
	}
}

func TestEventEmitWithComplexData(t *testing.T) {
	type ComplexData struct {
		ID    int      `json:"id"`
		Name  string   `json:"name"`
		Tags  []string `json:"tags"`
		Valid bool     `json:"valid"`
	}

	event := &Event{
		Module: "test",
		Name:   "complex",
	}

	data := ComplexData{
		ID:    42,
		Name:  "Test",
		Tags:  []string{"tag1", "tag2"},
		Valid: true,
	}

	marshalled, err := event.marshall(data)
	if err != nil {
		t.Fatalf("Failed to marshall complex data: %v", err)
	}

	var result map[string]any
	json.Unmarshal(marshalled, &result)

	params := result["params"].([]any)
	if len(params) != 1 {
		t.Errorf("Expected 1 param, got %d", len(params))
	}

	complexResult, ok := params[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected param to be map, got %T", params[0])
	}

	if complexResult["id"].(float64) != 42 {
		t.Errorf("Expected id 42, got %v", complexResult["id"])
	}

	if complexResult["name"] != "Test" {
		t.Errorf("Expected name 'Test', got %v", complexResult["name"])
	}
}
