package main

import (
	"context"
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

func (api *TestApi) Greet(ctx context.Context, in struct {
	Name string `json:"name"`
}) []string {
	return []string{
		"Hello, " + in.Name + "!",
		"Hi, " + in.Name + "!",
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

	apiMarshaler, err := webglue.NewApiMarshaler(options)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	result := apiMarshaler.Call(ctx, "test", "div", []byte(`[5, 2]`))
	println("result: " + string(result))

	result = apiMarshaler.Call(ctx, "test", "greet", []byte(`[{"name": "John"}]`))
	println("result: " + string(result))

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
