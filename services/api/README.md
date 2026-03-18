# Allude API

Go backend for the Allude MVP. The service now exposes a real GraphQL HTTP endpoint at `/` and implements the semantic-core operations used by the macOS client:

- `createThought`
- `updateThought`
- `thought`
- `searchThoughts`
- `graph`
- `concept`
- `relatedThoughts`
- `listThoughtVersions`

## Run

```bash
cd services/api
go run ./cmd/server
```

The current implementation uses an in-memory runtime store plus a Postgres `pgvector` schema at [`src/postgres/schema.sql`](/Users/bytedance/work/genai/Allude/services/api/src/postgres/schema.sql) for the production storage target. Background enrichment is driven through durable-style job records and a worker loop inside the Go monolith.
