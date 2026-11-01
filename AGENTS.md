# Repository instructions

## Project and scope

`amer-prototype` is a Go HTTP JSON service with a Task Tracking reference feature, PostgreSQL persistence through GORM, and an in-process completion-event queue. Use the Go version declared in `go.mod`.

Keep changes within the requested feature. Interview, voice, and transcript specifications under `docs/services/live-conversation/` describe future work; read them when that feature is requested, rather than treating them as implemented behavior. Redis and RabbitMQ are optional local Compose services, not dependencies of Task Tracking.

## Read the relevant sources

- For setup, environment variables, Docker, and operational behavior, read `README.md` and the applicable `Makefile` targets.
- For Task business changes, read `docs/services/amer-prototype/business/README.md` and `docs/services/amer-prototype/business/capabilities/task-tracking.md`.
- For Task architecture, persistence, event acceptance, time semantics, or shutdown changes, read `docs/design/task-tracking.md`.
- For HTTP changes, read `docs/api/openapi.yaml` and `docs/api/task.md`; update the contract, endpoint documentation, and `docs/api/CHANGELOG.md` together when the public contract changes.
- For original boilerplate requirements and verification limitations, consult `requirement.txt` and `docs/gonesis-audit.md`. The design record documents owner amendments to the original requirements; historical worktree and delivery instructions are context for the original task, not automatic instructions for every new task.

## Architecture and implementation

Keep the four layers directly at the repository root. Add features within these layers rather than introducing a parallel service layout.

| Location | Responsibility and dependency boundary |
| --- | --- |
| `domain/` | Aggregates, value objects, enums, and domain events. Owns invariants and lifecycle rules. Imports only domain packages and the standard-library imports permitted by `test/architecture/dependencies_test.go`. |
| `application/` | Commands in `usecase/`, reads in `query/`, DTOs, repository contracts, and service ports. Depends inward; excludes delivery, infrastructure, and GORM. |
| `delivery/` | HTTP parsing, validation of transport input, response formatting, and error mapping through application operation interfaces. Excludes infrastructure imports. |
| `infrastructure/` | Persistence and service adapters. Only `infrastructure/runtime/`, the composition root, may import delivery. |
| `cmd/app/` | Configuration, logging initialization, signals, and process shutdown enforcement. |

- Follow the existing operation pattern: an operation interface consumed by delivery, a plain implementation struct with injected ports, and detached application DTO results.
- Keep business rules in the domain and storage models in the persistence adapter. Preserve aggregate ownership by copying mutable time pointers at boundaries.
- Inject trusted time through `application/service.Clock`; preserve UTC millisecond precision and exact boundary behavior.
- Preserve transactional mutation through the repository `Update` callback. Completion reserves queue capacity before commit, cancels the reservation on transaction failure, and publishes only after successful commit.
- Keep the queue's process-local guarantees explicit. Durable delivery, automatic retries, and external brokers require an intentional design change.
- Keep startup and shutdown bounded. Read the runtime design before changing admission, draining, cancellation, or resource cleanup.
- Preserve the standard JSON envelope, strict request-field validation, nullable response timestamps, and safe error mapping. Public error codes live in `pkg/errs`; localized English and Indonesian messages live in `translations/errors.*.toml`.
- Document every authored named type, function, and method, including unexported declarations, interface methods, tests, and test helpers. `test/architecture/documentation_test.go` enforces this rule.
- Format changed Go files with `gofmt`. When changing mocked interfaces, update `.mockery.yml` as needed and regenerate `test/mocks/` using `make mocks`; retain generator-owned output and documentation.

## Verification

Choose tests for the changed behavior, then run the applicable repository checks. The `Makefile` owns the exact commands.

| Command | Purpose |
| --- | --- |
| `make test` | Domain/application race checks and coverage output. |
| `make test.all` | All packages with the race detector. |
| `make test.contract` | OpenAPI validation. |
| `make test.architecture` | Dependency-direction and declaration-documentation checks. |
| `make lint` | golangci-lint v2 checks. |
| `make test.security` | Dependency vulnerability checks with govulncheck. |
| `make check` | Combined all-package, contract, architecture, lint, and security checks. |
| `make test.integration` | Real repository and HTTP smoke tests using `TEST_DATABASE_URL`. |
| `make test.docker` | Isolated Compose PostgreSQL and real integration tests. |
| `make docker.verify` | Full isolated stack verification, including optional service readiness. |

- Cover changed invariants through public domain behavior, application orchestration through mocked ports, and HTTP behavior through mocked operation interfaces. Use real PostgreSQL tests for locking, rollback, persistence, and composition behavior.
- Use controlled time for due-date equality, overdue boundaries, and timestamp ownership. For completion changes, verify queue refusal, commit failure, and repeated completion leave state and publication consistent.
- Database tests can mutate schema and data. Set `TEST_DATABASE_URL` only for an explicitly authorized integration database. The isolated Docker verification targets are available for local database checks and require Docker Compose and Python 3.
- `go test ./...` and `make test.all` skip database suites when `TEST_DATABASE_URL` is absent. Report these checks as skipped, not as successful database verification; `make test.integration` and `make test.smoke` require the variable.
- Report exact verification commands, results, and remaining limitations. Record RED/GREEN evidence only when actually observed; current passing tests do not establish historical TDD evidence.

## Local resources and secrets

- Supply credentials through environment variables. Keep `.env` values, database URLs, and passwords out of committed files, logs, and command output; use the example environment files for setting names.
- Manual versioned SQL migrations are the approved schema policy (identity-access D8). T16–T17 capture the legacy schema and add tooling; runtime/test AutoMigrate remains transitional until T18 replaces it. Use only a suitable development database when launching this transitional binary.
- Follow `README.md` for Compose setup. Preserve normal development volumes; isolated verification scripts manage only their own resources.
- Keep unrelated working-tree changes intact. Remote publication, deployment, commits, and PR creation follow the user's current request.

<!-- postman-cli:installation:start -->
Postman CLI is installed on this device, and can be used for API Engineering work.
<!-- postman-cli:installation:end -->
