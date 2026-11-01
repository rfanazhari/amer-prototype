# amer-prototype

Go service foundation with the Task Tracking reference feature. The four root layers are `domain`, `application`, `delivery`, and `infrastructure`. Application operations expose interfaces and return DTOs; the domain owns lifecycle rules and uses a typed state enum. PostgreSQL/GORM persists tasks, and a process-local queue accepts completion events.

## Run locally

Use Go 1.26.10. Supply `DATABASE_URL` as a PostgreSQL connection URL through the environment. `HTTP_ADDRESS` defaults to `:8080`. `APP_ENV` defaults to development in logmanager; production disables its debug mode. Administrative startup requires `ACCESS_BOOTSTRAP_ADMIN_NAME`, `ACCESS_BOOTSTRAP_ADMIN_EMAIL`, `ACCESS_BOOTSTRAP_ADMIN_PASSWORD`, `ACCESS_HTTP_ORIGIN`, and `ACCESS_CSRF_SECRET` (at least 32 bytes). `TEST_DATABASE_URL` is only for explicitly authorized integration databases. Secrets must stay out of source files and command output.

Schema is installed explicitly through versioned SQL migrations (identity-access D8). Startup only reads migration metadata and requires clean version 3 before bootstrap/listening; it never creates or repairs schema. See the [migration runbook](docs/design/database-migrations.md) for preparation, populated adoption and recovery. Tasks have a UUID primary key, non-null title/status/creation/update times, optional due/completion times and a status check. Retained data requires backup/restore or reviewed forward repair; destructive reversal is for disposable databases. Data persists across service restarts. Startup connection and schema checks have a ten-second budget. HTTP health reports HTTP readiness only.

## Local Docker setup

Dockerfile builds a static binary and runs it as a non-root user in a minimal image. Compose starts the app and PostgreSQL by default; prepare schema with the separate migration job first. Set a URL-safe local `POSTGRES_PASSWORD` in the environment, then run:

```sh
make docker.config
docker compose up -d --wait database
make docker.migrate MIGRATION_VERSION=3
make docker.up
make docker.logs
make docker.down
```

`make docker.logs` shows structured startup stages: `service starting`, `database connected and schema compatible` (version 3), `administrative bootstrap complete`, then `http server ready` with the bound address and `event_queue=in_process`. The startup summary includes elapsed time. A failed startup logs `service startup failed` and never logs readiness. Lifecycle records omit database URLs, credentials and bootstrap account details.

The app listens at `http://localhost:8080`. `HTTP_PORT` and `POSTGRES_PORT` customize host ports. PostgreSQL uses a named volume; `docker.down` preserves it. The example environment file contains setting names and no credentials.

Optional services are infrastructure for future integrations, with no connections from the Task feature:

```sh
make docker.redis
# Set RABBITMQ_PASSWORD in the environment before either RabbitMQ command.
make docker.rabbitmq
make docker.optional
```

Redis uses the `redis` profile and port 6379; RabbitMQ uses `rabbitmq`, AMQP 5672 and management 15672 with username `amer`. Host bindings use loopback. Redis is intended for local development and has no authentication enabled. `REDIS_PORT`, `RABBITMQ_PORT`, and `RABBITMQ_MANAGEMENT_PORT` customize ports. No remote image publication or deployment is performed.

## Task API

See [OpenAPI](docs/api/openapi.yaml), [Task endpoints](docs/api/task.md), [administrative access](docs/api/identity-access.md), and [changelog](docs/api/CHANGELOG.md). The Task flow is `POST /tasks`, `POST /tasks/{id}/start`, `POST /tasks/{id}/complete`, and `GET /tasks/{id}`. `PATCH /tasks/{id}/due-date` changes or clears a date. `/health` is unauthenticated. Administrative routes use cookies, origin, and CSRF; they are not attached to Task or voice. Requests reject unknown fields; errors use the same JSON envelope, with English or Indonesian messages selected through `Accept-Language`.

Completion queue capacity is 100 with one consumer. Capacity is reserved before commit, released on transaction failure, and the event is enqueued only after successful commit. Events are volatile; subscriber failures are logged without automatic retry. There is no durable recovery or exactly-once guarantee.

## Verification

```sh
make test                # domain/application race checks and coverage.out
make test.all            # all packages, including runtime tests
make test.contract       # validate OpenAPI
make test.architecture   # repeatable import-direction checks
make lint                # golangci-lint v2
make test.security       # govulncheck
make mocks               # Mockery v2.53.6, built with the project Go version
make test.integration    # requires TEST_DATABASE_URL
make test.docker         # isolated Compose PostgreSQL + real integration tests
```

`test.docker` requires Python 3 and Docker Compose. It generates temporary credentials in memory, uses an isolated project and ephemeral host port, runs the actual composition/API and repository checks, and removes only its own containers and volumes. Normal Compose data is preserved. Direct `go test ./...` skips database suites without `TEST_DATABASE_URL`; a skip is not a passing database check. `make test.integration` and `make test.smoke` fail if the variable is absent.

## Shutdown

The executable installs a five-second process watchdog. HTTP admission stops, graceful request/event drain receives up to four seconds, and timeout cancels active request contexts and closes HTTP forcibly. The remaining budget covers worker/resource cleanup. A blocked cleanup causes exit code 1; event delivery and graceful database cleanup are not claimed in that case. `runtime.Run` alone does not install a process-exit policy. Ordinary subscriber errors are logged and processing continues; a panic is supervised as an unexpected worker failure.

See the owning [design and verification record](docs/design/task-tracking.md), [business rules](docs/services/amer-prototype/business/README.md), and [audit](docs/gonesis-audit.md). Historical TDD evidence and remaining acceptance limitations are recorded explicitly there.

`make docker.verify` also builds and starts the full isolated stack, verifies the container's HTTP health and optional service readiness, then runs the real integration suite. It removes only its own project resources afterward. Statement coverage and passing current scenarios do not reconstruct missing historical TDD evidence; see the design record for that limitation.

## Explicit migration tooling

Supply `MIGRATION_DATABASE_URL` through the environment with DDL permissions. Commands never accept or print credentials. From the repository root:

```sh
make migrate.status
make migrate.up MIGRATION_VERSION=3
make migrate.baseline
# Disposable databases only; removes data in the selected tables:
make migrate.down MIGRATION_STEPS=1 MIGRATION_DISPOSABLE=yes
```

Up requires target 1, 2 or 3 and never applies future candidate artifacts. Baseline requires absent metadata and exact legacy Task-only or three-table catalog equivalence. Stop legacy writers and take a restorable backup before adoption. Drift, partial/foreign schemas and dirty versions require inspection; no automatic force is provided. Baseline uses the same advisory lock as up/down. Commands allow 59 seconds for operations and one second for cleanup; lock waits are capped at five seconds. Baseline SQL uses explicit transactions and pinned checksums; change future schema through new versions rather than editing applied files. Run status after failures and use reviewed forward repair or restore.

`DATABASE_URL` is the DDL-denied application connection; `MIGRATION_DATABASE_URL` is the separate DDL-capable operator connection. Configure grants as described in the runbook. Normal local Compose uses its development owner for convenience; restricted-role runtime behavior is verified on disposable PostgreSQL. `TEST_MIGRATION_DATABASE_URL` is a separate explicitly authorized disposable database whose public schema is reset by migration acceptance tests. The isolated verification scripts supply it automatically.
