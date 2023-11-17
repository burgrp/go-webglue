package main

import (
	"embed"
	webglue "go-webglue/pkg"
	"net/http"
)

//go:embed client/*
var clientResources embed.FS

func main() {

	handler, err := webglue.NewHandler(webglue.Options{
		Modules: []webglue.Module{
			{
				Name:      "test",
				Resources: clientResources,
			},
		},
	})

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
