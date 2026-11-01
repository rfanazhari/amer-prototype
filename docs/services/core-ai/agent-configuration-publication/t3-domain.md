# T3 domain assessment — CORE-ACP-003

Implemented only instance version and ownership mechanics. T1 contracts,
production template fields/validators (T5), persistence and HTTP are unchanged.

## Design mapping and refinement

Technical-design §3–4 and §6 and tactical-strategy §3 remain the ownership map:
`AgentInstance` controls current draft replacement and active selection;
`Validation` owns detached exact-version findings; `PublishedRevision` exposes
immutable snapshots. Domain code imports only `errors` and `time`.

Types reside together in `domain/aggregate/agentconfig` to keep this feature's
version and ownership rules separate from Task. Responsibility is split across
instance, revision, validation, template and version files.

`Configuration[C]` is a pure value-copy contract: concrete configurations must
implement a deep `Clone()` including slices, nested references and time pointers.
The test implementation is a minimal editable fixture, not a mock or a production
schema. This internal refinement lets T3 enforce ownership without inventing Q1
fields or normalization. T5 must supply concrete domain values and test their
copying behavior. No arbitrary maps, JSON serialization or hashing are introduced.

The opaque `TemplateRef` includes template, template version and behavior version;
all must match the assessment. Positive `DraftVersion` is encapsulated and cannot
wrap on replacement. Creation accepts an explicit initial version/content;
it does not settle Q1 initial-draft policy or supported template identities.

New instances cannot carry active history. Separate restoration checks structural
identity, versions, ownership, timestamp ordering and provenance while preserving
stored content without invoking current eligibility/catalog/provider policy.
A restored active identity must belong to the instance/template and cannot refer
to a future draft version. Full revision/validation foreign-key consistency is
still the adapter's T4/T7 responsibility, not established by these value checks.

## Guarantees and downstream integration

- Replacement increments the expected version, rejects stale/zero/future
  expectations, copies input and preserves the active selection.
- Validation must match instance, draft, template and behavior interpretation.
  Ineligible assessments can be recorded but cannot publish.
- Publication copies the exact eligible draft, retains validation provenance,
  allocates the next revision number and changes only current active selection.
- Returned snapshots cannot mutate owned slices, optional times or active identity.
  Later replacement/publication leaves earlier revision values unchanged.
- Reusing the current active revision ID is rejected locally. Global/historical ID
  uniqueness, serialization and atomic commit remain repository responsibilities.
- Application replay must precede `Publish`; do not pass an old replay through
  new-publication eligibility or reselect an old revision.

T2 generic ports remain unchanged. T6/T7 can instantiate them with
`*AgentInstance[C]`, `Validation`, and `PublishedRevision[C]` (or detached snapshots
where their adapter mapping requires them). T5 supplies C; no DTO shape is settled
here. Transactions, authentication, warning policy, historical deployment support
and external consumer bindings remain T1/T4–T11 work as planned.

## Evidence and assessment

All commands use repository root
`/Users/rfanazhari/Projects/go-workspace/prototype`.

RED and GREEN scenario command:

```sh
go test ./domain/aggregate/agentconfig -run 'Test(DraftVersion|Instance|Validation|Revision)' -count=1
```

RED `2c5d335e08e10b36c79abd2049d9c32ec608ce16` observed executable assertion
failures: positive versions were rejected by inert construction stubs.
The checkpoint includes tests plus only the declarations/stubs needed to execute
them. GREEN implements those operations without weakening assertions.

Passed: scenario command, `make test`, `make test.all`,
`make test.architecture`, `make test.contract`, `make lint`,
`make test.security`. T3 statement coverage: 97.2%.
The first sandboxed `make test` was blocked by Go-cache permissions; the complete
sequence passed when rerun with cache access. Govulncheck found no affected
vulnerabilities in called code; it reported uncalled dependency findings.

Database and live voice suites remain opt-in/skipped without their environment;
no PostgreSQL, SQL-mock, HTTP runtime or live-provider claim is made for T3.
`make test.integration`, `make test.docker`, and `make docker.verify` are not run:
this task changes no persistence or runtime integration. No I/O ports are mocked,
so sqlmock and Mockery generation are not applicable here.

Assessment disposition: **retained with rationale**. Private aggregate state,
validated construction/restoration, immutable owned values, bounded snapshots
and purpose-specific operations keep invariant ownership local without a policy
engine or speculative I/O. No additional refactor or new decision is required.
The refinement above is the synchronization note for T5/T6/T7; broader template,
schema and integration acceptance remains pending under their existing gates.
