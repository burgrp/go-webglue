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

func (session *TestSession) Div(a int, b int) (any, any, error) {
	if b == 0 {
		return 0, 0, errors.New("division by zero")
	}
	return a / b, a % b, nil
}

func (session *TestSession) Greet(ctx context.Context, in struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}) []string {
	return []string{
		"Hello, " + in.FirstName + " " + in.LastName + "!",
		"Hi, " + in.FirstName + " " + in.LastName + "!",
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
	port := "8080"
	println("Listening on port " + port)
	err = http.ListenAndServe(":"+port, server)
	if err != nil {
		panic(err)
	}
}
