package actions_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	actions "github.com/bbw9n/allude/services/api/internal/actions"
	models "github.com/bbw9n/allude/services/api/internal/domains/models"
	"github.com/bbw9n/allude/services/api/internal/pkgs/ai"
	apiGraphql "github.com/bbw9n/allude/services/api/internal/pkgs/graphql"
	memstore "github.com/bbw9n/allude/services/api/internal/pkgs/storage/memory"
)

func TestCreateThoughtQueuesJobsAndPreservesVersions(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	created, err := service.CreateThought("Stoicism helps founders endure uncertainty and pressure.")
	if err != nil {
		t.Fatalf("create thought: %v", err)
	}
	if len(service.Jobs()) != 1 {
		t.Fatalf("expected queued job, got %d", len(service.Jobs()))
	}
	if err := service.DrainJobs(8); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}
	hydrated, err := service.Thought(created.ID)
	if err != nil {
		t.Fatalf("fetch thought: %v", err)
	}
	if hydrated.CurrentVersion.VersionNo != 1 {
		t.Fatalf("expected version 1, got %d", hydrated.CurrentVersion.VersionNo)
	}
	if hydrated.ProcessingStatus != models.ProcessingReady {
		t.Fatalf("expected ready status, got %s", hydrated.ProcessingStatus)
	}
	if len(hydrated.Concepts) == 0 {
		t.Fatal("expected extracted concepts")
	}

	updated, err := service.EditThought(created.ID, "Stoicism helps founders endure uncertainty and refine judgment.")
	if err != nil {
		t.Fatalf("edit thought: %v", err)
	}
	if updated.CurrentVersion.VersionNo != 2 {
		t.Fatalf("expected version 2, got %d", updated.CurrentVersion.VersionNo)
	}
}

func TestGraphAndConceptQueriesAfterEnrichment(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	first, _ := service.CreateThought("Stoicism and boxing both train discipline under discomfort.")
	second, _ := service.CreateThought("Founder psychology borrows discipline rituals from martial arts.")
	_, _ = service.CreateThought("Cities can make loneliness feel louder than solitude.")
	if err := service.DrainJobs(20); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}

	graph, err := service.Graph(first.ID, 2, 12)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if graph.Center.Thought.ID != first.ID {
		t.Fatalf("expected center %s, got %s", first.ID, graph.Center.Thought.ID)
	}
	hydratedFirst, err := service.Thought(first.ID)
	if err != nil {
		t.Fatalf("fetch hydrated thought: %v", err)
	}
	related := hydratedFirst.RelatedThoughts
	found := false
	for _, thought := range related {
		if thought.ID == second.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("expected linked related thought")
	}

	concept, err := service.Concept("", "", "discipline")
	if err != nil {
		t.Fatalf("concept lookup: %v", err)
	}
	if concept == nil || len(concept.TopThoughts) == 0 {
		t.Fatal("expected concept page to include top thoughts")
	}
	if concept.ThoughtCount == 0 {
		t.Fatal("expected concept page to include thought count")
	}
}

func TestHybridSearchAndCollections(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	created, _ := service.CreateThought("Boredom and creativity need silence.")
	if err := service.DrainJobs(8); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}
	collection, err := service.CreateCollection("Creative Inputs", "Notes about creative recovery")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if _, err := service.AddThoughtToCollection(collection.ID, created.ID); err != nil {
		t.Fatalf("add thought to collection: %v", err)
	}
	result, err := service.SearchThoughts("creativity silence")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Thoughts) == 0 {
		t.Fatal("expected search results")
	}

	collections, err := service.Collections()
	if err != nil {
		t.Fatalf("list collections: %v", err)
	}
	if len(collections) == 0 {
		t.Fatal("expected collections list to include created collection")
	}
}

func TestDraftSuggestionsReturnsConceptsAndRelatedThoughts(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	_, _ = service.CreateThought("Stoicism helps founders build discipline under pressure.")
	_, _ = service.CreateThought("Boxing turns discipline into a physical practice.")
	_, _ = service.CreateThought("Founders should not treat stoicism as emotional suppression.")
	if err := service.DrainJobs(24); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}

	suggestions, err := service.DraftSuggestions("Stoicism can help founders stay disciplined during hard periods.", "")
	if err != nil {
		t.Fatalf("draft suggestions: %v", err)
	}
	if len(suggestions.RelatedConcepts) == 0 {
		t.Fatal("expected related concepts")
	}
	if len(suggestions.SupportingThoughts) == 0 {
		t.Fatal("expected supporting thoughts")
	}
	if len(suggestions.Reframes) == 0 {
		t.Fatal("expected reframes")
	}
}

func TestMyThoughtsReturnsViewerThoughts(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	_, _ = service.CreateThought("Stoicism helps me recover focus.")
	_, _ = service.CreateThought("Boxing keeps the body aligned with discipline.")
	if err := service.DrainJobs(16); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}

	thoughts, err := service.MyThoughts(20)
	if err != nil {
		t.Fatalf("my thoughts: %v", err)
	}
	if len(thoughts) < 2 {
		t.Fatalf("expected at least 2 thoughts, got %d", len(thoughts))
	}
}

func TestCurrentsAndHomeReturnDiscoveryPayloads(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	createdOne, _ := service.CreateThought("Stoicism helps founders hold steady under pressure.")
	createdTwo, _ := service.CreateThought("Boxing turns discipline into a daily ritual.")
	createdThree, _ := service.CreateThought("Creativity often needs boredom and silence.")
	if err := service.DrainJobs(24); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}
	collection, err := service.CreateCollection("Founder Notes", "Notes worth revisiting")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if _, err := service.AddThoughtToCollection(collection.ID, createdOne.ID); err != nil {
		t.Fatalf("add first thought to collection: %v", err)
	}
	if _, err := service.AddThoughtToCollection(collection.ID, createdTwo.ID); err != nil {
		t.Fatalf("add second thought to collection: %v", err)
	}
	if _, err := service.RecordEngagement("thought", createdThree.ID, "open", 4200); err != nil {
		t.Fatalf("record engagement: %v", err)
	}

	currents, err := service.Currents(4)
	if err != nil {
		t.Fatalf("currents: %v", err)
	}
	if len(currents) == 0 {
		t.Fatal("expected at least one current")
	}
	if len(currents[0].Thoughts) == 0 {
		t.Fatal("expected current to include clustered thoughts")
	}

	home, err := service.Home(4)
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	if home.Viewer == nil {
		t.Fatal("expected home payload to include viewer")
	}
	if len(home.Currents) == 0 {
		t.Fatal("expected home payload to include currents")
	}
	if len(home.RecommendedThoughts) == 0 {
		t.Fatal("expected home payload to include recommended thoughts")
	}
	if len(home.RecommendedCollections) == 0 {
		t.Fatal("expected home payload to include recommended collections")
	}
}

func TestUserInterestsAndRankedHomeReflectBehavior(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	stoic, _ := service.CreateThought("Stoicism helps founders keep discipline under pressure.")
	creative, _ := service.CreateThought("Creativity needs boredom, silence, and recovery.")
	if err := service.DrainJobs(16); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}

	collection, err := service.CreateCollection("Founder Canon", "Thoughts worth keeping")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if _, err := service.AddThoughtToCollection(collection.ID, stoic.ID); err != nil {
		t.Fatalf("add thought to collection: %v", err)
	}
	if _, err := service.RecordEngagement("thought", stoic.ID, "open", 4200); err != nil {
		t.Fatalf("record engagement: %v", err)
	}
	if _, err := service.Telescope("connections between stoicism and discipline"); err != nil {
		t.Fatalf("telescope: %v", err)
	}

	interests, err := service.ViewerInterests(6)
	if err != nil {
		t.Fatalf("viewer interests: %v", err)
	}
	if len(interests) == 0 {
		t.Fatal("expected personalized user interests")
	}

	home, err := service.Home(4)
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	if len(home.RecommendedThoughts) == 0 {
		t.Fatal("expected recommended thoughts")
	}
	if home.RecommendedThoughts[0].ID != stoic.ID {
		t.Fatalf("expected personalized home to prefer stoic thought %s, got %s", stoic.ID, home.RecommendedThoughts[0].ID)
	}
	if len(home.Currents) == 0 {
		t.Fatal("expected ranked currents")
	}
	if len(home.Viewer.Interests) == 0 {
		t.Fatal("expected viewer interests to be populated from profile")
	}
	_ = creative
}

func TestTelescopeReturnsStructuredDiscoveryPayload(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	_, _ = service.CreateThought("Stoicism and boxing both train discipline under discomfort.")
	_, _ = service.CreateThought("Founder psychology borrows discipline rituals from martial arts.")
	_, _ = service.CreateThought("Founders sometimes misuse stoicism as emotional suppression.")
	if err := service.DrainJobs(24); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}

	result, err := service.Telescope("connections between stoicism and discipline")
	if err != nil {
		t.Fatalf("telescope: %v", err)
	}
	if result.Intent != "connect" {
		t.Fatalf("expected connect intent, got %s", result.Intent)
	}
	if len(result.SeedThoughts) == 0 {
		t.Fatal("expected telescope to return seed thoughts")
	}
	if len(result.Clusters) == 0 {
		t.Fatal("expected telescope to return clusters")
	}
	if result.Graph == nil || result.Graph.Center == nil {
		t.Fatal("expected telescope to include a graph neighborhood")
	}
	if len(result.SuggestedJumps) == 0 {
		t.Fatal("expected telescope to return suggested jumps")
	}
	if result.Narrative == "" {
		t.Fatal("expected telescope to return a narrative")
	}
}

func TestGraphQLServerSupportsQueriesAndMutations(t *testing.T) {
	service := actions.NewService(memstore.NewInMemoryRepository(), &ai.StubAIProvider{})
	server, err := apiGraphql.NewGraphQLServer(service)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"query": "mutation CreateThought($content: String!) { createThought(content: $content) { id processingStatus currentVersion { id version content } } }",
		"variables": map[string]interface{}{
			"content": "Boredom creates space for reflection.",
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if err := service.DrainJobs(8); err != nil {
		t.Fatalf("drain jobs: %v", err)
	}

	searchBody, _ := json.Marshal(map[string]interface{}{
		"query": "{ searchThoughts(query: \"reflection\") { thoughts { id concepts { canonicalName } } } }",
	})
	searchRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(searchBody))
	searchResponse := httptest.NewRecorder()
	server.ServeHTTP(searchResponse, searchRequest)
	if !bytes.Contains(searchResponse.Body.Bytes(), []byte("searchThoughts")) {
		t.Fatalf("expected searchThoughts payload, got %s", searchResponse.Body.String())
	}

	suggestionBody, _ := json.Marshal(map[string]interface{}{
		"query": "{ draftSuggestions(content: \"Stoicism builds discipline\") { relatedConcepts reframes } }",
	})
	suggestionRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(suggestionBody))
	suggestionResponse := httptest.NewRecorder()
	server.ServeHTTP(suggestionResponse, suggestionRequest)
	if !bytes.Contains(suggestionResponse.Body.Bytes(), []byte("draftSuggestions")) {
		t.Fatalf("expected draftSuggestions payload, got %s", suggestionResponse.Body.String())
	}

	collectionMutationBody, _ := json.Marshal(map[string]interface{}{
		"query": "mutation { createCollection(title: \"Research\") { id title } }",
	})
	collectionMutationRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(collectionMutationBody))
	collectionMutationResponse := httptest.NewRecorder()
	server.ServeHTTP(collectionMutationResponse, collectionMutationRequest)

	collectionsQueryBody, _ := json.Marshal(map[string]interface{}{
		"query": "{ collections { id title } }",
	})
	collectionsQueryRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(collectionsQueryBody))
	collectionsQueryResponse := httptest.NewRecorder()
	server.ServeHTTP(collectionsQueryResponse, collectionsQueryRequest)
	if !bytes.Contains(collectionsQueryResponse.Body.Bytes(), []byte("collections")) {
		t.Fatalf("expected collections payload, got %s", collectionsQueryResponse.Body.String())
	}

	myThoughtsBody, _ := json.Marshal(map[string]interface{}{
		"query": "{ myThoughts(limit: 10) { id } }",
	})
	myThoughtsRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(myThoughtsBody))
	myThoughtsResponse := httptest.NewRecorder()
	server.ServeHTTP(myThoughtsResponse, myThoughtsRequest)
	if !bytes.Contains(myThoughtsResponse.Body.Bytes(), []byte("myThoughts")) {
		t.Fatalf("expected myThoughts payload, got %s", myThoughtsResponse.Body.String())
	}

	currentsBody, _ := json.Marshal(map[string]interface{}{
		"query": "{ currents(limit: 4) { id title thoughts { id } } }",
	})
	currentsRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(currentsBody))
	currentsResponse := httptest.NewRecorder()
	server.ServeHTTP(currentsResponse, currentsRequest)
	if !bytes.Contains(currentsResponse.Body.Bytes(), []byte("currents")) {
		t.Fatalf("expected currents payload, got %s", currentsResponse.Body.String())
	}

	homeBody, _ := json.Marshal(map[string]interface{}{
		"query": "{ home(limit: 4) { viewer { id } currents { id } recommendedThoughts { id } } }",
	})
	homeRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(homeBody))
	homeResponse := httptest.NewRecorder()
	server.ServeHTTP(homeResponse, homeRequest)
	if !bytes.Contains(homeResponse.Body.Bytes(), []byte("recommendedThoughts")) {
		t.Fatalf("expected home payload, got %s", homeResponse.Body.String())
	}
}
