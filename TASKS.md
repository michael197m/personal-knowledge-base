# Project Improvement Tasks

- [x] 1. Add authentication and real user ownership (JWT/session + per-user data isolation).
- [ ] 2. Move embedding generation to async background jobs with retry/backoff and dead-letter handling.
- [ ] 3. Add pagination/cursor APIs for notes and search to keep performance stable as data grows.
- [ ] 4. Add observability: structured logs, request IDs, metrics (latency, fallback rate, Ollama availability), and dashboards.
- [ ] 5. Harden API contracts: request validation, stricter error codes, and OpenAPI spec generation.
- [ ] 6. Expand testing: API integration tests in CI (backend + Postgres + optional Ollama mock), plus frontend component tests for search/fallback states.
- [ ] 7. Improve search quality: hybrid ranking (vector + keyword), configurable thresholds, and basic relevance feedback.
