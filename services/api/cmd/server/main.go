package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bbw9n/allude/services/api/internal/allude"
)

func main() {
	var repository allude.Repository
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		postgresRepository, err := allude.NewPostgresRepository(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		repository = postgresRepository
	} else {
		repository = allude.NewInMemoryRepository()
	}
	repository = allude.NewCachedRepository(repository, 30*time.Second)
	service := allude.NewService(repository, &allude.StubAIProvider{})
	server, err := allude.NewGraphQLServer(service)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	service.StartWorkers(ctx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	mux := http.NewServeMux()
	mux.Handle("/", server)

	log.Printf("Allude API ready at http://127.0.0.1:%s/\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
