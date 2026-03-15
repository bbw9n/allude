# Allude API

Go backend for the Allude MVP. The service exposes a GraphQL-compatible HTTP endpoint at `/` and currently implements the MVP operations used by the macOS client:

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

The current implementation uses an in-memory repository plus a real Postgres `pgvector` schema file at [`src/postgres/schema.sql`](/Users/bytedance/work/genai/Allude/services/api/src/postgres/schema.sql) for the production storage target.
