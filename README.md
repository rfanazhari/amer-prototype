# amer-prototype

Go service foundation with the Task Tracking reference feature. The four root layers are `domain`, `application`, `delivery`, and `infrastructure`. Application operations expose interfaces and return DTOs; the domain owns lifecycle rules and uses a typed state enum. PostgreSQL/GORM persists tasks, and a process-local queue accepts completion events.

## Run locally

Use Go 1.26.10. Supply `DATABASE_URL` as a PostgreSQL connection URL through the environment. `HTTP_ADDRESS` defaults to `:8080`. `APP_ENV` defaults to development in logmanager; production disables its debug mode. Administrative startup requires `ACCESS_BOOTSTRAP_ADMIN_NAME`, `ACCESS_BOOTSTRAP_ADMIN_EMAIL`, `ACCESS_BOOTSTRAP_ADMIN_PASSWORD`, `ACCESS_HTTP_ORIGIN`, and `ACCESS_CSRF_SECRET` (at least 32 bytes). `TEST_DATABASE_URL` is only for explicitly authorized integration databases. Secrets must stay out of source files and command output.

The composition runs GORM AutoMigrate for `tasks` at startup. This follows the selected requirement; explicit SQL migration files remain a pending schema-policy amendment. Tasks have a UUID primary key, non-null title/status/creation/update times, optional due/completion times and a status check. Rollback is manual table removal. Data persists across service restarts. Startup connection and schema checks have a ten-second budget. HTTP health reports HTTP readiness only.

## Local Docker setup

Dockerfile builds a static binary and runs it as a non-root user in a minimal image. Compose starts the app and PostgreSQL by default. Set a URL-safe local `POSTGRES_PASSWORD` in the environment, then run:

```sh
make docker.config
make docker.up
make docker.logs
make docker.down
```

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
