package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bbw9n/allude/services/api/internal/actions"
	"github.com/bbw9n/allude/services/api/internal/domains/ports"
	ai "github.com/bbw9n/allude/services/api/internal/pkgs/ai"
	apiGraphql "github.com/bbw9n/allude/services/api/internal/pkgs/graphql"
	cache "github.com/bbw9n/allude/services/api/internal/pkgs/storage/cache"
	memstore "github.com/bbw9n/allude/services/api/internal/pkgs/storage/memory"
	pgstore "github.com/bbw9n/allude/services/api/internal/pkgs/storage/postgres"
)

func main() {
	var repository ports.Repository
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		postgresRepository, err := pgstore.NewPostgresRepository(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		repository = postgresRepository
	} else {
		repository = memstore.NewInMemoryRepository()
	}
	repository = cache.NewCachedRepository(repository, 30*time.Second)
	service := actions.NewService(repository, &ai.StubAIProvider{})
	server, err := apiGraphql.NewGraphQLServer(service)
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
