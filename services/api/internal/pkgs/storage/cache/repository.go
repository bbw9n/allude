package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/bbw9n/allude/services/api/internal/domains/models"
	"github.com/bbw9n/allude/services/api/internal/domains/ports"
)

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

type CachedRepository struct {
	next    ports.Repository
	ttl     time.Duration
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func NewCachedRepository(next ports.Repository, ttl time.Duration) *CachedRepository {
	return &CachedRepository{
		next:    next,
		ttl:     ttl,
		entries: map[string]cacheEntry{},
	}
}

func (repository *CachedRepository) GetViewer() *models.User {
	return repository.next.GetViewer()
}

func (repository *CachedRepository) CreateThought(authorID, content string) (*models.Thought, error) {
	thought, err := repository.next.CreateThought(authorID, content)
	if err == nil {
		repository.invalidateThought(thought.ID)
	}
	return thought, err
}

func (repository *CachedRepository) UpdateThought(thoughtID, content string) (*models.Thought, error) {
	thought, err := repository.next.UpdateThought(thoughtID, content)
	if err == nil {
		repository.invalidateThought(thoughtID)
	}
	return thought, err
}

func (repository *CachedRepository) GetThought(thoughtID string) (*models.Thought, error) {
	key := "thought:" + thoughtID
	if value, ok := repository.get(key); ok {
		return value.(*models.Thought), nil
	}
	thought, err := repository.next.GetThought(thoughtID)
	if err == nil && thought != nil {
		repository.set(key, thought)
	}
	return thought, err
}

func (repository *CachedRepository) ListThoughtVersions(thoughtID string) ([]*models.ThoughtVersion, error) {
	return repository.next.ListThoughtVersions(thoughtID)
}

func (repository *CachedRepository) SaveThoughtVersionEnrichment(versionID string, embedding []float64, conceptNames []string, status models.ProcessingStatus, notes []string) (*models.ThoughtVersion, error) {
	version, err := repository.next.SaveThoughtVersionEnrichment(versionID, embedding, conceptNames, status, notes)
	if err == nil {
		repository.invalidateAll()
	}
	return version, err
}

func (repository *CachedRepository) SearchThoughts(query string, embedding []float64, limit int) (*models.SearchThoughtsResult, error) {
	key := fmt.Sprintf("search:%s:%d", query, limit)
	if value, ok := repository.get(key); ok {
		return value.(*models.SearchThoughtsResult), nil
	}
	result, err := repository.next.SearchThoughts(query, embedding, limit)
	if err == nil && result != nil {
		repository.set(key, result)
	}
	return result, err
}

func (repository *CachedRepository) GetRelatedThoughts(thoughtID string, limit int) ([]*models.Thought, error) {
	return repository.next.GetRelatedThoughts(thoughtID, limit)
}

func (repository *CachedRepository) GetGraphNeighborhood(centerThoughtID string, hopCount, limit int) (*models.GraphNeighborhood, error) {
	key := fmt.Sprintf("graph:%s:%d:%d", centerThoughtID, hopCount, limit)
	if value, ok := repository.get(key); ok {
		return value.(*models.GraphNeighborhood), nil
	}
	graph, err := repository.next.GetGraphNeighborhood(centerThoughtID, hopCount, limit)
	if err == nil && graph != nil {
		repository.set(key, graph)
	}
	return graph, err
}

func (repository *CachedRepository) GetConceptByID(id string) (*models.Concept, error) {
	key := "concept:id:" + id
	if value, ok := repository.get(key); ok {
		return value.(*models.Concept), nil
	}
	concept, err := repository.next.GetConceptByID(id)
	if err == nil && concept != nil {
		repository.set(key, concept)
	}
	return concept, err
}

func (repository *CachedRepository) GetConceptBySlug(slug string) (*models.Concept, error) {
	key := "concept:slug:" + slug
	if value, ok := repository.get(key); ok {
		return value.(*models.Concept), nil
	}
	concept, err := repository.next.GetConceptBySlug(slug)
	if err == nil && concept != nil {
		repository.set(key, concept)
	}
	return concept, err
}

func (repository *CachedRepository) GetConceptByName(name string) (*models.Concept, error) {
	key := "concept:name:" + name
	if value, ok := repository.get(key); ok {
		return value.(*models.Concept), nil
	}
	concept, err := repository.next.GetConceptByName(name)
	if err == nil && concept != nil {
		repository.set(key, concept)
	}
	return concept, err
}

func (repository *CachedRepository) GetConceptThoughts(conceptID string, limit int) ([]*models.Thought, error) {
	return repository.next.GetConceptThoughts(conceptID, limit)
}

func (repository *CachedRepository) GetRelatedConcepts(conceptID string, limit int) ([]*models.Concept, error) {
	return repository.next.GetRelatedConcepts(conceptID, limit)
}

func (repository *CachedRepository) ReplaceThoughtLinks(thoughtID string, links []*models.ThoughtLink) error {
	err := repository.next.ReplaceThoughtLinks(thoughtID, links)
	if err == nil {
		repository.invalidateAll()
	}
	return err
}

func (repository *CachedRepository) CreateCollection(curatorID, title, description string) (*models.Collection, error) {
	collection, err := repository.next.CreateCollection(curatorID, title, description)
	if err == nil {
		repository.invalidateAll()
	}
	return collection, err
}

func (repository *CachedRepository) AddThoughtToCollection(collectionID, thoughtID string) (*models.Collection, error) {
	collection, err := repository.next.AddThoughtToCollection(collectionID, thoughtID)
	if err == nil {
		repository.invalidateAll()
	}
	return collection, err
}

func (repository *CachedRepository) GetCollection(id string) (*models.Collection, error) {
	key := "collection:" + id
	if value, ok := repository.get(key); ok {
		return value.(*models.Collection), nil
	}
	collection, err := repository.next.GetCollection(id)
	if err == nil && collection != nil {
		repository.set(key, collection)
	}
	return collection, err
}

func (repository *CachedRepository) ListCollections() ([]*models.Collection, error) {
	key := "collections:list"
	if value, ok := repository.get(key); ok {
		return value.([]*models.Collection), nil
	}
	collections, err := repository.next.ListCollections()
	if err == nil && collections != nil {
		repository.set(key, collections)
	}
	return collections, err
}

func (repository *CachedRepository) RecordEngagement(event *models.EngagementEvent) (*models.EngagementEvent, error) {
	return repository.next.RecordEngagement(event)
}

func (repository *CachedRepository) EnqueueJob(job *models.Job) (*models.Job, error) {
	return repository.next.EnqueueJob(job)
}

func (repository *CachedRepository) LeasePendingJob(workerID string) (*models.Job, error) {
	return repository.next.LeasePendingJob(workerID)
}

func (repository *CachedRepository) CompleteJob(jobID string) error {
	return repository.next.CompleteJob(jobID)
}

func (repository *CachedRepository) FailJob(jobID, message string) error {
	return repository.next.FailJob(jobID, message)
}

func (repository *CachedRepository) ListJobs() []*models.Job {
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
