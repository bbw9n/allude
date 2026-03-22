package graphql

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/bbw9n/allude/services/api/internal/actions"
	"github.com/bbw9n/allude/services/api/internal/pkgs/ai"
	memstore "github.com/bbw9n/allude/services/api/internal/pkgs/storage/memory"
)

func TestSchemaSDLRootFieldsMatchRuntimeSchema(t *testing.T) {
	server, err := NewGraphQLServer(actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{}))
	if err != nil {
		t.Fatalf("new graphql server: %v", err)
	}

	schemaSDL := readSchemaSDL(t)

	assertRootFieldSetMatches(t,
		"Query",
		parseRootFieldsFromSDL(t, schemaSDL, "Query"),
		mapKeys(server.schema.QueryType().Fields()),
	)
	assertRootFieldSetMatches(t,
		"Mutation",
		parseRootFieldsFromSDL(t, schemaSDL, "Mutation"),
		mapKeys(server.schema.MutationType().Fields()),
	)
}

func readSchemaSDL(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine current file path")
	}

	schemaPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../../../packages/schema/src/schema.graphql"))
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema SDL: %v", err)
	}
	return string(content)
}

func parseRootFieldsFromSDL(t *testing.T, schemaSDL, typeName string) []string {
	t.Helper()

	blockStart := "type " + typeName + " {"
	startIndex := strings.Index(schemaSDL, blockStart)
	if startIndex < 0 {
		t.Fatalf("schema SDL is missing %s block", typeName)
	}

	block := schemaSDL[startIndex+len(blockStart):]
	endIndex := strings.Index(block, "\n}")
	if endIndex < 0 {
		t.Fatalf("schema SDL %s block is not terminated", typeName)
	}

	lines := strings.Split(block[:endIndex], "\n")
	fields := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fieldName := trimmed
		if argumentIndex := strings.Index(fieldName, "("); argumentIndex >= 0 {
			fieldName = fieldName[:argumentIndex]
		}
		if typeIndex := strings.Index(fieldName, ":"); typeIndex >= 0 {
			fieldName = fieldName[:typeIndex]
		}
		fieldName = strings.TrimSpace(fieldName)
		if fieldName != "" {
			fields = append(fields, fieldName)
		}
	}

	slices.Sort(fields)
	return fields
}

func mapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func assertRootFieldSetMatches(t *testing.T, typeName string, expected, actual []string) {
	t.Helper()

	if !slices.Equal(expected, actual) {
		t.Fatalf("%s root fields drifted\nexpected: %v\nactual:   %v", typeName, expected, actual)
	}
}
