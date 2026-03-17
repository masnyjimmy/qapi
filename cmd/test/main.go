package main

import (
	"context"
	"log"
	"net/http"

	"github.com/masnyjimmy/qapi/swagger"
)

func main() {
	s, err := swagger.NewWithWatcher("private/api.yaml", context.Background(), swagger.DefaultOptions())

	if err != nil {
		log.Fatalf("skibidi: %v", err)
	}

	log.Fatal(http.ListenAndServe(":1234", s.Handler(nil)))
}
