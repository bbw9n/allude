package allude

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type GraphQLHandler struct {
	service *Service
}

func NewGraphQLHandler(service *Service) *GraphQLHandler {
	return &GraphQLHandler{service: service}
}

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type graphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []graphQLError `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

func (handler *GraphQLHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload graphQLRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		handler.writeError(writer, http.StatusBadRequest, err)
		return
	}

	data, err := handler.route(payload)
	if err != nil {
		handler.writeError(writer, http.StatusOK, err)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(graphQLResponse{Data: data})
}

func (handler *GraphQLHandler) route(payload graphQLRequest) (map[string]interface{}, error) {
	switch {
	case strings.Contains(payload.Query, "createThought"):
		content := getString(payload.Variables, "content")
		thought, err := handler.service.CreateThought(content)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"createThought": thought}, nil
	case strings.Contains(payload.Query, "updateThought"):
		thought, err := handler.service.UpdateThought(getString(payload.Variables, "thoughtId"), getString(payload.Variables, "content"))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"updateThought": thought}, nil
	case strings.Contains(payload.Query, "searchThoughts"):
		result, err := handler.service.SearchThoughts(getString(payload.Variables, "query"))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"searchThoughts": result}, nil
	case strings.Contains(payload.Query, "graph("):
		graph, err := handler.service.Graph(
			getString(payload.Variables, "centerThoughtId"),
			getInt(payload.Variables, "distance", 2),
			getInt(payload.Variables, "limit", 12),
		)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"graph": graph}, nil
	case strings.Contains(payload.Query, "concept("):
		concept, err := handler.service.Concept(getString(payload.Variables, "id"), getString(payload.Variables, "name"))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"concept": concept}, nil
	case strings.Contains(payload.Query, "listThoughtVersions"):
		versions, err := handler.service.ThoughtVersions(getString(payload.Variables, "thoughtId"))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"listThoughtVersions": versions}, nil
	case strings.Contains(payload.Query, "relatedThoughts"):
		thoughts, err := handler.service.RelatedThoughts(getString(payload.Variables, "thoughtId"), getInt(payload.Variables, "limit", 8))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"relatedThoughts": thoughts}, nil
	case strings.Contains(payload.Query, "thought("):
		thought, err := handler.service.Thought(getString(payload.Variables, "id"))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"thought": thought}, nil
	case strings.Contains(payload.Query, "viewer"):
		return map[string]interface{}{"viewer": handler.service.Viewer()}, nil
	default:
		return nil, errors.New("unsupported GraphQL operation for the MVP handler")
	}
}

func (handler *GraphQLHandler) writeError(writer http.ResponseWriter, status int, err error) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(graphQLResponse{
		Errors: []graphQLError{{Message: err.Error()}},
	})
}

func getString(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	if value, exists := values[key]; exists {
		switch typed := value.(type) {
		case string:
			return typed
		}
	}
	return ""
}

func getInt(values map[string]interface{}, key string, fallback int) int {
	if values == nil {
		return fallback
	}
	if value, exists := values[key]; exists {
		switch typed := value.(type) {
		case float64:
			return int(typed)
		case int:
			return typed
		}
	}
	return fallback
}
