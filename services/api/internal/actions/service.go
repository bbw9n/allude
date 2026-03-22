package actions

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

func (service *Service) ViewerInterests(limit int) ([]*models.UserInterest, error) {
	return service.repository.ListUserInterests(shared.ViewerID, limit)
}

func (service *Service) MyThoughts(limit int) ([]*models.Thought, error) {
	return service.repository.ListThoughtsByAuthor(shared.ViewerID, limit)
}

func (service *Service) Currents(limit int) ([]*models.IdeaCurrent, error) {
	thoughts, err := service.repository.ListRecentThoughts(max(limit*4, 24))
	if err != nil {
		return nil, err
	}
	return buildIdeaCurrents(thoughts, limit), nil
}

func (service *Service) Home(limit int) (*models.HomePayload, error) {
	currents, err := service.Currents(max(limit, 4))
	if err != nil {
		return nil, err
	}
	collections, err := service.repository.ListCollections()
	if err != nil {
		return nil, err
	}
	profile, err := service.repository.ListUserInterests(shared.ViewerID, 12)
	if err != nil {
		return nil, err
	}
	currents = rankCurrentsForProfile(currents, profile)

	recommendedThoughts := make([]*models.Thought, 0, limit)
	seenThoughts := map[string]struct{}{}
	for _, current := range currents {
		for _, thought := range current.Thoughts {
			if _, exists := seenThoughts[thought.ID]; exists {
				continue
			}
			seenThoughts[thought.ID] = struct{}{}
			recommendedThoughts = append(recommendedThoughts, thought)
			if len(recommendedThoughts) == limit {
				break
			}
		}
		if len(recommendedThoughts) == limit {
			break
		}
	}

	if len(recommendedThoughts) < limit {
		recentThoughts, err := service.repository.ListRecentThoughts(limit * 2)
		if err == nil {
			recentThoughts = rankThoughtsForProfile(recentThoughts, profile)
			for _, thought := range recentThoughts {
				if _, exists := seenThoughts[thought.ID]; exists {
					continue
				}
				seenThoughts[thought.ID] = struct{}{}
				recommendedThoughts = append(recommendedThoughts, thought)
				if len(recommendedThoughts) == limit {
					break
				}
			}
		}
	}

	collections = rankCollectionsForProfile(collections, profile)
	if len(collections) > limit {
		collections = collections[:limit]
	}

	return &models.HomePayload{
		Viewer:                 service.Viewer(),
		Currents:               currents,
		RecommendedThoughts:    recommendedThoughts,
		RecommendedCollections: collections,
	}, nil
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

func (service *Service) DraftSuggestions(content, thoughtID string) (*models.DraftSuggestions, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return &models.DraftSuggestions{}, nil
	}

	analysis, err := service.ai.AnalyzeThought(trimmed)
	if err != nil {
		return nil, err
	}

	draftThought := &models.Thought{
		ID: thoughtID,
		CurrentVersion: &models.ThoughtVersion{
			Content:   trimmed,
			Embedding: analysis.Embedding,
		},
		Concepts: conceptsFromNames(analysis.Concepts),
	}

	result := &models.DraftSuggestions{
		RelatedConcepts: uniqueLimitedStrings(analysis.Concepts, 6),
		Notes:           append([]string(nil), analysis.Notes...),
	}

	searchResult, err := service.repository.SearchThoughts(trimmed, analysis.Embedding, 8)
	if err != nil {
		return result, nil
	}

	for _, candidate := range searchResult.Thoughts {
		if candidate.ID == thoughtID || candidate.CurrentVersion == nil {
			continue
		}
		score := semantics.CombinedRelationshipScore(draftThought, candidate)
		if score < 0.15 {
			continue
		}
		if semantics.RelationTypeForThoughts(draftThought, candidate) == models.RelationContradict {
			if len(result.CounterThoughts) < 3 {
				result.CounterThoughts = append(result.CounterThoughts, candidate)
			}
			continue
		}
		if len(result.SupportingThoughts) < 3 {
			result.SupportingThoughts = append(result.SupportingThoughts, candidate)
		}
	}

	result.Reframes = buildReframes(result.RelatedConcepts, result.SupportingThoughts, result.CounterThoughts)
	return result, nil
}

func (service *Service) Telescope(query string) (*models.TelescopeResult, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return &models.TelescopeResult{
			Query:     "",
			Intent:    "explore",
			Narrative: "Ask Allude about a tension, concept, contradiction, or connection to explore the idea graph.",
		}, nil
	}

	analysis, err := service.ai.AnalyzeThought(trimmed)
	if err != nil {
		return nil, err
	}
	searchResult, err := service.repository.SearchThoughts(trimmed, analysis.Embedding, 12)
	if err != nil {
		return nil, err
	}

	result := &models.TelescopeResult{
		Query:    trimmed,
		Intent:   telescopeIntent(trimmed),
		Clusters: searchResult.Clusters,
	}

	result.SeedThoughts = limitedThoughts(searchResult.Thoughts, 4)
	result.SeedConcepts = service.resolveSeedConcepts(trimmed, analysis.Concepts, searchResult.Clusters)
	if len(result.SeedThoughts) > 0 {
		graph, err := service.repository.GetGraphNeighborhood(result.SeedThoughts[0].ID, 2, 12)
		if err == nil {
			result.Graph = graph
		}
	}

	currents, err := service.Currents(6)
	if err == nil {
		result.RelatedCurrents = filterCurrentsForTelescope(currents, result.SeedConcepts, result.SeedThoughts, 3)
	}
	result.SuggestedJumps = buildTelescopeSuggestedJumps(trimmed, result.Intent, result.SeedConcepts, searchResult.Clusters, result.SeedThoughts)
	result.Narrative = buildTelescopeNarrative(trimmed, result.Intent, result.SeedConcepts, result.SeedThoughts, searchResult.Clusters)
	for _, concept := range result.SeedConcepts[:min(len(result.SeedConcepts), 3)] {
		_ = service.repository.AdjustUserInterest(shared.ViewerID, concept.ID, "telescope_query", 0.18)
	}
	return result, nil
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
	collection, err := service.repository.AddThoughtToCollection(collectionID, thoughtID)
	if err != nil {
		return nil, err
	}
	if thought, err := service.repository.GetThought(thoughtID); err == nil {
		_ = service.applyThoughtInterestDelta(shared.ViewerID, thought, 0.9, "collection")
	}
	return collection, nil
}

func (service *Service) Collection(id string) (*models.Collection, error) {
	return service.repository.GetCollection(id)
}

func (service *Service) Collections() ([]*models.Collection, error) {
	return service.repository.ListCollections()
}

func (service *Service) RecordEngagement(entityType, entityID, actionType string, dwellMS int) (*models.EngagementEvent, error) {
	event, err := service.repository.RecordEngagement(&models.EngagementEvent{
		UserID:     shared.ViewerID,
		EntityType: entityType,
		EntityID:   entityID,
		ActionType: actionType,
		DwellMS:    dwellMS,
	})
	if err != nil {
		return nil, err
	}
	switch entityType {
	case "thought":
		if thought, err := service.repository.GetThought(entityID); err == nil {
			weight := 0.2
			if dwellMS >= 3000 {
				weight = 0.45
			}
			_ = service.applyThoughtInterestDelta(shared.ViewerID, thought, weight, "engagement")
		}
	case "concept":
		_ = service.repository.AdjustUserInterest(shared.ViewerID, entityID, "engagement", 0.35)
	}
	return event, nil
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
	_ = service.applyThoughtInterestDelta(latestThought.AuthorID, latestThought, 1.2, "authored")
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

func conceptsFromNames(names []string) []*models.Concept {
	concepts := make([]*models.Concept, 0, len(names))
	for _, name := range uniqueLimitedStrings(names, 6) {
		concepts = append(concepts, &models.Concept{
			ID:            strings.ToLower(strings.ReplaceAll(name, " ", "-")),
			CanonicalName: name,
			Slug:          strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		})
	}
	return concepts
}

func uniqueLimitedStrings(values []string, limit int) []string {
	seen := map[string]struct{}{}
	capacity := len(values)
	if limit < capacity {
		capacity = limit
	}
	result := make([]string, 0, capacity)
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
		if len(result) == limit {
			break
		}
	}
	return result
}

func buildIdeaCurrents(thoughts []*models.Thought, limit int) []*models.IdeaCurrent {
	type bucket struct {
		concept  *models.Concept
		thoughts []*models.Thought
		quality  float64
	}

	buckets := map[string]*bucket{}
	for _, thought := range thoughts {
		if thought == nil || thought.CurrentVersion == nil {
			continue
		}
		var concept *models.Concept
		if len(thought.Concepts) > 0 {
			concept = thought.Concepts[0]
		} else {
			concept = &models.Concept{
				ID:            "uncategorized",
				CanonicalName: "Unsorted Ideas",
				Slug:          "unsorted-ideas",
			}
		}
		entry := buckets[concept.ID]
		if entry == nil {
			entry = &bucket{concept: concept}
			buckets[concept.ID] = entry
		}
		entry.thoughts = append(entry.thoughts, thought)
		entry.quality += semantics.QualityScore(thought)
	}

	currents := make([]*models.IdeaCurrent, 0, len(buckets))
	for _, entry := range buckets {
		if len(entry.thoughts) == 0 {
			continue
		}
		if len(entry.thoughts) > 4 {
			entry.thoughts = entry.thoughts[:4]
		}
		quality := entry.quality / float64(len(entry.thoughts))
		freshness := 1.0 / float64(len(currents)+1)
		title := fmt.Sprintf("Current: %s", entry.concept.CanonicalName)
		summary := buildCurrentSummary(entry.concept, entry.thoughts)
		current := &models.IdeaCurrent{
			ID:             entry.concept.ID,
			Title:          title,
			Summary:        summary,
			ClusterKey:     entry.concept.Slug,
			FreshnessScore: freshness,
			QualityScore:   quality,
			Concepts:       []*models.Concept{entry.concept},
			Thoughts:       entry.thoughts,
			CreatedAt:      entry.thoughts[0].CreatedAt,
			UpdatedAt:      entry.thoughts[0].UpdatedAt,
		}
		currents = append(currents, current)
	}

	sort.Slice(currents, func(i, j int) bool {
		left := currents[i].QualityScore + currents[i].FreshnessScore
		right := currents[j].QualityScore + currents[j].FreshnessScore
		if left == right {
			return currents[i].UpdatedAt > currents[j].UpdatedAt
		}
		return left > right
	})
	if len(currents) > limit {
		currents = currents[:limit]
	}
	return currents
}

func buildCurrentSummary(concept *models.Concept, thoughts []*models.Thought) string {
	phrases := make([]string, 0, min(len(thoughts), 2))
	for _, thought := range thoughts {
		content := strings.TrimSpace(thought.CurrentVersion.Content)
		if content == "" {
			continue
		}
		if len(content) > 88 {
			content = content[:88] + "..."
		}
		phrases = append(phrases, content)
		if len(phrases) == 2 {
			break
		}
	}
	if len(phrases) == 0 {
		return fmt.Sprintf("A developing cluster around %s.", strings.ToLower(concept.CanonicalName))
	}
	return fmt.Sprintf("%s is surfacing through %d connected thought%s, including %s",
		concept.CanonicalName,
		len(thoughts),
		pluralSuffix(len(thoughts)),
		strings.Join(phrases, " and "),
	)
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func limitedThoughts(thoughts []*models.Thought, limit int) []*models.Thought {
	if len(thoughts) <= limit {
		return thoughts
	}
	return thoughts[:limit]
}

func telescopeIntent(query string) string {
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "disagree"), strings.Contains(lower, "contradict"), strings.Contains(lower, "oppose"):
		return "contradict"
	case strings.Contains(lower, "compare"), strings.Contains(lower, "versus"), strings.Contains(lower, "vs"):
		return "compare"
	case strings.Contains(lower, "connect"), strings.Contains(lower, "connection"), strings.Contains(lower, "between"):
		return "connect"
	default:
		return "explore"
	}
}

func (service *Service) resolveSeedConcepts(query string, conceptNames []string, clusters []*models.SearchCluster) []*models.Concept {
	seen := map[string]struct{}{}
	seeds := make([]*models.Concept, 0, 6)
	appendConcept := func(concept *models.Concept) {
		if concept == nil {
			return
		}
		if _, exists := seen[concept.ID]; exists {
			return
		}
		seen[concept.ID] = struct{}{}
		seeds = append(seeds, concept)
	}

	for _, name := range uniqueLimitedStrings(conceptNames, 4) {
		concept, err := service.repository.GetConceptByName(name)
		if err == nil && concept != nil {
			appendConcept(concept)
		}
	}
	for _, cluster := range clusters {
		for _, concept := range cluster.Concepts {
			appendConcept(concept)
			if len(seeds) == 6 {
				return seeds
			}
		}
	}

	queryWords := uniqueLimitedStrings(strings.Fields(strings.ToLower(query)), 4)
	for _, word := range queryWords {
		concept, err := service.repository.GetConceptByName(word)
		if err == nil && concept != nil {
			appendConcept(concept)
			if len(seeds) == 6 {
				return seeds
			}
		}
	}
	return seeds
}

func filterCurrentsForTelescope(currents []*models.IdeaCurrent, concepts []*models.Concept, thoughts []*models.Thought, limit int) []*models.IdeaCurrent {
	conceptIDs := map[string]struct{}{}
	thoughtIDs := map[string]struct{}{}
	for _, concept := range concepts {
		conceptIDs[concept.ID] = struct{}{}
	}
	for _, thought := range thoughts {
		thoughtIDs[thought.ID] = struct{}{}
	}

	filtered := make([]*models.IdeaCurrent, 0, limit)
	for _, current := range currents {
		matched := false
		for _, concept := range current.Concepts {
			if _, exists := conceptIDs[concept.ID]; exists {
				matched = true
				break
			}
		}
		if !matched {
			for _, thought := range current.Thoughts {
				if _, exists := thoughtIDs[thought.ID]; exists {
					matched = true
					break
				}
			}
		}
		if matched {
			filtered = append(filtered, current)
			if len(filtered) == limit {
				break
			}
		}
	}
	return filtered
}

func buildTelescopeSuggestedJumps(query, intent string, seedConcepts []*models.Concept, clusters []*models.SearchCluster, thoughts []*models.Thought) []*models.TelescopeJump {
	jumps := make([]*models.TelescopeJump, 0, 4)
	appendJump := func(label, jumpQuery, reason string, thoughtIDs []string) {
		for _, existing := range jumps {
			if existing.Query == jumpQuery {
				return
			}
		}
		jumps = append(jumps, &models.TelescopeJump{
			Label:      label,
			Query:      jumpQuery,
			Reason:     reason,
			ThoughtIDs: thoughtIDs,
		})
	}

	if len(seedConcepts) > 0 {
		concept := seedConcepts[0]
		appendJump(
			"Go deeper on "+concept.CanonicalName,
			"adjacent ideas to "+strings.ToLower(concept.CanonicalName),
			"Expand the strongest concept cluster from this result.",
			nil,
		)
		appendJump(
			"Find disagreement",
			"who disagrees with "+strings.ToLower(concept.CanonicalName),
			"Surface contradiction and alternative takes.",
			nil,
		)
	}
	if len(clusters) > 1 {
		appendJump(
			"Compare clusters",
			"connections between "+strings.ToLower(clusters[0].Label)+" and "+strings.ToLower(clusters[1].Label),
			"Follow the strongest bridge between the top clusters.",
			appendUniqueThoughtIDs(thoughts, 3),
		)
	}
	if intent != "contradict" {
		appendJump(
			"Look for tensions",
			"contradictions in "+query,
			"Ask for the edge cases and objections.",
			appendUniqueThoughtIDs(thoughts, 3),
		)
	}
	return jumps
}

func buildTelescopeNarrative(query, intent string, seedConcepts []*models.Concept, thoughts []*models.Thought, clusters []*models.SearchCluster) string {
	if len(thoughts) == 0 {
		return fmt.Sprintf("Allude couldn’t find a strong cluster for %q yet. Try naming a concept directly or asking for a comparison or contradiction.", query)
	}

	parts := []string{
		fmt.Sprintf("For %q, Allude found %d thought%s across %d cluster%s.", query, len(thoughts), pluralSuffix(len(thoughts)), len(clusters), pluralSuffix(len(clusters))),
	}
	if len(seedConcepts) > 0 {
		conceptNames := make([]string, 0, min(len(seedConcepts), 3))
		for _, concept := range seedConcepts[:min(len(seedConcepts), 3)] {
			conceptNames = append(conceptNames, concept.CanonicalName)
		}
		parts = append(parts, fmt.Sprintf("The strongest concepts are %s.", strings.Join(conceptNames, ", ")))
	}

	switch intent {
	case "contradict":
		parts = append(parts, "This query is framed as disagreement, so the best next step is to inspect opposing thoughts and edge-case clusters.")
	case "compare":
		parts = append(parts, "This query is framed as a comparison, so the best next step is to inspect where the top clusters overlap and diverge.")
	case "connect":
		parts = append(parts, "This query is framed as a connection search, so the best next step is to follow the bridge thoughts between the leading concepts.")
	default:
		parts = append(parts, "This query is exploratory, so the best next step is to pivot into the strongest cluster and then branch into adjacent ideas.")
	}
	return strings.Join(parts, " ")
}

func appendUniqueThoughtIDs(thoughts []*models.Thought, limit int) []string {
	ids := make([]string, 0, min(len(thoughts), limit))
	seen := map[string]struct{}{}
	for _, thought := range thoughts {
		if _, exists := seen[thought.ID]; exists {
			continue
		}
		seen[thought.ID] = struct{}{}
		ids = append(ids, thought.ID)
		if len(ids) == limit {
			break
		}
	}
	return ids
}

func (service *Service) applyThoughtInterestDelta(userID string, thought *models.Thought, baseDelta float64, source string) error {
	if userID == "" || thought == nil {
		return nil
	}
	for index, concept := range thought.Concepts {
		weight := baseDelta / float64(index+1)
		if err := service.repository.AdjustUserInterest(userID, concept.ID, source, weight); err != nil {
			return err
		}
	}
	return nil
}

func profileAffinityMap(profile []*models.UserInterest) map[string]float64 {
	affinity := map[string]float64{}
	for _, interest := range profile {
		affinity[interest.ConceptID] = interest.AffinityScore
	}
	return affinity
}

func rankThoughtsForProfile(thoughts []*models.Thought, profile []*models.UserInterest) []*models.Thought {
	affinity := profileAffinityMap(profile)
	type rankedThought struct {
		thought *models.Thought
		score   float64
	}
	ranked := make([]rankedThought, 0, len(thoughts))
	for _, thought := range thoughts {
		score := semantics.QualityScore(thought)
		for _, concept := range thought.Concepts {
			score += affinity[concept.ID]
		}
		ranked = append(ranked, rankedThought{thought: thought, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].thought.UpdatedAt > ranked[j].thought.UpdatedAt
		}
		return ranked[i].score > ranked[j].score
	})
	result := make([]*models.Thought, 0, len(ranked))
	for _, item := range ranked {
		result = append(result, item.thought)
	}
	return result
}

func rankCurrentsForProfile(currents []*models.IdeaCurrent, profile []*models.UserInterest) []*models.IdeaCurrent {
	affinity := profileAffinityMap(profile)
	ranked := append([]*models.IdeaCurrent(nil), currents...)
	sort.Slice(ranked, func(i, j int) bool {
		left := ranked[i].QualityScore + ranked[i].FreshnessScore + currentAffinity(ranked[i], affinity)
		right := ranked[j].QualityScore + ranked[j].FreshnessScore + currentAffinity(ranked[j], affinity)
		if left == right {
			return ranked[i].UpdatedAt > ranked[j].UpdatedAt
		}
		return left > right
	})
	return ranked
}

func rankCollectionsForProfile(collections []*models.Collection, profile []*models.UserInterest) []*models.Collection {
	affinity := profileAffinityMap(profile)
	ranked := append([]*models.Collection(nil), collections...)
	sort.Slice(ranked, func(i, j int) bool {
		left := collectionAffinity(ranked[i], affinity)
		right := collectionAffinity(ranked[j], affinity)
		if left == right {
			return ranked[i].UpdatedAt > ranked[j].UpdatedAt
		}
		return left > right
	})
	return ranked
}

func currentAffinity(current *models.IdeaCurrent, affinity map[string]float64) float64 {
	score := 0.0
	for _, concept := range current.Concepts {
		score += affinity[concept.ID]
	}
	return score
}

func collectionAffinity(collection *models.Collection, affinity map[string]float64) float64 {
	score := 0.0
	for _, item := range collection.Items {
		if item.Thought == nil {
			continue
		}
		for _, concept := range item.Thought.Concepts {
			score += affinity[concept.ID]
		}
	}
	return score
}

func buildReframes(concepts []string, supportingThoughts, counterThoughts []*models.Thought) []string {
	reframes := make([]string, 0, 3)
	if len(concepts) > 0 {
		reframes = append(reframes, "Anchor this thought more explicitly in "+concepts[0]+".")
	}
	if len(supportingThoughts) > 0 {
		reframes = append(reframes, "Make the argument more concrete by adding an example or mechanism.")
	}
	if len(counterThoughts) > 0 {
		reframes = append(reframes, "Name the strongest tension or counterargument directly in the draft.")
	}
	if len(reframes) == 0 {
		reframes = append(reframes, "Tighten the claim by making the core idea more specific.")
	}
	if len(concepts) > 0 {
		reframes[0] = "Anchor this thought more explicitly in " + concepts[0] + "."
	}
	return reframes
}
