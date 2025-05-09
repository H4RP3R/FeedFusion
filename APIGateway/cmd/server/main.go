package main

import (
	"log"
	"net/http"

	"gateway/pkg/api"
)

func main() {
	api := api.New()
	err := http.ListenAndServe(":8080", api.Router())
	if err != nil {
		log.Fatal(err)
	}
}
