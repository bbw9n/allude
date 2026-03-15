# Allude

Allude is an AI-native idea network for composing thoughts, extracting concepts, linking ideas, and exploring a living graph.

## Workspace

- `services/api`: GraphQL API, analysis pipeline, and repository adapters
- `packages/schema`: Shared GraphQL SDL and example operations
- `apps/macos`: SwiftUI macOS client

## Development

### API

```bash
cd services/api
go run ./cmd/server
```

The current backend is implemented in Go and serves a GraphQL-compatible MVP endpoint at `http://127.0.0.1:4000/`. The in-memory repository is active by default, and the Postgres + pgvector target schema lives at [`schema.sql`](/Users/bytedance/work/genai/Allude/services/api/src/postgres/schema.sql).

### macOS app

```bash
cd apps/macos
swift run
```
