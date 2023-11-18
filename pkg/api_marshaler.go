package webglue

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

type ApiMarshaler struct {
}

func NewApiMarshaler(options Options) (*ApiMarshaler, error) {
	return &ApiMarshaler{}, nil
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
	Result any `json:"result"`
}

func (marshaler *ApiMarshaler) Call(receiver any, ctx context.Context, function string, request []byte) []byte {

	fncGoName := strings.ToUpper(function[0:1]) + function[1:]

	modPtrType := (reflect.TypeOf(receiver))

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
	unmParams := make([]any, numIn)
	unmToAllMap := make(map[int]int, numIn)
	unmParamsLen := 0

	for i := 0; i < len(allParams); i++ {

		paramType := fncType.In(i)

		if reflect.TypeOf(ctx).AssignableTo(paramType) {
			allParams[i] = reflect.ValueOf(ctx)
			continue
		}

		if reflect.TypeOf(receiver).AssignableTo(paramType) {
			allParams[i] = reflect.ValueOf(receiver)
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

	results := make([]any, 0)
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
