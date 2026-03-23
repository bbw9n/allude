package graphql

import (
	"encoding/json"
	"net/http"

	"github.com/bbw9n/allude/services/api/internal/actions"
	"github.com/bbw9n/allude/services/api/internal/domains/models"
	"github.com/graphql-go/graphql"
)

type Thought = models.Thought
type ThoughtVersion = models.ThoughtVersion
type ThoughtLink = models.ThoughtLink
type Concept = models.Concept

type GraphQLServer struct {
	service *actions.Service
	schema  graphql.Schema
}

func NewGraphQLServer(service *actions.Service) (*GraphQLServer, error) {
	server := &GraphQLServer{service: service}
	schema, err := server.buildSchema()
	if err != nil {
		return nil, err
	}
	server.schema = schema
	return server, nil
}

func (server *GraphQLServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{
			"errors": []map[string]string{{"message": err.Error()}},
		})
		return
	}

	result := graphql.Do(graphql.Params{
		Schema:         server.schema,
		RequestString:  payload.Query,
		OperationName:  payload.OperationName,
		VariableValues: payload.Variables,
		Context:        request.Context(),
	})

	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(result)
}

func (server *GraphQLServer) buildSchema() (graphql.Schema, error) {
	var thoughtType *graphql.Object
	var conceptType *graphql.Object
	var collectionType *graphql.Object
	var captureType *graphql.Object
	var ideaCurrentType *graphql.Object
	var telescopeResultType *graphql.Object

	userType := graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"username":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"displayName": &graphql.Field{Type: graphql.String},
			"bio":         &graphql.Field{Type: graphql.String},
			"avatarUrl":   &graphql.Field{Type: graphql.String},
			"interests":   &graphql.Field{Type: graphql.NewList(graphql.String)},
			"createdAt":   &graphql.Field{Type: graphql.String},
			"updatedAt":   &graphql.Field{Type: graphql.String},
		},
	})

	userInterestType := graphql.NewObject(graphql.ObjectConfig{
		Name: "UserInterest",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"userId":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"conceptId":     &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"affinityScore": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
				"source":        &graphql.Field{Type: graphql.String},
				"updatedAt":     &graphql.Field{Type: graphql.String},
				"concept":       &graphql.Field{Type: conceptType},
			}
		}),
	})

	thoughtVersionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ThoughtVersion",
		Fields: graphql.Fields{
			"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"thoughtId": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"version": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*models.ThoughtVersion).VersionNo, nil
				},
			},
			"content":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"language":         &graphql.Field{Type: graphql.String},
			"tokenCount":       &graphql.Field{Type: graphql.Int},
			"processingStatus": &graphql.Field{Type: graphql.String},
			"processingNotes":  &graphql.Field{Type: graphql.NewList(graphql.String)},
			"createdAt":        &graphql.Field{Type: graphql.String},
		},
	})

	thoughtLinkType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ThoughtLink",
		Fields: graphql.Fields{
			"id":              &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"sourceThoughtId": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"targetThoughtId": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"relationType":    &graphql.Field{Type: graphql.String},
			"weight":          &graphql.Field{Type: graphql.Float},
			"score": &graphql.Field{
				Type: graphql.Float,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*models.ThoughtLink).Weight, nil
				},
			},
			"source": &graphql.Field{Type: graphql.String},
			"origin": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*models.ThoughtLink).Source, nil
				},
			},
			"explanation": &graphql.Field{Type: graphql.String},
			"createdAt":   &graphql.Field{Type: graphql.String},
		},
	})

	graphEdgeType := graphql.NewObject(graphql.ObjectConfig{
		Name: "GraphEdge",
		Fields: graphql.Fields{
			"link": &graphql.Field{Type: thoughtLinkType},
		},
	})

	conceptAliasType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ConceptAlias",
		Fields: graphql.Fields{
			"id":              &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"conceptId":       &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"alias":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"normalizedAlias": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	conceptType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Concept",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"canonicalName": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"name": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return p.Source.(*models.Concept).CanonicalName, nil
					},
				},
				"slug":                  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"description":           &graphql.Field{Type: graphql.String},
				"conceptType":           &graphql.Field{Type: graphql.String},
				"thoughtCount":          &graphql.Field{Type: graphql.Int},
				"aliases":               &graphql.Field{Type: graphql.NewList(conceptAliasType)},
				"relatedConcepts":       &graphql.Field{Type: graphql.NewList(conceptType)},
				"topThoughts":           &graphql.Field{Type: graphql.NewList(thoughtType)},
				"contradictionThoughts": &graphql.Field{Type: graphql.NewList(thoughtType)},
				"createdAt":             &graphql.Field{Type: graphql.String},
				"updatedAt":             &graphql.Field{Type: graphql.String},
			}
		}),
	})

	collectionItemType := graphql.NewObject(graphql.ObjectConfig{
		Name: "CollectionItem",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"collectionId": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"thoughtId":    &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"position":     &graphql.Field{Type: graphql.Int},
				"addedAt":      &graphql.Field{Type: graphql.String},
				"thought":      &graphql.Field{Type: thoughtType},
			}
		}),
	})

	collectionType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Collection",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"curatorId":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"title":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description": &graphql.Field{Type: graphql.String},
			"visibility":  &graphql.Field{Type: graphql.String},
			"items":       &graphql.Field{Type: graphql.NewList(collectionItemType)},
			"createdAt":   &graphql.Field{Type: graphql.String},
			"updatedAt":   &graphql.Field{Type: graphql.String},
		},
	})

	captureType = graphql.NewObject(graphql.ObjectConfig{
		Name: "CaptureItem",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"id":                &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"authorId":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"content":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"sourceType":        &graphql.Field{Type: graphql.String},
				"sourceTitle":       &graphql.Field{Type: graphql.String},
				"sourceUrl":         &graphql.Field{Type: graphql.String},
				"sourceApp":         &graphql.Field{Type: graphql.String},
				"status":            &graphql.Field{Type: graphql.String},
				"promotedThoughtId": &graphql.Field{Type: graphql.ID},
				"promotedThought":   &graphql.Field{Type: thoughtType},
				"createdAt":         &graphql.Field{Type: graphql.String},
				"updatedAt":         &graphql.Field{Type: graphql.String},
			}
		}),
	})

	thoughtType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Thought",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"id":               &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"author":           &graphql.Field{Type: userType},
				"status":           &graphql.Field{Type: graphql.String},
				"visibility":       &graphql.Field{Type: graphql.String},
				"currentVersion":   &graphql.Field{Type: thoughtVersionType},
				"versions":         &graphql.Field{Type: graphql.NewList(thoughtVersionType)},
				"concepts":         &graphql.Field{Type: graphql.NewList(conceptType)},
				"relatedThoughts":  &graphql.Field{Type: graphql.NewList(thoughtType)},
				"links":            &graphql.Field{Type: graphql.NewList(thoughtLinkType)},
				"collections":      &graphql.Field{Type: graphql.NewList(collectionType)},
				"processingStatus": &graphql.Field{Type: graphql.String},
				"processingNotes":  &graphql.Field{Type: graphql.NewList(graphql.String)},
				"createdAt":        &graphql.Field{Type: graphql.String},
				"updatedAt":        &graphql.Field{Type: graphql.String},
			}
		}),
	})

	graphNodeType := graphql.NewObject(graphql.ObjectConfig{
		Name: "GraphNode",
		Fields: graphql.Fields{
			"thought":  &graphql.Field{Type: thoughtType},
			"x":        &graphql.Field{Type: graphql.Float},
			"y":        &graphql.Field{Type: graphql.Float},
			"distance": &graphql.Field{Type: graphql.Int},
		},
	})

	graphNeighborhoodType := graphql.NewObject(graphql.ObjectConfig{
		Name: "GraphNeighborhood",
		Fields: graphql.Fields{
			"center": &graphql.Field{Type: graphNodeType},
			"nodes":  &graphql.Field{Type: graphql.NewList(graphNodeType)},
			"edges":  &graphql.Field{Type: graphql.NewList(graphEdgeType)},
		},
	})

	searchClusterType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SearchCluster",
		Fields: graphql.Fields{
			"label":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"concepts":   &graphql.Field{Type: graphql.NewList(conceptType)},
			"thoughtIds": &graphql.Field{Type: graphql.NewList(graphql.ID)},
		},
	})

	searchResultType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SearchResult",
		Fields: graphql.Fields{
			"thoughts": &graphql.Field{Type: graphql.NewList(thoughtType)},
			"clusters": &graphql.Field{Type: graphql.NewList(searchClusterType)},
		},
	})

	draftSuggestionsType := graphql.NewObject(graphql.ObjectConfig{
		Name: "DraftSuggestions",
		Fields: graphql.Fields{
			"relatedConcepts":    &graphql.Field{Type: graphql.NewList(graphql.String)},
			"supportingThoughts": &graphql.Field{Type: graphql.NewList(thoughtType)},
			"counterThoughts":    &graphql.Field{Type: graphql.NewList(thoughtType)},
			"reframes":           &graphql.Field{Type: graphql.NewList(graphql.String)},
			"notes":              &graphql.Field{Type: graphql.NewList(graphql.String)},
		},
	})

	ideaCurrentType = graphql.NewObject(graphql.ObjectConfig{
		Name: "IdeaCurrent",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"id":             &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"title":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"summary":        &graphql.Field{Type: graphql.String},
				"clusterKey":     &graphql.Field{Type: graphql.String},
				"freshnessScore": &graphql.Field{Type: graphql.Float},
				"qualityScore":   &graphql.Field{Type: graphql.Float},
				"concepts":       &graphql.Field{Type: graphql.NewList(conceptType)},
				"thoughts":       &graphql.Field{Type: graphql.NewList(thoughtType)},
				"createdAt":      &graphql.Field{Type: graphql.String},
				"updatedAt":      &graphql.Field{Type: graphql.String},
			}
		}),
	})

	telescopeJumpType := graphql.NewObject(graphql.ObjectConfig{
		Name: "TelescopeJump",
		Fields: graphql.Fields{
			"label":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"query":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"reason":     &graphql.Field{Type: graphql.String},
			"thoughtIds": &graphql.Field{Type: graphql.NewList(graphql.ID)},
		},
	})

	telescopeResultType = graphql.NewObject(graphql.ObjectConfig{
		Name: "TelescopeResult",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"query":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"intent":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"seedConcepts":    &graphql.Field{Type: graphql.NewList(conceptType)},
				"seedThoughts":    &graphql.Field{Type: graphql.NewList(thoughtType)},
				"graph":           &graphql.Field{Type: graphNeighborhoodType},
				"clusters":        &graphql.Field{Type: graphql.NewList(searchClusterType)},
				"narrative":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"suggestedJumps":  &graphql.Field{Type: graphql.NewList(telescopeJumpType)},
				"relatedCurrents": &graphql.Field{Type: graphql.NewList(ideaCurrentType)},
			}
		}),
	})

	homeType := graphql.NewObject(graphql.ObjectConfig{
		Name: "HomePayload",
		Fields: graphql.Fields{
			"viewer":                 &graphql.Field{Type: userType},
			"currents":               &graphql.Field{Type: graphql.NewList(ideaCurrentType)},
			"recommendedThoughts":    &graphql.Field{Type: graphql.NewList(thoughtType)},
			"recommendedCollections": &graphql.Field{Type: graphql.NewList(collectionType)},
		},
	})

	engagementType := graphql.NewObject(graphql.ObjectConfig{
		Name: "EngagementEvent",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"userId":     &graphql.Field{Type: graphql.ID},
			"entityType": &graphql.Field{Type: graphql.String},
			"entityId":   &graphql.Field{Type: graphql.ID},
			"actionType": &graphql.Field{Type: graphql.String},
			"dwellMs":    &graphql.Field{Type: graphql.Int},
			"createdAt":  &graphql.Field{Type: graphql.String},
		},
	})

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"me": &graphql.Field{
				Type: userType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Viewer(), nil
				},
			},
			"viewer": &graphql.Field{
				Type: userType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Viewer(), nil
				},
			},
			"myThoughts": &graphql.Field{
				Type: graphql.NewList(thoughtType),
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.MyThoughts(optionalInt(p.Args, "limit", 40))
				},
			},
			"inbox": &graphql.Field{
				Type: graphql.NewList(captureType),
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Inbox(optionalInt(p.Args, "limit", 40))
				},
			},
			"capture": &graphql.Field{
				Type: captureType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Capture(p.Args["id"].(string))
				},
			},
			"viewerInterests": &graphql.Field{
				Type: graphql.NewList(userInterestType),
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.ViewerInterests(optionalInt(p.Args, "limit", 12))
				},
			},
			"currents": &graphql.Field{
				Type: graphql.NewList(ideaCurrentType),
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Currents(optionalInt(p.Args, "limit", 6))
				},
			},
			"home": &graphql.Field{
				Type: homeType,
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Home(optionalInt(p.Args, "limit", 6))
				},
			},
			"thought": &graphql.Field{
				Type: thoughtType,
				Args: graphql.FieldConfigArgument{"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)}},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Thought(p.Args["id"].(string))
				},
			},
			"concept": &graphql.Field{
				Type: conceptType,
				Args: graphql.FieldConfigArgument{
					"id":   &graphql.ArgumentConfig{Type: graphql.ID},
					"slug": &graphql.ArgumentConfig{Type: graphql.String},
					"name": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Concept(optionalString(p.Args, "id"), optionalString(p.Args, "slug"), optionalString(p.Args, "name"))
				},
			},
			"search": &graphql.Field{
				Type: searchResultType,
				Args: graphql.FieldConfigArgument{"query": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)}},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.SearchThoughts(p.Args["query"].(string))
				},
			},
			"searchThoughts": &graphql.Field{
				Type: searchResultType,
				Args: graphql.FieldConfigArgument{"query": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)}},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.SearchThoughts(p.Args["query"].(string))
				},
			},
			"draftSuggestions": &graphql.Field{
				Type: draftSuggestionsType,
				Args: graphql.FieldConfigArgument{
					"content":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"thoughtId": &graphql.ArgumentConfig{Type: graphql.ID},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.DraftSuggestions(
						p.Args["content"].(string),
						optionalString(p.Args, "thoughtId"),
					)
				},
			},
			"telescope": &graphql.Field{
				Type: telescopeResultType,
				Args: graphql.FieldConfigArgument{
					"query": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Telescope(p.Args["query"].(string))
				},
			},
			"graph": &graphql.Field{
				Type: graphNeighborhoodType,
				Args: graphql.FieldConfigArgument{
					"thoughtId":       &graphql.ArgumentConfig{Type: graphql.ID},
					"centerThoughtId": &graphql.ArgumentConfig{Type: graphql.ID},
					"hopCount":        &graphql.ArgumentConfig{Type: graphql.Int},
					"distance":        &graphql.ArgumentConfig{Type: graphql.Int},
					"limit":           &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					thoughtID := optionalString(p.Args, "thoughtId")
					if thoughtID == "" {
						thoughtID = optionalString(p.Args, "centerThoughtId")
					}
					hopCount := optionalInt(p.Args, "hopCount", 2)
					if _, exists := p.Args["distance"]; exists {
						hopCount = optionalInt(p.Args, "distance", 2)
					}
					return server.service.Graph(thoughtID, hopCount, optionalInt(p.Args, "limit", 12))
				},
			},
			"collection": &graphql.Field{
				Type: collectionType,
				Args: graphql.FieldConfigArgument{"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)}},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Collection(p.Args["id"].(string))
				},
			},
			"relatedThoughts": &graphql.Field{
				Type: graphql.NewList(thoughtType),
				Args: graphql.FieldConfigArgument{
					"thoughtId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"limit":     &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.RelatedThoughts(p.Args["thoughtId"].(string), optionalInt(p.Args, "limit", 8))
				},
			},
			"listThoughtVersions": &graphql.Field{
				Type: graphql.NewList(thoughtVersionType),
				Args: graphql.FieldConfigArgument{
					"thoughtId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.ThoughtVersions(p.Args["thoughtId"].(string))
				},
			},
			"collections": &graphql.Field{
				Type: graphql.NewList(collectionType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.Collections()
				},
			},
		},
	})

	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createThought": &graphql.Field{
				Type: thoughtType,
				Args: graphql.FieldConfigArgument{"content": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)}},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.CreateThought(p.Args["content"].(string))
				},
			},
			"editThought": &graphql.Field{
				Type: thoughtType,
				Args: graphql.FieldConfigArgument{
					"thoughtId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"content":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.EditThought(p.Args["thoughtId"].(string), p.Args["content"].(string))
				},
			},
			"updateThought": &graphql.Field{
				Type: thoughtType,
				Args: graphql.FieldConfigArgument{
					"thoughtId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"content":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.EditThought(p.Args["thoughtId"].(string), p.Args["content"].(string))
				},
			},
			"createCollection": &graphql.Field{
				Type: collectionType,
				Args: graphql.FieldConfigArgument{
					"title":       &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"description": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.CreateCollection(p.Args["title"].(string), optionalString(p.Args, "description"))
				},
			},
			"addThoughtToCollection": &graphql.Field{
				Type: collectionType,
				Args: graphql.FieldConfigArgument{
					"collectionId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"thoughtId":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.AddThoughtToCollection(p.Args["collectionId"].(string), p.Args["thoughtId"].(string))
				},
			},
			"recordEngagement": &graphql.Field{
				Type: engagementType,
				Args: graphql.FieldConfigArgument{
					"entityType": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"entityId":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"actionType": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"dwellMs":    &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.RecordEngagement(
						p.Args["entityType"].(string),
						p.Args["entityId"].(string),
						p.Args["actionType"].(string),
						optionalInt(p.Args, "dwellMs", 0),
					)
				},
			},
			"createCapture": &graphql.Field{
				Type: captureType,
				Args: graphql.FieldConfigArgument{
					"content":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"sourceType":  &graphql.ArgumentConfig{Type: graphql.String},
					"sourceTitle": &graphql.ArgumentConfig{Type: graphql.String},
					"sourceUrl":   &graphql.ArgumentConfig{Type: graphql.String},
					"sourceApp":   &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.CreateCapture(
						p.Args["content"].(string),
						models.CaptureSourceType(optionalString(p.Args, "sourceType")),
						optionalString(p.Args, "sourceTitle"),
						optionalString(p.Args, "sourceUrl"),
						optionalString(p.Args, "sourceApp"),
					)
				},
			},
			"archiveCapture": &graphql.Field{
				Type: captureType,
				Args: graphql.FieldConfigArgument{
					"captureId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.ArchiveCapture(p.Args["captureId"].(string))
				},
			},
			"promoteCapture": &graphql.Field{
				Type: captureType,
				Args: graphql.FieldConfigArgument{
					"captureId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return server.service.PromoteCapture(p.Args["captureId"].(string))
				},
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
}

func optionalString(values map[string]interface{}, key string) string {
	if value, exists := values[key]; exists && value != nil {
		return value.(string)
	}
	return ""
}

func optionalInt(values map[string]interface{}, key string, fallback int) int {
	if value, exists := values[key]; exists && value != nil {
		if number, ok := value.(int); ok {
			return number
		}
		if number, ok := value.(float64); ok {
			return int(number)
		}
	}
	return fallback
}
