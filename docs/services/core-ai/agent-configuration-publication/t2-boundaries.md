# T2 boundary assessment — CORE-ACP-002

Implemented only the application foundation in T2. No authentication, domain
configuration, SQL adapter, HTTP route, or public API is supplied.

## Design mapping and synchronization

Technical-design §3–5 and tactical-strategy §3 remain authoritative for ownership:
AgentInstance owns version/eligibility/active-selection invariants; application
ports describe I/O; the future adapter enforces atomicity and ownership constraints.

The concrete repository contract is split into `Reader`, `Mutation`,
`Transaction`, and `ReplayReader` in `application/repository/agentconfigport`.
This refines the proposed single `application/repository/agentconfiguration.go`
location to keep the feature contracts and uncertain-outcome classification together.
`AgentConfigurationAuthority` and `AgentReferenceReader` live in application/service.

Generic snapshot parameters deliberately defer instance, validation, revision,
and reference-fact shapes to T3/T5 and Q1. They are not configuration DTOs or an
arbitrary-map schema engine. T3/T6/T7 must instantiate them with the accepted
domain types; T1 must settle command scope, key syntax and intent normalization.
CommandIdentity merely carries that eventual normalized identity.

## Contract guarantees

- Reads are bounded and detached; historical revision restoration does not run
  current creation/provider policy.
- Mutation serializes instance-owned changes; callback and known commit failures
  roll back. A successful callback alone is not a successful durable result.
- Publication checks command replay before current-draft eligibility. Revision,
  active pointer and command result commit together; historical revisions have
  no update operation.
- Unique-conflict losers reconcile outside the aborted transaction. Uncertain
  commits retain the original key and preserve technical error classification;
  they do not assert rollback or authorize another publication identity.
- Principal is resolved by the trusted adapter, without a client role field.
  Every operation including replay must authorize before disclosure. Real
  expiry, revocation, candidate denial and CSRF controls remain Q2/T8 work.
- References resolve exact immutable versions outside transactions. Missing facts
  are distinct from dependency errors; model availability is excluded.

## Evidence and assessment

Mockery v2.53.6 generated and registered six interfaces, including generic ports.
Executable RED/GREEN tests cover uncertain-commit cause preservation and safe
messages, including unknown causes. Interface contracts themselves have no
adapter behavior to execute in T2. Generated mocks compile with the repository.

Retained after assessment: narrow read/mutation/reconciliation responsibilities,
no ORM dependency, no guessed configuration shape, and no publication behavior
outside the aggregate. No additional refactor was justified. SQL mocking is not
applicable in T2 because no SQL is authored; use sqlmock for future SQL unit tests.
Mock compilation and contract comments do not establish rollback, locking,
real authentication or PostgreSQL guarantees. Those proofs remain T6–T9.
