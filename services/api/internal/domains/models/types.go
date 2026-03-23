package models

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

type JobStatus string

const (
	JobPending   JobStatus = "PENDING"
	JobLeased    JobStatus = "LEASED"
	JobCompleted JobStatus = "COMPLETED"
	JobDead      JobStatus = "DEAD"
)

type JobType string

const (
	JobEmbedThoughtVersion   JobType = "embed_thought_version"
	JobExtractConcepts       JobType = "extract_concepts"
	JobLinkThought           JobType = "link_thought"
	JobRefreshConceptSummary JobType = "refresh_concept_summary"
	JobRefreshCurrents       JobType = "refresh_currents"
)

type CaptureStatus string

const (
	CaptureInbox    CaptureStatus = "inbox"
	CaptureArchived CaptureStatus = "archived"
	CapturePromoted CaptureStatus = "promoted"
)

type CaptureSourceType string

const (
	CaptureSourceText  CaptureSourceType = "text"
	CaptureSourceQuote CaptureSourceType = "quote"
	CaptureSourceLink  CaptureSourceType = "link"
	CaptureSourceNote  CaptureSourceType = "note"
)

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName,omitempty"`
	Bio         string   `json:"bio,omitempty"`
	AvatarURL   string   `json:"avatarUrl,omitempty"`
	Interests   []string `json:"interests"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type UserInterest struct {
	UserID        string   `json:"userId"`
	ConceptID     string   `json:"conceptId"`
	AffinityScore float64  `json:"affinityScore"`
	Source        string   `json:"source"`
	UpdatedAt     string   `json:"updatedAt"`
	Concept       *Concept `json:"concept,omitempty"`
}

type Thought struct {
	ID               string            `json:"id"`
	Author           *User             `json:"author,omitempty"`
	AuthorID         string            `json:"-"`
	Status           string            `json:"status"`
	Visibility       string            `json:"visibility"`
	CurrentVersionID string            `json:"-"`
	CurrentVersion   *ThoughtVersion   `json:"currentVersion"`
	Versions         []*ThoughtVersion `json:"versions,omitempty"`
	Concepts         []*Concept        `json:"concepts"`
	RelatedThoughts  []*Thought        `json:"relatedThoughts,omitempty"`
	Links            []*ThoughtLink    `json:"links,omitempty"`
	Collections      []*Collection     `json:"collections,omitempty"`
	ProcessingStatus ProcessingStatus  `json:"processingStatus"`
	ProcessingNotes  []string          `json:"processingNotes"`
	CreatedAt        string            `json:"createdAt"`
	UpdatedAt        string            `json:"updatedAt"`
}

type ThoughtVersion struct {
	ID               string           `json:"id"`
	ThoughtID        string           `json:"thoughtId"`
	VersionNo        int              `json:"version"`
	Content          string           `json:"content"`
	Embedding        []float64        `json:"-"`
	Language         string           `json:"language,omitempty"`
	TokenCount       int              `json:"tokenCount"`
	ProcessingStatus ProcessingStatus `json:"processingStatus"`
	ProcessingNotes  []string         `json:"processingNotes"`
	CreatedAt        string           `json:"createdAt"`
}

type Concept struct {
	ID                    string          `json:"id"`
	CanonicalName         string          `json:"canonicalName"`
	Slug                  string          `json:"slug"`
	Description           string          `json:"description,omitempty"`
	Embedding             []float64       `json:"-"`
	ConceptType           string          `json:"conceptType,omitempty"`
	ThoughtCount          int             `json:"thoughtCount"`
	Aliases               []*ConceptAlias `json:"aliases,omitempty"`
	RelatedConcepts       []*Concept      `json:"relatedConcepts,omitempty"`
	TopThoughts           []*Thought      `json:"topThoughts,omitempty"`
	ContradictionThoughts []*Thought      `json:"contradictionThoughts,omitempty"`
	CreatedAt             string          `json:"createdAt"`
	UpdatedAt             string          `json:"updatedAt"`
}

type ConceptAlias struct {
	ID              string `json:"id"`
	ConceptID       string `json:"conceptId"`
	Alias           string `json:"alias"`
	NormalizedAlias string `json:"normalizedAlias"`
}

type ThoughtConcept struct {
	ThoughtVersionID string  `json:"thoughtVersionId"`
	ConceptID        string  `json:"conceptId"`
	Weight           float64 `json:"weight"`
	Source           string  `json:"source"`
	CreatedAt        string  `json:"createdAt"`
}

type ThoughtLink struct {
	ID              string       `json:"id"`
	SourceThoughtID string       `json:"sourceThoughtId"`
	TargetThoughtID string       `json:"targetThoughtId"`
	RelationType    RelationType `json:"relationType"`
	Weight          float64      `json:"weight"`
	Source          string       `json:"source"`
	Explanation     string       `json:"explanation,omitempty"`
	CreatedAt       string       `json:"createdAt"`
}

type ConceptLink struct {
	ID              string       `json:"id"`
	SourceConceptID string       `json:"sourceConceptId"`
	TargetConceptID string       `json:"targetConceptId"`
	RelationType    RelationType `json:"relationType"`
	Weight          float64      `json:"weight"`
	Source          string       `json:"source"`
	Explanation     string       `json:"explanation,omitempty"`
	CreatedAt       string       `json:"createdAt"`
}

type Collection struct {
	ID          string            `json:"id"`
	CuratorID   string            `json:"curatorId"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Visibility  string            `json:"visibility"`
	Items       []*CollectionItem `json:"items,omitempty"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

type CollectionItem struct {
	CollectionID string   `json:"collectionId"`
	ThoughtID    string   `json:"thoughtId"`
	Position     int      `json:"position"`
	AddedAt      string   `json:"addedAt"`
	Thought      *Thought `json:"thought,omitempty"`
}

type EngagementEvent struct {
	ID         string `json:"id"`
	UserID     string `json:"userId,omitempty"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	ActionType string `json:"actionType"`
	DwellMS    int    `json:"dwellMs,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

type CaptureItem struct {
	ID                string            `json:"id"`
	AuthorID          string            `json:"authorId"`
	Content           string            `json:"content"`
	SourceType        CaptureSourceType `json:"sourceType"`
	SourceTitle       string            `json:"sourceTitle,omitempty"`
	SourceURL         string            `json:"sourceUrl,omitempty"`
	SourceApp         string            `json:"sourceApp,omitempty"`
	Status            CaptureStatus     `json:"status"`
	PromotedThoughtID string            `json:"promotedThoughtId,omitempty"`
	PromotedThought   *Thought          `json:"promotedThought,omitempty"`
	CreatedAt         string            `json:"createdAt"`
	UpdatedAt         string            `json:"updatedAt"`
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

type DraftSuggestions struct {
	RelatedConcepts    []string   `json:"relatedConcepts"`
	SupportingThoughts []*Thought `json:"supportingThoughts"`
	CounterThoughts    []*Thought `json:"counterThoughts"`
	Reframes           []string   `json:"reframes"`
	Notes              []string   `json:"notes"`
}

type IdeaCurrent struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary,omitempty"`
	ClusterKey     string     `json:"clusterKey,omitempty"`
	FreshnessScore float64    `json:"freshnessScore"`
	QualityScore   float64    `json:"qualityScore"`
	Concepts       []*Concept `json:"concepts,omitempty"`
	Thoughts       []*Thought `json:"thoughts,omitempty"`
	CreatedAt      string     `json:"createdAt"`
	UpdatedAt      string     `json:"updatedAt"`
}

type HomePayload struct {
	Viewer                 *User          `json:"viewer,omitempty"`
	Currents               []*IdeaCurrent `json:"currents"`
	RecommendedThoughts    []*Thought     `json:"recommendedThoughts"`
	RecommendedCollections []*Collection  `json:"recommendedCollections"`
}

type TelescopeJump struct {
	Label      string   `json:"label"`
	Query      string   `json:"query"`
	Reason     string   `json:"reason,omitempty"`
	ThoughtIDs []string `json:"thoughtIds,omitempty"`
}

type TelescopeResult struct {
	Query           string             `json:"query"`
	Intent          string             `json:"intent"`
	SeedConcepts    []*Concept         `json:"seedConcepts,omitempty"`
	SeedThoughts    []*Thought         `json:"seedThoughts,omitempty"`
	Graph           *GraphNeighborhood `json:"graph,omitempty"`
	Clusters        []*SearchCluster   `json:"clusters,omitempty"`
	Narrative       string             `json:"narrative"`
	SuggestedJumps  []*TelescopeJump   `json:"suggestedJumps,omitempty"`
	RelatedCurrents []*IdeaCurrent     `json:"relatedCurrents,omitempty"`
}

type Job struct {
	ID           string            `json:"id"`
	Type         JobType           `json:"type"`
	EntityType   string            `json:"entityType"`
	EntityID     string            `json:"entityId"`
	Status       JobStatus         `json:"status"`
	AttemptCount int               `json:"attemptCount"`
	MaxAttempts  int               `json:"maxAttempts"`
	Payload      map[string]string `json:"payload"`
	LastError    string            `json:"lastError,omitempty"`
	LeaseOwner   string            `json:"leaseOwner,omitempty"`
	VisibleAt    string            `json:"visibleAt"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}
