package main

import (
	"embed"
	"errors"
	webglue "go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

type TestApi struct {
}

func (api *TestApi) Add(a struct{ V int }, b int) any {
	return a.V + b
}

func (api *TestApi) Div(a int, b int) (any, any, error) {
	if b == 0 {
		return 0, 0, errors.New("division by zero")
	}
	return a / b, a % b, nil
}

func (api *TestApi) Greet(in struct{ name string }) []string {
	return []string{
		"Hello, " + in.name + "!",
		"Hi, " + in.name + "!",
	}
}

func main() {

	options := webglue.Options{
		Modules: []webglue.Module{
			{
				Name:      "test",
				Resources: clientResources,
				Api:       &TestApi{},
			},
		},
	}

	apiDispatcher, err := webglue.NewApiDispatcher(options)
	if err != nil {
		panic(err)
	}

	result := apiDispatcher.Call("test", "div", []byte(`[5, 2]`))
	print("result: " + string(result))

	// handler, err := webglue.NewHandler(options)

	// if err != nil {
	// 	panic(err)
	// }

	// server := http.NewServeMux()
	// server.Handle("/", handler)
	// err = http.ListenAndServe(":8080", server)
	// if err != nil {
	// 	panic(err)
	// }
}
