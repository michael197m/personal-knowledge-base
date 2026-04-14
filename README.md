# Personal Knowledge Base

Personal note-taking and semantic search application scaffolded for a React +
TypeScript frontend, Go backend, and PostgreSQL with `pgvector`.

## Structure

- `frontend/`: React + TypeScript UI
- `backend/`: Go REST API
- `db/`: database initialization SQL
- `deploy/`: local Docker orchestration and Kubernetes manifests
- `TASK_LOG.md`: persistent implementation log for future Codex sessions

## Local Development

1. Copy `.env.example` values into your environment or an env file.
2. Start the stack with Docker Compose:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

3. Visit:
   - Frontend: `http://localhost:3000`
   - Backend health: `http://localhost:8080/api/v1/health`
4. The project Postgres container is published on host port `55432` to avoid
   collisions with any local Postgres already using `5432`.

## Authentication

- Notes and search endpoints are now authenticated and scoped per user.
- Auth now uses an `HttpOnly` cookie (`pkb_auth_token`) instead of storing JWTs
  in frontend storage.
- Use the frontend auth form (login/register), or call:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password123"}'
```

- Then pass the returned token:

```bash
curl -i http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password123"}'
```

- Use the returned `Set-Cookie` value for protected endpoints when using CLI:

```bash
curl http://localhost:8080/api/v1/notes \
  -H 'Cookie: pkb_auth_token=<token>'
```

- For production, run the frontend and backend behind HTTPS and set:
  - `AUTH_COOKIE_SECURE=true`
  - `FRONTEND_ORIGIN=https://<your-frontend-origin>`

## Backend Testing

- Run the default backend test suite:

```bash
cd backend
GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go test ./...
```

- Run the Postgres-backed store integration tests against the local Docker DB:

```bash
cd backend
PKB_TEST_DATABASE_URL='postgres://kb_user:kb_password@127.0.0.1:55432/knowledge_base?sslmode=disable' \
GOCACHE=/tmp/go-build \
GOMODCACHE=/tmp/go-mod-cache \
go test ./internal/store -run Integration -v
```

## Ollama Embeddings

- The backend can generate note embeddings on create/update using Ollama.
- Pull the default embedding model:

```bash
ollama pull nomic-embed-text
```

- For local backend runs, set:

```bash
export OLLAMA_BASE_URL=http://localhost:11434
export OLLAMA_EMBED_MODEL=nomic-embed-text
```

- The Docker backend service is configured to reach a host Ollama instance via
  `http://host.docker.internal:11434` by default.
- To backfill embeddings for existing notes with `NULL` vectors:

```bash
cd backend
OLLAMA_BASE_URL=http://localhost:11434 \
OLLAMA_EMBED_MODEL=nomic-embed-text \
GOCACHE=/tmp/go-build \
GOMODCACHE=/tmp/go-mod-cache \
go run ./cmd/backfill_embeddings
```

## Notes

- The current frontend supports note create, update, and delete flows against
  the backend note API with token-based authentication.
- The current frontend also includes a search bar backed by
  `/api/v1/search?q=...`.
- The backend currently provides health plus note list/create/update/delete
  endpoints, plus search and auth endpoints (`/auth/register`, `/auth/login`).
- `/api/v1/health` now reports both database status and embedding-provider
  status (`ok`, `unavailable`, `unconfigured`) so semantic fallback behavior is
  easier to diagnose.
- When Ollama is configured, note create/update operations also attempt to
  generate and persist embeddings using `nomic-embed-text`.
- When embeddings exist, `/api/v1/search` uses pgvector cosine similarity for
  semantic search. If embeddings are unavailable, it falls back to plain
  title/content/tag matching.
- Search responses now include lightweight result metadata so the UI can show
  whether a result came from semantic ranking or text fallback, plus a semantic
  similarity score when available.
- Backend search logs now include diagnostics for mode selection (`semantic`,
  `text`, or `list`) and text-fallback reasons (for example
  `semantic_no_match`, `embedding_failed`, or `embedder_unconfigured`).
- PostgreSQL is initialized with the `vector` extension and starter schema.
- Kubernetes manifests are included as a base for Minikube deployment later.
