package webglue

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHandler(t *testing.T) {
	event := NewEvent("test")

	options := Options{
		Modules: []*Module{
			{
				Name:      "test",
				Resources: &testClientResources,
				Events:    []*Event{event},
				Api:       &TestApi{},
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

func TestNewHandlerWithMultipleModules(t *testing.T) {
	options := Options{
		Modules: []*Module{
			{
				Name: "module1",
				Api:  &TestApi{},
			},
			{
				Name: "module2",
				Api:  &AuthedTestApi{},
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

func TestNewHandlerWithCustomIndexHtml(t *testing.T) {
	customHTML := `<!DOCTYPE html>
<html>
<head>
<title>Custom</title>
{WEBGLUE}
</head>
<body>Test</body>
</html>`

	options := Options{
		Modules: []*Module{
			{
				Name:      "test",
				Resources: &testClientResources,
				Api:       &TestApi{},
			},
		},
		IndexHtml: customHTML,
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Test that custom HTML is used
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "<title>Custom</title>") {
		t.Error("Expected custom HTML to be used")
	}
}

func TestNewHandlerRoutingRoot(t *testing.T) {
	options := Options{
		Modules: []*Module{
			{
				Name:      "test",
				Resources: &testClientResources,
				Api:       &TestApi{},
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "html") {
		t.Errorf("Expected HTML content type, got '%s'", contentType)
	}
}

func TestNewHandlerRoutingApi(t *testing.T) {
	options := Options{
		Modules: []*Module{
			{
				Name: "test",
				Api:  &TestApi{},
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/test/add", strings.NewReader("[1,2]"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "json") {
		t.Errorf("Expected JSON content type, got '%s'", contentType)
	}
}

func TestNewHandlerRoutingEvents(t *testing.T) {
	event := NewEvent("test")

	options := Options{
		Modules: []*Module{
			{
				Name:   "test",
				Events: []*Event{event},
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// SSE connection blocks, so we need to handle it differently
	// Just verify the route is set up
	if handler == nil {
		t.Error("Handler should be configured for /events route")
	}
}

func TestNewHandlerCoreModuleIncluded(t *testing.T) {
	options := Options{
		Modules: []*Module{
			{
				Name: "mymodule",
				Api:  &TestApi{},
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Test that core module's discover function is available
	req := httptest.NewRequest("POST", "/api/webglue/discover", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected core module API to be available, got status %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "mymodule") {
		t.Error("Expected discover to return user modules")
	}
}

func TestNewHandlerEmptyModules(t *testing.T) {
	options := Options{
		Modules: []*Module{},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	if handler == nil {
		t.Fatal("Expected handler even with no modules")
	}

	// Core module should still be available
	req := httptest.NewRequest("POST", "/api/webglue/discover", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected core module to work, got status %d", w.Code)
	}
}

func TestNewHandlerModuleWithoutApi(t *testing.T) {
	options := Options{
		Modules: []*Module{
			{
				Name:      "staticonly",
				Resources: &testClientResources,
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	if handler == nil {
		t.Fatal("Expected handler with module without API")
	}
}

func TestNewHandlerModuleWithoutResources(t *testing.T) {
	options := Options{
		Modules: []*Module{
			{
				Name: "apionly",
				Api:  &TestApi{},
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	if handler == nil {
		t.Fatal("Expected handler with module without resources")
	}
}

func TestNewHandlerStaticResource(t *testing.T) {
	options := Options{
		Modules: []*Module{
			{
				Name:      "test",
				Resources: &testClientResources,
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("GET", "/webglue.js", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for static resource, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "javascript") {
		t.Errorf("Expected JavaScript content type, got '%s'", contentType)
	}
}

func TestNewHandlerDiscoverAllModules(t *testing.T) {
	event1 := NewEvent("event1")
	event2 := NewEvent("event2")

	options := Options{
		Modules: []*Module{
			{
				Name:   "module1",
				Api:    &TestApi{},
				Events: []*Event{event1},
			},
			{
				Name:   "module2",
				Api:    &AuthedTestApi{},
				Events: []*Event{event2},
			},
		},
	}

	handler, err := NewHandler(options)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/webglue/discover", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "module1") {
		t.Error("Expected module1 in discovery")
	}

	if !strings.Contains(body, "module2") {
		t.Error("Expected module2 in discovery")
	}

	if !strings.Contains(body, "event1") {
		t.Error("Expected event1 in discovery")
	}

	if !strings.Contains(body, "event2") {
		t.Error("Expected event2 in discovery")
	}
}

func TestModuleStruct(t *testing.T) {
	event := NewEvent("test")

	module := &Module{
		Name:      "test",
		Resources: &testClientResources,
		Events:    []*Event{event},
		Api:       &TestApi{},
	}

	if module.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", module.Name)
	}

	if module.Resources == nil {
		t.Error("Expected resources to be set")
	}

	if len(module.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(module.Events))
	}

	if module.Api == nil {
		t.Error("Expected API to be set")
	}
}

func TestOptionsStruct(t *testing.T) {
	customHTML := "<html></html>"

	options := Options{
		Modules: []*Module{
			{Name: "test"},
		},
		IndexHtml: customHTML,
	}

	if len(options.Modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(options.Modules))
	}

	if options.IndexHtml != customHTML {
		t.Error("Expected IndexHtml to be set")
	}
}
