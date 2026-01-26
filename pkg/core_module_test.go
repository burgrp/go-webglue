package webglue

import (
	"testing"
)

func TestNewCoreModule(t *testing.T) {
	options := &Options{
		Modules: []*Module{
			{
				Name: "test1",
				Api:  &TestApi{},
			},
			{
				Name: "test2",
				Api:  &TestApi{},
			},
		},
	}

	coreModule := newCoreModule(options)

	if coreModule.Name != "webglue" {
		t.Errorf("Expected core module name 'webglue', got '%s'", coreModule.Name)
	}

	if coreModule.Resources == nil {
		t.Error("Expected core module to have resources")
	}

	if coreModule.Api == nil {
		t.Error("Expected core module to have API")
	}

	coreApi, ok := coreModule.Api.(*CoreApi)
	if !ok {
		t.Fatalf("Expected CoreApi type, got %T", coreModule.Api)
	}

	if coreApi.Options != options {
		t.Error("Expected CoreApi to reference options")
	}
}

func TestCoreApiDiscover(t *testing.T) {
	event1 := NewEvent("event1")
	event2 := NewEvent("event2")

	options := &Options{
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

	coreApi := &CoreApi{Options: options}
	discovered := coreApi.Discover()

	if len(discovered) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(discovered))
	}

	// Check module1
	module1, ok := discovered["module1"]
	if !ok {
		t.Fatal("Expected module1 to be discovered")
	}

	if len(module1.Functions) == 0 {
		t.Error("Expected module1 to have functions")
	}

	// Check that function names are camelCase
	hasAdd := false
	for _, fn := range module1.Functions {
		if fn == "add" {
			hasAdd = true
		}
		// First character should be lowercase
		if len(fn) > 0 && fn[0] >= 'A' && fn[0] <= 'Z' {
			t.Errorf("Expected function name to be camelCase, got '%s'", fn)
		}
	}

	if !hasAdd {
		t.Error("Expected 'add' function in module1")
	}

	// Check events
	if len(module1.Events) != 1 {
		t.Errorf("Expected 1 event in module1, got %d", len(module1.Events))
	}

	if module1.Events[0] != "event1" {
		t.Errorf("Expected event 'event1', got '%s'", module1.Events[0])
	}
}

func TestCoreApiDiscoverNoApi(t *testing.T) {
	options := &Options{
		Modules: []*Module{
			{
				Name: "noApi",
				Api:  nil,
			},
		},
	}

	coreApi := &CoreApi{Options: options}
	discovered := coreApi.Discover()

	module, ok := discovered["noApi"]
	if !ok {
		t.Fatal("Expected noApi module to be discovered")
	}

	if len(module.Functions) != 0 {
		t.Errorf("Expected no functions, got %d", len(module.Functions))
	}
}

func TestCoreApiDiscoverNoEvents(t *testing.T) {
	options := &Options{
		Modules: []*Module{
			{
				Name:   "noEvents",
				Api:    &TestApi{},
				Events: []*Event{},
			},
		},
	}

	coreApi := &CoreApi{Options: options}
	discovered := coreApi.Discover()

	module := discovered["noEvents"]
	if len(module.Events) != 0 {
		t.Errorf("Expected no events, got %d", len(module.Events))
	}
}

func TestCoreApiDiscoverMultipleFunctions(t *testing.T) {
	options := &Options{
		Modules: []*Module{
			{
				Name: "test",
				Api:  &TestApi{},
			},
		},
	}

	coreApi := &CoreApi{Options: options}
	discovered := coreApi.Discover()

	module := discovered["test"]

	// TestApi should have multiple methods
	expectedFunctions := []string{"add", "subtract", "divide", "multipleReturns", "getCounter", "incrementCounter", "withContext", "complexParam"}

	for _, expected := range expectedFunctions {
		found := false
		for _, fn := range module.Functions {
			if fn == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected function '%s' to be discovered", expected)
		}
	}
}

type SingleMethodApi struct{}

func (api *SingleMethodApi) GetUserData() {}

func TestCoreApiDiscoverPascalToCamelCase(t *testing.T) {
	options := &Options{
		Modules: []*Module{
			{
				Name: "test",
				Api:  &SingleMethodApi{},
			},
		},
	}

	coreApi := &CoreApi{Options: options}
	discovered := coreApi.Discover()

	module := discovered["test"]

	if len(module.Functions) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(module.Functions))
	}

	if module.Functions[0] != "getUserData" {
		t.Errorf("Expected 'getUserData', got '%s'", module.Functions[0])
	}
}

func TestCoreApiDiscoverMultipleModules(t *testing.T) {
	event1 := NewEvent("evt1")
	event2 := NewEvent("evt2")
	event3 := NewEvent("evt3")

	options := &Options{
		Modules: []*Module{
			{
				Name:   "mod1",
				Api:    &TestApi{},
				Events: []*Event{event1},
			},
			{
				Name:   "mod2",
				Api:    &AuthedTestApi{},
				Events: []*Event{event2, event3},
			},
			{
				Name: "mod3",
				Api:  &TestApi{},
			},
		},
	}

	coreApi := &CoreApi{Options: options}
	discovered := coreApi.Discover()

	if len(discovered) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(discovered))
	}

	if _, ok := discovered["mod1"]; !ok {
		t.Error("Expected mod1 to be discovered")
	}

	if _, ok := discovered["mod2"]; !ok {
		t.Error("Expected mod2 to be discovered")
	}

	if _, ok := discovered["mod3"]; !ok {
		t.Error("Expected mod3 to be discovered")
	}

	if len(discovered["mod2"].Events) != 2 {
		t.Errorf("Expected mod2 to have 2 events, got %d", len(discovered["mod2"].Events))
	}
}

func TestDiscoveredModuleStructure(t *testing.T) {
	event := NewEvent("testEvent")

	options := &Options{
		Modules: []*Module{
			{
				Name:   "test",
				Api:    &TestApi{},
				Events: []*Event{event},
			},
		},
	}

	coreApi := &CoreApi{Options: options}
	discovered := coreApi.Discover()

	module := discovered["test"]

	// Test that the structure matches DiscoveredModule
	if module.Functions == nil {
		t.Error("Expected Functions field to be present")
	}

	if module.Events == nil {
		t.Error("Expected Events field to be present")
	}
}
