package webglue

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

type ApiDispatcher struct {
	modules map[string]interface{}
}

func NewApiDispatcher(options Options) (*ApiDispatcher, error) {

	modules := make(map[string]interface{})
	for _, module := range options.Modules {
		modules[module.Name] = module.Api
	}

	return &ApiDispatcher{
		modules: modules,
	}, nil
}

type ErrorReply struct {
	Error string `json:"error"`
}

func MarshalError(err error) []byte {
	json, err := json.Marshal(ErrorReply{
		Error: err.Error(),
	})
	if err != nil {
		panic(err)
	}
	return json
}

type ResultReply struct {
	Result interface{} `json:"result"`
}

func (marshaler *ApiDispatcher) Call(module string, function string, request []byte) []byte {

	mod, ok := marshaler.modules[module]
	if !ok {
		return MarshalError(errors.New("module not found"))
	}

	fncGoName := strings.ToUpper(function[0:1]) + function[1:]

	modPtrType := (reflect.TypeOf(mod))

	fncValue, ok := modPtrType.MethodByName(fncGoName)
	if !ok {
		return MarshalError(errors.New("function not found"))
	}

	fncType := fncValue.Type

	if fncType.NumIn() < 1 {
		return MarshalError(errors.New("function must have receiver"))
	}

	numParams := fncType.NumIn() - 1

	params := make([]interface{}, numParams)
	for i := 0; i < numParams; i++ {
		paramType := fncType.In(i + 1)
		param := reflect.New(paramType)
		params[i] = param.Interface()
	}

	err := json.Unmarshal(request, &params)
	if err != nil {
		return MarshalError(err)
	}

	if len(params) != numParams {
		return MarshalError(errors.New("wrong number of parameters"))
	}

	rcvAndArgValues := []reflect.Value{reflect.ValueOf(mod)}
	for _, param := range params {
		rcvAndArgValues = append(rcvAndArgValues, reflect.ValueOf(param).Elem())
	}

	allResults := fncValue.Func.Call(rcvAndArgValues)

	results := make([]interface{}, 0)
	for _, result := range allResults {
		if result.Type().AssignableTo(reflect.TypeOf((*error)(nil)).Elem()) {
			if !result.IsNil() {
				return MarshalError(result.Interface().(error))
			}
		} else {
			results = append(results, result.Interface())
		}
	}

	resultReply := ResultReply{}

	if len(results) == 1 {
		resultReply.Result = results[0]
	}

	if len(results) > 1 {
		resultReply.Result = results
	}

	resultJson, err := json.Marshal(resultReply)
	if err != nil {
		return MarshalError(err)
	}

	return resultJson
}
