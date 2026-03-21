package allude

import (
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

type CachedRepository struct {
	next    Repository
	ttl     time.Duration
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func NewCachedRepository(next Repository, ttl time.Duration) *CachedRepository {
	return &CachedRepository{
		next:    next,
		ttl:     ttl,
		entries: map[string]cacheEntry{},
	}
}

func (repository *CachedRepository) GetViewer() *User {
	return repository.next.GetViewer()
}

func (repository *CachedRepository) CreateThought(authorID, content string) (*Thought, error) {
	thought, err := repository.next.CreateThought(authorID, content)
	if err == nil {
		repository.invalidateThought(thought.ID)
	}
	return thought, err
}

func (repository *CachedRepository) UpdateThought(thoughtID, content string) (*Thought, error) {
	thought, err := repository.next.UpdateThought(thoughtID, content)
	if err == nil {
		repository.invalidateThought(thoughtID)
	}
	return thought, err
}

func (repository *CachedRepository) GetThought(thoughtID string) (*Thought, error) {
	key := "thought:" + thoughtID
	if value, ok := repository.get(key); ok {
		return value.(*Thought), nil
	}
	thought, err := repository.next.GetThought(thoughtID)
	if err == nil && thought != nil {
		repository.set(key, thought)
	}
	return thought, err
}

func (repository *CachedRepository) ListThoughtVersions(thoughtID string) ([]*ThoughtVersion, error) {
	return repository.next.ListThoughtVersions(thoughtID)
}

func (repository *CachedRepository) SaveThoughtVersionEnrichment(versionID string, embedding []float64, conceptNames []string, status ProcessingStatus, notes []string) (*ThoughtVersion, error) {
	version, err := repository.next.SaveThoughtVersionEnrichment(versionID, embedding, conceptNames, status, notes)
	if err == nil {
		repository.invalidateAll()
	}
	return version, err
}

func (repository *CachedRepository) SearchThoughts(query string, embedding []float64, limit int) (*SearchThoughtsResult, error) {
	key := fmt.Sprintf("search:%s:%d", query, limit)
	if value, ok := repository.get(key); ok {
		return value.(*SearchThoughtsResult), nil
	}
	result, err := repository.next.SearchThoughts(query, embedding, limit)
	if err == nil && result != nil {
		repository.set(key, result)
	}
	return result, err
}

func (repository *CachedRepository) GetRelatedThoughts(thoughtID string, limit int) ([]*Thought, error) {
	return repository.next.GetRelatedThoughts(thoughtID, limit)
}

func (repository *CachedRepository) GetGraphNeighborhood(centerThoughtID string, hopCount, limit int) (*GraphNeighborhood, error) {
	key := fmt.Sprintf("graph:%s:%d:%d", centerThoughtID, hopCount, limit)
	if value, ok := repository.get(key); ok {
		return value.(*GraphNeighborhood), nil
	}
	graph, err := repository.next.GetGraphNeighborhood(centerThoughtID, hopCount, limit)
	if err == nil && graph != nil {
		repository.set(key, graph)
	}
	return graph, err
}

func (repository *CachedRepository) GetConceptByID(id string) (*Concept, error) {
	key := "concept:id:" + id
	if value, ok := repository.get(key); ok {
		return value.(*Concept), nil
	}
	concept, err := repository.next.GetConceptByID(id)
	if err == nil && concept != nil {
		repository.set(key, concept)
	}
	return concept, err
}

func (repository *CachedRepository) GetConceptBySlug(slug string) (*Concept, error) {
	key := "concept:slug:" + slug
	if value, ok := repository.get(key); ok {
		return value.(*Concept), nil
	}
	concept, err := repository.next.GetConceptBySlug(slug)
	if err == nil && concept != nil {
		repository.set(key, concept)
	}
	return concept, err
}

func (repository *CachedRepository) GetConceptByName(name string) (*Concept, error) {
	key := "concept:name:" + name
	if value, ok := repository.get(key); ok {
		return value.(*Concept), nil
	}
	concept, err := repository.next.GetConceptByName(name)
	if err == nil && concept != nil {
		repository.set(key, concept)
	}
	return concept, err
}

func (repository *CachedRepository) GetConceptThoughts(conceptID string, limit int) ([]*Thought, error) {
	return repository.next.GetConceptThoughts(conceptID, limit)
}

func (repository *CachedRepository) GetRelatedConcepts(conceptID string, limit int) ([]*Concept, error) {
	return repository.next.GetRelatedConcepts(conceptID, limit)
}

func (repository *CachedRepository) ReplaceThoughtLinks(thoughtID string, links []*ThoughtLink) error {
	err := repository.next.ReplaceThoughtLinks(thoughtID, links)
	if err == nil {
		repository.invalidateAll()
	}
	return err
}

func (repository *CachedRepository) CreateCollection(curatorID, title, description string) (*Collection, error) {
	collection, err := repository.next.CreateCollection(curatorID, title, description)
	if err == nil {
		repository.invalidateAll()
	}
	return collection, err
}

func (repository *CachedRepository) AddThoughtToCollection(collectionID, thoughtID string) (*Collection, error) {
	collection, err := repository.next.AddThoughtToCollection(collectionID, thoughtID)
	if err == nil {
		repository.invalidateAll()
	}
	return collection, err
}

func (repository *CachedRepository) GetCollection(id string) (*Collection, error) {
	key := "collection:" + id
	if value, ok := repository.get(key); ok {
		return value.(*Collection), nil
	}
	collection, err := repository.next.GetCollection(id)
	if err == nil && collection != nil {
		repository.set(key, collection)
	}
	return collection, err
}

func (repository *CachedRepository) RecordEngagement(event *EngagementEvent) (*EngagementEvent, error) {
	return repository.next.RecordEngagement(event)
}

func (repository *CachedRepository) EnqueueJob(job *Job) (*Job, error) {
	return repository.next.EnqueueJob(job)
}

func (repository *CachedRepository) LeasePendingJob(workerID string) (*Job, error) {
	return repository.next.LeasePendingJob(workerID)
}

func (repository *CachedRepository) CompleteJob(jobID string) error {
	return repository.next.CompleteJob(jobID)
}

func (repository *CachedRepository) FailJob(jobID, message string) error {
	return repository.next.FailJob(jobID, message)
}

func (repository *CachedRepository) ListJobs() []*Job {
	return repository.next.ListJobs()
}

func (repository *CachedRepository) get(key string) (interface{}, bool) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	entry, exists := repository.entries[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

func (repository *CachedRepository) set(key string, value interface{}) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.entries[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(repository.ttl),
	}
}

func (repository *CachedRepository) invalidateThought(thoughtID string) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	delete(repository.entries, "thought:"+thoughtID)
}

func (repository *CachedRepository) invalidateAll() {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.entries = map[string]cacheEntry{}
}
