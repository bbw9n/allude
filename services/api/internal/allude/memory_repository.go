package allude

import (
	"errors"
	"math"
	"sort"
	"strings"
	"sync"
)

type thoughtRecord struct {
	Thought *Thought
}

type InMemoryRepository struct {
	mu              sync.RWMutex
	viewer          *User
	thoughts        map[string]*Thought
	versions        map[string][]*ThoughtVersion
	concepts        map[string]*Concept
	thoughtConcepts map[string]map[string]struct{}
	thoughtLinks    map[string]*ThoughtLink
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		viewer: &User{
			ID:        ViewerID,
			Username:  "allude-dev",
			Bio:       "A local development identity for Allude.",
			Interests: []string{"philosophy", "creativity", "systems"},
		},
		thoughts:        map[string]*Thought{},
		versions:        map[string][]*ThoughtVersion{},
		concepts:        map[string]*Concept{},
		thoughtConcepts: map[string]map[string]struct{}{},
		thoughtLinks:    map[string]*ThoughtLink{},
	}
}

func (repository *InMemoryRepository) GetViewer() *User {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	return cloneUser(repository.viewer)
}

func (repository *InMemoryRepository) CreateThought(authorID, content string) (*Thought, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	now := nowISO()
	thoughtID := createID("thought")
	versionID := createID("version")
	version := &ThoughtVersion{
		ID:        versionID,
		ThoughtID: thoughtID,
		Version:   1,
		Content:   content,
		CreatedAt: now,
	}
	thought := &Thought{
		ID:               thoughtID,
		AuthorID:         authorID,
		CurrentVersionID: versionID,
		Embedding:        nil,
		ProcessingStatus: ProcessingProcessing,
		ProcessingNotes:  []string{"Queued for analysis"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	repository.thoughts[thoughtID] = thought
	repository.versions[thoughtID] = []*ThoughtVersion{version}
	repository.thoughtConcepts[thoughtID] = map[string]struct{}{}
	return repository.buildThoughtLocked(thoughtID, true, true)
}

func (repository *InMemoryRepository) UpdateThought(thoughtID, content string) (*Thought, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	thought, exists := repository.thoughts[thoughtID]
	if !exists {
		return nil, errors.New("thought not found")
	}

	versionID := createID("version")
	version := &ThoughtVersion{
		ID:        versionID,
		ThoughtID: thoughtID,
		Version:   len(repository.versions[thoughtID]) + 1,
		Content:   content,
		CreatedAt: nowISO(),
	}
	repository.versions[thoughtID] = append(repository.versions[thoughtID], version)
	thought.CurrentVersionID = versionID
	thought.ProcessingStatus = ProcessingProcessing
	thought.ProcessingNotes = []string{"Queued for analysis"}
	thought.UpdatedAt = nowISO()
	return repository.buildThoughtLocked(thoughtID, true, true)
}

func (repository *InMemoryRepository) GetThought(thoughtID string) (*Thought, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	return repository.buildThoughtLocked(thoughtID, true, true)
}

func (repository *InMemoryRepository) ListThoughtVersions(thoughtID string) ([]*ThoughtVersion, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	versions, exists := repository.versions[thoughtID]
	if !exists {
		return nil, errors.New("thought not found")
	}

	clones := make([]*ThoughtVersion, 0, len(versions))
	for _, version := range versions {
		clones = append(clones, cloneVersion(version))
	}
	return clones, nil
}

func (repository *InMemoryRepository) SaveThoughtAnalysis(thoughtID string, embedding []float64, conceptNames []string, status ProcessingStatus, notes []string) (*Thought, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	thought, exists := repository.thoughts[thoughtID]
	if !exists {
		return nil, errors.New("thought not found")
	}

	thought.Embedding = append([]float64(nil), embedding...)
	thought.ProcessingStatus = status
	thought.ProcessingNotes = append([]string(nil), notes...)
	thought.UpdatedAt = nowISO()
	repository.thoughtConcepts[thoughtID] = map[string]struct{}{}

	for _, raw := range conceptNames {
		normalized := normalizeConceptName(raw)
		if normalized == "" {
			continue
		}
		var concept *Concept
		for _, existing := range repository.concepts {
			if existing.NormalizedName == normalized {
				concept = existing
				break
			}
		}
		if concept == nil {
			concept = &Concept{
				ID:             createID("concept"),
				Name:           strings.TrimSpace(raw),
				NormalizedName: normalized,
				CreatedAt:      nowISO(),
			}
			repository.concepts[concept.ID] = concept
		}
		repository.thoughtConcepts[thoughtID][concept.ID] = struct{}{}
	}

	return repository.buildThoughtLocked(thoughtID, true, true)
}

func (repository *InMemoryRepository) SearchThoughtsByEmbedding(embedding []float64, _ string, limit int) (*SearchThoughtsResult, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	type scored struct {
		ThoughtID string
		Score     float64
	}
	var scoredThoughts []scored
	for thoughtID, thought := range repository.thoughts {
		if len(thought.Embedding) == 0 {
			continue
		}
		scoredThoughts = append(scoredThoughts, scored{
			ThoughtID: thoughtID,
			Score:     cosineSimilarity(embedding, thought.Embedding),
		})
	}
	sort.Slice(scoredThoughts, func(i, j int) bool {
		return scoredThoughts[i].Score > scoredThoughts[j].Score
	})
	if len(scoredThoughts) > limit {
		scoredThoughts = scoredThoughts[:limit]
	}

	thoughts := make([]*Thought, 0, len(scoredThoughts))
	conceptCounts := map[string]struct {
		Concept    *Concept
		ThoughtIDs map[string]struct{}
	}{}
	for _, entry := range scoredThoughts {
		thought, err := repository.buildThoughtLocked(entry.ThoughtID, false, false)
		if err != nil {
			return nil, err
		}
		thoughts = append(thoughts, thought)
		for _, concept := range thought.Concepts {
			current := conceptCounts[concept.ID]
			if current.ThoughtIDs == nil {
				current = struct {
					Concept    *Concept
					ThoughtIDs map[string]struct{}
				}{
					Concept:    cloneConceptBase(concept),
					ThoughtIDs: map[string]struct{}{},
				}
			}
			current.ThoughtIDs[thought.ID] = struct{}{}
			conceptCounts[concept.ID] = current
		}
	}

	var clusters []*SearchCluster
	for _, entry := range conceptCounts {
		var thoughtIDs []string
		for thoughtID := range entry.ThoughtIDs {
			thoughtIDs = append(thoughtIDs, thoughtID)
		}
		sort.Strings(thoughtIDs)
		clusters = append(clusters, &SearchCluster{
			Label:      entry.Concept.Name,
			Concepts:   []*Concept{cloneConceptBase(entry.Concept)},
			ThoughtIDs: thoughtIDs,
		})
	}
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].ThoughtIDs) > len(clusters[j].ThoughtIDs)
	})
	if len(clusters) > 4 {
		clusters = clusters[:4]
	}

	return &SearchThoughtsResult{
		Thoughts: thoughts,
		Clusters: clusters,
	}, nil
}

func (repository *InMemoryRepository) GetRelatedThoughts(thoughtID string, limit int) ([]*Thought, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	var links []*ThoughtLink
	for _, link := range repository.thoughtLinks {
		if link.SourceThoughtID == thoughtID || link.TargetThoughtID == thoughtID {
			links = append(links, link)
		}
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].Score > links[j].Score
	})
	if len(links) > limit {
		links = links[:limit]
	}
	seen := map[string]struct{}{}
	var related []*Thought
	for _, link := range links {
		targetID := link.SourceThoughtID
		if targetID == thoughtID {
			targetID = link.TargetThoughtID
		}
		if _, exists := seen[targetID]; exists {
			continue
		}
		seen[targetID] = struct{}{}
		thought, err := repository.buildThoughtLocked(targetID, false, false)
		if err == nil {
			related = append(related, thought)
		}
	}
	return related, nil
}

func (repository *InMemoryRepository) GetGraphNeighborhood(centerThoughtID string, distance, limit int) (*GraphNeighborhood, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	type queueItem struct {
		ThoughtID string
		Distance  int
	}

	queue := []queueItem{{ThoughtID: centerThoughtID, Distance: 0}}
	visited := map[string]struct{}{}
	var nodes []*GraphNode
	edgeMap := map[string]*ThoughtLink{}

	for len(queue) > 0 && len(nodes) < limit {
		current := queue[0]
		queue = queue[1:]
		if _, seen := visited[current.ThoughtID]; seen || current.Distance > distance {
			continue
		}
		visited[current.ThoughtID] = struct{}{}
		thought, err := repository.buildThoughtLocked(current.ThoughtID, false, false)
		if err != nil {
			continue
		}
		nodes = append(nodes, repository.layoutNode(thought, current.Distance, len(nodes)))

		for _, link := range repository.thoughtLinks {
			if link.SourceThoughtID != current.ThoughtID && link.TargetThoughtID != current.ThoughtID {
				continue
			}
			edgeMap[link.ID] = cloneLink(link)
			nextID := link.SourceThoughtID
			if nextID == current.ThoughtID {
				nextID = link.TargetThoughtID
			}
			if _, seen := visited[nextID]; !seen {
				queue = append(queue, queueItem{ThoughtID: nextID, Distance: current.Distance + 1})
			}
		}
	}

	if len(nodes) == 0 {
		return nil, errors.New("thought not found")
	}

	center := nodes[0]
	var edges []*GraphEdge
	for _, link := range edgeMap {
		if _, ok := visited[link.SourceThoughtID]; ok {
			if _, ok := visited[link.TargetThoughtID]; ok {
				edges = append(edges, &GraphEdge{Link: cloneLink(link)})
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Link.Score > edges[j].Link.Score
	})

	return &GraphNeighborhood{
		Center: center,
		Nodes:  nodes,
		Edges:  edges,
	}, nil
}

func (repository *InMemoryRepository) GetConceptByID(id string) (*Concept, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	concept, exists := repository.concepts[id]
	if !exists {
		return nil, nil
	}
	return repository.buildConceptLocked(concept), nil
}

func (repository *InMemoryRepository) GetConceptByName(name string) (*Concept, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	normalized := normalizeConceptName(name)
	for _, concept := range repository.concepts {
		if concept.NormalizedName == normalized {
			return repository.buildConceptLocked(concept), nil
		}
	}
	return nil, nil
}

func (repository *InMemoryRepository) GetConceptThoughts(conceptID string, limit int) ([]*Thought, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	var thoughts []*Thought
	for thoughtID, conceptIDs := range repository.thoughtConcepts {
		if _, exists := conceptIDs[conceptID]; !exists {
			continue
		}
		thought, err := repository.buildThoughtLocked(thoughtID, false, false)
		if err == nil {
			thoughts = append(thoughts, thought)
		}
	}
	sort.Slice(thoughts, func(i, j int) bool {
		return thoughts[i].UpdatedAt > thoughts[j].UpdatedAt
	})
	if len(thoughts) > limit {
		thoughts = thoughts[:limit]
	}
	return thoughts, nil
}

func (repository *InMemoryRepository) GetRelatedConcepts(conceptID string, limit int) ([]*Concept, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	counts := map[string]int{}
	for _, conceptIDs := range repository.thoughtConcepts {
		if _, exists := conceptIDs[conceptID]; !exists {
			continue
		}
		for relatedID := range conceptIDs {
			if relatedID == conceptID {
				continue
			}
			counts[relatedID]++
		}
	}
	type scored struct {
		ConceptID string
		Count     int
	}
	var ranked []scored
	for conceptID, count := range counts {
		ranked = append(ranked, scored{ConceptID: conceptID, Count: count})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Count > ranked[j].Count
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	var related []*Concept
	for _, entry := range ranked {
		if concept, exists := repository.concepts[entry.ConceptID]; exists {
			related = append(related, cloneConceptBase(concept))
		}
	}
	return related, nil
}

func (repository *InMemoryRepository) ReplaceThoughtLinks(thoughtID string, links []*ThoughtLink) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	for id, link := range repository.thoughtLinks {
		if link.Origin == "analysis" && (link.SourceThoughtID == thoughtID || link.TargetThoughtID == thoughtID) {
			delete(repository.thoughtLinks, id)
		}
	}

	for _, next := range links {
		pair := normalizedPair(next.SourceThoughtID, next.TargetThoughtID)
		var existingID string
		for id, link := range repository.thoughtLinks {
			if normalizedPair(link.SourceThoughtID, link.TargetThoughtID) == pair && link.RelationType == next.RelationType {
				existingID = id
				break
			}
		}
		if existingID != "" {
			if repository.thoughtLinks[existingID].Score < next.Score {
				repository.thoughtLinks[existingID].Score = next.Score
			}
			repository.thoughtLinks[existingID].Origin = next.Origin
			continue
		}
		link := cloneLink(next)
		link.ID = createID("link")
		link.CreatedAt = nowISO()
		repository.thoughtLinks[link.ID] = link
	}
	return nil
}

func normalizedPair(left, right string) string {
	if left < right {
		return left + ":" + right
	}
	return right + ":" + left
}

func (repository *InMemoryRepository) buildConceptLocked(base *Concept) *Concept {
	concept := cloneConceptBase(base)
	related, _ := repository.GetRelatedConcepts(base.ID, 8)
	topThoughts, _ := repository.GetConceptThoughts(base.ID, 8)
	concept.RelatedConcepts = related
	concept.TopThoughts = topThoughts
	return concept
}

func (repository *InMemoryRepository) buildThoughtLocked(thoughtID string, includeVersions bool, includeRelated bool) (*Thought, error) {
	base, exists := repository.thoughts[thoughtID]
	if !exists {
		return nil, errors.New("thought not found")
	}
	thought := &Thought{
		ID:               base.ID,
		AuthorID:         base.AuthorID,
		Author:           cloneUser(repository.viewer),
		CurrentVersionID: base.CurrentVersionID,
		Embedding:        append([]float64(nil), base.Embedding...),
		ProcessingStatus: base.ProcessingStatus,
		ProcessingNotes:  append([]string(nil), base.ProcessingNotes...),
		CreatedAt:        base.CreatedAt,
		UpdatedAt:        base.UpdatedAt,
	}
	for _, version := range repository.versions[thoughtID] {
		if version.ID == base.CurrentVersionID {
			thought.CurrentVersion = cloneVersion(version)
		}
		if includeVersions {
			thought.Versions = append(thought.Versions, cloneVersion(version))
		}
	}
	for conceptID := range repository.thoughtConcepts[thoughtID] {
		if concept, exists := repository.concepts[conceptID]; exists {
			thought.Concepts = append(thought.Concepts, cloneConceptBase(concept))
		}
	}
	sort.Slice(thought.Concepts, func(i, j int) bool {
		return thought.Concepts[i].Name < thought.Concepts[j].Name
	})
	var links []*ThoughtLink
	for _, link := range repository.thoughtLinks {
		if link.SourceThoughtID == thoughtID || link.TargetThoughtID == thoughtID {
			links = append(links, cloneLink(link))
		}
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].Score > links[j].Score
	})
	thought.Links = links
	if includeRelated {
		thought.RelatedThoughts = repository.relatedThoughtsLocked(thoughtID, 8)
	}
	if thought.CurrentVersion == nil {
		return nil, errors.New("current version missing")
	}
	return thought, nil
}

func (repository *InMemoryRepository) relatedThoughtsLocked(thoughtID string, limit int) []*Thought {
	var links []*ThoughtLink
	for _, link := range repository.thoughtLinks {
		if link.SourceThoughtID == thoughtID || link.TargetThoughtID == thoughtID {
			links = append(links, link)
		}
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].Score > links[j].Score
	})
	if len(links) > limit {
		links = links[:limit]
	}
	seen := map[string]struct{}{}
	var related []*Thought
	for _, link := range links {
		targetID := link.SourceThoughtID
		if targetID == thoughtID {
			targetID = link.TargetThoughtID
		}
		if _, exists := seen[targetID]; exists {
			continue
		}
		seen[targetID] = struct{}{}
		thought, err := repository.buildThoughtLocked(targetID, false, false)
		if err == nil {
			related = append(related, thought)
		}
	}
	return related
}

func (repository *InMemoryRepository) layoutNode(thought *Thought, distance, index int) *GraphNode {
	radius := 0.0
	if distance > 0 {
		radius = 160.0 * float64(distance)
	}
	angle := 0.0
	if distance > 0 {
		angle = math.Mod(float64(index)*math.Pi*0.7, math.Pi*2)
	}
	return &GraphNode{
		Thought:   thought,
		ThoughtID: thought.ID,
		X:         math.Round(math.Cos(angle)*radius*100) / 100,
		Y:         math.Round(math.Sin(angle)*radius*100) / 100,
		Distance:  clamp(distance, 0, 4),
	}
}

func cloneUser(user *User) *User {
	if user == nil {
		return nil
	}
	clone := *user
	clone.Interests = append([]string(nil), user.Interests...)
	return &clone
}

func cloneVersion(version *ThoughtVersion) *ThoughtVersion {
	if version == nil {
		return nil
	}
	clone := *version
	return &clone
}

func cloneConceptBase(concept *Concept) *Concept {
	if concept == nil {
		return nil
	}
	return &Concept{
		ID:             concept.ID,
		Name:           concept.Name,
		NormalizedName: concept.NormalizedName,
		CreatedAt:      concept.CreatedAt,
	}
}

func cloneLink(link *ThoughtLink) *ThoughtLink {
	if link == nil {
		return nil
	}
	clone := *link
	return &clone
}
