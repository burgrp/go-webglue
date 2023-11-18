package webglue

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

type ApiMarshaler struct {
	modules map[string]interface{}
}

func NewApiMarshaler(options Options) (*ApiMarshaler, error) {

	modules := make(map[string]interface{})
	for _, module := range options.Modules {
		modules[module.Name] = module.Api
	}

	return &ApiMarshaler{
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

func (marshaler *ApiMarshaler) Call(ctx context.Context, module string, function string, request []byte) []byte {

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

	numIn := fncType.NumIn()
	allParams := make([]reflect.Value, numIn)
	unmParams := make([]interface{}, numIn)
	unmToAllMap := make(map[int]int, numIn)
	unmParamsLen := 0

	for i := 0; i < len(allParams); i++ {

		paramType := fncType.In(i)

		if reflect.TypeOf(ctx).AssignableTo(paramType) {
			allParams[i] = reflect.ValueOf(ctx)
			continue
		}

		if reflect.TypeOf(mod).AssignableTo(paramType) {
			allParams[i] = reflect.ValueOf(mod)
			continue
		}

		param := reflect.New(paramType)
		unmParams[unmParamsLen] = param.Interface()
		unmToAllMap[unmParamsLen] = i
		unmParamsLen++
	}
	unmParams = unmParams[:unmParamsLen]

	beforeUnmarshal := len(unmParams)

	err := json.Unmarshal(request, &unmParams)
	if err != nil {
		return MarshalError(err)
	}

	if len(unmParams) != beforeUnmarshal {
		return MarshalError(errors.New("wrong number of parameters"))
	}

	for i := 0; i < len(unmParams); i++ {
		allParams[unmToAllMap[i]] = reflect.ValueOf(unmParams[i]).Elem()
	}

	allResults := fncValue.Func.Call(allParams)

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
