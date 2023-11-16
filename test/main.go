package main

import (
	"embed"
	webglue "go-webglue/pkg"
	"net/http"
)

//go:embed client/*
var clientResources embed.FS

func main() {
	server := http.NewServeMux()

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

	server.Handle("/", handler)
	http.ListenAndServe(":8080", server)
}
