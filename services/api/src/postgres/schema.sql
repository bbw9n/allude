CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE users (
  id UUID PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT,
  bio TEXT,
  avatar_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE thoughts (
  id UUID PRIMARY KEY,
  author_id UUID NOT NULL REFERENCES users(id),
  status TEXT NOT NULL,
  current_version_id UUID,
  visibility TEXT NOT NULL DEFAULT 'public',
  processing_status TEXT NOT NULL DEFAULT 'PENDING',
  processing_notes TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE thought_versions (
  id UUID PRIMARY KEY,
  thought_id UUID NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
  version_no INTEGER NOT NULL,
  content TEXT NOT NULL,
  embedding vector(16),
  language TEXT,
  token_count INTEGER,
  processing_status TEXT NOT NULL DEFAULT 'PENDING',
  processing_notes TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(thought_id, version_no)
);

ALTER TABLE thoughts
  ADD CONSTRAINT thoughts_current_version_fk
  FOREIGN KEY (current_version_id) REFERENCES thought_versions(id);

CREATE TABLE concepts (
  id UUID PRIMARY KEY,
  canonical_name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  description TEXT,
  embedding vector(16),
  concept_type TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  concept_id UUID NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
  affinity_score DOUBLE PRECISION NOT NULL,
  source TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, concept_id)
);

CREATE TABLE concept_aliases (
  id UUID PRIMARY KEY,
  concept_id UUID NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
  alias TEXT NOT NULL,
  normalized_alias TEXT NOT NULL,
  UNIQUE (concept_id, normalized_alias),
  UNIQUE (normalized_alias)
);

CREATE TABLE thought_concepts (
  thought_version_id UUID NOT NULL REFERENCES thought_versions(id) ON DELETE CASCADE,
  concept_id UUID NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
  weight DOUBLE PRECISION NOT NULL,
  source TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (thought_version_id, concept_id)
);

CREATE TABLE thought_links (
  id UUID PRIMARY KEY,
  source_thought_id UUID NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
  target_thought_id UUID NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
  relation_type TEXT NOT NULL,
  weight DOUBLE PRECISION NOT NULL,
  source TEXT NOT NULL,
  explanation TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(source_thought_id, target_thought_id, relation_type)
);

CREATE TABLE concept_links (
  id UUID PRIMARY KEY,
  source_concept_id UUID NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
  target_concept_id UUID NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
  relation_type TEXT NOT NULL,
  weight DOUBLE PRECISION NOT NULL,
  source TEXT NOT NULL,
  explanation TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(source_concept_id, target_concept_id, relation_type)
);

CREATE TABLE collections (
  id UUID PRIMARY KEY,
  curator_id UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  description TEXT,
  visibility TEXT NOT NULL DEFAULT 'public',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE collection_items (
  collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
  thought_id UUID NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
  position INTEGER,
  added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (collection_id, thought_id)
);

CREATE TABLE engagement_events (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  entity_type TEXT NOT NULL,
  entity_id UUID NOT NULL,
  action_type TEXT NOT NULL,
  dwell_ms INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE embedding_jobs (
  id UUID PRIMARY KEY,
  entity_type TEXT NOT NULL,
  entity_id UUID NOT NULL,
  model_name TEXT NOT NULL,
  model_version TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE jobs (
  id UUID PRIMARY KEY,
  type TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id UUID NOT NULL,
  status TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 3,
  last_error TEXT,
  lease_owner TEXT,
  visible_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO users (id, username, display_name, bio)
VALUES ('00000000-0000-0000-0000-000000000001', 'allude-dev', 'Allude Dev', 'Seeded development identity')
ON CONFLICT (username) DO NOTHING;

CREATE INDEX thought_versions_embedding_idx ON thought_versions USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX concepts_embedding_idx ON concepts USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX user_interests_user_idx ON user_interests (user_id, affinity_score DESC);
CREATE INDEX thought_links_source_idx ON thought_links (source_thought_id);
CREATE INDEX thought_links_target_idx ON thought_links (target_thought_id);
CREATE INDEX concept_links_source_idx ON concept_links (source_concept_id);
CREATE INDEX concept_links_target_idx ON concept_links (target_concept_id);
CREATE INDEX thought_versions_thought_id_idx ON thought_versions (thought_id);
CREATE INDEX thought_concepts_concept_idx ON thought_concepts (concept_id);
CREATE INDEX collection_items_thought_idx ON collection_items (thought_id);
CREATE INDEX engagement_events_entity_idx ON engagement_events (entity_type, entity_id);
CREATE INDEX jobs_status_visible_idx ON jobs (status, visible_at);
