package webglue

import (
	"embed"
	"reflect"
	"strings"
)

//go:embed client
var clientResources embed.FS

type CoreApi struct {
	Options *Options
}

func newCoreModule(Options *Options) *Module {
	return &Module{
		Name:      "webglue",
		Resources: &clientResources,
		Api: &CoreApi{
			Options: Options,
		},
	}
}

type DiscoveredModule struct {
	Functions []string `json:"functions"`
	Events    []string `json:"events"`
}

func (api *CoreApi) Discover() map[string]DiscoveredModule {
	result := make(map[string]DiscoveredModule)
	for _, module := range api.Options.Modules {

		var functions []string = make([]string, 0)

		if module.Api != nil {
			apiType := reflect.TypeOf(module.Api)
			functions = make([]string, apiType.NumMethod())
			for i := 0; i < len(functions); i++ {
				name := apiType.Method(i).Name
				name = strings.ToLower(name[0:1]) + name[1:]
				functions[i] = name
			}
		}

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
