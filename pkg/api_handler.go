package webglue

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
)

const (
	ContentTypeHeader   = "Content-Type"
	ContentTypeJson     = "application/json"
	ContentLengthHeader = "Content-Length"
)

type ApiHandler struct {
	modules []*Module
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

func newApiHandler(modules []*Module) (*ApiHandler, error) {

	apiHandler := &ApiHandler{
		modules: modules,
	}

	return apiHandler, nil
}

func (ah *ApiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	pathSplit := strings.Split(request.URL.Path, "/")
	if len(pathSplit) < 3 {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	moduleName := pathSplit[len(pathSplit)-2]
	functionName := pathSplit[len(pathSplit)-1]

	responseHeaders := writer.Header()
	responseHeaders.Set(ContentTypeHeader, ContentTypeJson)

	if request.Method == http.MethodHead {
		responseHeaders.Set(ContentLengthHeader, "0")
		return
	}

	goFunctionName := strings.ToUpper(functionName[0:1]) + functionName[1:]

	var receiver any
	for _, module := range ah.modules {
		if module.Name == moduleName {
			receiver = module.Api
			break
		}
	}
	if receiver == nil {
		MarshalError(errors.New("module API not found"), writer)
		return
	}

	modPtrType := (reflect.TypeOf(receiver))

	fncValue, ok := modPtrType.MethodByName(goFunctionName)
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

	ctx := request.Context()

	typedParams := []any{
		ctx,
		receiver,
	}

outer:
	for i := 0; i < len(allParams); i++ {

		paramType := fncType.In(i)

		for j := 0; j < len(typedParams); j++ {
			typedParam := typedParams[j]
			if reflect.TypeOf(typedParam).AssignableTo(paramType) {
				allParams[i] = reflect.ValueOf(typedParam)
				continue outer
			}
		}

		param := reflect.New(paramType)
		unmParams[unmParamsLen] = param.Interface()
		unmToAllMap[unmParamsLen] = i
		unmParamsLen++
	}
	unmParams = unmParams[:unmParamsLen]

	beforeUnmarshal := len(unmParams)

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

	err := json.NewEncoder(writer).Encode(resultReply)
	if err != nil {
		MarshalError(err, writer)
		return
	}
}
