package graphql_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	actions "github.com/bbw9n/allude/services/api/internal/actions"
	"github.com/bbw9n/allude/services/api/internal/pkgs/ai"
	api "github.com/bbw9n/allude/services/api/internal/pkgs/graphql"
	memstore "github.com/bbw9n/allude/services/api/internal/pkgs/storage/memory"
)

type graphQLResponse struct {
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

func TestGraphQLThoughtLifecycleFlow(t *testing.T) {
	h := newGraphQLHarness(t)
	h.seedThoughts(t)

	editPayload := h.exec(t, mutationEditThought, map[string]interface{}{
		"thoughtId": h.firstThought,
		"content":   "Stoicism and boxing both train discipline under deliberate discomfort.",
	})
	assertGraphQLHasData(t, editPayload, "editThought")

	h.drain(t, 16, "after edit thought")

	payload := h.exec(t, queryThoughtLifecycle, map[string]interface{}{"id": h.firstThought})
	assertGraphQLHasData(t, payload, "thought", "relatedThoughts", "listThoughtVersions")
	assertPayloadContains(t, payload, h.firstThought)
	assertPayloadContains(t, payload, h.secondThought)
}

func TestGraphQLDiscoveryFlow(t *testing.T) {
	h := newGraphQLHarness(t)
	h.seedThoughts(t)
	h.createCollection(t)
	h.addThoughtToCollection(t, h.firstThought)

	searchPayload := h.exec(t, queryDiscoverySearch, map[string]interface{}{"query": "discipline"})
	assertGraphQLHasData(t, searchPayload, "search", "searchThoughts")

	conceptPayload := h.exec(t, queryConceptByName, map[string]interface{}{"name": "discipline"})
	assertGraphQLHasData(t, conceptPayload, "concept")

	conceptSlugPayload := h.exec(t, queryConceptBySlug, map[string]interface{}{"slug": "discipline"})
	assertGraphQLHasData(t, conceptSlugPayload, "concept")

	graphPayload := h.exec(t, queryDiscoveryGraph, map[string]interface{}{"thoughtId": h.firstThought})
	assertGraphQLHasData(t, graphPayload, "graph")

	discoveryPayload := h.exec(t, queryDiscoverySurface, map[string]interface{}{
		"query":        "connections between stoicism and discipline",
		"thoughtId":    h.firstThought,
		"collectionId": h.collectionID,
	})
	assertGraphQLHasData(t, discoveryPayload, "currents", "home", "collection", "collections", "draftSuggestions", "telescope")
	assertPayloadContains(t, discoveryPayload, "narrative")
}

func TestGraphQLCollectionsAndPersonalizationFlow(t *testing.T) {
	h := newGraphQLHarness(t)
	h.seedThoughts(t)

	createPayload := h.exec(t, mutationCreateThought, map[string]interface{}{
		"content": "Boredom creates space for reflection.",
	})
	createdID := mustExtractID(t, createPayload, "createThought")
	h.drain(t, 8, "after create thought")

	updatePayload := h.exec(t, mutationUpdateThought, map[string]interface{}{
		"thoughtId": createdID,
		"content":   "Boredom creates space for reflection, recovery, and new ideas.",
	})
	assertGraphQLHasData(t, updatePayload, "updateThought")
	h.drain(t, 8, "after update thought")

	h.createCollection(t)
	h.addThoughtToCollection(t, createdID)
	h.recordThoughtEngagement(t, createdID)

	payload := h.exec(t, queryViewerAndPersonalization, map[string]interface{}{"collectionId": h.collectionID})
	assertGraphQLHasData(t, payload, "me", "viewer", "viewerInterests", "myThoughts", "currents", "home", "collection", "collections")
	assertPayloadContains(t, payload, h.collectionID)
}

func TestGraphQLCaptureInboxFlow(t *testing.T) {
	h := newGraphQLHarness(t)

	createPayload := h.exec(t, mutationCreateCapture, map[string]interface{}{
		"content":     "Clip this quote about boredom and creativity.",
		"sourceType":  "quote",
		"sourceTitle": "On Boredom",
		"sourceUrl":   "https://example.com/boredom",
		"sourceApp":   "Safari",
	})
	captureID := mustExtractID(t, createPayload, "createCapture")

	inboxPayload := h.exec(t, queryInbox, nil)
	assertGraphQLHasData(t, inboxPayload, "inbox")
	assertPayloadContains(t, inboxPayload, captureID)

	promotePayload := h.exec(t, mutationPromoteCapture, map[string]interface{}{
		"captureId": captureID,
	})
	assertGraphQLHasData(t, promotePayload, "promoteCapture")
	assertPayloadContains(t, promotePayload, "promoted")

	h.drain(t, 8, "after promote capture")

	secondCreatePayload := h.exec(t, mutationCreateCapture, map[string]interface{}{
		"content": "Archive me",
	})
	secondCaptureID := mustExtractID(t, secondCreatePayload, "createCapture")

	archivePayload := h.exec(t, mutationArchiveCapture, map[string]interface{}{
		"captureId": secondCaptureID,
	})
	assertGraphQLHasData(t, archivePayload, "archiveCapture")
	assertPayloadContains(t, archivePayload, "archived")

	previewPayload := h.exec(t, queryCapturePreview, map[string]interface{}{
		"id": captureID,
	})
	assertGraphQLHasData(t, previewPayload, "capture")
	assertPayloadContains(t, previewPayload, "preview")
}

func TestGraphQLCurrentsExposeMaterializedDiscoveryFields(t *testing.T) {
	h := newGraphQLHarness(t)

	_, _ = h.service.CreateThought("Stoicism helps founders build discipline under pressure.")
	_, _ = h.service.CreateThought("Boxing makes discipline tangible through repeated practice.")
	h.drain(t, 24, "after currents seed")

	payload := h.exec(t, queryCurrentsRich, nil)

	var response graphQLResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode graphql payload: %v", err)
	}

	rawCurrents, exists := response.Data["currents"]
	if !exists {
		t.Fatalf("expected currents in payload %s", string(payload))
	}

	var currents []struct {
		ID             string  `json:"id"`
		Title          string  `json:"title"`
		Summary        string  `json:"summary"`
		ClusterKey     string  `json:"clusterKey"`
		FreshnessScore float64 `json:"freshnessScore"`
		QualityScore   float64 `json:"qualityScore"`
		Thoughts       []struct {
			ID string `json:"id"`
		} `json:"thoughts"`
		Concepts []struct {
			CanonicalName string `json:"canonicalName"`
		} `json:"concepts"`
	}
	if err := json.Unmarshal(rawCurrents, &currents); err != nil {
		t.Fatalf("decode currents payload: %v", err)
	}
	if len(currents) == 0 {
		t.Fatal("expected at least one current")
	}
	if currents[0].Summary == "" {
		t.Fatal("expected current summary to be populated")
	}
	if currents[0].ClusterKey == "" {
		t.Fatal("expected current cluster key to be populated")
	}
	if len(currents[0].Thoughts) == 0 {
		t.Fatal("expected current thoughts to be populated")
	}
	if len(currents[0].Concepts) == 0 {
		t.Fatal("expected current concepts to be populated")
	}
}

func TestGraphQLServerHandlesMethodAndPayloadErrors(t *testing.T) {
	server, _ := newTestGraphQLServer(t)

	methodRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	methodResponse := httptest.NewRecorder()
	server.ServeHTTP(methodResponse, methodRequest)
	if methodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d", methodResponse.Code)
	}

	badJSONRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{not-json"))
	badJSONResponse := httptest.NewRecorder()
	server.ServeHTTP(badJSONResponse, badJSONRequest)
	if badJSONResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d", badJSONResponse.Code)
	}

	var payload graphQLResponse
	if err := json.Unmarshal(badJSONResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bad json response: %v", err)
	}
	if len(payload.Errors) == 0 {
		t.Fatalf("expected graphql-style errors, got %s", badJSONResponse.Body.String())
	}
}

func newGraphQLHarness(t *testing.T) *graphQLHarness {
	t.Helper()
	server, service := newTestGraphQLServer(t)
	return &graphQLHarness{server: server, service: service}
}

func (h *graphQLHarness) seedThoughts(t *testing.T) {
	t.Helper()
	firstPayload := h.exec(t, mutationCreateThought, map[string]interface{}{
		"content": "Stoicism and boxing both train discipline under discomfort.",
	})
	h.firstThought = mustExtractID(t, firstPayload, "createThought")

	secondPayload := h.exec(t, mutationCreateThought, map[string]interface{}{
		"content": "Founder psychology borrows discipline rituals from martial arts.",
	})
	h.secondThought = mustExtractID(t, secondPayload, "createThought")

	h.exec(t, mutationCreateThought, map[string]interface{}{
		"content": "Creativity often needs boredom and silence.",
	})

	h.drain(t, 24, "after seeding thoughts")
}

func (h *graphQLHarness) createCollection(t *testing.T) {
	t.Helper()
	payload := h.exec(t, mutationCreateCollection, map[string]interface{}{
		"title":       "Research",
		"description": "Ideas worth keeping",
	})
	h.collectionID = mustExtractID(t, payload, "createCollection")
}

func (h *graphQLHarness) addThoughtToCollection(t *testing.T, thoughtID string) {
	t.Helper()
	payload := h.exec(t, mutationAddThoughtToCollection, map[string]interface{}{
		"collectionId": h.collectionID,
		"thoughtId":    thoughtID,
	})
	assertGraphQLHasData(t, payload, "addThoughtToCollection")
}

func (h *graphQLHarness) recordThoughtEngagement(t *testing.T, thoughtID string) {
	t.Helper()
	payload := h.exec(t, mutationRecordEngagement, map[string]interface{}{
		"entityType": "thought",
		"entityId":   thoughtID,
		"actionType": "open",
		"dwellMs":    3200,
	})
	assertGraphQLHasData(t, payload, "recordEngagement")
}

func (h *graphQLHarness) drain(t *testing.T, maxJobs int, label string) {
	t.Helper()
	if err := h.service.DrainJobs(maxJobs); err != nil {
		t.Fatalf("drain jobs %s: %v", label, err)
	}
}

func (h *graphQLHarness) exec(t *testing.T, query string, variables map[string]interface{}) []byte {
	t.Helper()
	return executeGraphQL(t, h.server, query, variables)
}

func newTestGraphQLServer(t *testing.T) (*api.GraphQLServer, *actions.Service) {
	t.Helper()
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	server, err := api.NewGraphQLServer(service)
	if err != nil {
		t.Fatalf("new graphql server: %v", err)
	}
	return server, service
}

func executeGraphQL(t *testing.T, server *api.GraphQLServer, query string, variables map[string]interface{}) []byte {
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
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", response.Code, response.Body.String())
	}
	var payload graphQLResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode graphql response: %v", err)
	}
	if len(payload.Errors) > 0 {
		t.Fatalf("unexpected graphql errors: %s", response.Body.String())
	}
	return response.Body.Bytes()
}

func assertGraphQLHasData(t *testing.T, payload []byte, keys ...string) {
	t.Helper()
	var response graphQLResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode graphql payload: %v", err)
	}
	for _, key := range keys {
		if _, exists := response.Data[key]; !exists {
			t.Fatalf("expected graphql data key %q in payload %s", key, string(payload))
		}
	}
}

func mustExtractID(t *testing.T, payload []byte, rootKey string) string {
	t.Helper()
	var response graphQLResponse
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
