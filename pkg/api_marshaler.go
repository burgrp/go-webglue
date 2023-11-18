package webglue

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
)

type ApiMarshaler struct {
}

func newApiMarshaler(options Options) (*ApiMarshaler, error) {
	return &ApiMarshaler{}, nil
}

type ErrorReply struct {
	Error string `json:"error"`
}

func MarshalError(err error, writer io.Writer) {
	err2 := json.NewEncoder(writer).Encode(ErrorReply{
		Error: err.Error(),
	})
	if err2 != nil {
		panic(err)
	}
}

type ResultReply struct {
	Result any `json:"result"`
}

func (marshaler *ApiMarshaler) describe(receiver any, writer io.Writer) {

	modPtrType := (reflect.TypeOf(receiver))

	list := make([]string, modPtrType.NumMethod())
	for i := 0; i < len(list); i++ {
		name := modPtrType.Method(i).Name
		list[i] = strings.ToLower(name[0:1]) + name[1:]
	}
	json.NewEncoder(writer).Encode(list)
}

func (marshaler *ApiMarshaler) call(receiver any, ctx context.Context, function string, reader io.Reader, writer io.Writer) {

	fncGoName := strings.ToUpper(function[0:1]) + function[1:]

	modPtrType := (reflect.TypeOf(receiver))

	fncValue, ok := modPtrType.MethodByName(fncGoName)
	if !ok {
		MarshalError(errors.New("function not found"), writer)
		return
	}

	fncType := fncValue.Type

	if fncType.NumIn() < 1 {
		MarshalError(errors.New("function must have receiver"), writer)
		return
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

	err := json.NewDecoder(reader).Decode(&unmParams)
	if err != nil {
		MarshalError(err, writer)
		return
	}

	if len(unmParams) != beforeUnmarshal {
		MarshalError(errors.New("wrong number of parameters"), writer)
		return
	}

	for i := 0; i < len(unmParams); i++ {
		allParams[unmToAllMap[i]] = reflect.ValueOf(unmParams[i]).Elem()
	}

	allResults := fncValue.Func.Call(allParams)

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

	resultReply := ResultReply{}

	if len(results) == 1 {
		resultReply.Result = results[0]
	}

	if len(results) > 1 {
		resultReply.Result = results
	}

	err = json.NewEncoder(writer).Encode(resultReply)
	if err != nil {
		MarshalError(err, writer)
		return
	}
}
