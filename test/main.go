package main

import (
	"context"
	"embed"
	"errors"
	webglue "go-webglue/pkg"
	"net/http"
)

//go:embed client/*
var clientResources embed.FS

type TestSession struct {
	Id      string
	Counter int
}

func (session *TestSession) Add(a struct{ V int }, b int) any {
	return a.V + b
}

func (session *TestSession) Div(a int, b int) (any, any, error) {
	if b == 0 {
		return 0, 0, errors.New("division by zero")
	}
	return a / b, a % b, nil
}

func (session *TestSession) Greet(ctx context.Context, in struct {
	Name string `json:"name"`
}) []string {
	return []string{
		"Hello, " + in.Name + "!",
		"Hi, " + in.Name + "!",
	}
}

func (session *TestSession) GetId() string {
	return session.Id
}

func (session *TestSession) Inc(inc int) int {
	session.Counter += inc
	return session.Counter
}

func main() {

	options := webglue.Options{
		Modules: []webglue.Module{
			{
				Name:      "test",
				Resources: clientResources,
			},
		},
		SessionFactory: func(id string) any {
			return &TestSession{
				Id: id,
			}
		},
	}

	handler, err := webglue.NewHandler(options)
	if err != nil {
		panic(err)
	}

	server := http.NewServeMux()
	server.Handle("/", handler)
	err = http.ListenAndServe(":8080", server)
	if err != nil {
		panic(err)
	}
}
