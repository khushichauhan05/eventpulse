# Release Notes

This release is the repository stabilization pass focused on reproducibility, documentation, and clean interview presentation.

## Stabilization Release

This release focuses on making EventPulse reproducible, documented, and verifiable end to end.

### What Works

- Full Kafka pipeline from `POST /events` to PostgreSQL alert persistence.
- API endpoints for `/health`, `/events`, `/alerts`, and `/alert?id=`.
- Clean-clone Compose startup without a required local `.env` file.
- Structured logging and startup retries for Postgres.
- PostgreSQL schema initialization via `db/init.sql`.
- Screenshot artifacts for the architecture and validation evidence are included in `docs/screenshots/`.

### Fixes Included

- Removed table creation from Go services.
- Added Docker healthchecks and service restart policies.
- Introduced shared internal packages for config, database, Kafka, logging, models, handlers, retry, and service orchestration.
- Switched Dockerfiles to multi-stage builds.
- Added a professional README and operational notes.
- Tightened the repository status and validation notes for portfolio readiness.

### Remaining Gaps

- The system is not yet covered by automated tests.
- No dead-letter queue or retry-topic topology.
- No CI/CD pipeline.
- No observability stack yet.

### Suggested Next Release

1. Integration test coverage.
2. DLQ support for malformed Kafka payloads.
3. CI checks for build, compose, and smoke tests.