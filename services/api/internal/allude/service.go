package allude

import (
	"context"
	"errors"
	"strings"
)

type Service struct {
	repository Repository
	ai         AIProvider
	runner     *JobRunner
}

func NewService(repository Repository, ai AIProvider) *Service {
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

func (service *Service) Viewer() *User {
	return service.repository.GetViewer()
}

func (service *Service) CreateThought(content string) (*Thought, error) {
	thought, err := service.repository.CreateThought(ViewerID, strings.TrimSpace(content))
	if err != nil {
		return nil, err
	}
	_, err = service.repository.EnqueueJob(&Job{
		Type:        JobEmbedThoughtVersion,
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

func (service *Service) EditThought(thoughtID, content string) (*Thought, error) {
	thought, err := service.repository.UpdateThought(thoughtID, strings.TrimSpace(content))
	if err != nil {
		return nil, err
	}
	_, err = service.repository.EnqueueJob(&Job{
		Type:        JobEmbedThoughtVersion,
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

func (service *Service) Thought(id string) (*Thought, error) {
	return service.repository.GetThought(id)
}

func (service *Service) SearchThoughts(query string) (*SearchThoughtsResult, error) {
	embedding, err := service.ai.EmbedQuery(query)
	if err != nil {
		return nil, err
	}
	return service.repository.SearchThoughts(query, embedding, 12)
}

func (service *Service) Graph(centerThoughtID string, hopCount, limit int) (*GraphNeighborhood, error) {
	return service.repository.GetGraphNeighborhood(centerThoughtID, hopCount, limit)
}

func (service *Service) Concept(id, slug, name string) (*Concept, error) {
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

func (service *Service) CreateCollection(title, description string) (*Collection, error) {
	return service.repository.CreateCollection(ViewerID, title, description)
}

func (service *Service) AddThoughtToCollection(collectionID, thoughtID string) (*Collection, error) {
	return service.repository.AddThoughtToCollection(collectionID, thoughtID)
}

func (service *Service) Collection(id string) (*Collection, error) {
	return service.repository.GetCollection(id)
}

func (service *Service) RecordEngagement(entityType, entityID, actionType string, dwellMS int) (*EngagementEvent, error) {
	return service.repository.RecordEngagement(&EngagementEvent{
		UserID:     ViewerID,
		EntityType: entityType,
		EntityID:   entityID,
		ActionType: actionType,
		DwellMS:    dwellMS,
	})
}

func (service *Service) Jobs() []*Job {
	return service.repository.ListJobs()
}

func (service *Service) enrichThoughtVersion(versionID, thoughtID string) error {
	thought, err := service.repository.GetThought(thoughtID)
	if err != nil {
		return err
	}
	var version *ThoughtVersion
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
		_, _ = service.repository.SaveThoughtVersionEnrichment(versionID, version.Embedding, nil, ProcessingPartial, []string{err.Error()})
		return err
	}
	enrichedVersion, err := service.repository.SaveThoughtVersionEnrichment(versionID, analysis.Embedding, analysis.Concepts, ProcessingReady, analysis.Notes)
	if err != nil {
		return err
	}
	if _, err := service.repository.EnqueueJob(&Job{
		Type:        JobLinkThought,
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
		_, _ = service.repository.EnqueueJob(&Job{
			Type:        JobRefreshConceptSummary,
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

	var links []*ThoughtLink
	for _, candidate := range searchResult.Thoughts {
		if candidate.ID == thought.ID || candidate.CurrentVersion == nil {
			continue
		}
		score := combinedRelationshipScore(thought, candidate)
		if score < 0.28 {
			continue
		}
		links = append(links, &ThoughtLink{
			SourceThoughtID: thought.ID,
			TargetThoughtID: candidate.ID,
			RelationType:    relationTypeForThoughts(thought, candidate),
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

func combinedRelationshipScore(source, target *Thought) float64 {
	base := cosineSimilarity(source.CurrentVersion.Embedding, target.CurrentVersion.Embedding)
	overlap := conceptOverlap(source, target)
	return (base * 0.7) + (overlap * 0.3)
}

func conceptOverlap(source, target *Thought) float64 {
	if len(source.Concepts) == 0 || len(target.Concepts) == 0 {
		return 0
	}
	sourceConcepts := map[string]struct{}{}
	for _, concept := range source.Concepts {
		sourceConcepts[concept.CanonicalName] = struct{}{}
	}
	matches := 0
	for _, concept := range target.Concepts {
		if _, exists := sourceConcepts[concept.CanonicalName]; exists {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	total := len(source.Concepts)
	if len(target.Concepts) > total {
		total = len(target.Concepts)
	}
	return float64(matches) / float64(total)
}

func relationTypeForThoughts(source, target *Thought) RelationType {
	sourceContent := strings.ToLower(source.CurrentVersion.Content)
	targetContent := strings.ToLower(target.CurrentVersion.Content)
	if strings.Contains(sourceContent, "not") || strings.Contains(targetContent, "not") || strings.Contains(sourceContent, "against") {
		return RelationContradict
	}
	sourceConcepts := map[string]struct{}{}
	for _, concept := range source.Concepts {
		sourceConcepts[concept.ID] = struct{}{}
	}
	for _, concept := range target.Concepts {
		if _, exists := sourceConcepts[concept.ID]; exists {
			return RelationExtends
		}
	}
	return RelationRelated
}
