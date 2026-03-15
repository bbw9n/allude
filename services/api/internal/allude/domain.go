package allude

type ProcessingStatus string

const (
	ProcessingPending    ProcessingStatus = "PENDING"
	ProcessingProcessing ProcessingStatus = "PROCESSING"
	ProcessingReady      ProcessingStatus = "READY"
	ProcessingPartial    ProcessingStatus = "PARTIAL"
	ProcessingFailed     ProcessingStatus = "FAILED"
)

type RelationType string

const (
	RelationRelated    RelationType = "related"
	RelationExtends    RelationType = "extends"
	RelationContradict RelationType = "contradicts"
	RelationExampleOf  RelationType = "example_of"
)

type User struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Bio       string   `json:"bio,omitempty"`
	Interests []string `json:"interests"`
}

type Thought struct {
	ID               string            `json:"id"`
	Author           *User             `json:"author,omitempty"`
	AuthorID         string            `json:"-"`
	CurrentVersionID string            `json:"-"`
	Embedding        []float64         `json:"-"`
	CurrentVersion   *ThoughtVersion   `json:"currentVersion"`
	Versions         []*ThoughtVersion `json:"versions,omitempty"`
	Concepts         []*Concept        `json:"concepts"`
	RelatedThoughts  []*Thought        `json:"relatedThoughts,omitempty"`
	Links            []*ThoughtLink    `json:"links,omitempty"`
	ProcessingStatus ProcessingStatus  `json:"processingStatus"`
	ProcessingNotes  []string          `json:"processingNotes"`
	CreatedAt        string            `json:"createdAt"`
	UpdatedAt        string            `json:"updatedAt"`
}

type ThoughtVersion struct {
	ID        string `json:"id"`
	ThoughtID string `json:"thoughtId"`
	Version   int    `json:"version"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type Concept struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	NormalizedName  string     `json:"-"`
	Embedding       []float64  `json:"-"`
	CreatedAt       string     `json:"createdAt"`
	RelatedConcepts []*Concept `json:"relatedConcepts,omitempty"`
	TopThoughts     []*Thought `json:"topThoughts,omitempty"`
}

type ThoughtLink struct {
	ID              string       `json:"id"`
	SourceThoughtID string       `json:"sourceThoughtId"`
	TargetThoughtID string       `json:"targetThoughtId"`
	RelationType    RelationType `json:"relationType"`
	Score           float64      `json:"score"`
	Origin          string       `json:"origin"`
	CreatedAt       string       `json:"createdAt"`
}

type Collection struct {
	ID          string `json:"id"`
	CuratorID   string `json:"curatorId"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type GraphNode struct {
	Thought   *Thought `json:"thought"`
	ThoughtID string   `json:"-"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	Distance  int      `json:"distance"`
}

type GraphEdge struct {
	Link *ThoughtLink `json:"link"`
}

type GraphNeighborhood struct {
	Center *GraphNode   `json:"center"`
	Nodes  []*GraphNode `json:"nodes"`
	Edges  []*GraphEdge `json:"edges"`
}

type SearchCluster struct {
	Label      string     `json:"label"`
	Concepts   []*Concept `json:"concepts"`
	ThoughtIDs []string   `json:"thoughtIds"`
}

type SearchThoughtsResult struct {
	Thoughts []*Thought       `json:"thoughts"`
	Clusters []*SearchCluster `json:"clusters"`
}
