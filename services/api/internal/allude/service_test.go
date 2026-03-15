package allude

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func eventually(t *testing.T, assertion func() bool) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if assertion() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !assertion() {
		t.Fatal("condition not met before deadline")
	}
}

func TestCreateThoughtStoresVersionAndFinishesAnalysis(t *testing.T) {
	service := NewService(NewInMemoryRepository(), &StubAIProvider{})
	created, err := service.CreateThought("Stoicism helps founders endure uncertainty and pressure.")
	if err != nil {
		t.Fatalf("create thought: %v", err)
	}
	if created.CurrentVersion.Version != 1 {
		t.Fatalf("expected version 1, got %d", created.CurrentVersion.Version)
	}
	if created.ProcessingStatus != ProcessingProcessing {
		t.Fatalf("expected processing status, got %s", created.ProcessingStatus)
	}

	eventually(t, func() bool {
		hydrated, _ := service.Thought(created.ID)
		return hydrated != nil && hydrated.ProcessingStatus == ProcessingReady && len(hydrated.Concepts) > 0
	})
}

func TestUpdateThoughtCreatesNewVersion(t *testing.T) {
	service := NewService(NewInMemoryRepository(), &StubAIProvider{})
	created, _ := service.CreateThought("Creativity blooms when boredom is allowed room.")
	eventually(t, func() bool {
		hydrated, _ := service.Thought(created.ID)
		return hydrated != nil && hydrated.ProcessingStatus == ProcessingReady
	})

	updated, err := service.UpdateThought(created.ID, "Creativity blooms when boredom and solitude are given room.")
	if err != nil {
		t.Fatalf("update thought: %v", err)
	}
	if updated.CurrentVersion.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.CurrentVersion.Version)
	}
	versions, err := service.ThoughtVersions(created.ID)
	if err != nil {
		t.Fatalf("versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
}

func TestSemanticSearchAndRelatedThoughts(t *testing.T) {
	service := NewService(NewInMemoryRepository(), &StubAIProvider{})
	first, _ := service.CreateThought("Stoicism and boxing both train discipline under discomfort.")
	second, _ := service.CreateThought("Founder psychology often borrows discipline rituals from martial arts.")
	_, _ = service.CreateThought("Cities can make loneliness feel louder than solitude.")

	eventually(t, func() bool {
		related, _ := service.RelatedThoughts(first.ID, 8)
		for _, thought := range related {
			if thought.ID == second.ID {
				return true
			}
		}
		return false
	})

	results, err := service.SearchThoughts("discipline and martial arts")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results.Thoughts) < 2 {
		t.Fatalf("expected search results, got %d", len(results.Thoughts))
	}
}

func TestGraphNeighborhoodAndConceptPage(t *testing.T) {
	service := NewService(NewInMemoryRepository(), &StubAIProvider{})
	first, _ := service.CreateThought("Stoicism values discipline and reflection.")
	_, _ = service.CreateThought("Discipline in martial arts can sharpen reflection.")

	eventually(t, func() bool {
		concept, _ := service.Concept("", "discipline")
		return concept != nil
	})

	graph, err := service.Graph(first.ID, 2, 12)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if graph.Center.Thought.ID != first.ID {
		t.Fatalf("expected center thought %s, got %s", first.ID, graph.Center.Thought.ID)
	}

	concept, err := service.Concept("", "discipline")
	if err != nil {
		t.Fatalf("concept: %v", err)
	}
	if concept == nil || len(concept.TopThoughts) == 0 {
		t.Fatal("expected concept page to include top thoughts")
	}
}

func TestGraphQLHandlerCreateThought(t *testing.T) {
	service := NewService(NewInMemoryRepository(), &StubAIProvider{})
	handler := NewGraphQLHandler(service)

	body, _ := json.Marshal(map[string]interface{}{
		"query": "mutation CreateThought($content: String!) { createThought(content: $content) { id processingStatus currentVersion { version } } }",
		"variables": map[string]interface{}{
			"content": "Boredom creates space for reflection.",
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("createThought")) {
		t.Fatalf("expected GraphQL payload, got %s", response.Body.String())
	}
}
