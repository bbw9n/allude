package memory

import (
	"errors"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/bbw9n/allude/services/api/internal/domains/models"
	"github.com/bbw9n/allude/services/api/internal/domains/semantics"
	"github.com/bbw9n/allude/services/api/internal/pkgs/shared"
)

const ViewerID = shared.ViewerID

var (
	nowISO               = shared.NowISO
	createID             = shared.CreateID
	normalizeConceptName = shared.NormalizeConceptName
	cosineSimilarity     = shared.CosineSimilarity
	clamp                = shared.Clamp
)

const (
	ProcessingPending    = models.ProcessingPending
	ProcessingProcessing = models.ProcessingProcessing
	ProcessingReady      = models.ProcessingReady
	ProcessingPartial    = models.ProcessingPartial
	ProcessingFailed     = models.ProcessingFailed
	RelationRelated      = models.RelationRelated
	RelationExtends      = models.RelationExtends
	RelationContradict   = models.RelationContradict
	RelationExampleOf    = models.RelationExampleOf
	JobPending           = models.JobPending
	JobLeased            = models.JobLeased
	JobCompleted         = models.JobCompleted
	JobDead              = models.JobDead
)

type User = models.User
type Thought = models.Thought
type ThoughtVersion = models.ThoughtVersion
type Concept = models.Concept
type ConceptAlias = models.ConceptAlias
type ThoughtConcept = models.ThoughtConcept
type ThoughtLink = models.ThoughtLink
type ConceptLink = models.ConceptLink
type Collection = models.Collection
type CollectionItem = models.CollectionItem
type EngagementEvent = models.EngagementEvent
type GraphNode = models.GraphNode
type GraphEdge = models.GraphEdge
type GraphNeighborhood = models.GraphNeighborhood
type SearchCluster = models.SearchCluster
type SearchThoughtsResult = models.SearchThoughtsResult
type Job = models.Job
type ProcessingStatus = models.ProcessingStatus
type RelationType = models.RelationType
type JobStatus = models.JobStatus
type JobType = models.JobType

type InMemoryRepository struct {
	mu                sync.RWMutex
	viewer            *User
	thoughts          map[string]*Thought
	versions          map[string]*ThoughtVersion
	versionsByThought map[string][]string
	concepts          map[string]*Concept
	aliases           map[string]*ConceptAlias
	thoughtConcepts   map[string]map[string]*ThoughtConcept
	thoughtLinks      map[string]*ThoughtLink
	conceptLinks      map[string]*ConceptLink
	collections       map[string]*Collection
	collectionItems   map[string][]*CollectionItem
	engagementEvents  map[string]*EngagementEvent
	jobs              map[string]*Job
	jobOrder          []string
}

func NewInMemoryRepository() *InMemoryRepository {
	now := nowISO()
	viewer := &User{
		ID:          ViewerID,
		Username:    "allude-dev",
		DisplayName: "Allude Dev",
		Bio:         "Seeded development identity",
		Interests:   []string{"philosophy", "creativity", "systems"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return &InMemoryRepository{
		viewer:            viewer,
		thoughts:          map[string]*Thought{},
		versions:          map[string]*ThoughtVersion{},
		versionsByThought: map[string][]string{},
		concepts:          map[string]*Concept{},
		aliases:           map[string]*ConceptAlias{},
		thoughtConcepts:   map[string]map[string]*ThoughtConcept{},
		thoughtLinks:      map[string]*ThoughtLink{},
		conceptLinks:      map[string]*ConceptLink{},
		collections:       map[string]*Collection{},
		collectionItems:   map[string][]*CollectionItem{},
		engagementEvents:  map[string]*EngagementEvent{},
		jobs:              map[string]*Job{},
		jobOrder:          []string{},
	}
}

func (repository *InMemoryRepository) GetViewer() *User {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	return cloneUser(repository.viewer)
}

func (repository *InMemoryRepository) ListThoughtsByAuthor(authorID string, limit int) ([]*Thought, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	thoughts := make([]*Thought, 0)
	for thoughtID, thought := range repository.thoughts {
		if thought.AuthorID != authorID {
			continue
		}
		hydrated, err := repository.hydrateThoughtLocked(thoughtID, false, false)
		if err == nil {
			thoughts = append(thoughts, hydrated)
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

func (repository *InMemoryRepository) CreateThought(authorID, content string) (*Thought, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	now := nowISO()
	thoughtID := createID("thought")
	versionID := createID("version")
	version := &ThoughtVersion{
		ID:               versionID,
		ThoughtID:        thoughtID,
		VersionNo:        1,
		Content:          content,
		TokenCount:       len(strings.Fields(content)),
		Language:         "en",
		ProcessingStatus: ProcessingPending,
		ProcessingNotes:  []string{"Queued for enrichment"},
		CreatedAt:        now,
	}
	thought := &Thought{
		ID:               thoughtID,
		AuthorID:         authorID,
		Status:           "active",
		Visibility:       "public",
		CurrentVersionID: versionID,
		ProcessingStatus: ProcessingPending,
		ProcessingNotes:  []string{"Queued for enrichment"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	repository.thoughts[thoughtID] = thought
	repository.versions[versionID] = version
	repository.versionsByThought[thoughtID] = []string{versionID}
	repository.thoughtConcepts[versionID] = map[string]*ThoughtConcept{}
	return repository.hydrateThoughtLocked(thoughtID, true, true)
}

func (repository *InMemoryRepository) UpdateThought(thoughtID, content string) (*Thought, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	thought, exists := repository.thoughts[thoughtID]
	if !exists {
		return nil, errors.New("thought not found")
	}

	now := nowISO()
	versionID := createID("version")
	version := &ThoughtVersion{
		ID:               versionID,
		ThoughtID:        thoughtID,
		VersionNo:        len(repository.versionsByThought[thoughtID]) + 1,
		Content:          content,
		TokenCount:       len(strings.Fields(content)),
		Language:         "en",
		ProcessingStatus: ProcessingPending,
		ProcessingNotes:  []string{"Queued for enrichment"},
		CreatedAt:        now,
	}
	repository.versions[versionID] = version
	repository.versionsByThought[thoughtID] = append(repository.versionsByThought[thoughtID], versionID)
	repository.thoughtConcepts[versionID] = map[string]*ThoughtConcept{}
	thought.CurrentVersionID = versionID
	thought.ProcessingStatus = ProcessingPending
	thought.ProcessingNotes = []string{"Queued for enrichment"}
	thought.UpdatedAt = now
	return repository.hydrateThoughtLocked(thoughtID, true, true)
}

func (repository *InMemoryRepository) GetThought(thoughtID string) (*Thought, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	return repository.hydrateThoughtLocked(thoughtID, true, true)
}

func (repository *InMemoryRepository) ListThoughtVersions(thoughtID string) ([]*ThoughtVersion, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	ids, exists := repository.versionsByThought[thoughtID]
	if !exists {
		return nil, errors.New("thought not found")
	}
	versions := make([]*ThoughtVersion, 0, len(ids))
	for _, id := range ids {
		versions = append(versions, cloneVersion(repository.versions[id]))
	}
	return versions, nil
}

func (repository *InMemoryRepository) SaveThoughtVersionEnrichment(versionID string, embedding []float64, conceptNames []string, status ProcessingStatus, notes []string) (*ThoughtVersion, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	version, exists := repository.versions[versionID]
	if !exists {
		return nil, errors.New("version not found")
	}

	version.Embedding = append([]float64(nil), embedding...)
	version.ProcessingStatus = status
	version.ProcessingNotes = append([]string(nil), notes...)
	repository.thoughtConcepts[versionID] = map[string]*ThoughtConcept{}

	for _, raw := range semantics.UniqueStrings(conceptNames) {
		concept := repository.upsertConceptLocked(raw)
		repository.thoughtConcepts[versionID][concept.ID] = &ThoughtConcept{
			ThoughtVersionID: versionID,
			ConceptID:        concept.ID,
			Weight:           1.0,
			Source:           "ai",
			CreatedAt:        nowISO(),
		}
	}

	thought := repository.thoughts[version.ThoughtID]
	if thought.CurrentVersionID == versionID {
		thought.ProcessingStatus = status
		thought.ProcessingNotes = append([]string(nil), notes...)
		thought.UpdatedAt = nowISO()
	}

	repository.refreshConceptLinksLocked()
	return cloneVersion(version), nil
}

func (repository *InMemoryRepository) SearchThoughts(query string, embedding []float64, limit int) (*SearchThoughtsResult, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	type scoredThought struct {
		Thought *Thought
		Score   float64
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	var ranked []scoredThought
	for thoughtID := range repository.thoughts {
		thought, err := repository.hydrateThoughtLocked(thoughtID, false, false)
		if err != nil || thought.CurrentVersion == nil {
			continue
		}
		lexical := semantics.LexicalScore(queryLower, thought.CurrentVersion.Content, thought.Concepts)
		semantic := cosineSimilarity(embedding, thought.CurrentVersion.Embedding)
		quality := repository.qualityScoreLocked(thought.ID)
		score := (0.35 * lexical) + (0.45 * semantic) + (0.20 * quality)
		if score <= 0 {
			continue
		}
		ranked = append(ranked, scoredThought{Thought: thought, Score: score})
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	result := &SearchThoughtsResult{}
	clusterMap := map[string]*SearchCluster{}
	for _, entry := range ranked {
		result.Thoughts = append(result.Thoughts, entry.Thought)
		for _, concept := range entry.Thought.Concepts {
			cluster, exists := clusterMap[concept.ID]
			if !exists {
				cluster = &SearchCluster{
					Label:      concept.CanonicalName,
					Concepts:   []*Concept{cloneConceptBase(concept)},
					ThoughtIDs: []string{},
				}
				clusterMap[concept.ID] = cluster
			}
			cluster.ThoughtIDs = semantics.AppendUniqueString(cluster.ThoughtIDs, entry.Thought.ID)
		}
	}

	for _, cluster := range clusterMap {
		result.Clusters = append(result.Clusters, cluster)
	}
	sort.Slice(result.Clusters, func(i, j int) bool {
		return len(result.Clusters[i].ThoughtIDs) > len(result.Clusters[j].ThoughtIDs)
	})
	if len(result.Clusters) > 4 {
		result.Clusters = result.Clusters[:4]
	}
	return result, nil
}

func (repository *InMemoryRepository) GetRelatedThoughts(thoughtID string, limit int) ([]*Thought, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	return repository.relatedThoughtsLocked(thoughtID, limit), nil
}

func (repository *InMemoryRepository) GetGraphNeighborhood(centerThoughtID string, hopCount, limit int) (*GraphNeighborhood, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	type queueItem struct {
		ThoughtID string
		Distance  int
	}

	queue := []queueItem{{ThoughtID: centerThoughtID, Distance: 0}}
	visited := map[string]struct{}{}
	dominantConceptCount := map[string]int{}
	nodes := []*GraphNode{}
	edges := []*GraphEdge{}
	edgeSeen := map[string]struct{}{}

	for len(queue) > 0 && len(nodes) < limit {
		current := queue[0]
		queue = queue[1:]
		if current.Distance > hopCount {
			continue
		}
		if _, exists := visited[current.ThoughtID]; exists {
			continue
		}
		thought, err := repository.hydrateThoughtLocked(current.ThoughtID, false, false)
		if err != nil {
			continue
		}
		dominantConcept := semantics.DominantConceptName(thought)
		if dominantConcept != "" && current.Distance > 0 && dominantConceptCount[dominantConcept] >= 2 {
			continue
		}
		visited[current.ThoughtID] = struct{}{}
		if dominantConcept != "" {
			dominantConceptCount[dominantConcept]++
		}
		nodes = append(nodes, repository.layoutNode(thought, current.Distance, len(nodes)))

		for _, link := range repository.sortedLinksForThoughtLocked(current.ThoughtID) {
			if _, exists := edgeSeen[link.ID]; !exists {
				edges = append(edges, &GraphEdge{Link: cloneLink(link)})
				edgeSeen[link.ID] = struct{}{}
			}
			nextID := link.SourceThoughtID
			if nextID == current.ThoughtID {
				nextID = link.TargetThoughtID
			}
			if _, exists := visited[nextID]; !exists {
				queue = append(queue, queueItem{ThoughtID: nextID, Distance: current.Distance + 1})
			}
		}
	}

	if len(nodes) == 0 {
		return nil, errors.New("thought not found")
	}

	return &GraphNeighborhood{
		Center: nodes[0],
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
	return repository.hydrateConceptLocked(concept), nil
}

func (repository *InMemoryRepository) GetConceptBySlug(slug string) (*Concept, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	for _, concept := range repository.concepts {
		if concept.Slug == slug {
			return repository.hydrateConceptLocked(concept), nil
		}
	}
	return nil, nil
}

func (repository *InMemoryRepository) GetConceptByName(name string) (*Concept, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	normalized := normalizeConceptName(name)
	for _, concept := range repository.concepts {
		if normalizeConceptName(concept.CanonicalName) == normalized {
			return repository.hydrateConceptLocked(concept), nil
		}
	}
	if alias, exists := repository.aliases[normalized]; exists {
		return repository.hydrateConceptLocked(repository.concepts[alias.ConceptID]), nil
	}
	return nil, nil
}

func (repository *InMemoryRepository) GetConceptThoughts(conceptID string, limit int) ([]*Thought, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	scoredThoughts := []*Thought{}
	for thoughtID, thought := range repository.thoughts {
		if thought.CurrentVersionID == "" {
			continue
		}
		if _, exists := repository.thoughtConcepts[thought.CurrentVersionID][conceptID]; !exists {
			continue
		}
		hydrated, err := repository.hydrateThoughtLocked(thoughtID, false, false)
		if err == nil {
			scoredThoughts = append(scoredThoughts, hydrated)
		}
	}
	sort.Slice(scoredThoughts, func(i, j int) bool {
		return repository.qualityScoreLocked(scoredThoughts[i].ID) > repository.qualityScoreLocked(scoredThoughts[j].ID)
	})
	if len(scoredThoughts) > limit {
		scoredThoughts = scoredThoughts[:limit]
	}
	return scoredThoughts, nil
}

func (repository *InMemoryRepository) GetRelatedConcepts(conceptID string, limit int) ([]*Concept, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	var related []*Concept
	for _, link := range repository.conceptLinks {
		if link.SourceConceptID == conceptID {
			if concept, exists := repository.concepts[link.TargetConceptID]; exists {
				related = append(related, cloneConceptBase(concept))
			}
		}
	}
	if len(related) > limit {
		related = related[:limit]
	}
	return related, nil
}

func (repository *InMemoryRepository) ReplaceThoughtLinks(thoughtID string, links []*ThoughtLink) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	for id, link := range repository.thoughtLinks {
		if link.Source == "analysis" && (link.SourceThoughtID == thoughtID || link.TargetThoughtID == thoughtID) {
			delete(repository.thoughtLinks, id)
		}
	}

	for _, next := range links {
		pair := semantics.NormalizedPair(next.SourceThoughtID, next.TargetThoughtID)
		var existing *ThoughtLink
		for _, link := range repository.thoughtLinks {
			if semantics.NormalizedPair(link.SourceThoughtID, link.TargetThoughtID) == pair && link.RelationType == next.RelationType {
				existing = link
				break
			}
		}
		if existing != nil {
			existing.Weight = semantics.MaxFloat(existing.Weight, next.Weight)
			existing.Source = next.Source
			existing.Explanation = next.Explanation
			continue
		}
		link := cloneLink(next)
		link.ID = createID("link")
		link.CreatedAt = nowISO()
		repository.thoughtLinks[link.ID] = link
	}
	return nil
}

func (repository *InMemoryRepository) CreateCollection(curatorID, title, description string) (*Collection, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	now := nowISO()
	collection := &Collection{
		ID:          createID("collection"),
		CuratorID:   curatorID,
		Title:       title,
		Description: description,
		Visibility:  "public",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repository.collections[collection.ID] = collection
	repository.collectionItems[collection.ID] = []*CollectionItem{}
	return cloneCollectionBase(collection), nil
}

func (repository *InMemoryRepository) AddThoughtToCollection(collectionID, thoughtID string) (*Collection, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	collection, exists := repository.collections[collectionID]
	if !exists {
		return nil, errors.New("collection not found")
	}
	if _, exists := repository.thoughts[thoughtID]; !exists {
		return nil, errors.New("thought not found")
	}
	item := &CollectionItem{
		CollectionID: collectionID,
		ThoughtID:    thoughtID,
		Position:     len(repository.collectionItems[collectionID]),
		AddedAt:      nowISO(),
	}
	repository.collectionItems[collectionID] = append(repository.collectionItems[collectionID], item)
	collection.UpdatedAt = nowISO()
	return repository.hydrateCollectionLocked(collectionID)
}

func (repository *InMemoryRepository) GetCollection(id string) (*Collection, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	return repository.hydrateCollectionLocked(id)
}

func (repository *InMemoryRepository) ListCollections() ([]*Collection, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	collections := make([]*Collection, 0, len(repository.collections))
	for id := range repository.collections {
		collection, err := repository.hydrateCollectionLocked(id)
		if err == nil {
			collections = append(collections, collection)
		}
	}
	sort.Slice(collections, func(i, j int) bool {
		return collections[i].UpdatedAt > collections[j].UpdatedAt
	})
	return collections, nil
}

func (repository *InMemoryRepository) RecordEngagement(event *EngagementEvent) (*EngagementEvent, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	clone := *event
	if clone.ID == "" {
		clone.ID = createID("engagement")
	}
	if clone.CreatedAt == "" {
		clone.CreatedAt = nowISO()
	}
	repository.engagementEvents[clone.ID] = &clone
	return &clone, nil
}

func (repository *InMemoryRepository) EnqueueJob(job *Job) (*Job, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	clone := *job
	now := nowISO()
	if clone.ID == "" {
		clone.ID = createID("job")
	}
	clone.Status = JobPending
	clone.VisibleAt = now
	clone.CreatedAt = now
	clone.UpdatedAt = now
	if clone.MaxAttempts == 0 {
		clone.MaxAttempts = 3
	}
	if clone.Payload == nil {
		clone.Payload = map[string]string{}
	}
	repository.jobs[clone.ID] = &clone
	repository.jobOrder = append(repository.jobOrder, clone.ID)
	return cloneJob(&clone), nil
}

func (repository *InMemoryRepository) LeasePendingJob(workerID string) (*Job, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	for _, id := range repository.jobOrder {
		job := repository.jobs[id]
		if job == nil || job.Status != JobPending {
			continue
		}
		job.Status = JobLeased
		job.LeaseOwner = workerID
		job.AttemptCount++
		job.UpdatedAt = nowISO()
		return cloneJob(job), nil
	}
	return nil, nil
}

func (repository *InMemoryRepository) CompleteJob(jobID string) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	job, exists := repository.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	job.Status = JobCompleted
	job.UpdatedAt = nowISO()
	return nil
}

func (repository *InMemoryRepository) FailJob(jobID, message string) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	job, exists := repository.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	job.LastError = message
	job.UpdatedAt = nowISO()
	if job.AttemptCount >= job.MaxAttempts {
		job.Status = JobDead
		return nil
	}
	job.Status = JobPending
	job.LeaseOwner = ""
	return nil
}

func (repository *InMemoryRepository) ListJobs() []*Job {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	jobs := make([]*Job, 0, len(repository.jobOrder))
	for _, id := range repository.jobOrder {
		jobs = append(jobs, cloneJob(repository.jobs[id]))
	}
	return jobs
}

func (repository *InMemoryRepository) hydrateThoughtLocked(thoughtID string, includeVersions, includeRelated bool) (*Thought, error) {
	base, exists := repository.thoughts[thoughtID]
	if !exists {
		return nil, errors.New("thought not found")
	}
	thought := &Thought{
		ID:               base.ID,
		AuthorID:         base.AuthorID,
		Author:           cloneUser(repository.viewer),
		Status:           base.Status,
		Visibility:       base.Visibility,
		CurrentVersionID: base.CurrentVersionID,
		ProcessingStatus: base.ProcessingStatus,
		ProcessingNotes:  append([]string(nil), base.ProcessingNotes...),
		CreatedAt:        base.CreatedAt,
		UpdatedAt:        base.UpdatedAt,
	}
	for _, versionID := range repository.versionsByThought[thoughtID] {
		version := repository.versions[versionID]
		if version.ID == base.CurrentVersionID {
			thought.CurrentVersion = cloneVersion(version)
		}
		if includeVersions {
			thought.Versions = append(thought.Versions, cloneVersion(version))
		}
	}
	if thought.CurrentVersion == nil {
		return nil, errors.New("current version missing")
	}
	for conceptID := range repository.thoughtConcepts[thought.CurrentVersion.ID] {
		if concept, exists := repository.concepts[conceptID]; exists {
			thought.Concepts = append(thought.Concepts, cloneConceptBase(concept))
		}
	}
	sort.Slice(thought.Concepts, func(i, j int) bool {
		return thought.Concepts[i].CanonicalName < thought.Concepts[j].CanonicalName
	})
	for _, link := range repository.sortedLinksForThoughtLocked(thoughtID) {
		thought.Links = append(thought.Links, cloneLink(link))
	}
	for collectionID, items := range repository.collectionItems {
		for _, item := range items {
			if item.ThoughtID == thoughtID {
				if collection, exists := repository.collections[collectionID]; exists {
					thought.Collections = append(thought.Collections, cloneCollectionBase(collection))
				}
			}
		}
	}
	if includeRelated {
		thought.RelatedThoughts = repository.relatedThoughtsLocked(thoughtID, 8)
	}
	return thought, nil
}

func (repository *InMemoryRepository) hydrateConceptLocked(base *Concept) *Concept {
	concept := cloneConceptBase(base)
	for _, alias := range repository.aliases {
		if alias.ConceptID == concept.ID {
			clone := *alias
			concept.Aliases = append(concept.Aliases, &clone)
		}
	}
	concept.TopThoughts, _ = repository.GetConceptThoughts(concept.ID, 8)
	concept.RelatedConcepts, _ = repository.GetRelatedConcepts(concept.ID, 8)
	concept.ContradictionThoughts = repository.contradictionThoughtsForConceptLocked(concept.ID, 6)
	concept.ThoughtCount = repository.thoughtCountForConceptLocked(concept.ID)
	return concept
}

func (repository *InMemoryRepository) hydrateCollectionLocked(id string) (*Collection, error) {
	base, exists := repository.collections[id]
	if !exists {
		return nil, errors.New("collection not found")
	}
	collection := cloneCollectionBase(base)
	for _, item := range repository.collectionItems[id] {
		clone := *item
		if thought, err := repository.hydrateThoughtLocked(item.ThoughtID, false, false); err == nil {
			clone.Thought = thought
		}
		collection.Items = append(collection.Items, &clone)
	}
	return collection, nil
}

func (repository *InMemoryRepository) sortedLinksForThoughtLocked(thoughtID string) []*ThoughtLink {
	var links []*ThoughtLink
	for _, link := range repository.thoughtLinks {
		if link.SourceThoughtID == thoughtID || link.TargetThoughtID == thoughtID {
			links = append(links, link)
		}
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].Weight > links[j].Weight
	})
	return links
}

func (repository *InMemoryRepository) relatedThoughtsLocked(thoughtID string, limit int) []*Thought {
	links := repository.sortedLinksForThoughtLocked(thoughtID)
	if len(links) > limit {
		links = links[:limit]
	}
	seen := map[string]struct{}{}
	var thoughts []*Thought
	for _, link := range links {
		relatedID := link.SourceThoughtID
		if relatedID == thoughtID {
			relatedID = link.TargetThoughtID
		}
		if _, exists := seen[relatedID]; exists {
			continue
		}
		seen[relatedID] = struct{}{}
		thought, err := repository.hydrateThoughtLocked(relatedID, false, false)
		if err == nil {
			thoughts = append(thoughts, thought)
		}
	}
	return thoughts
}

func (repository *InMemoryRepository) refreshConceptLinksLocked() {
	repository.conceptLinks = map[string]*ConceptLink{}
	counts := map[string]float64{}
	for _, conceptEdges := range repository.thoughtConcepts {
		var ids []string
		for conceptID := range conceptEdges {
			ids = append(ids, conceptID)
		}
		sort.Strings(ids)
		for index := 0; index < len(ids); index++ {
			for other := index + 1; other < len(ids); other++ {
				key := ids[index] + ":" + ids[other]
				counts[key]++
			}
		}
	}
	for pair, weight := range counts {
		parts := strings.Split(pair, ":")
		repository.conceptLinks[createID("concept_link")] = &ConceptLink{
			ID:              createID("concept_link"),
			SourceConceptID: parts[0],
			TargetConceptID: parts[1],
			RelationType:    RelationRelated,
			Weight:          weight,
			Source:          "co_occurrence",
			CreatedAt:       nowISO(),
		}
	}
}

func (repository *InMemoryRepository) upsertConceptLocked(raw string) *Concept {
	normalized := normalizeConceptName(raw)
	if alias, exists := repository.aliases[normalized]; exists {
		return repository.concepts[alias.ConceptID]
	}
	for _, concept := range repository.concepts {
		if normalizeConceptName(concept.CanonicalName) == normalized {
			return concept
		}
	}
	now := nowISO()
	concept := &Concept{
		ID:            createID("concept"),
		CanonicalName: strings.TrimSpace(raw),
		Slug:          semantics.Slugify(raw),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	repository.concepts[concept.ID] = concept
	alias := &ConceptAlias{
		ID:              createID("alias"),
		ConceptID:       concept.ID,
		Alias:           concept.CanonicalName,
		NormalizedAlias: normalized,
	}
	repository.aliases[normalized] = alias
	return concept
}

func (repository *InMemoryRepository) qualityScoreLocked(thoughtID string) float64 {
	score := 0.2
	for _, items := range repository.collectionItems {
		for _, item := range items {
			if item.ThoughtID == thoughtID {
				score += 0.2
			}
		}
	}
	for _, link := range repository.thoughtLinks {
		if link.SourceThoughtID == thoughtID || link.TargetThoughtID == thoughtID {
			score += 0.1
		}
	}
	if score > 1 {
		score = 1
	}
	return score
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
	clone.Embedding = append([]float64(nil), version.Embedding...)
	clone.ProcessingNotes = append([]string(nil), version.ProcessingNotes...)
	return &clone
}

func cloneConceptBase(concept *Concept) *Concept {
	if concept == nil {
		return nil
	}
	return &Concept{
		ID:            concept.ID,
		CanonicalName: concept.CanonicalName,
		Slug:          concept.Slug,
		Description:   concept.Description,
		ConceptType:   concept.ConceptType,
		ThoughtCount:  concept.ThoughtCount,
		CreatedAt:     concept.CreatedAt,
		UpdatedAt:     concept.UpdatedAt,
	}
}

func (repository *InMemoryRepository) thoughtCountForConceptLocked(conceptID string) int {
	count := 0
	for _, thought := range repository.thoughts {
		if thought.CurrentVersionID == "" {
			continue
		}
		if _, exists := repository.thoughtConcepts[thought.CurrentVersionID][conceptID]; exists {
			count++
		}
	}
	return count
}

func (repository *InMemoryRepository) contradictionThoughtsForConceptLocked(conceptID string, limit int) []*Thought {
	scored := map[string]float64{}
	for _, link := range repository.thoughtLinks {
		if link.RelationType != RelationContradict {
			continue
		}
		source := repository.thoughts[link.SourceThoughtID]
		target := repository.thoughts[link.TargetThoughtID]
		if source == nil || target == nil || source.CurrentVersionID == "" || target.CurrentVersionID == "" {
			continue
		}
		sourceHas := repository.thoughtConcepts[source.CurrentVersionID][conceptID] != nil
		targetHas := repository.thoughtConcepts[target.CurrentVersionID][conceptID] != nil
		if !sourceHas && !targetHas {
			continue
		}
		if sourceHas {
			scored[source.ID] = semantics.MaxFloat(scored[source.ID], link.Weight)
		}
		if targetHas {
			scored[target.ID] = semantics.MaxFloat(scored[target.ID], link.Weight)
		}
	}

	type rankedThought struct {
		id    string
		score float64
	}
	ranked := make([]rankedThought, 0, len(scored))
	for id, score := range scored {
		ranked = append(ranked, rankedThought{id: id, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	result := make([]*Thought, 0, len(ranked))
	for _, item := range ranked {
		thought, err := repository.hydrateThoughtLocked(item.id, false, false)
		if err == nil {
			result = append(result, thought)
		}
	}
	return result
}

func cloneCollectionBase(collection *Collection) *Collection {
	if collection == nil {
		return nil
	}
	clone := *collection
	clone.Items = nil
	return &clone
}

func cloneLink(link *ThoughtLink) *ThoughtLink {
	if link == nil {
		return nil
	}
	clone := *link
	return &clone
}

func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	clone := *job
	clone.Payload = map[string]string{}
	for key, value := range job.Payload {
		clone.Payload[key] = value
	}
	return &clone
}
