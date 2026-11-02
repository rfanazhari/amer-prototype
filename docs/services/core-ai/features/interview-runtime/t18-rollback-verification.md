# T18 rollback follow-up — CORE-IRT-018

## Change and approval

The owner explicitly approved `Setujui koreksi down v10` on 2026-11-03 after reviewing the proposed dependency-order patch. This is a narrow exception to the immutable migration policy: only `000010_interview_runtime.down.sql` and its two checksum records change. Every up artifact and release version 11 remain unchanged. Rollback stays guarded by disposable confirmation; no runtime migration, new schema version, CASCADE or production-data repair is introduced.

The old down file dropped transcript entries before coverage citations referencing those entries. PostgreSQL refused with SQLSTATE `2BP01`; the explicit transaction then returned `25P02`. Coverage now drops before transcript, and transcript/checkpoint dependents drop before sessions. Progress remains last. Only the eight runtime tables are removed; prior execution pins survive.

The former artifact test merely reversed creation order, which missed FKs added after table creation. It now asserts actual child/parent dependencies and rejects extra drops or CASCADE. The PostgreSQL scenario populates all eight tables, including a coverage answer citation and session/checkpoint dependents, before down; it verifies preserved pins and empty re-created runtime tables after up.

## RED and GREEN

Cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
GOCACHE=/private/tmp/t18-go-cache make test.docker
GOCACHE=/private/tmp/t18-go-cache go test ./infrastructure/database/schema -run TestInterviewRuntimeMigrationArtifacts -count=1
```

Observed RED: Compose reproduced the FK failure. The new artifact regression independently failed `coverage citations before transcript`. No implementation scaffolding was needed.

- RED SHA: `e0f9a10296e5cee8eb9c24421738921f78791b2e`
- RED subject: `test: [CORE-IRT-018][rfanazhari] RED cover interview rollback foreign key order`
- GREEN subject: `fix: [CORE-IRT-018][rfanazhari] GREEN correct disposable interview rollback order`
- The GREEN SHA is reported in the final handoff; resolve it by this exact subject in Git.

After correction, both commands pass. Additional checks:

```sh
GOCACHE=/private/tmp/t18-go-cache make test test.contract test.architecture
GOCACHE=/private/tmp/t18-go-cache make test.all
GOCACHE=/private/tmp/t18-go-cache GOLANGCI_LINT_CACHE=/private/tmp/t18-lint-cache make lint test.security
```

Results: **passed**. Lint reports 0 issues; security reports 0 reachable vulnerabilities. `make test.all` without database variables skips resource suites; the successful Compose command separately executes actual PostgreSQL migration, integration/HTTP and smoke suites, plus backup/restore, guarded adoption and retained-row comparison. The script removed only its own project containers, network and volume on completion. No required check remains failed or skipped; the intentional RED failures are resolved.

No interfaces or dependencies changed, so Mockery regeneration was not applicable. Existing sqlmock tests remain; the new rollback proof runs real PostgreSQL rather than substituting mocked SQL semantics.

## Integrity and assessment

- Previous down digest: `049a54c3f2191d627a9b9778ec542e441710cd2725457b6799ba38c4464a7fab`.
- Approved corrected down digest: `8af2b71e7103d14326e475b885dc86c4c2ec558e124c3c20846c45e73f81702f`.
- Version-10 up digest remains `bf1bde8379e4f705be46a238bee5744e746380195a164b37fd73421d9e55cdce`.

Design retained: dependency-ordered SQL inside the existing explicit transaction, checksum enforcement, advisory locking, bounded commands and disposable-only destructive policy. The correction fixes artifact order without changing aggregate/application/HTTP behavior. No additional refactor was selected. The previously recorded T18 version-10 down limitation is resolved by this approved follow-up; earlier checkpoint evidence remains historical. Unrelated T17/pre-existing working-tree edits are preserved.
