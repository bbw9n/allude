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

func TestGraphQLQueriesCoverPrimaryReadSurface(t *testing.T) {
	server, service := newTestGraphQLServer(t)

	first, _ := service.CreateThought("Stoicism and boxing both train discipline under discomfort.")
	second, _ := service.CreateThought("Founder psychology borrows discipline rituals from martial arts.")
	_, _ = service.CreateThought("Creativity often needs boredom and silence.")
	if err := service.DrainJobs(24); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}
	collection, err := service.CreateCollection("Research", "Ideas worth keeping")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if _, err := service.AddThoughtToCollection(collection.ID, first.ID); err != nil {
		t.Fatalf("add thought to collection: %v", err)
	}

	assertGraphQLHasData(t, executeGraphQL(t, server, "query { me { id username } viewer { id username } }", nil), "me", "viewer")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query { myThoughts(limit: 10) { id currentVersion { content } } }", nil), "myThoughts")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query Thought($id: ID!) { thought(id: $id) { id concepts { canonicalName } relatedThoughts { id } versions { id version } } }", map[string]interface{}{"id": first.ID}), "thought")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query ConceptByName($name: String!) { concept(name: $name) { id canonicalName topThoughts { id } relatedConcepts { id } thoughtCount } }", map[string]interface{}{"name": "discipline"}), "concept")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query ConceptBySlug($slug: String!) { concept(slug: $slug) { id canonicalName } }", map[string]interface{}{"slug": "discipline"}), "concept")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query Search($query: String!) { search(query: $query) { thoughts { id } clusters { label thoughtIds } } searchThoughts(query: $query) { thoughts { id } } }", map[string]interface{}{"query": "discipline"}), "search", "searchThoughts")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query Draft($content: String!) { draftSuggestions(content: $content) { relatedConcepts reframes supportingThoughts { id } counterThoughts { id } notes } }", map[string]interface{}{"content": "Stoicism builds discipline"}), "draftSuggestions")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query Telescope($query: String!) { telescope(query: $query) { query intent narrative seedConcepts { id canonicalName } seedThoughts { id } graph { center { thought { id } } } clusters { label } suggestedJumps { label query } relatedCurrents { id } } }", map[string]interface{}{"query": "connections between stoicism and discipline"}), "telescope")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query Graph($thoughtId: ID!) { graph(thoughtId: $thoughtId, hopCount: 2, limit: 12) { center { thought { id } } nodes { thought { id } } edges { link { id relationType } } } }", map[string]interface{}{"thoughtId": first.ID}), "graph")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query Collection($id: ID!) { collection(id: $id) { id title items { thought { id } } } collections { id title } }", map[string]interface{}{"id": collection.ID}), "collection", "collections")
	assertGraphQLHasData(t, executeGraphQL(t, server, "query Discovery { currents(limit: 4) { id title thoughts { id } concepts { id } } home(limit: 4) { viewer { id } currents { id } recommendedThoughts { id } recommendedCollections { id } } }", nil), "currents", "home")

	relatedPayload := executeGraphQL(t, server, "query Thought($id: ID!) { thought(id: $id) { relatedThoughts { id } } }", map[string]interface{}{"id": first.ID})
	if !bytes.Contains(relatedPayload, []byte(second.ID)) {
		t.Fatalf("expected related thought %s in payload %s", second.ID, string(relatedPayload))
	}
}

func TestGraphQLMutationsCoverPrimaryWriteSurface(t *testing.T) {
	server, service := newTestGraphQLServer(t)

	createPayload := executeGraphQL(t, server, "mutation CreateThought($content: String!) { createThought(content: $content) { id currentVersion { id version content } processingStatus } }", map[string]interface{}{"content": "Boredom creates space for reflection."})
	createdID := mustExtractID(t, createPayload, "createThought")
	if err := service.DrainJobs(8); err != nil {
		t.Fatalf("drain jobs after create: %v", err)
	}

	assertGraphQLHasData(t, executeGraphQL(t, server, "mutation EditThought($thoughtId: ID!, $content: String!) { editThought(thoughtId: $thoughtId, content: $content) { id currentVersion { version content } } }", map[string]interface{}{"thoughtId": createdID, "content": "Boredom creates space for reflection and creative recovery."}), "editThought")
	if err := service.DrainJobs(8); err != nil {
		t.Fatalf("drain jobs after edit: %v", err)
	}

	assertGraphQLHasData(t, executeGraphQL(t, server, "mutation UpdateThought($thoughtId: ID!, $content: String!) { updateThought(thoughtId: $thoughtId, content: $content) { id currentVersion { version content } } }", map[string]interface{}{"thoughtId": createdID, "content": "Boredom creates space for reflection, recovery, and new ideas."}), "updateThought")
	if err := service.DrainJobs(8); err != nil {
		t.Fatalf("drain jobs after update: %v", err)
	}

	collectionPayload := executeGraphQL(t, server, "mutation CreateCollection($title: String!, $description: String) { createCollection(title: $title, description: $description) { id title description } }", map[string]interface{}{"title": "Creative Inputs", "description": "Notes about creative recovery"})
	collectionID := mustExtractID(t, collectionPayload, "createCollection")

	assertGraphQLHasData(t, executeGraphQL(t, server, "mutation AddToCollection($collectionId: ID!, $thoughtId: ID!) { addThoughtToCollection(collectionId: $collectionId, thoughtId: $thoughtId) { id items { thought { id } } } }", map[string]interface{}{"collectionId": collectionID, "thoughtId": createdID}), "addThoughtToCollection")
	assertGraphQLHasData(t, executeGraphQL(t, server, "mutation RecordEngagement($entityType: String!, $entityId: ID!, $actionType: String!, $dwellMs: Int) { recordEngagement(entityType: $entityType, entityId: $entityId, actionType: $actionType, dwellMs: $dwellMs) { id entityType entityId actionType dwellMs } }", map[string]interface{}{"entityType": "thought", "entityId": createdID, "actionType": "open", "dwellMs": 3200}), "recordEngagement")
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
