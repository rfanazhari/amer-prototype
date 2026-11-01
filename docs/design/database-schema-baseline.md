# Legacy database schema baseline

Tickets: CORE-IAM-016 / CORE-IAM-017. Captured 2026-11-01 on disposable PostgreSQL 17.11, GORM v1.25.12 and postgres driver v1.5.11.

Observed by executing `db.AutoMigrate(&taskrepository.Model{})` followed by `accessrepository.AutoMigrate(db)` against an empty isolated PostgreSQL 17 container. No deployed database was inspected. This describes generated synthetic schema, not deployed-target compatibility.

## Frozen facts

The complete executable catalog is [legacy_automigrate.sql](../../test/integration/testdata/schema/legacy_automigrate.sql). Machine-readable normalized column, constraint and index facts are [tasks.json](../../infrastructure/database/schema/tasks.json) and [administrative.json](../../infrastructure/database/schema/administrative.json); [catalog.sql](../../infrastructure/database/schema/catalog.sql) records the extraction query.

All columns have no database default. All timestamps are `timestamp with time zone` with precision 6, not a database millisecond truncation. UUIDs have no generated default.

| Table | Required columns | Nullable columns |
| --- | --- | --- |
| tasks | id uuid PK; title/status text; created_at/updated_at timestamptz | due_date/completed_at timestamptz |
| administrative_accounts | id uuid PK; name/email/password_hash/role/status text; created_at/updated_at timestamptz | created_by uuid |
| administrative_sessions | id uuid PK; account_id uuid; token_hash text; created_at/expires_at timestamptz | revoked_at timestamptz |

Checks: Task status in todo/in_progress/done; account role ADMIN/HR, status ACTIVE/DISABLED, nonempty password hash; session token lowercase 64-character hex, expiry strictly after creation, revocation absent or at/after creation. Exact names/expressions are frozen in SQL and catalog JSON.

Primary keys are tasks_pkey, administrative_accounts_pkey, administrative_sessions_pkey. Both creator self-reference and session account FK use ON UPDATE RESTRICT / ON DELETE RESTRICT. Five additional btree indexes: unique administrative_accounts_email_INDEX and administrative_sessions_token_hash_INDEX; nonunique administrative_accounts_created_by_INDEX, administrative_sessions_account_id_INDEX, administrative_sessions_expires_at_INDEX. There are no seed rows.

## Caller inventory and allocation

- Production: infrastructure/runtime/resources.go calls Task AutoMigrate and accessrepository.AutoMigrate; infrastructure/repository/accessrepository/migrate.go installs both administrative models and 13 additive index/constraint statements.
- Tests: test/integration/repository_test.go, access_repository_test.go and access_http_test.go invoke the same installers. lifecycle_test.go exercises runtime DDL cancellation indirectly.
- No other executable installer was found by `rg 'AutoMigrate|CREATE TABLE|ALTER TABLE|CREATE.*INDEX' infrastructure test/integration`.
- Reserve 000001 for Task and 000002 for administrative access. Existing proposed 000003/000004 candidate artifacts remain untouched. Future assignment ordering remains its owner's responsibility.

## Policy and limits

D8 approves manual migrations; root instructions, README, original requirement amendment, Task design and audit current-policy note are synchronized. Runtime/test switching remains T18. Source inspection or sqlmock is not PostgreSQL proof. Capturing this disposable schema authorizes no deployed-target adoption; that requires target inspection, stopped legacy writers and a restorable backup.
