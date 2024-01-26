package main

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"time"

	webglue "github.com/burgrp/go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

type TestApi struct {
	Counter int
}

func (api *TestApi) Div(a int, b int) (any, any, error) {
	if b == 0 {
		return 0, 0, errors.New("division by zero")
	}
	return a / b, a % b, nil
}

func (api *TestApi) Greet(ctx context.Context, in struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}) []string {
	return []string{
		"Hello, " + in.FirstName + " " + in.LastName + "!",
		"Hi, " + in.FirstName + " " + in.LastName + "!",
	}
}

func (api *TestApi) GetId() string {
	return "not implemented"
}

func (api *TestApi) Inc(inc int) int {
	api.Counter += inc
	return api.Counter
}

func main() {

	tickEvent := webglue.NewEvent("tick")

	options := webglue.Options{
		Modules: []webglue.Module{
			{
				Name:      "test",
				Resources: &clientResources,
				Events: []*webglue.Event{
					tickEvent,
				},
				Api: &TestApi{},
			},
		},
	}

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			tickEvent.TryEmit(time.Now().UnixMicro())
		}
	}()

	handler, err := webglue.NewHandler(options)
	if err != nil {
		panic(err)
	}

	port := "8080"
	println("Listening on port " + port)
	err = http.ListenAndServe(":"+port, handler)
	if err != nil {
		panic(err)
	}
}
