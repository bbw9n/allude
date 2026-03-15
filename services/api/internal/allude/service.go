package allude

import (
	"errors"
	"strings"
)

type Service struct {
	repository Repository
	ai         AIProvider
}

func NewService(repository Repository, ai AIProvider) *Service {
	return &Service{
		repository: repository,
		ai:         ai,
	}
}

func (service *Service) Viewer() *User {
	return service.repository.GetViewer()
}

func (service *Service) CreateThought(content string) (*Thought, error) {
	thought, err := service.repository.CreateThought(ViewerID, strings.TrimSpace(content))
	if err != nil {
		return nil, err
	}
	go service.AnalyzeThought(thought.ID)
	return thought, nil
}

func (service *Service) UpdateThought(thoughtID, content string) (*Thought, error) {
	thought, err := service.repository.UpdateThought(thoughtID, strings.TrimSpace(content))
	if err != nil {
		return nil, err
	}
	go service.AnalyzeThought(thought.ID)
	return thought, nil
}

func (service *Service) Thought(id string) (*Thought, error) {
	return service.repository.GetThought(id)
}

func (service *Service) SearchThoughts(query string) (*SearchThoughtsResult, error) {
	embedding, err := service.ai.EmbedQuery(query)
	if err != nil {
		return nil, err
	}
	return service.repository.SearchThoughtsByEmbedding(embedding, query, 12)
}

func (service *Service) Graph(centerThoughtID string, distance, limit int) (*GraphNeighborhood, error) {
	return service.repository.GetGraphNeighborhood(centerThoughtID, distance, limit)
}

func (service *Service) RelatedThoughts(thoughtID string, limit int) ([]*Thought, error) {
	return service.repository.GetRelatedThoughts(thoughtID, limit)
}

func (service *Service) ThoughtVersions(thoughtID string) ([]*ThoughtVersion, error) {
	return service.repository.ListThoughtVersions(thoughtID)
}

func (service *Service) Concept(id, name string) (*Concept, error) {
	if id != "" {
		return service.repository.GetConceptByID(id)
	}
	if name != "" {
		return service.repository.GetConceptByName(name)
	}
	return nil, errors.New("concept requires id or name")
}

func (service *Service) AnalyzeThought(thoughtID string) {
	thought, err := service.repository.GetThought(thoughtID)
	if err != nil || thought == nil || thought.CurrentVersion == nil {
		return
	}

	analysis, err := service.ai.AnalyzeThought(thought.CurrentVersion.Content)
	if err != nil {
		_, _ = service.repository.SaveThoughtAnalysis(thoughtID, thought.Embedding, conceptNames(thought.Concepts), ProcessingPartial, []string{err.Error()})
		return
	}

	updated, err := service.repository.SaveThoughtAnalysis(thoughtID, analysis.Embedding, analysis.Concepts, ProcessingReady, analysis.Notes)
	if err != nil {
		return
	}

	searchResult, err := service.repository.SearchThoughtsByEmbedding(analysis.Embedding, updated.CurrentVersion.Content, 6)
	if err != nil {
		return
	}

	var links []*ThoughtLink
	for _, candidate := range searchResult.Thoughts {
		if candidate.ID == updated.ID || len(candidate.Embedding) == 0 {
			continue
		}
		score := combinedRelationshipScore(updated, candidate, analysis.Embedding)
		if score <= 0.28 {
			continue
		}
		links = append(links, &ThoughtLink{
			SourceThoughtID: updated.ID,
			TargetThoughtID: candidate.ID,
			RelationType:    relationTypeForThoughts(updated, candidate),
			Score:           score,
			Origin:          "analysis",
		})
	}
	_ = service.repository.ReplaceThoughtLinks(updated.ID, links)
}

func conceptNames(concepts []*Concept) []string {
	var names []string
	for _, concept := range concepts {
		names = append(names, concept.Name)
	}
	return names
}

func combinedRelationshipScore(source, target *Thought, sourceEmbedding []float64) float64 {
	base := cosineSimilarity(sourceEmbedding, target.Embedding)
	overlap := conceptOverlap(source, target)
	return (base * 0.7) + (overlap * 0.3)
}

func conceptOverlap(source, target *Thought) float64 {
	if len(source.Concepts) == 0 || len(target.Concepts) == 0 {
		return 0
	}
	sourceConcepts := map[string]struct{}{}
	for _, concept := range source.Concepts {
		sourceConcepts[concept.Name] = struct{}{}
	}
	matches := 0
	for _, concept := range target.Concepts {
		if _, exists := sourceConcepts[concept.Name]; exists {
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
	if strings.Contains(strings.ToLower(source.CurrentVersion.Content), "not") ||
		strings.Contains(strings.ToLower(target.CurrentVersion.Content), "not") ||
		strings.Contains(strings.ToLower(source.CurrentVersion.Content), "against") {
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
