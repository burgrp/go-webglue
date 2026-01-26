package webglue

import (
	"embed"
	"reflect"
	"strings"
)

//go:embed client
var clientResources embed.FS

// CoreApi provides the discovery API that allows clients to enumerate
// all available modules, their functions, and events.
type CoreApi struct {
	Options *Options
}

// newCoreModule creates the built-in "webglue" module that provides
// API discovery functionality and core client resources.
func newCoreModule(Options *Options) *Module {
	return &Module{
		Name:      "webglue",
		Resources: &clientResources,
		Api: &CoreApi{
			Options: Options,
		},
	}
}

// DiscoveredModule describes the API surface of a module, listing
// all available functions and events for client-side code generation.
type DiscoveredModule struct {
	Functions []string `json:"functions"`
	Events    []string `json:"events"`
}

// Discover returns a map of all modules with their available functions and events.
// Function names are converted from PascalCase to camelCase for JavaScript compatibility.
// This method is called automatically by the client during initialization.
func (api *CoreApi) Discover() map[string]DiscoveredModule {
	result := make(map[string]DiscoveredModule)
	for _, module := range api.Options.Modules {

		var functions []string = make([]string, 0)

		// Use reflection to discover all exported methods on the API struct
		if module.Api != nil {
			apiType := reflect.TypeOf(module.Api)
			functions = make([]string, apiType.NumMethod())
			for i := 0; i < len(functions); i++ {
				name := apiType.Method(i).Name
				// Convert PascalCase to camelCase (GetUser -> getUser)
				name = strings.ToLower(name[0:1]) + name[1:]
				functions[i] = name
			}
		}

		// Extract event names
		events := make([]string, len(module.Events))
		for i, event := range module.Events {
			events[i] = event.Name
		}

		result[module.Name] = struct {
			Functions []string `json:"functions"`
			Events    []string `json:"events"`
		}{
			Functions: functions,
			Events:    events,
		}
	}
	return result
}
