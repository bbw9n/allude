CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  bio TEXT,
  interests TEXT[] NOT NULL DEFAULT '{}'
);

CREATE TABLE thoughts (
  id TEXT PRIMARY KEY,
  author_id TEXT NOT NULL REFERENCES users(id),
  current_version_id TEXT,
  embedding vector(16),
  processing_status TEXT NOT NULL,
  processing_notes TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE thought_versions (
  id TEXT PRIMARY KEY,
  thought_id TEXT NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(thought_id, version)
);

ALTER TABLE thoughts
  ADD CONSTRAINT thoughts_current_version_fk
  FOREIGN KEY (current_version_id) REFERENCES thought_versions(id);

CREATE TABLE concepts (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  normalized_name TEXT NOT NULL UNIQUE,
  embedding vector(16),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE thought_concepts (
  thought_id TEXT NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
  concept_id TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
  PRIMARY KEY (thought_id, concept_id)
);

CREATE TABLE thought_links (
  id TEXT PRIMARY KEY,
  source_thought_id TEXT NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
  target_thought_id TEXT NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
  relation_type TEXT NOT NULL,
  score DOUBLE PRECISION NOT NULL,
  origin TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(source_thought_id, target_thought_id, relation_type)
);

CREATE TABLE collections (
  id TEXT PRIMARY KEY,
  curator_id TEXT NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);

CREATE INDEX thoughts_embedding_idx ON thoughts USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX concepts_embedding_idx ON concepts USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX thought_links_source_idx ON thought_links (source_thought_id);
CREATE INDEX thought_links_target_idx ON thought_links (target_thought_id);
