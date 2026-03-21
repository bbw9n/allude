package ports

import "github.com/bbw9n/allude/services/api/internal/domains/models"

type Repository interface {
	GetViewer() *models.User
	CreateThought(authorID, content string) (*models.Thought, error)
	UpdateThought(thoughtID, content string) (*models.Thought, error)
	GetThought(thoughtID string) (*models.Thought, error)
	ListThoughtVersions(thoughtID string) ([]*models.ThoughtVersion, error)
	SaveThoughtVersionEnrichment(versionID string, embedding []float64, conceptNames []string, status models.ProcessingStatus, notes []string) (*models.ThoughtVersion, error)
	SearchThoughts(query string, embedding []float64, limit int) (*models.SearchThoughtsResult, error)
	GetRelatedThoughts(thoughtID string, limit int) ([]*models.Thought, error)
	GetGraphNeighborhood(centerThoughtID string, hopCount, limit int) (*models.GraphNeighborhood, error)
	GetConceptByID(id string) (*models.Concept, error)
	GetConceptBySlug(slug string) (*models.Concept, error)
	GetConceptByName(name string) (*models.Concept, error)
	GetConceptThoughts(conceptID string, limit int) ([]*models.Thought, error)
	GetRelatedConcepts(conceptID string, limit int) ([]*models.Concept, error)
	ReplaceThoughtLinks(thoughtID string, links []*models.ThoughtLink) error
	CreateCollection(curatorID, title, description string) (*models.Collection, error)
	AddThoughtToCollection(collectionID, thoughtID string) (*models.Collection, error)
	GetCollection(id string) (*models.Collection, error)
	RecordEngagement(event *models.EngagementEvent) (*models.EngagementEvent, error)
	EnqueueJob(job *models.Job) (*models.Job, error)
	LeasePendingJob(workerID string) (*models.Job, error)
	CompleteJob(jobID string) error
	FailJob(jobID, message string) error
	ListJobs() []*models.Job
}
