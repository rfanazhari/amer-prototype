.PHONY: check test test.all test.contract test.architecture test.smoke lint test.security mocks format
check: test.all test.contract test.architecture lint test.security

test:
	go test -race ./domain/... ./application/... -coverprofile=coverage.out

test.all:
	go test -race ./...

test.contract:
	go test ./test/contract/... -v

test.architecture:
	go test ./test/architecture/... -v

test.smoke:
	@test -n "$$TEST_DATABASE_URL" || (echo 'TEST_DATABASE_URL is required; database verification is unfinished' >&2; exit 1)
	go test ./test/smoke/... -count=1 -v

lint:
	golangci-lint run --timeout 5m

test.security:
	govulncheck ./...

mocks:
	mockery

format:
	gofmt -w application domain delivery infrastructure cmd pkg translations test

.PHONY: docker.build docker.config docker.up docker.down docker.logs docker.redis docker.rabbitmq docker.optional
docker.build:
	docker build -t amer-prototype:local .

docker.config:
	docker compose config --quiet

docker.up:
	docker compose build app
	docker compose up -d --wait database
	@schema_status="$$(docker compose run --rm --no-deps migrate status)" && test "$$schema_status" = 'version=9 dirty=false' || (echo 'Prepare schema first: make docker.migrate MIGRATION_VERSION=9 (existing volumes: see database-migrations runbook)' >&2; exit 1)
	docker compose up -d

docker.down:
	docker compose --profile redis --profile rabbitmq down

docker.logs:
	docker compose logs -f app

docker.redis:
	docker compose --profile redis up -d

docker.rabbitmq:
	@test -n "$$RABBITMQ_PASSWORD" || (echo 'RABBITMQ_PASSWORD is required for rabbitmq' >&2; exit 1)
	docker compose --profile rabbitmq up -d

docker.optional:
	@test -n "$$RABBITMQ_PASSWORD" || (echo 'RABBITMQ_PASSWORD is required for rabbitmq' >&2; exit 1)
	docker compose --profile redis --profile rabbitmq up -d

.PHONY: test.integration test.docker
test.integration:
	@test -n "$$TEST_DATABASE_URL" || (echo 'TEST_DATABASE_URL is required' >&2; exit 1)
	go test -race -p 1 ./test/integration/... ./test/smoke/... -count=1 -v

test.docker:
	python3 scripts/verify_compose.py

.PHONY: docker.verify
docker.verify:
	python3 scripts/verify_compose.py --stack

.PHONY: migrate.status migrate.up migrate.baseline migrate.down
migrate.status:
	@go run ./cmd/migrate status
migrate.up:
	@test -n "$(MIGRATION_VERSION)" || (echo 'MIGRATION_VERSION is required' >&2; exit 1)
	@MIGRATION_VERSION="$(MIGRATION_VERSION)" go run ./cmd/migrate up
migrate.baseline:
	@go run ./cmd/migrate baseline
migrate.down:
	@test "$(MIGRATION_DISPOSABLE)" = yes || (echo 'MIGRATION_DISPOSABLE=yes required' >&2; exit 1)
	@test -n "$(MIGRATION_STEPS)" || (echo 'MIGRATION_STEPS is required' >&2; exit 1)
	@MIGRATION_STEPS="$(MIGRATION_STEPS)" MIGRATION_DISPOSABLE=yes go run ./cmd/migrate down

.PHONY: docker.migrate
docker.migrate:
	@test -n "$(MIGRATION_VERSION)" || (echo 'MIGRATION_VERSION is required' >&2; exit 1)
	@MIGRATION_VERSION="$(MIGRATION_VERSION)" docker compose --profile migration run --build --rm migrate
