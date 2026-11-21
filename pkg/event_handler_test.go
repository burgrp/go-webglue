package webglue

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
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
	t.Cleanup(handler.Close)

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

type blockedEventWriter struct {
	*httptest.ResponseRecorder
	flushing chan struct{}
	release  chan struct{}
}

func (writer *blockedEventWriter) SetWriteDeadline(time.Time) error {
	return nil
}

func (writer *blockedEventWriter) Flush() {
	select {
	case writer.flushing <- struct{}{}:
	default:
	}
	<-writer.release
	writer.ResponseRecorder.Flush()
}

type streamedEvent struct {
	Module string            `json:"module"`
	Name   string            `json:"name"`
	Params []json.RawMessage `json:"params"`
}

func openEventStream(t *testing.T, client *http.Client, url string) (*bufio.Scanner, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/events?stream=webglue", nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	closeStream := func() {
		response.Body.Close()
		cancel()
	}
	t.Cleanup(closeStream)
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("SSE response = %d %v", response.StatusCode, response.Header)
	}
	return bufio.NewScanner(response.Body), closeStream
}

func nextStreamEvent(t *testing.T, scanner *bufio.Scanner) streamedEvent {
	t.Helper()
	for scanner.Scan() {
		if data, ok := strings.CutPrefix(scanner.Text(), "data: "); ok {
			var event streamedEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				t.Fatal(err)
			}
			return event
		}
	}
	t.Fatalf("SSE ended before the next event: %v", scanner.Err())
	return streamedEvent{}
}

func TestSlowEventSubscriberDoesNotBlockOthers(t *testing.T) {
	event := NewEvent("updated")
	handler, err := newEventHandler([]*Module{{Name: "test", Events: []*Event{event}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(handler.Close)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	ctx, cancel := context.WithCancel(context.Background())
	writer := &blockedEventWriter{
		ResponseRecorder: httptest.NewRecorder(),
		flushing:         make(chan struct{}, 1),
		release:          make(chan struct{}),
	}
	done := make(chan struct{})
	var released sync.Once
	release := func() { released.Do(func() { close(writer.release) }) }
	go func() {
		defer close(done)
		handler.ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/events?stream=webglue", nil).WithContext(ctx))
	}()
	t.Cleanup(func() {
		cancel()
		release()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("blocked event handler did not stop")
		}
	})
	select {
	case <-writer.flushing:
	case <-time.After(5 * time.Second):
		t.Fatal("blocked subscriber did not connect")
	}
	first, _ := openEventStream(t, server.Client(), server.URL)
	for value := 0; value < 130; value++ {
		event.Emit(value)
		reported := nextStreamEvent(t, first)
		if reported.Module != "test" || reported.Name != "updated" || len(reported.Params) != 1 {
			t.Fatalf("event = %#v", reported)
		}
		var actual int
		if err := json.Unmarshal(reported.Params[0], &actual); err != nil {
			t.Fatal(err)
		}
		if actual != value {
			t.Fatalf("event value = %d, want %d", actual, value)
		}
	}
	second, _ := openEventStream(t, server.Client(), server.URL)
	event.Emit("still-live")
	for _, scanner := range []*bufio.Scanner{first, second} {
		reported := nextStreamEvent(t, scanner)
		if len(reported.Params) != 1 || string(reported.Params[0]) != `"still-live"` {
			t.Fatalf("event after overflow = %#v", reported)
		}
	}
	release()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("overflowed connection remained open")
	}
}

type failingEventWriter struct {
	*httptest.ResponseRecorder
	event     *Event
	operation string
	deadline  time.Time
	started   bool
}

func (writer *failingEventWriter) SetWriteDeadline(deadline time.Time) error {
	writer.deadline = deadline
	return nil
}

func (writer *failingEventWriter) Write(data []byte) (int, error) {
	if writer.operation == "write" {
		time.Sleep(time.Until(writer.deadline))
		return 0, os.ErrDeadlineExceeded
	}
	if writer.operation == "short-write" {
		return len(data) - 1, nil
	}
	return writer.ResponseRecorder.Write(data)
}

func (writer *failingEventWriter) FlushError() error {
	if !writer.started {
		writer.started = true
		writer.event.Emit("payload")
		return nil
	}
	time.Sleep(time.Until(writer.deadline))
	return os.ErrDeadlineExceeded
}

func TestEventWriteFailuresRemoveSubscriber(t *testing.T) {
	for _, operation := range []string{"write", "flush", "short-write"} {
		t.Run(operation, func(t *testing.T) {
			event := NewEvent("updated")
			handler, err := newEventHandler([]*Module{{Name: "test", Events: []*Event{event}}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(handler.Close)
			writer := &failingEventWriter{ResponseRecorder: httptest.NewRecorder(), event: event, operation: operation}
			start := time.Now()
			handler.ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/events?stream=webglue", nil))
			if operation != "short-write" {
				if elapsed := time.Since(start); elapsed < 5*time.Second || elapsed > 7*time.Second {
					t.Fatalf("blocked %s lasted %v, want a five-second deadline", operation, elapsed)
				}
			}
			if len(handler.clients) != 0 {
				t.Fatal("failed event connection retained its subscriber")
			}
		})
	}
}

func TestIdleHTTP2EventStreamRemainsOpen(t *testing.T) {
	event := NewEvent("updated")
	handler, err := newEventHandler([]*Module{{Name: "test", Events: []*Event{event}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(handler.Close)
	protocol := make(chan int, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		protocol <- request.ProtoMajor
		handler.ServeHTTP(writer, request)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	scanner, _ := openEventStream(t, server.Client(), server.URL)
	if version := <-protocol; version != 2 {
		t.Fatalf("protocol major = %d, want HTTP/2", version)
	}
	time.Sleep(6 * time.Second)
	event.Emit("after-idle")
	reported := nextStreamEvent(t, scanner)
	if len(reported.Params) != 1 || string(reported.Params[0]) != `"after-idle"` {
		t.Fatalf("event after idle period = %#v", reported)
	}
}

func TestEventStreamCancellationReconnectAndClose(t *testing.T) {
	event := NewEvent("updated")
	handler, err := newEventHandler([]*Module{{Name: "test", Events: []*Event{event}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(handler.Close)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	first, closeFirst := openEventStream(t, server.Client(), server.URL)
	event.Emit("first")
	if reported := nextStreamEvent(t, first); string(reported.Params[0]) != `"first"` {
		t.Fatalf("first event = %#v", reported)
	}
	closeFirst()
	event.Emit("between-connections")
	second, _ := openEventStream(t, server.Client(), server.URL)
	event.Emit("second")
	if reported := nextStreamEvent(t, second); string(reported.Params[0]) != `"second"` {
		t.Fatalf("reconnected stream replayed an old event: %#v", reported)
	}
	handler.Close()
	handler.Close()
	event.Emit("closed")
	for second.Scan() {
		if strings.HasPrefix(second.Text(), "data: ") {
			t.Fatalf("closed stream still delivered data: %s", second.Text())
		}
	}
	if err := second.Err(); err != nil {
		t.Fatalf("closed stream ended with an error: %v", err)
	}
	if len(handler.clients) != 0 {
		t.Fatal("closed event handler retained subscribers")
	}
}

func TestConcurrentEventEmissionAndRegistration(t *testing.T) {
	event := NewEvent("updated")
	started := make(chan struct{})
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		event.Emit(0)
		close(started)
		for value := 1; ; value++ {
			select {
			case <-stop:
				return
			default:
				event.Emit(value)
			}
		}

	}()
	<-started
	defer func() {
		close(stop)
		<-done
	}()
	for index := 0; index < 64; index++ {
		handler, err := newEventHandler([]*Module{{Name: "test", Events: []*Event{event}}})
		if err != nil {
			t.Fatal(err)
		}
		handler.Close()
	}
}

func TestEventStreamRejectsInvalidRequests(t *testing.T) {
	handler, err := newEventHandler(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(handler.Close)
	for _, item := range []struct {
		path string
		id   string
	}{
		{path: "/events"},
		{path: "/events?stream=missing"},
		{path: "/events?stream=webglue", id: "invalid"},
	} {
		request := httptest.NewRequest(http.MethodGet, item.path, nil)
		request.Header.Set("Last-Event-ID", item.id)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || response.Body.Len() == 0 {
			t.Errorf("%s, Last-Event-ID=%q: response = %d %q", item.path, item.id, response.Code, response.Body.String())
		}
	}
	if len(handler.clients) != 0 {
		t.Fatal("invalid requests created event subscriptions")
	}
}
