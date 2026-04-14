# Task Log

## 2026-04-13

- Reviewed `../sideProjects.md` and selected the Option 1 requirements as the
  scaffold target.
- Inspected the `personal-knowledge-base` directory and confirmed it was empty
  before scaffolding.
- Chose the initial stack layout:
  - Frontend: React + TypeScript with Vite
  - Backend: Go REST API with `net/http`
  - Database: PostgreSQL with `pgvector`
  - Infrastructure: Docker Compose and starter Kubernetes manifests
- Created the base repository structure for frontend, backend, database, and
  deployment assets.
- Added root documentation and environment templates.
- Added local Docker Compose orchestration for frontend, backend, and postgres.
- Added starter Kubernetes manifests for namespace, config, postgres, backend,
  and frontend services.
- Added frontend base files:
  - Vite/TypeScript package manifest and TS configs
  - Main React entrypoint and starter dashboard UI
  - API client layer and shared frontend types
  - Frontend Dockerfile and nginx config
- Added backend base files:
  - Go module definition
  - API entrypoint with graceful shutdown
  - Environment-based config loader
  - Router, CORS middleware, health handler, and starter note handlers
  - Shared note model and PostgreSQL pool setup
  - Backend Dockerfile
- Added database initialization SQL:
  - `pgvector` and `pgcrypto` extensions
  - Starter `users`, `notes`, `tags`, and `note_tags` tables
  - Baseline indexes including a vector similarity index
- Verified `deploy/docker-compose.yml` with `docker compose config`.
- Could not run `gofmt` or local Go compilation in this shell because Go tooling
  is not installed in the environment used for this Codex session.
- Verified the backend Go toolchain is now available in the current environment
  (`go version go1.26.2 linux/amd64`).
- Ran `go mod tidy` in `backend/`, which generated `backend/go.sum` and fetched
  the declared module dependencies.
- Verified the backend compiles with
  `env GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go build ./...`.
- Verified the backend test packages load successfully with
  `env GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go test ./...`.
- Replaced the placeholder in-memory note handler with a Postgres-backed note
  store that:
  - ensures a demo user exists until auth is implemented
  - lists notes with tag aggregation
  - creates notes and tag links transactionally
- Updated the frontend to create notes through the backend API and render the
  persisted note list instead of static seeded responses.
- Ran `gofmt` on the updated backend files.
- Frontend build verification is still pending local dependency installation;
  `npm run build` currently fails with `tsc: not found` because `node_modules`
  has not been installed in this workspace.
- Installed frontend dependencies, verified `npm run build`, and later re-ran
  the build after dependency security updates.
- Updated the frontend React plugin to the latest Vite 8-compatible
  `@vitejs/plugin-react` release (`6.0.1`).
- Evaluated `@vitejs/plugin-react-oxc`, but did not adopt it because the latest
  published version currently supports Vite 6/7 rather than the Vite 8 version
  now present in this project.
- Added a lightweight frontend ESLint setup using flat config plus
  TypeScript/React Hooks rules, with a new `npm run lint` script.
- Split the main frontend view into smaller `NoteForm` and `NoteList`
  components so `App.tsx` only coordinates data loading and submission.
- Verified both `npm run lint` and `npm run build` pass after the refactor.
- Added backend HTTP-layer tests for:
  - note list and create handler success/error behavior
  - invalid note payload validation
  - health endpoint database status reporting
  - CORS middleware preflight and pass-through behavior
- Introduced small handler interfaces so the HTTP tests can use mocks instead of
  a live database pool.
- Verified the backend still passes both `go test ./...` and `go build ./...`.
- Added unit tests for `backend/internal/store/notes.go` covering:
  - tag normalization behavior
  - demo-user ID retrieval
  - note listing row mapping
  - note creation trimming/tag normalization behavior
  - transaction startup and query error handling
- Refactored `NoteStore` to depend on a small internal database interface so the
  store logic can be tested with mocks without requiring a running Postgres
  instance.
- Added opt-in Postgres-backed integration tests for `NoteStore` that:
  - apply the database init SQL
  - reset the schema tables between test cases
  - verify note create/list behavior against a real database
  - verify `ListNotes` ordering
- Gated the integration tests behind `PKB_TEST_DATABASE_URL` so the default test
  suite remains fast and does not fail in environments without a reachable
  Postgres instance.
- Diagnosed a host Postgres collision on `127.0.0.1:5432`; the Docker database
  was healthy, but host-side `psql` was reaching a separate local Postgres
  process instead of the project container.
- Updated Docker Compose to publish the project Postgres container on host port
  `55432` to avoid the collision, and documented the host port in
  `.env.example`.
- Verified host access to the Docker Postgres database through `127.0.0.1:55432`
  after the port change.
- Ran the real `NoteStore` integration tests successfully against the Docker
  Postgres instance using `PKB_TEST_DATABASE_URL`.
- Updated `README.md` with backend test commands, including the Postgres-backed
  integration-test invocation.
- Added note update/delete behavior to the backend store and HTTP handlers,
  including not-found handling and tag replacement on update.
- Added matching HTTP handler tests, store unit tests, and Postgres-backed
  integration tests for note update/delete behavior.
- Verified `go test ./...` and the live `NoteStore` integration tests pass with
  the expanded note behavior.
- Wired the frontend to the backend note update/delete endpoints.
- Reused the note form for both create and edit flows, and added note-card edit
  and delete actions in the UI.
- Verified the frontend still passes both `npm run lint` and `npm run build`
  after the CRUD wiring changes.
- Updated the frontend Dockerfile to copy `package-lock.json` and use `npm ci`
  instead of `npm install` so Docker builds are more deterministic and faster on
  rebuilds.
- Added `frontend/.dockerignore` to keep Docker build context small and avoid
  sending local artifacts such as `node_modules`, `dist`, and tsbuildinfo files
  into the frontend image build.
- Added a first plain-text search slice:
  - `NoteStore.SearchNotes` with title/content/tag matching and basic relevance
    ordering
  - `GET /api/v1/search?q=...` backend handler and route
  - frontend search bar wired to the new endpoint
- Added matching handler tests, store unit tests, and Postgres-backed
  integration tests for search behavior.
- Verified `go test ./...`, the live store integration tests, `npm run lint`,
  and `npm run build` all pass with the search slice in place.
- Added an Ollama embedding client targeting `/api/embed` with
  `nomic-embed-text` as the default configured model.
- Wired note create/update operations to attempt embedding generation and store
  the resulting vector in `notes.embedding` when Ollama is configured.
- Kept embedding generation non-blocking for CRUD: if Ollama is unavailable or
  embedding generation fails, note writes still succeed and the embedding stays
  `NULL`.
- Added backend config and Docker Compose wiring for `OLLAMA_BASE_URL`,
  `OLLAMA_EMBED_MODEL`, and `OLLAMA_TIMEOUT`, including a Docker default that
  reaches a host Ollama instance through `host.docker.internal`.
- Verified `go test ./...` and the live store integration tests still pass with
  the optional embedding path enabled in the codebase.
- Upgraded `NoteStore.SearchNotes` to prefer semantic vector search when the
  query can be embedded and stored note embeddings exist.
- Implemented pgvector cosine-distance ranking for semantic search, with
  fallback to the existing plain text title/content/tag search path when
  embeddings are unavailable.
- Added store unit tests and Postgres-backed integration tests covering the
  semantic search path and vector-similarity ordering.
- Verified `go test ./...` and the live store integration tests pass with the
  semantic search path enabled.
- Upgraded the search API response shape to return note plus lightweight search
  metadata (`matchType` and semantic similarity score when available).
- Updated the frontend search UI to render semantic/text fallback badges and
  scores for ranked results instead of treating search responses as plain notes.
- Verified `go test ./...`, `npm run lint`, and `npm run build` all pass with
  the explainable-search response shape.
- Added backend search diagnostics so each `/api/v1/search` request logs the
  selected search mode (`semantic`, `text`, `list`) and any text fallback
  reason.
- Added `NoteStore.SearchNoteResultsWithDiagnostics` and unit tests that verify
  semantic-mode detection and semantic-to-text fallback reason reporting.
- Added a new backfill command at `backend/cmd/backfill_embeddings` to generate
  embeddings for existing notes that still have `NULL` vectors.
- Added store unit tests for embedding backfill behavior (success, partial
  failures, and embedder-unconfigured guard).
- Extended `/api/v1/health` to include embedding-provider health alongside
  database health.
- Added Ollama health probing (`GET /api/tags`) and tests, and surfaced
  `embeddings.status` plus configured model in the health payload.
- Updated the frontend backend-status card to render database and embeddings
  status so semantic fallback is visible from the UI.

## How To Extend This Log

- Append a new dated section for each future Codex session.
- Record what changed, why it changed, and any unresolved follow-up items.
