package allude

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type PostgresRepository struct {
	db *bun.DB
}

type userRow struct {
	bun.BaseModel `bun:"table:users"`
	ID            string   `bun:"id,pk"`
	Username      string   `bun:"username"`
	DisplayName   string   `bun:"display_name"`
	Bio           string   `bun:"bio"`
	AvatarURL     string   `bun:"avatar_url"`
	Interests     []string `bun:"interests,array"`
	CreatedAt     string   `bun:"created_at"`
	UpdatedAt     string   `bun:"updated_at"`
}

type thoughtRow struct {
	bun.BaseModel    `bun:"table:thoughts"`
	ID               string   `bun:"id,pk"`
	AuthorID         string   `bun:"author_id"`
	Status           string   `bun:"status"`
	Visibility       string   `bun:"visibility"`
	CurrentVersionID string   `bun:"current_version_id"`
	ProcessingStatus string   `bun:"processing_status"`
	ProcessingNotes  []string `bun:"processing_notes,array"`
	CreatedAt        string   `bun:"created_at"`
	UpdatedAt        string   `bun:"updated_at"`
}

type thoughtVersionRow struct {
	bun.BaseModel    `bun:"table:thought_versions"`
	ID               string   `bun:"id,pk"`
	ThoughtID        string   `bun:"thought_id"`
	VersionNo        int      `bun:"version_no"`
	Content          string   `bun:"content"`
	Language         string   `bun:"language"`
	TokenCount       int      `bun:"token_count"`
	ProcessingStatus string   `bun:"processing_status"`
	ProcessingNotes  []string `bun:"processing_notes,array"`
	CreatedAt        string   `bun:"created_at"`
	Embedding        string   `bun:"embedding"`
}

type conceptRow struct {
	bun.BaseModel `bun:"table:concepts"`
	ID            string `bun:"id,pk"`
	CanonicalName string `bun:"canonical_name"`
	Slug          string `bun:"slug"`
	Description   string `bun:"description"`
	ConceptType   string `bun:"concept_type"`
	CreatedAt     string `bun:"created_at"`
	UpdatedAt     string `bun:"updated_at"`
}

type conceptAliasRow struct {
	bun.BaseModel   `bun:"table:concept_aliases"`
	ID              string `bun:"id,pk"`
	ConceptID       string `bun:"concept_id"`
	Alias           string `bun:"alias"`
	NormalizedAlias string `bun:"normalized_alias"`
}

type thoughtLinkRow struct {
	bun.BaseModel   `bun:"table:thought_links"`
	ID              string  `bun:"id,pk"`
	SourceThoughtID string  `bun:"source_thought_id"`
	TargetThoughtID string  `bun:"target_thought_id"`
	RelationType    string  `bun:"relation_type"`
	Weight          float64 `bun:"weight"`
	Source          string  `bun:"source"`
	Explanation     string  `bun:"explanation"`
	CreatedAt       string  `bun:"created_at"`
}

type conceptLinkRow struct {
	bun.BaseModel   `bun:"table:concept_links"`
	ID              string  `bun:"id,pk"`
	SourceConceptID string  `bun:"source_concept_id"`
	TargetConceptID string  `bun:"target_concept_id"`
	RelationType    string  `bun:"relation_type"`
	Weight          float64 `bun:"weight"`
	Source          string  `bun:"source"`
	Explanation     string  `bun:"explanation"`
	CreatedAt       string  `bun:"created_at"`
}

type collectionRow struct {
	bun.BaseModel `bun:"table:collections"`
	ID            string `bun:"id,pk"`
	CuratorID     string `bun:"curator_id"`
	Title         string `bun:"title"`
	Description   string `bun:"description"`
	Visibility    string `bun:"visibility"`
	CreatedAt     string `bun:"created_at"`
	UpdatedAt     string `bun:"updated_at"`
}

type collectionItemRow struct {
	bun.BaseModel `bun:"table:collection_items"`
	CollectionID  string `bun:"collection_id,pk"`
	ThoughtID     string `bun:"thought_id,pk"`
	Position      int    `bun:"position"`
	AddedAt       string `bun:"added_at"`
}

type engagementEventRow struct {
	bun.BaseModel `bun:"table:engagement_events"`
	ID            string `bun:"id,pk"`
	UserID        string `bun:"user_id"`
	EntityType    string `bun:"entity_type"`
	EntityID      string `bun:"entity_id"`
	ActionType    string `bun:"action_type"`
	DwellMS       int    `bun:"dwell_ms"`
	CreatedAt     string `bun:"created_at"`
}

type jobRow struct {
	bun.BaseModel `bun:"table:jobs"`
	ID            string `bun:"id,pk"`
	Type          string `bun:"type"`
	EntityType    string `bun:"entity_type"`
	EntityID      string `bun:"entity_id"`
	Status        string `bun:"status"`
	Payload       []byte `bun:"payload"`
	AttemptCount  int    `bun:"attempt_count"`
	MaxAttempts   int    `bun:"max_attempts"`
	LastError     string `bun:"last_error"`
	LeaseOwner    string `bun:"lease_owner"`
	VisibleAt     string `bun:"visible_at"`
	CreatedAt     string `bun:"created_at"`
	UpdatedAt     string `bun:"updated_at"`
}

func NewPostgresRepository(databaseURL string) (*PostgresRepository, error) {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db := bun.NewDB(sqlDB, pgdialect.New())
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresRepository{db: db}, nil
}

func (repository *PostgresRepository) GetViewer() *User {
	ctx := context.Background()
	row := new(userRow)
	err := repository.db.NewSelect().
		Model(row).
		Where("username = ?", "allude-dev").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return &User{ID: ViewerID, Username: "allude-dev", DisplayName: "Allude Dev"}
	}
	return userFromRow(row)
}

func (repository *PostgresRepository) CreateThought(authorID, content string) (*Thought, error) {
	ctx := context.Background()
	thoughtID := newUUID()
	versionID := newUUID()
	err := repository.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		thought := &thoughtRow{
			ID:               thoughtID,
			AuthorID:         authorID,
			Status:           "active",
			Visibility:       "public",
			CurrentVersionID: versionID,
			ProcessingStatus: string(ProcessingPending),
			ProcessingNotes:  []string{"Queued for enrichment"},
		}
		version := &thoughtVersionRow{
			ID:               versionID,
			ThoughtID:        thoughtID,
			VersionNo:        1,
			Content:          content,
			Language:         "en",
			TokenCount:       len(strings.Fields(content)),
			ProcessingStatus: string(ProcessingPending),
			ProcessingNotes:  []string{"Queued for enrichment"},
		}
		if _, err := tx.NewInsert().Model(thought).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewInsert().Model(version).Exec(ctx); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return repository.GetThought(thoughtID)
}

func (repository *PostgresRepository) UpdateThought(thoughtID, content string) (*Thought, error) {
	ctx := context.Background()
	versions, err := repository.ListThoughtVersions(thoughtID)
	if err != nil {
		return nil, err
	}
	versionID := newUUID()
	err = repository.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		version := &thoughtVersionRow{
			ID:               versionID,
			ThoughtID:        thoughtID,
			VersionNo:        len(versions) + 1,
			Content:          content,
			Language:         "en",
			TokenCount:       len(strings.Fields(content)),
			ProcessingStatus: string(ProcessingPending),
			ProcessingNotes:  []string{"Queued for enrichment"},
		}
		if _, err := tx.NewInsert().Model(version).Exec(ctx); err != nil {
			return err
		}
		_, err := tx.NewUpdate().
			Model((*thoughtRow)(nil)).
			Set("current_version_id = ?", versionID).
			Set("processing_status = ?", string(ProcessingPending)).
			Set("processing_notes = ?", []string{"Queued for enrichment"}).
			Set("updated_at = NOW()").
			Where("id = ?", thoughtID).
			Exec(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return repository.GetThought(thoughtID)
}

func (repository *PostgresRepository) GetThought(thoughtID string) (*Thought, error) {
	ctx := context.Background()
	row := new(thoughtRow)
	if err := repository.db.NewSelect().Model(row).Where("id = ?", thoughtID).Scan(ctx); err != nil {
		return nil, err
	}
	versions, err := repository.ListThoughtVersions(thoughtID)
	if err != nil {
		return nil, err
	}
	thought := thoughtFromRow(row, repository.GetViewer())
	thought.Versions = versions
	for _, version := range versions {
		if version.ID == row.CurrentVersionID {
			thought.CurrentVersion = version
			break
		}
	}
	thought.Concepts, _ = repository.currentThoughtConcepts(row.CurrentVersionID)
	thought.Links, _ = repository.currentThoughtLinks(thought.ID)
	thought.RelatedThoughts, _ = repository.GetRelatedThoughts(thought.ID, 8)
	thought.Collections, _ = repository.collectionsForThought(thought.ID)
	return thought, nil
}

func (repository *PostgresRepository) ListThoughtVersions(thoughtID string) ([]*ThoughtVersion, error) {
	ctx := context.Background()
	rows := []*thoughtVersionRow{}
	err := repository.db.NewSelect().
		Model(&rows).
		Column("id", "thought_id", "version_no", "content", "language", "token_count", "processing_status", "processing_notes", "created_at").
		ColumnExpr("COALESCE(embedding::text, '') AS embedding").
		Where("thought_id = ?", thoughtID).
		Order("version_no ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	versions := make([]*ThoughtVersion, 0, len(rows))
	for _, row := range rows {
		versions = append(versions, thoughtVersionFromRow(row))
	}
	return versions, nil
}

func (repository *PostgresRepository) SaveThoughtVersionEnrichment(versionID string, embedding []float64, conceptNames []string, status ProcessingStatus, notes []string) (*ThoughtVersion, error) {
	ctx := context.Background()
	err := repository.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model((*thoughtVersionRow)(nil)).
			Set("embedding = ?::vector", vectorLiteral(embedding)).
			Set("processing_status = ?", string(status)).
			Set("processing_notes = ?", notes).
			Where("id = ?", versionID).
			Exec(ctx); err != nil {
			return err
		}
		var thoughtID string
		if err := tx.NewSelect().Model((*thoughtVersionRow)(nil)).Column("thought_id").Where("id = ?", versionID).Scan(ctx, &thoughtID); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model((*struct {
			bun.BaseModel `bun:"table:thought_concepts"`
		})(nil)).Where("thought_version_id = ?", versionID).Exec(ctx); err != nil {
			return err
		}
		for _, conceptName := range uniqueStrings(conceptNames) {
			conceptID, err := repository.upsertConceptTx(ctx, tx, conceptName)
			if err != nil {
				return err
			}
			if _, err := tx.NewRaw(`
				INSERT INTO thought_concepts (thought_version_id, concept_id, weight, source)
				VALUES (?, ?, 1.0, 'ai')
				ON CONFLICT (thought_version_id, concept_id) DO UPDATE SET weight = EXCLUDED.weight, source = EXCLUDED.source`,
				versionID, conceptID,
			).Exec(ctx); err != nil {
				return err
			}
		}
		if _, err := tx.NewUpdate().
			Model((*thoughtRow)(nil)).
			Set("processing_status = ?", string(status)).
			Set("processing_notes = ?", notes).
			Set("updated_at = NOW()").
			Where("current_version_id = ?", versionID).
			Exec(ctx); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var version thoughtVersionRow
	if err := repository.db.NewSelect().
		Model(&version).
		Column("id", "thought_id", "version_no", "content", "language", "token_count", "processing_status", "processing_notes", "created_at").
		ColumnExpr("COALESCE(embedding::text, '') AS embedding").
		Where("id = ?", versionID).
		Scan(ctx); err != nil {
		return nil, err
	}
	return thoughtVersionFromRow(&version), nil
}

func (repository *PostgresRepository) SearchThoughts(query string, embedding []float64, limit int) (*SearchThoughtsResult, error) {
	ctx := context.Background()
	ids := []string{}
	if err := repository.db.NewSelect().
		Model((*thoughtRow)(nil)).
		Column("id").
		Order("updated_at DESC").
		Limit(200).
		Scan(ctx, &ids); err != nil {
		return nil, err
	}
	type scored struct {
		thought *Thought
		score   float64
	}
	var ranked []scored
	for _, thoughtID := range ids {
		thought, err := repository.GetThought(thoughtID)
		if err != nil || thought.CurrentVersion == nil {
			continue
		}
		score := (0.35 * lexicalScore(strings.ToLower(query), thought.CurrentVersion.Content, thought.Concepts)) +
			(0.45 * cosineSimilarity(embedding, thought.CurrentVersion.Embedding)) +
			(0.20 * qualityScore(thought))
		if score > 0 {
			ranked = append(ranked, scored{thought: thought, score: score})
		}
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	result := &SearchThoughtsResult{}
	clusterMap := map[string]*SearchCluster{}
	for _, entry := range ranked {
		result.Thoughts = append(result.Thoughts, entry.thought)
		for _, concept := range entry.thought.Concepts {
			cluster := clusterMap[concept.ID]
			if cluster == nil {
				cluster = &SearchCluster{Label: concept.CanonicalName, Concepts: []*Concept{cloneConceptBase(concept)}}
				clusterMap[concept.ID] = cluster
			}
			cluster.ThoughtIDs = appendUniqueString(cluster.ThoughtIDs, entry.thought.ID)
		}
	}
	for _, cluster := range clusterMap {
		result.Clusters = append(result.Clusters, cluster)
	}
	return result, nil
}

func (repository *PostgresRepository) GetRelatedThoughts(thoughtID string, limit int) ([]*Thought, error) {
	ctx := context.Background()
	var relatedIDs []string
	if err := repository.db.NewRaw(`
		SELECT CASE
			WHEN source_thought_id = ? THEN target_thought_id
			ELSE source_thought_id
		END AS related_id
		FROM thought_links
		WHERE source_thought_id = ? OR target_thought_id = ?
		ORDER BY weight DESC
		LIMIT ?`, thoughtID, thoughtID, thoughtID, limit).Scan(ctx, &relatedIDs); err != nil {
		return nil, err
	}
	var thoughts []*Thought
	for _, relatedID := range relatedIDs {
		thought, err := repository.GetThought(relatedID)
		if err == nil {
			thought.RelatedThoughts = nil
			thoughts = append(thoughts, thought)
		}
	}
	return thoughts, nil
}

func (repository *PostgresRepository) GetGraphNeighborhood(centerThoughtID string, hopCount, limit int) (*GraphNeighborhood, error) {
	thought, err := repository.GetThought(centerThoughtID)
	if err != nil {
		return nil, err
	}
	nodes := []*GraphNode{{Thought: thought, ThoughtID: thought.ID, Distance: 0}}
	edges := []*GraphEdge{}
	queue := []string{centerThoughtID}
	distances := map[string]int{centerThoughtID: 0}
	seenEdges := map[string]struct{}{}
	for len(queue) > 0 && len(nodes) < limit {
		current := queue[0]
		queue = queue[1:]
		if distances[current] >= hopCount {
			continue
		}
		links, err := repository.currentThoughtLinks(current)
		if err != nil {
			return nil, err
		}
		for _, link := range links {
			if _, exists := seenEdges[link.ID]; !exists {
				edges = append(edges, &GraphEdge{Link: link})
				seenEdges[link.ID] = struct{}{}
			}
			nextID := link.SourceThoughtID
			if nextID == current {
				nextID = link.TargetThoughtID
			}
			if _, exists := distances[nextID]; exists {
				continue
			}
			nextThought, err := repository.GetThought(nextID)
			if err != nil {
				continue
			}
			distance := distances[current] + 1
			distances[nextID] = distance
			nodes = append(nodes, &GraphNode{Thought: nextThought, ThoughtID: nextThought.ID, X: float64(distance * 160), Y: float64(len(nodes) * 45), Distance: distance})
			queue = append(queue, nextID)
			if len(nodes) >= limit {
				break
			}
		}
	}
	return &GraphNeighborhood{Center: nodes[0], Nodes: nodes, Edges: edges}, nil
}

func (repository *PostgresRepository) GetConceptByID(id string) (*Concept, error) {
	return repository.loadConcept("id = ?", id)
}

func (repository *PostgresRepository) GetConceptBySlug(slug string) (*Concept, error) {
	return repository.loadConcept("slug = ?", slug)
}

func (repository *PostgresRepository) GetConceptByName(name string) (*Concept, error) {
	concept, err := repository.loadConcept("lower(canonical_name) = lower(?)", name)
	if err == nil && concept != nil {
		return concept, nil
	}
	ctx := context.Background()
	var conceptID string
	err = repository.db.NewSelect().
		Model((*conceptAliasRow)(nil)).
		Column("concept_id").
		Where("normalized_alias = ?", normalizeConceptName(name)).
		Limit(1).
		Scan(ctx, &conceptID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return repository.GetConceptByID(conceptID)
}

func (repository *PostgresRepository) GetConceptThoughts(conceptID string, limit int) ([]*Thought, error) {
	ctx := context.Background()
	var thoughtIDs []string
	if err := repository.db.NewRaw(`
		SELECT DISTINCT t.id
		FROM thought_concepts tc
		JOIN thought_versions tv ON tv.id = tc.thought_version_id
		JOIN thoughts t ON t.current_version_id = tv.id
		WHERE tc.concept_id = ?
		ORDER BY t.updated_at DESC
		LIMIT ?`, conceptID, limit).Scan(ctx, &thoughtIDs); err != nil {
		return nil, err
	}
	var thoughts []*Thought
	for _, thoughtID := range thoughtIDs {
		thought, err := repository.GetThought(thoughtID)
		if err == nil {
			thoughts = append(thoughts, thought)
		}
	}
	return thoughts, nil
}

func (repository *PostgresRepository) GetRelatedConcepts(conceptID string, limit int) ([]*Concept, error) {
	ctx := context.Background()
	var conceptIDs []string
	if err := repository.db.NewRaw(`
		SELECT CASE
			WHEN source_concept_id = ? THEN target_concept_id
			ELSE source_concept_id
		END
		FROM concept_links
		WHERE source_concept_id = ? OR target_concept_id = ?
		ORDER BY weight DESC
		LIMIT ?`, conceptID, conceptID, conceptID, limit).Scan(ctx, &conceptIDs); err != nil {
		return nil, err
	}
	var concepts []*Concept
	for _, relatedID := range conceptIDs {
		concept, err := repository.GetConceptByID(relatedID)
		if err == nil && concept != nil {
			concepts = append(concepts, concept)
		}
	}
	return concepts, nil
}

func (repository *PostgresRepository) ReplaceThoughtLinks(thoughtID string, links []*ThoughtLink) error {
	ctx := context.Background()
	return repository.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().
			Model((*thoughtLinkRow)(nil)).
			Where("source = ?", "analysis").
			Where("(source_thought_id = ? OR target_thought_id = ?)", thoughtID, thoughtID).
			Exec(ctx); err != nil {
			return err
		}
		for _, link := range links {
			row := thoughtLinkRow{
				ID:              newUUID(),
				SourceThoughtID: link.SourceThoughtID,
				TargetThoughtID: link.TargetThoughtID,
				RelationType:    string(link.RelationType),
				Weight:          link.Weight,
				Source:          link.Source,
				Explanation:     link.Explanation,
			}
			if _, err := tx.NewRaw(`
				INSERT INTO thought_links (id, source_thought_id, target_thought_id, relation_type, weight, source, explanation)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (source_thought_id, target_thought_id, relation_type)
				DO UPDATE SET weight = EXCLUDED.weight, source = EXCLUDED.source, explanation = EXCLUDED.explanation`,
				row.ID, row.SourceThoughtID, row.TargetThoughtID, row.RelationType, row.Weight, row.Source, row.Explanation,
			).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (repository *PostgresRepository) CreateCollection(curatorID, title, description string) (*Collection, error) {
	ctx := context.Background()
	row := &collectionRow{ID: newUUID(), CuratorID: curatorID, Title: title, Description: description, Visibility: "public"}
	if _, err := repository.db.NewInsert().Model(row).Exec(ctx); err != nil {
		return nil, err
	}
	return repository.GetCollection(row.ID)
}

func (repository *PostgresRepository) AddThoughtToCollection(collectionID, thoughtID string) (*Collection, error) {
	ctx := context.Background()
	if _, err := repository.db.NewRaw(`
		INSERT INTO collection_items (collection_id, thought_id, position)
		VALUES (?, ?, COALESCE((SELECT MAX(position) + 1 FROM collection_items WHERE collection_id = ?), 0))
		ON CONFLICT (collection_id, thought_id) DO NOTHING`, collectionID, thoughtID, collectionID).Exec(ctx); err != nil {
		return nil, err
	}
	return repository.GetCollection(collectionID)
}

func (repository *PostgresRepository) GetCollection(id string) (*Collection, error) {
	ctx := context.Background()
	row := new(collectionRow)
	if err := repository.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, err
	}
	items := []*collectionItemRow{}
	if err := repository.db.NewSelect().Model(&items).Where("collection_id = ?", id).Order("position ASC").Scan(ctx); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	collection := collectionFromRow(row)
	for _, item := range items {
		entry := &CollectionItem{
			CollectionID: item.CollectionID,
			ThoughtID:    item.ThoughtID,
			Position:     item.Position,
			AddedAt:      item.AddedAt,
		}
		entry.Thought, _ = repository.GetThought(entry.ThoughtID)
		collection.Items = append(collection.Items, entry)
	}
	return collection, nil
}

func (repository *PostgresRepository) RecordEngagement(event *EngagementEvent) (*EngagementEvent, error) {
	ctx := context.Background()
	if event.ID == "" {
		event.ID = newUUID()
	}
	row := &engagementEventRow{
		ID:         event.ID,
		UserID:     event.UserID,
		EntityType: event.EntityType,
		EntityID:   event.EntityID,
		ActionType: event.ActionType,
		DwellMS:    event.DwellMS,
	}
	if _, err := repository.db.NewInsert().Model(row).Exec(ctx); err != nil {
		return nil, err
	}
	return &EngagementEvent{
		ID:         row.ID,
		UserID:     row.UserID,
		EntityType: row.EntityType,
		EntityID:   row.EntityID,
		ActionType: row.ActionType,
		DwellMS:    row.DwellMS,
		CreatedAt:  row.CreatedAt,
	}, nil
}

func (repository *PostgresRepository) EnqueueJob(job *Job) (*Job, error) {
	ctx := context.Background()
	if job.ID == "" {
		job.ID = newUUID()
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}
	payload, _ := json.Marshal(job.Payload)
	row := &jobRow{
		ID:           job.ID,
		Type:         string(job.Type),
		EntityType:   job.EntityType,
		EntityID:     job.EntityID,
		Status:       string(JobPending),
		Payload:      payload,
		AttemptCount: 0,
		MaxAttempts:  job.MaxAttempts,
	}
	if _, err := repository.db.NewInsert().Model(row).Exec(ctx); err != nil {
		return nil, err
	}
	job.Status = JobPending
	return job, nil
}

func (repository *PostgresRepository) LeasePendingJob(workerID string) (*Job, error) {
	ctx := context.Background()
	row := new(jobRow)
	err := repository.db.NewSelect().
		Model(row).
		Where("status = ?", string(JobPending)).
		Order("created_at ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	row.Status = string(JobLeased)
	row.LeaseOwner = workerID
	row.AttemptCount++
	if _, err := repository.db.NewUpdate().
		Model(row).
		Column("status", "lease_owner", "attempt_count", "updated_at").
		WherePK().
		Exec(ctx); err != nil {
		return nil, err
	}
	return jobFromRow(row), nil
}

func (repository *PostgresRepository) CompleteJob(jobID string) error {
	_, err := repository.db.NewUpdate().
		Model((*jobRow)(nil)).
		Set("status = ?", string(JobCompleted)).
		Set("updated_at = NOW()").
		Where("id = ?", jobID).
		Exec(context.Background())
	return err
}

func (repository *PostgresRepository) FailJob(jobID, message string) error {
	_, err := repository.db.NewRaw(`
		UPDATE jobs
		SET last_error = ?,
		    status = CASE WHEN attempt_count >= max_attempts THEN 'DEAD' ELSE 'PENDING' END,
		    lease_owner = NULL,
		    updated_at = NOW()
		WHERE id = ?`, message, jobID).Exec(context.Background())
	return err
}

func (repository *PostgresRepository) ListJobs() []*Job {
	ctx := context.Background()
	rows := []*jobRow{}
	if err := repository.db.NewSelect().Model(&rows).Order("created_at ASC").Scan(ctx); err != nil {
		return nil
	}
	jobs := make([]*Job, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, jobFromRow(row))
	}
	return jobs
}

func (repository *PostgresRepository) loadConcept(where string, arg string) (*Concept, error) {
	ctx := context.Background()
	row := new(conceptRow)
	if err := repository.db.NewSelect().Model(row).Where(where, arg).Limit(1).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	concept := conceptFromRow(row)
	aliases := []*conceptAliasRow{}
	if err := repository.db.NewSelect().Model(&aliases).Where("concept_id = ?", concept.ID).Scan(ctx); err == nil {
		for _, alias := range aliases {
			concept.Aliases = append(concept.Aliases, &ConceptAlias{
				ID: alias.ID, ConceptID: alias.ConceptID, Alias: alias.Alias, NormalizedAlias: alias.NormalizedAlias,
			})
		}
	}
	concept.TopThoughts, _ = repository.GetConceptThoughts(concept.ID, 8)
	concept.RelatedConcepts, _ = repository.GetRelatedConcepts(concept.ID, 8)
	return concept, nil
}

func (repository *PostgresRepository) currentThoughtConcepts(versionID string) ([]*Concept, error) {
	ctx := context.Background()
	rows := []*conceptRow{}
	err := repository.db.NewRaw(`
		SELECT c.id, c.canonical_name, c.slug, COALESCE(c.description, '') AS description, COALESCE(c.concept_type, '') AS concept_type, c.created_at, c.updated_at
		FROM thought_concepts tc
		JOIN concepts c ON c.id = tc.concept_id
		WHERE tc.thought_version_id = ?
		ORDER BY c.canonical_name ASC`, versionID).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	concepts := make([]*Concept, 0, len(rows))
	for _, row := range rows {
		concepts = append(concepts, conceptFromRow(row))
	}
	return concepts, nil
}

func (repository *PostgresRepository) currentThoughtLinks(thoughtID string) ([]*ThoughtLink, error) {
	ctx := context.Background()
	rows := []*thoughtLinkRow{}
	err := repository.db.NewSelect().
		Model(&rows).
		Where("source_thought_id = ? OR target_thought_id = ?", thoughtID, thoughtID).
		Order("weight DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	links := make([]*ThoughtLink, 0, len(rows))
	for _, row := range rows {
		links = append(links, thoughtLinkFromRow(row))
	}
	return links, nil
}

func (repository *PostgresRepository) collectionsForThought(thoughtID string) ([]*Collection, error) {
	ctx := context.Background()
	rows := []*collectionRow{}
	err := repository.db.NewRaw(`
		SELECT c.id, c.curator_id, c.title, COALESCE(c.description, '') AS description, c.visibility, c.created_at, c.updated_at
		FROM collection_items ci
		JOIN collections c ON c.id = ci.collection_id
		WHERE ci.thought_id = ?`, thoughtID).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	collections := make([]*Collection, 0, len(rows))
	for _, row := range rows {
		collections = append(collections, collectionFromRow(row))
	}
	return collections, nil
}

func (repository *PostgresRepository) upsertConceptTx(ctx context.Context, tx bun.Tx, conceptName string) (string, error) {
	normalized := normalizeConceptName(conceptName)
	var conceptID string
	err := tx.NewSelect().Model((*conceptAliasRow)(nil)).Column("concept_id").Where("normalized_alias = ?", normalized).Limit(1).Scan(ctx, &conceptID)
	if err == nil {
		return conceptID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	concept := &conceptRow{
		ID:            newUUID(),
		CanonicalName: conceptName,
		Slug:          slugify(conceptName),
	}
	if _, err := tx.NewInsert().Model(concept).Ignore().Exec(ctx); err != nil {
		return "", err
	}
	alias := &conceptAliasRow{
		ID:              newUUID(),
		ConceptID:       concept.ID,
		Alias:           conceptName,
		NormalizedAlias: normalized,
	}
	if _, err := tx.NewInsert().Model(alias).Ignore().Exec(ctx); err != nil {
		return "", err
	}
	return concept.ID, nil
}

func thoughtVersionFromRow(row *thoughtVersionRow) *ThoughtVersion {
	return &ThoughtVersion{
		ID:               row.ID,
		ThoughtID:        row.ThoughtID,
		VersionNo:        row.VersionNo,
		Content:          row.Content,
		Embedding:        parseVector(row.Embedding),
		Language:         row.Language,
		TokenCount:       row.TokenCount,
		ProcessingStatus: ProcessingStatus(row.ProcessingStatus),
		ProcessingNotes:  append([]string(nil), row.ProcessingNotes...),
		CreatedAt:        row.CreatedAt,
	}
}

func thoughtFromRow(row *thoughtRow, author *User) *Thought {
	return &Thought{
		ID:               row.ID,
		Author:           author,
		AuthorID:         row.AuthorID,
		Status:           row.Status,
		Visibility:       row.Visibility,
		CurrentVersionID: row.CurrentVersionID,
		ProcessingStatus: ProcessingStatus(row.ProcessingStatus),
		ProcessingNotes:  append([]string(nil), row.ProcessingNotes...),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func conceptFromRow(row *conceptRow) *Concept {
	return &Concept{
		ID:            row.ID,
		CanonicalName: row.CanonicalName,
		Slug:          row.Slug,
		Description:   row.Description,
		ConceptType:   row.ConceptType,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func collectionFromRow(row *collectionRow) *Collection {
	return &Collection{
		ID:          row.ID,
		CuratorID:   row.CuratorID,
		Title:       row.Title,
		Description: row.Description,
		Visibility:  row.Visibility,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func thoughtLinkFromRow(row *thoughtLinkRow) *ThoughtLink {
	return &ThoughtLink{
		ID:              row.ID,
		SourceThoughtID: row.SourceThoughtID,
		TargetThoughtID: row.TargetThoughtID,
		RelationType:    RelationType(row.RelationType),
		Weight:          row.Weight,
		Source:          row.Source,
		Explanation:     row.Explanation,
		CreatedAt:       row.CreatedAt,
	}
}

func userFromRow(row *userRow) *User {
	return &User{
		ID:          row.ID,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		Bio:         row.Bio,
		AvatarURL:   row.AvatarURL,
		Interests:   append([]string(nil), row.Interests...),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func jobFromRow(row *jobRow) *Job {
	job := &Job{
		ID:           row.ID,
		Type:         JobType(row.Type),
		EntityType:   row.EntityType,
		EntityID:     row.EntityID,
		Status:       JobStatus(row.Status),
		AttemptCount: row.AttemptCount,
		MaxAttempts:  row.MaxAttempts,
		LastError:    row.LastError,
		LeaseOwner:   row.LeaseOwner,
		VisibleAt:    row.VisibleAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		Payload:      map[string]string{},
	}
	_ = json.Unmarshal(row.Payload, &job.Payload)
	return job
}

func qualityScore(thought *Thought) float64 {
	score := 0.2
	score += float64(len(thought.Collections)) * 0.2
	score += float64(len(thought.Links)) * 0.1
	if score > 1 {
		score = 1
	}
	return score
}

func parseVector(input string) []float64 {
	input = strings.TrimSpace(strings.Trim(input, "[]"))
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	vector := make([]float64, 0, len(parts))
	for _, part := range parts {
		var value float64
		fmt.Sscanf(strings.TrimSpace(part), "%f", &value)
		vector = append(vector, value)
	}
	return vector
}

func vectorLiteral(values []float64) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%f", value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
