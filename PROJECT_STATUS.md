# Project Status

This repository is currently stabilized for portfolio use and clean-clone reproduction.

## Verified Functionality

- `go build ./...` completes successfully.
- `docker compose config` resolves without requiring a local `.env` file.
- `docker compose up -d --build` starts Kafka, PostgreSQL, API Gateway, Analytics Service, and Alert Service successfully.
- Health checks are exposed for all services.
- `POST /events` publishes to Kafka topic `events.raw`.
- Analytics consumes `events.raw`, scores the event, and publishes to `events.processed`.
- Alert Service consumes `events.processed`, publishes alerts to Kafka topic `alerts`, and stores them in PostgreSQL.
- `GET /alerts` returns persisted alerts.

## Validation Artifacts

- [Architecture screenshot](docs/screenshots/architecture.svg)
- [Validation screenshot](docs/screenshots/validation.svg)

## Issues Fixed During Stabilization

- Removed the hard dependency on a local `.env` file for Compose startup.
- Added PostgreSQL initialization SQL in `db/init.sql`.
- Added multi-stage Dockerfiles for all services.
- Added structured JSON logging.
- Added health endpoints and Docker healthchecks.
- Added connection retry logic for PostgreSQL startup.
- Added screenshot artifacts for architecture and validation evidence.

## Remaining Gaps

- No automated test suite yet.
- No dead-letter topic for malformed Kafka messages.
- No metrics or tracing.
- No idempotency protection for duplicate alert writes.
- No CI pipeline defined in the repository.

## Recommended Next Steps

1. Add integration tests with Testcontainers.
2. Add a dead-letter topic for poison messages.
3. Add idempotency keys or deduplication for alerts.
4. Add CI for build and compose validation.
5. Add metrics and tracing after the repo is test-covered.