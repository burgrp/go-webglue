package webglue

import (
	"embed"
	"net/http"
)

// Module represents a logical component of a webglue application.
// Each module can provide an API (Go methods), events (SSE), and client resources (HTML/CSS/JS).
type Module struct {
	Name      string      // Module identifier used in URLs and API paths
	Resources *embed.FS   // Embedded client-side files (optional)
	Events    []*Event    // Server-to-client events (optional)
	Api       any         // Struct with exported methods to expose as API (optional)
}

// Options configures the webglue handler.
type Options struct {
	Modules   []*Module  // Application modules
	IndexHtml string     // Custom HTML template (optional, uses default if empty)
}

// NewHandler creates the main HTTP handler for a webglue application.
// It sets up three route handlers:
//   - / -> static file serving and SPA fallback
//   - /api/* -> API method routing
//   - /events -> Server-Sent Events stream
//
// The core "webglue" module is automatically added for API discovery.
func NewHandler(options Options) (*http.ServeMux, error) {

	// Add core module that provides discovery API and client library
	allModules := append([]*Module{
		newCoreModule(&options),
	}, options.Modules...)

	// Create static file handler (serves HTML, CSS, JS, images)
	staticHandler, err := newStaticHandler(allModules, options.IndexHtml)
	if err != nil {
		return nil, err
	}

	// Create API handler (routes HTTP calls to Go methods)
	apiHandler, err := newApiHandler(allModules)
	if err != nil {
		return nil, err
	}

	// Create event handler (SSE streaming)
	eventHandler, err := newEventHandler(allModules)
	if err != nil {
		return nil, err
	}

	// Set up routing
	mux := http.ServeMux{}

	mux.Handle("/", staticHandler)
	mux.Handle("/api/", apiHandler)
	mux.Handle("/events", eventHandler)

	return &mux, nil
}
