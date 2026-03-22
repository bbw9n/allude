package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	actions "github.com/bbw9n/allude/services/api/internal/actions"
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

	first, err := service.CreateThought("Stoicism helps founders hold discipline under pressure.")
	if err != nil {
		t.Fatalf("create first thought: %v", err)
	}
	second, err := service.CreateThought("Boxing turns discipline into a daily physical practice.")
	if err != nil {
		t.Fatalf("create second thought: %v", err)
	}
	if err := service.DrainJobs(24); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}

	hydrated, err := service.Thought(first.ID)
	if err != nil {
		t.Fatalf("fetch thought: %v", err)
	}
	if hydrated.CurrentVersion == nil || len(hydrated.Concepts) == 0 {
		t.Fatalf("expected hydrated thought with concepts, got %+v", hydrated)
	}

	search, err := service.SearchThoughts("discipline")
	if err != nil {
		t.Fatalf("search thoughts: %v", err)
	}
	if len(search.Thoughts) == 0 {
		t.Fatal("expected search results from postgres repository")
	}

	graph, err := service.Graph(first.ID, 2, 12)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if graph.Center == nil || graph.Center.Thought == nil {
		t.Fatal("expected graph center")
	}

	concept, err := service.Concept("", "", "discipline")
	if err != nil {
		t.Fatalf("concept lookup: %v", err)
	}
	if concept == nil || concept.ThoughtCount == 0 {
		t.Fatal("expected concept page data from postgres repository")
	}

	collection, err := service.CreateCollection("Founder Notes", "Important founder ideas")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if _, err := service.AddThoughtToCollection(collection.ID, first.ID); err != nil {
		t.Fatalf("add thought to collection: %v", err)
	}
	if _, err := service.RecordEngagement("thought", second.ID, "open", 4200); err != nil {
		t.Fatalf("record engagement: %v", err)
	}

	home, err := service.Home(4)
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	if len(home.RecommendedThoughts) == 0 || len(home.Currents) == 0 {
		t.Fatalf("expected personalized home payload, got %+v", home)
	}

	telescope, err := service.Telescope("connections between stoicism and discipline")
	if err != nil {
		t.Fatalf("telescope: %v", err)
	}
	if telescope.Graph == nil || len(telescope.SeedThoughts) == 0 {
		t.Fatalf("expected telescope payload, got %+v", telescope)
	}
}

func resetDatabase(t *testing.T, databaseURL string) {
	t.Helper()

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
