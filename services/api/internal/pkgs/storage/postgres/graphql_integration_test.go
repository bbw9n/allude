package postgres_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	actions "github.com/bbw9n/allude/services/api/internal/actions"
	"github.com/bbw9n/allude/services/api/internal/pkgs/ai"
	api "github.com/bbw9n/allude/services/api/internal/pkgs/graphql"
	pgstore "github.com/bbw9n/allude/services/api/internal/pkgs/storage/postgres"
)

type graphQLIntegrationResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type graphQLHarness struct {
	server        *api.GraphQLServer
	service       *actions.Service
	firstThought  string
	secondThought string
	collectionID  string
}

func TestPostgresGraphQLE2EThoughtLifecycle(t *testing.T) {
	h := newGraphQLHarness(t)
	h.seedThoughts(t)

	editPayload := h.exec(t, mutationEditThought, map[string]interface{}{
		"thoughtId": h.firstThought,
		"content":   "Stoicism helps founders keep discipline under pressure and recover judgment.",
	})
	assertGraphQLRootKeys(t, editPayload, "editThought")

	h.drain(t, 24, "after edit thought")

	payload := h.exec(t, queryThoughtLifecycle, map[string]interface{}{"id": h.firstThought})
	assertGraphQLRootKeys(t, payload, "thought", "relatedThoughts", "listThoughtVersions")
	assertPayloadContains(t, payload, h.firstThought)
	assertPayloadContains(t, payload, h.secondThought)
	assertPayloadContains(t, payload, "canonicalName")
}

func TestPostgresGraphQLE2EDiscoveryFlow(t *testing.T) {
	h := newGraphQLHarness(t)
	h.seedThoughts(t)

	searchPayload := h.exec(t, queryDiscoverySearch, map[string]interface{}{"query": "discipline"})
	assertGraphQLRootKeys(t, searchPayload, "search", "searchThoughts")
	assertPayloadContains(t, searchPayload, h.firstThought)

	graphPayload := h.exec(t, queryDiscoveryGraph, map[string]interface{}{"thoughtId": h.firstThought})
	assertGraphQLRootKeys(t, graphPayload, "graph")
	assertPayloadContains(t, graphPayload, h.firstThought)

	conceptPayload := h.exec(t, queryDiscoveryConcept, map[string]interface{}{"name": "discipline"})
	assertGraphQLRootKeys(t, conceptPayload, "concept")
	assertPayloadContains(t, conceptPayload, "discipline")

	discoveryPayload := h.exec(t, queryDiscoveryTelescope, map[string]interface{}{
		"query":     "connections between stoicism and discipline",
		"thoughtId": h.firstThought,
	})
	assertGraphQLRootKeys(t, discoveryPayload, "draftSuggestions", "telescope")
	assertPayloadContains(t, discoveryPayload, "narrative")
}

func TestPostgresGraphQLE2ECollectionsAndPersonalization(t *testing.T) {
	h := newGraphQLHarness(t)
	h.seedThoughts(t)

	h.createCollection(t)
	h.addThoughtToCollection(t, h.firstThought)
	h.recordThoughtEngagement(t, h.secondThought)

	payload := h.exec(t, queryPersonalization, map[string]interface{}{"collectionId": h.collectionID})
	assertGraphQLRootKeys(t, payload, "me", "viewer", "viewerInterests", "myThoughts", "currents", "home", "collection", "collections")
	assertPayloadContains(t, payload, h.collectionID)
	assertPayloadContains(t, payload, h.firstThought)
}

func newGraphQLHarness(t *testing.T) *graphQLHarness {
	t.Helper()

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
	server, err := api.NewGraphQLServer(service)
	if err != nil {
		t.Fatalf("new graphql server: %v", err)
	}

	return &graphQLHarness{
		server:  server,
		service: service,
	}
}

func (h *graphQLHarness) seedThoughts(t *testing.T) {
	t.Helper()

	firstPayload := h.exec(t, mutationCreateThought, map[string]interface{}{
		"content": "Stoicism helps founders keep discipline under pressure.",
	})
	h.firstThought = mustExtractGraphQLID(t, firstPayload, "createThought")

	secondPayload := h.exec(t, mutationCreateThought, map[string]interface{}{
		"content": "Boxing makes discipline tangible through repeated practice.",
	})
	h.secondThought = mustExtractGraphQLID(t, secondPayload, "createThought")

	h.drain(t, 24, "after seeding thoughts")
}

func (h *graphQLHarness) createCollection(t *testing.T) {
	t.Helper()

	payload := h.exec(t, mutationCreateCollection, map[string]interface{}{
		"title":       "Founder Canon",
		"description": "Thoughts worth keeping",
	})
	h.collectionID = mustExtractGraphQLID(t, payload, "createCollection")
}

func (h *graphQLHarness) addThoughtToCollection(t *testing.T, thoughtID string) {
	t.Helper()
	payload := h.exec(t, mutationAddThoughtToCollection, map[string]interface{}{
		"collectionId": h.collectionID,
		"thoughtId":    thoughtID,
	})
	assertGraphQLRootKeys(t, payload, "addThoughtToCollection")
}

func (h *graphQLHarness) recordThoughtEngagement(t *testing.T, thoughtID string) {
	t.Helper()
	payload := h.exec(t, mutationRecordEngagement, map[string]interface{}{
		"entityType": "thought",
		"entityId":   thoughtID,
		"actionType": "open",
		"dwellMs":    4200,
	})
	assertGraphQLRootKeys(t, payload, "recordEngagement")
}

func (h *graphQLHarness) drain(t *testing.T, maxJobs int, label string) {
	t.Helper()
	if err := h.service.DrainJobs(maxJobs); err != nil {
		t.Fatalf("drain jobs %s: %v", label, err)
	}
}

func (h *graphQLHarness) exec(t *testing.T, query string, variables map[string]interface{}) []byte {
	t.Helper()

	body, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		t.Fatalf("marshal graphql body: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	response := httptest.NewRecorder()
	h.server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload graphQLIntegrationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode graphql response: %v", err)
	}
	if len(payload.Errors) > 0 {
		t.Fatalf("unexpected graphql errors: %s", response.Body.String())
	}
	return response.Body.Bytes()
}

func assertGraphQLRootKeys(t *testing.T, payload []byte, keys ...string) {
	t.Helper()

	var response graphQLIntegrationResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode graphql payload: %v", err)
	}

	for _, key := range keys {
		if _, exists := response.Data[key]; !exists {
			t.Fatalf("expected graphql data key %q in payload %s", key, string(payload))
		}
	}
}

func mustExtractGraphQLID(t *testing.T, payload []byte, rootKey string) string {
	t.Helper()

	var response graphQLIntegrationResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode graphql payload: %v", err)
	}

	raw, exists := response.Data[rootKey]
	if !exists {
		t.Fatalf("expected root key %q in payload %s", rootKey, string(payload))
	}

	var data struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("decode %s payload: %v", rootKey, err)
	}
	if data.ID == "" {
		t.Fatalf("expected non-empty id in %s payload %s", rootKey, string(raw))
	}
	return data.ID
}

func assertPayloadContains(t *testing.T, payload []byte, needle string) {
	t.Helper()
	if !bytes.Contains(payload, []byte(needle)) {
		t.Fatalf("expected payload to contain %q, got %s", needle, string(payload))
	}
}
