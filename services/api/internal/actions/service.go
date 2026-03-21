package actions

import (
	"context"
	"errors"
	"strings"

	"github.com/bbw9n/allude/services/api/internal/domains/models"
	"github.com/bbw9n/allude/services/api/internal/domains/ports"
	"github.com/bbw9n/allude/services/api/internal/domains/semantics"
	"github.com/bbw9n/allude/services/api/internal/pkgs/shared"
)

type Service struct {
	repository ports.Repository
	ai         ports.AIProvider
	runner     *JobRunner
}

func NewService(repository ports.Repository, ai ports.AIProvider) *Service {
	service := &Service{
		repository: repository,
		ai:         ai,
	}
	service.runner = NewJobRunner(repository, service, "worker-dev")
	return service
}

func (service *Service) StartWorkers(ctx context.Context) {
	go service.runner.Start(ctx)
}

func (service *Service) DrainJobs(maxJobs int) error {
	return service.runner.Drain(context.Background(), maxJobs)
}

func (service *Service) Viewer() *models.User {
	return service.repository.GetViewer()
}

func (service *Service) CreateThought(content string) (*models.Thought, error) {
	thought, err := service.repository.CreateThought(shared.ViewerID, strings.TrimSpace(content))
	if err != nil {
		return nil, err
	}
	_, err = service.repository.EnqueueJob(&models.Job{
		Type:        models.JobEmbedThoughtVersion,
		EntityType:  "thought_version",
		EntityID:    thought.CurrentVersion.ID,
		MaxAttempts: 3,
		Payload: map[string]string{
			"thoughtVersionId": thought.CurrentVersion.ID,
			"thoughtId":        thought.ID,
		},
	})
	return thought, err
}

func (service *Service) EditThought(thoughtID, content string) (*models.Thought, error) {
	thought, err := service.repository.UpdateThought(thoughtID, strings.TrimSpace(content))
	if err != nil {
		return nil, err
	}
	_, err = service.repository.EnqueueJob(&models.Job{
		Type:        models.JobEmbedThoughtVersion,
		EntityType:  "thought_version",
		EntityID:    thought.CurrentVersion.ID,
		MaxAttempts: 3,
		Payload: map[string]string{
			"thoughtVersionId": thought.CurrentVersion.ID,
			"thoughtId":        thought.ID,
		},
	})
	return thought, err
}

func (service *Service) Thought(id string) (*models.Thought, error) {
	return service.repository.GetThought(id)
}

func (service *Service) SearchThoughts(query string) (*models.SearchThoughtsResult, error) {
	embedding, err := service.ai.EmbedQuery(query)
	if err != nil {
		return nil, err
	}
	return service.repository.SearchThoughts(query, embedding, 12)
}

func (service *Service) Graph(centerThoughtID string, hopCount, limit int) (*models.GraphNeighborhood, error) {
	return service.repository.GetGraphNeighborhood(centerThoughtID, hopCount, limit)
}

func (service *Service) Concept(id, slug, name string) (*models.Concept, error) {
	if id != "" {
		return service.repository.GetConceptByID(id)
	}
	if slug != "" {
		return service.repository.GetConceptBySlug(slug)
	}
	if name != "" {
		return service.repository.GetConceptByName(name)
	}
	return nil, errors.New("concept requires id, slug, or name")
}

func (service *Service) CreateCollection(title, description string) (*models.Collection, error) {
	return service.repository.CreateCollection(shared.ViewerID, title, description)
}

func (service *Service) AddThoughtToCollection(collectionID, thoughtID string) (*models.Collection, error) {
	return service.repository.AddThoughtToCollection(collectionID, thoughtID)
}

func (service *Service) Collection(id string) (*models.Collection, error) {
	return service.repository.GetCollection(id)
}

func (service *Service) RecordEngagement(entityType, entityID, actionType string, dwellMS int) (*models.EngagementEvent, error) {
	return service.repository.RecordEngagement(&models.EngagementEvent{
		UserID:     shared.ViewerID,
		EntityType: entityType,
		EntityID:   entityID,
		ActionType: actionType,
		DwellMS:    dwellMS,
	})
}

func (service *Service) Jobs() []*models.Job {
	return service.repository.ListJobs()
}

func (service *Service) enrichThoughtVersion(versionID, thoughtID string) error {
	thought, err := service.repository.GetThought(thoughtID)
	if err != nil {
		return err
	}
	var version *models.ThoughtVersion
	for _, candidate := range thought.Versions {
		if candidate.ID == versionID {
			version = candidate
			break
		}
	}
	if version == nil {
		return errors.New("version not found")
	}

	analysis, err := service.ai.AnalyzeThought(version.Content)
	if err != nil {
		_, _ = service.repository.SaveThoughtVersionEnrichment(versionID, version.Embedding, nil, models.ProcessingPartial, []string{err.Error()})
		return err
	}
	enrichedVersion, err := service.repository.SaveThoughtVersionEnrichment(versionID, analysis.Embedding, analysis.Concepts, models.ProcessingReady, analysis.Notes)
	if err != nil {
		return err
	}
	if _, err := service.repository.EnqueueJob(&models.Job{
		Type:        models.JobLinkThought,
		EntityType:  "thought",
		EntityID:    thoughtID,
		MaxAttempts: 3,
		Payload: map[string]string{
			"thoughtVersionId": enrichedVersion.ID,
			"thoughtId":        thoughtID,
		},
	}); err != nil {
		return err
	}
	latestThought, err := service.repository.GetThought(thoughtID)
	if err != nil {
		return nil
	}
	for _, concept := range latestThought.Concepts {
		_, _ = service.repository.EnqueueJob(&models.Job{
			Type:        models.JobRefreshConceptSummary,
			EntityType:  "concept",
			EntityID:    concept.ID,
			MaxAttempts: 3,
			Payload:     map[string]string{"conceptId": concept.ID},
		})
	}
	return nil
}

func (service *Service) linkThought(thoughtID string) error {
	thought, err := service.repository.GetThought(thoughtID)
	if err != nil {
		return err
	}
	if thought.CurrentVersion == nil {
		return nil
	}
	searchResult, err := service.repository.SearchThoughts(thought.CurrentVersion.Content, thought.CurrentVersion.Embedding, 12)
	if err != nil {
		return err
	}

	var links []*models.ThoughtLink
	for _, candidate := range searchResult.Thoughts {
		if candidate.ID == thought.ID || candidate.CurrentVersion == nil {
			continue
		}
		score := semantics.CombinedRelationshipScore(thought, candidate)
		if score < 0.28 {
			continue
		}
		links = append(links, &models.ThoughtLink{
			SourceThoughtID: thought.ID,
			TargetThoughtID: candidate.ID,
			RelationType:    semantics.RelationTypeForThoughts(thought, candidate),
			Weight:          score,
			Source:          "analysis",
			Explanation:     "Ranked from vector similarity and concept overlap",
		})
	}
	return service.repository.ReplaceThoughtLinks(thought.ID, links)
}

func (service *Service) refreshConceptSummary(_ string) error {
	return nil
}
