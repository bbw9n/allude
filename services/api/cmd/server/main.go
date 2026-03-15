package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bytedance/allude/services/api/internal/allude"
)

func main() {
	service := allude.NewService(allude.NewInMemoryRepository(), &allude.StubAIProvider{})
	handler := allude.NewGraphQLHandler(service)
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	log.Printf("Allude API ready at http://127.0.0.1:%s/\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
