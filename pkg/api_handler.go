// Package webglue provides a web framework that automatically bridges Go backend and JavaScript frontend code.
// It exposes Go struct methods as HTTP APIs, streams real-time events via SSE, and serves optimized static resources.
package webglue

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	json "github.com/json-iterator/go"
)

// HTTP header constants used throughout the API handler.
const (
	ContentTypeHeader   = "Content-Type"
	ContentTypeJson     = "application/json"
	ContentLengthHeader = "Content-Length"
)

// ApiHandler routes HTTP requests to Go methods using reflection.
// It handles parameter injection, JSON marshaling/unmarshaling, and result formatting.
type ApiHandler struct {
	modules []*Module
}

// ErrorReply is the JSON structure returned when an API call fails.
type ErrorReply struct {
	Error string `json:"error"`
}

// MarshalError writes an error as JSON to the response writer.
// If encoding fails, it panics to prevent silent failures.
func MarshalError(err error, writer io.Writer) {
	err2 := json.NewEncoder(writer).Encode(ErrorReply{
		Error: err.Error(),
	})
	if err2 != nil {
		panic(err)
	}
}

// CallChecker is an optional interface that API structs can implement to perform
// authentication, authorization, or parameter injection before each API call.
// The returned slice of values will be injected into the function call as typed parameters.
type CallChecker interface {
	CheckCall(request *http.Request, functionName string) ([]any, error)
}

// ResultReply is the JSON structure returned when an API call succeeds.
type ResultReply struct {
	Result any `json:"result"`
}

// newApiHandler creates a new API handler for the given modules.
func newApiHandler(modules []*Module) (*ApiHandler, error) {
	apiHandler := &ApiHandler{
		modules: modules,
	}

	return apiHandler, nil
}

// ServeHTTP handles API requests by routing them to the appropriate Go method.
// URL format: /api/{module}/{function}
// Request body: JSON array of parameters
// Response: JSON object with "result" or "error" field
func (ah *ApiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	// Parse URL path to extract module and function names
	pathSplit := strings.Split(request.URL.Path, "/")
	if len(pathSplit) < 3 {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	moduleName := pathSplit[len(pathSplit)-2]
	functionName := pathSplit[len(pathSplit)-1]
	// Convert camelCase to PascalCase (JavaScript "getUser" -> Go "GetUser")
	functionName = strings.ToUpper(functionName[0:1]) + functionName[1:]

	responseHeaders := writer.Header()
	responseHeaders.Set(ContentTypeHeader, ContentTypeJson)

	// Handle HEAD requests (used for API discovery)
	if request.Method == http.MethodHead {
		responseHeaders.Set(ContentLengthHeader, "0")
		return
	}

	// Find the requested module
	var module *Module
	for _, m := range ah.modules {
		if m.Name == moduleName {
			module = m
			break
		}
	}

	if module == nil {
		MarshalError(errors.New("module not found"), writer)
		return
	}

	api := module.Api
	if api == nil {
		MarshalError(errors.New("module API not found"), writer)
		return
	}

	// Use reflection to find the method
	modPtrType := (reflect.TypeOf(api))

	fncValue, ok := modPtrType.MethodByName(functionName)
	if !ok {
		MarshalError(errors.New("function not found"), writer)
		return
	}

	fncType := fncValue.Type

	if fncType.NumIn() < 1 {
		MarshalError(errors.New("function must have receiver"), writer)
		return
	}

	// Build parameter list: some injected (context, custom types), some from JSON
	numIn := fncType.NumIn()
	allParams := make([]reflect.Value, numIn)
	unmParams := make([]any, numIn)
	unmToAllMap := make(map[int]int, numIn)
	unmParamsLen := 0

	ctx := request.Context()

	// Typed parameters that can be auto-injected
	typedParams := []any{
		ctx,
		api,
	}

	// If API implements CallChecker, call it for authentication/authorization
	if callChecker, ok := api.(CallChecker); ok {
		if functionName == "CheckCall" {
			MarshalError(errors.New("CheckCall function is not allowed"), writer)
			return
		}
		tp, err := callChecker.CheckCall(request, functionName)
		if err != nil {
			MarshalError(err, writer)
			return
		}
		typedParams = append(typedParams, tp...)
	}

	// Match function parameters: try typed injection first, then prepare for JSON unmarshaling
outer:
	for i := 0; i < len(allParams); i++ {

		paramType := fncType.In(i)

		// Try to match with typed parameters (context, API instance, or CallChecker results)
		for j := 0; j < len(typedParams); j++ {
			typedParam := typedParams[j]
			if reflect.TypeOf(typedParam).AssignableTo(paramType) {
				allParams[i] = reflect.ValueOf(typedParam)
				continue outer
			}
		}

		// Parameter needs to be unmarshaled from JSON
		param := reflect.New(paramType)
		unmParams[unmParamsLen] = param.Interface()
		unmToAllMap[unmParamsLen] = i
		unmParamsLen++
	}
	unmParams = unmParams[:unmParamsLen]

	beforeUnmarshal := len(unmParams)

	// Unmarshal JSON parameters from request body
	if beforeUnmarshal > 0 {
		err := json.NewDecoder(request.Body).Decode(&unmParams)
		if err != nil {
			MarshalError(err, writer)
			return
		}
	}

	if len(unmParams) != beforeUnmarshal {
		MarshalError(errors.New("wrong number of parameters"), writer)
		return
	}

	// Extract values from pointer wrappers created by JSON unmarshaling
	for i := 0; i < len(unmParams); i++ {
		paramValue := reflect.ValueOf(unmParams[i])
		paramKind := paramValue.Kind()
		if paramKind == reflect.Ptr || paramKind == reflect.Interface {
			paramValue = paramValue.Elem()
		} else {
			MarshalError(errors.New("can not unmarshal parameter"), writer)
			return
		}
		allParams[unmToAllMap[i]] = paramValue
	}

	// Invoke the method with all parameters
	allResults := fncValue.Func.Call(allParams)

	// Process results: separate errors from regular return values
	results := make([]any, 0)
	for _, result := range allResults {
		if result.Type().AssignableTo(reflect.TypeOf((*error)(nil)).Elem()) {
			if !result.IsNil() {
				MarshalError(result.Interface().(error), writer)
				return
			}
		} else {
			results = append(results, result.Interface())
		}
	}

	// Format result based on number of return values
	resultReply := ResultReply{}

	if len(results) == 1 {
		resultReply.Result = results[0]
	}

	if len(results) > 1 {
		resultReply.Result = results
	}

	// Marshal and write response
	err := json.NewEncoder(writer).Encode(resultReply)
	if err != nil {
		MarshalError(err, writer)
		return
	}
}
