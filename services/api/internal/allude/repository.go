package allude

type Repository interface {
	GetViewer() *User
	CreateThought(authorID, content string) (*Thought, error)
	UpdateThought(thoughtID, content string) (*Thought, error)
	GetThought(thoughtID string) (*Thought, error)
	ListThoughtVersions(thoughtID string) ([]*ThoughtVersion, error)
	SaveThoughtAnalysis(thoughtID string, embedding []float64, conceptNames []string, status ProcessingStatus, notes []string) (*Thought, error)
	SearchThoughtsByEmbedding(embedding []float64, query string, limit int) (*SearchThoughtsResult, error)
	GetRelatedThoughts(thoughtID string, limit int) ([]*Thought, error)
	GetGraphNeighborhood(centerThoughtID string, distance, limit int) (*GraphNeighborhood, error)
	GetConceptByID(id string) (*Concept, error)
	GetConceptByName(name string) (*Concept, error)
	GetConceptThoughts(conceptID string, limit int) ([]*Thought, error)
	GetRelatedConcepts(conceptID string, limit int) ([]*Concept, error)
	ReplaceThoughtLinks(thoughtID string, links []*ThoughtLink) error
}
