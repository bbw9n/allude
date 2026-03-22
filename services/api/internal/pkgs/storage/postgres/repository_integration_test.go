package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	actions "github.com/bbw9n/allude/services/api/internal/actions"
	"github.com/bbw9n/allude/services/api/internal/domains/models"
	"github.com/bbw9n/allude/services/api/internal/pkgs/ai"
	pgstore "github.com/bbw9n/allude/services/api/internal/pkgs/storage/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresRepositoryReadWritePath(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	resetDatabase(t, databaseURL)

	repository, err := pgstore.NewPostgresRepository(databaseURL)
	if err != nil {
		t.Fatalf("new postgres repository: %v", err)
	}
	service := actions.NewService(repository, &ai.StubAIProvider{})

	step := func(label string, fn func() error) {
		t.Helper()
		start := time.Now()
		fmt.Printf("[postgres-integration] START %s\n", label)
		t.Logf("starting %s", label)
		if err := fn(); err != nil {
			fmt.Printf("[postgres-integration] FAIL %s after %s: %v\n", label, time.Since(start), err)
			t.Fatalf("%s: %v", label, err)
		}
		fmt.Printf("[postgres-integration] DONE %s in %s\n", label, time.Since(start))
		t.Logf("finished %s in %s", label, time.Since(start))
	}

	var first, second *models.Thought
	step("create first thought", func() error {
		created, createErr := service.CreateThought("Stoicism helps founders hold discipline under pressure.")
		first = created
		return createErr
	})
	step("create second thought", func() error {
		created, createErr := service.CreateThought("Boxing turns discipline into a daily physical practice.")
		second = created
		return createErr
	})
	step("drain jobs", func() error {
		fmt.Printf("[postgres-integration] queued jobs before drain: %d\n", len(service.Jobs()))
		err := service.DrainJobs(24)
		fmt.Printf("[postgres-integration] queued jobs after drain: %d\n", len(service.Jobs()))
		return err
	})

	var hydrated *models.Thought
	step("fetch thought", func() error {
		loaded, loadErr := service.Thought(first.ID)
		hydrated = loaded
		return loadErr
	})
	if hydrated.CurrentVersion == nil || len(hydrated.Concepts) == 0 {
		t.Fatalf("expected hydrated thought with concepts, got %+v", hydrated)
	}

	var search *models.SearchThoughtsResult
	step("search thoughts", func() error {
		result, searchErr := service.SearchThoughts("discipline")
		search = result
		return searchErr
	})
	if len(search.Thoughts) == 0 {
		t.Fatal("expected search results from postgres repository")
	}

	var graph *models.GraphNeighborhood
	step("graph", func() error {
		result, graphErr := service.Graph(first.ID, 2, 12)
		graph = result
		return graphErr
	})
	if graph.Center == nil || graph.Center.Thought == nil {
		t.Fatal("expected graph center")
	}

	var concept *models.Concept
	step("concept lookup", func() error {
		result, conceptErr := service.Concept("", "", "discipline")
		concept = result
		return conceptErr
	})
	if concept == nil || concept.ThoughtCount == 0 {
		t.Fatal("expected concept page data from postgres repository")
	}

	var collection *models.Collection
	step("create collection", func() error {
		result, collectionErr := service.CreateCollection("Founder Notes", "Important founder ideas")
		collection = result
		return collectionErr
	})
	step("add thought to collection", func() error {
		_, addErr := service.AddThoughtToCollection(collection.ID, first.ID)
		return addErr
	})
	step("record engagement", func() error {
		_, engagementErr := service.RecordEngagement("thought", second.ID, "open", 4200)
		return engagementErr
	})

	var home *models.HomePayload
	step("home", func() error {
		result, homeErr := service.Home(4)
		home = result
		return homeErr
	})
	if len(home.RecommendedThoughts) == 0 || len(home.Currents) == 0 {
		t.Fatalf("expected personalized home payload, got %+v", home)
	}

	var telescope *models.TelescopeResult
	step("telescope", func() error {
		result, telescopeErr := service.Telescope("connections between stoicism and discipline")
		telescope = result
		return telescopeErr
	})
	if telescope.Graph == nil || len(telescope.SeedThoughts) == 0 {
		t.Fatalf("expected telescope payload, got %+v", telescope)
	}
}

func resetDatabase(t *testing.T, databaseURL string) {
	t.Helper()
	fmt.Printf("[postgres-integration] reset database start\n")
	start := time.Now()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	resetStatements := []string{
		"DROP SCHEMA IF EXISTS public CASCADE",
		"CREATE SCHEMA public",
		"CREATE EXTENSION IF NOT EXISTS vector",
	}
	for _, statement := range resetStatements {
		fmt.Printf("[postgres-integration] exec reset statement: %s\n", statement)
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("exec reset statement %q: %v", statement, err)
		}
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine current file path")
	}
	schemaPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../../src/postgres/schema.sql"))
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	for _, statement := range splitSQLStatements(string(schemaSQL)) {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("exec schema statement %q: %v", statement, err)
		}
	}
	fmt.Printf("[postgres-integration] reset database done in %s\n", time.Since(start))
}

func splitSQLStatements(schema string) []string {
	parts := strings.Split(schema, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}
	return statements
}
