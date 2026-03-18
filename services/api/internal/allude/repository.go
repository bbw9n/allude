package allude

type Repository interface {
	GetViewer() *User
	CreateThought(authorID, content string) (*Thought, error)
	UpdateThought(thoughtID, content string) (*Thought, error)
	GetThought(thoughtID string) (*Thought, error)
	ListThoughtVersions(thoughtID string) ([]*ThoughtVersion, error)
	SaveThoughtVersionEnrichment(versionID string, embedding []float64, conceptNames []string, status ProcessingStatus, notes []string) (*ThoughtVersion, error)
	SearchThoughts(query string, embedding []float64, limit int) (*SearchThoughtsResult, error)
	GetRelatedThoughts(thoughtID string, limit int) ([]*Thought, error)
	GetGraphNeighborhood(centerThoughtID string, hopCount, limit int) (*GraphNeighborhood, error)
	GetConceptByID(id string) (*Concept, error)
	GetConceptBySlug(slug string) (*Concept, error)
	GetConceptByName(name string) (*Concept, error)
	GetConceptThoughts(conceptID string, limit int) ([]*Thought, error)
	GetRelatedConcepts(conceptID string, limit int) ([]*Concept, error)
	ReplaceThoughtLinks(thoughtID string, links []*ThoughtLink) error
	CreateCollection(curatorID, title, description string) (*Collection, error)
	AddThoughtToCollection(collectionID, thoughtID string) (*Collection, error)
	GetCollection(id string) (*Collection, error)
	RecordEngagement(event *EngagementEvent) (*EngagementEvent, error)
	EnqueueJob(job *Job) (*Job, error)
	LeasePendingJob(workerID string) (*Job, error)
	CompleteJob(jobID string) error
	FailJob(jobID, message string) error
	ListJobs() []*Job
}
