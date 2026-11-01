# Technical Design: Identity and Access

## 1. Overview

Date: 2026-11-01. Mode A — new access capability in the existing Go service. Status: planned, with owner-approved business scope and table properties; engineering choices below are assessed design proposals, not runtime evidence. Authority: [Identity and Access](../business/capabilities/identity-access.md), BR-CORE-020..039, the owner's conversation approvals, and [platform design §9–10 / TD-21](../../../design/core-ai/technical-design.md).

### Goals

- Provision ADMIN/HR administrative accounts and real revocable sessions; allow ADMIN to add HR accounts, without HR business authority.
- Supply current HR-only authority to Agent Configuration through the existing application port, including reads and publication replay.
- Preserve reusable candidate identity and issue assignment-scoped invitation/session access without permanent candidate accounts.
- Keep access establishment separate from consent, execution, and provider sessions.

### Non-Goals

Multiple companies/workspaces, public signup, configurable role/permission tables, combined roles, SSO/MFA, account recovery UI, public candidate passwords, mail delivery infrastructure, assignment/execution/consent implementation, interview provider integration, and replacing Task/voice authentication behavior. A DISABLED state does not authorize adding account-lifecycle endpoints. Do not implement a placeholder assignment table or permissive authorizer to unblock candidate delivery.

### Verified project context and impact

| Source inspected | Observed fact and design consequence |
|---|---|
| Root `AGENTS.md`, Architecture / Verification / Local resources | Four root layers; application-owned ports; operation interfaces; detached DTOs; trusted UTC millisecond time; all declarations documented; HTTP tests through mocked operations; bounded lifecycle; additive AutoMigrate. No scoped AGENTS.md or CLAUDE.md found. |
| `README.md`, `go.mod`, `Makefile` | Go 1.26.10, GORM 1.25.12 / postgres driver 1.5.11, PostgreSQL, net/http, testify, Mockery; current startup migrates Task. Redis/RabbitMQ are not access dependencies. |
| `application/service/agentconfiguration.go` | `AdminPrincipal{SubjectID}` and `AgentConfigurationAuthority.Principal/Authorize` exist; there is no real access adapter. Preserve their public shape unless evidence requires a separate synchronized change. |
| `application/usecase/taskusecase/create.go`, `application/repository/task.go` | Narrow delivery operation interface, plain injected implementation, trusted Clock and detached result are suitable patterns. Task business policy is not identity policy. |
| `infrastructure/repository/taskrepository/task.go` | Separate storage model; row locking; explicit updates; restoration errors classified as technical. Task's callback is not an invitation/session repository. |
| `infrastructure/runtime/resources.go`, `http.go`; `delivery/http/handler/taskhandler/handler.go` | Shared database lifecycle and shutdown admission; strict bounded JSON, envelope and safe EN/ID errors. Handler helpers are private and Task-oriented. |
| `docs/api/openapi.yaml`, `docs/api/task.md`, `docs/api/CHANGELOG.md` | Current routes have no auth. Administrative/candidate endpoints here are planned additions, not already exposed APIs. |
| Repository source inventory, `.mockery.yml`, `test/architecture/` | No accounts/candidates/invitations/session persistence or assignment implementation found. Authority mocks exist, not behavior. Domain imports exclude crypto, JSON, net/http and GORM. No migration directory found. |

No live database was inspected. New-access schema assertions are planned, not claims about deployed tables. Existing ACP T2/T3 foundations remain Done; this feature consumes their authority contract without changing publication atomicity, subject-scoped keys, or stable task IDs. The companion ACP tripack receives a documents-only Mode C synchronization.

Project instructions override bundled Backend Service Standard defaults: retain AutoMigrate, test delivery with mocked operations, keep repository ports in application, and preserve encapsulated domain state. No structural cleanup of Task/voice is authorized. Binding contracts are BR rules, roles/scope, expiry, revocation, session separation, confidentiality and ACP error ordering. Internal package/helper choices can be refined with preserved-contract evidence; unresolved product choices in §9 block their dependent work.

## 2. API / Interface Contract Changes

The existing `/admin` prefix means the administrative channel, not the ADMIN role. HR uses this channel too. Proposed additive REST directions:

| Method/path | Authority | Result / responsibility |
|---|---|---|
| POST `/admin/sessions` | Provisioned ADMIN or HR credentials | 201; establish cookie after session commit; account/session metadata only |
| GET `/admin/session` | Valid administrative session | 200; safe account, role, session expiry and CSRF material |
| DELETE `/admin/sessions/current` | Valid administrative session + CSRF | 200 standard envelope after persisted revocation, then clear cookie |
| POST `/admin/hr-accounts` | Current ADMIN + CSRF | 201; create HR, with role fixed by operation, not caller input |
| POST `/admin/candidates` | Current HR + CSRF | 201; record candidate identity |
| GET `/admin/candidates/{id}` | Current HR | 200; detached candidate contact view for selection by known identity |
| POST `/admin/interview-assignments/{id}/invitations` | Current HR + assignment eligibility + CSRF | 201; persist new invitation/revoke former invitation atomically; reveal link/token once |
| POST `/candidate/access/exchange` | Valid invitation + pre-session origin protection | 201 after new session commit; scoped cookie and safe assignment identity; repeat semantics Q3 |
| GET `/candidate/session` | Valid candidate session + accessible assignment | 200; safe candidate/assignment identity, expiry and CSRF material; no execution creation |

Administrative account/session schemas and limits are approved in §6; candidate-specific accepted inputs remain Q3/Q4. No collection listing/search or lifecycle mutation endpoints are silently added. The Interview capability owns `/candidate/interview`, consent, initiation, assignment creation/revocation and its full candidate UI. A known `candidate_id` enables reuse without introducing deduplication by email.

Implement one operation interface per delivery use case/query in `application/usecase/accessusecase/` and `application/query/accessquery/`, returning detached `application/dto` values. Purpose-specific repositories in `application/repository/accessport/` hide ORM handles. Ports in `application/service/` isolate password verification, secure token generation/digests, session resolution and assignment access facts.

The real `AgentConfigurationAuthority` adapter resolves a trusted account/session, returns `AdminPrincipal.SubjectID = administrative_accounts.id`, and consults current account/session state for authorization. ADMIN resolves an administrative principal but fails configuration authorization. Candidate cookies never resolve one. Authenticate in delivery and recheck authority at application boundaries, including non-HTTP operations and replay. Never trust JSON role, account ID header or a caller-constructed principal without current-session binding. Unknown operation strings are denied.

Update `docs/api/openapi.yaml`, new `docs/api/identity-access.md`, and `docs/api/CHANGELOG.md` together in implementation tasks, with cookie security schemes and per-operation security. Existing Task/voice/health behavior remains compatible; do not attach global mandatory auth. This planning task does not publish unimplemented routes in the active OpenAPI.

## 3. Data Model & Schema Changes

### Concepts, identity and consistency

AdministrativeAccount is an aggregate root with stable UUID, role and status; administrative sessions are independently identified access records referencing it. Account identity must never be recycled or derived from email/session. Candidate is a separate identity aggregate with no password or administrative role. HR can reference one candidate from several assignments. Candidate contact changes must not rewrite assignment-bound historical input snapshots owned by Interview.

Invitation is an entity whose replacement is serialized under the assignment consistency boundary. CandidateSession is a separate access entity bound to one assignment and source invitation. Issuance, exchange and assignment revocation must coordinate with the assignment owner; they cannot be made safe by independent table CRUD. Assignment keeps at most one execution under the platform contract. No access operation creates, restarts or substitutes that execution.

Role, AccountStatus, SessionLifetime and immutable assignment-access facts have value semantics. Session/Invitation domain predicates own `revoked_at == nil && now < expires_at`; administrative access also requires ACTIVE account. Trusted `Clock` supplies `now`, UTC truncated to milliseconds. Creation requires expiry after creation, and session expiry exactly creation + 2 hours; invitation expiry creation + 24 hours. Restoration validates stored identity, time order and references without resetting deadlines, copying all mutable pointers. Crypto/encoding remain infrastructure/application work; entities do not import crypto/JSON/HTTP.

### Approved five-table target

All IDs are UUID primary keys; timestamps are PostgreSQL `timestamptz`, canonical UTC milliseconds. All fields are NOT NULL unless explicitly nullable. FK removal uses RESTRICT; no soft deletion or cascade purge is selected.

| Table | Columns and constraints | Indexes / ownership |
|---|---|---|
| `administrative_accounts` | `id`; `name text`; `email text` unique canonical login; `password_hash text`; `role text CHECK IN ('ADMIN','HR')`; `status text CHECK IN ('ACTIVE','DISABLED')`; `created_by uuid NULL` self-FK for bootstrap exception; `created_at`, `updated_at` | Unique canonical email; stable subject. Normalization and public field bounds §6.1. Plaintext passwords never persisted. |
| `administrative_sessions` | `id`; `account_id uuid` FK accounts; `token_hash text` unique; `created_at`; `expires_at CHECK > created_at`; `revoked_at NULL` | Account FK index; expiry index for later cleanup. Current role/status read from account, not session snapshots. |
| `candidates` | `id`; `name text`; `email text` contact, deliberately not unique; `created_by uuid` FK accounts; `created_at`, `updated_at` | Creator FK index. HR-created; no role/password/login account. Matching is Q4, not an implicit email upsert. |
| `interview_invitations` | `id`; `assignment_id uuid` FK external assignments; `token_hash text` unique; `expires_at CHECK > created_at`; `revoked_at NULL`; `created_by uuid` FK accounts; `created_at` | `(assignment_id, id)` unique for same-assignment session FK; assignment index; partial unique `(assignment_id) WHERE revoked_at IS NULL` to enforce one current unrevoked invitation. An expired row is explicitly revoked during replacement. |
| `candidate_sessions` | `id`; `assignment_id uuid` FK external assignments; `invitation_id uuid`; `token_hash text` unique; `created_at`; `expires_at CHECK > created_at`; `revoked_at NULL` | Composite `(assignment_id, invitation_id)` FK to invitation `(assignment_id, id)`; assignment and expiry indexes. Candidate identity derives through assignment, not caller input or duplicated candidate FK. |

Use nonempty role/status and canonical digest checks in adapter schema; existing-row collision behavior follows §6.4; named DDL/indexes and mappings are proven in T4/T8. Creation-time creator-role authority is enforced in application, not a static FK which would break when an employee's role later changes. `revoked_at` is nullable and must not precede creation. Digests are 64-character lowercase SHA-256 hex for 32-byte CSPRNG tokens; each credential kind has its own generation/lookup namespace. Record IDs are public identifiers, not credentials.

**Count:** five access-owned tables. `interview_assignments` is a sixth table only when counting the separate Interview dependency; its full schema is not invented here. Required facts are stable assignment ID, candidate reference, pinned published revision, current access-revocation state, and serialized admission/revocation ownership. Consent, executions, provider sessions and agent configuration tables are excluded from the access count.

### Adapter, constraints and installation

Use GORM 1.25.12 CRUD with separate models and explicit maps/Select updates for `revoked_at`, status and zero/null fields. Do not let `Updates(struct)` omission semantics control revocation/reset; do not use unconstrained Save/upsert or automatic association writes. Bounded explicit joins for account/session and assignment/session resolution avoid N+1 and history loading. PostgreSQL enforces unique digests, email and composite ownership in addition to application checks. A small adapter-owned SQL statement may install the partial unique index if GORM migration cannot express it clearly; it is not a general switch to raw SQL. The [GORM update documentation](https://gorm.io/docs/update.html) explains omission behavior; T4/T8 must prove mappings against the pinned repository version.

Retain additive AutoMigrate under `infrastructure/runtime/resources.go` and current startup deadlines. Install accounts/sessions first (two tables) and candidate identity independently (one more). Install invitation/session tables only after the actual assignment schema/contract is available (two more). Do not mount unresolved candidate routes or install unenforceable references to fake assignment tables. Test empty and existing Task databases; no Task backfill. Unique bootstrap/account operations must handle concurrent processes.

Rollback unmounts affected routes and restores compatible code while retaining tables and historical links. Old binaries must start with additive tables present. Do not drop candidates, referenced accounts or published-command subjects. Purge/retention Q5 needs a separate approved policy before physical removal.

### Transaction boundaries and races

- Bootstrap: hash outside the database transaction; serialize initialization with a database mechanism under the bounded startup budget. Create the initial ADMIN only when initialization is applicable; never overwrite an existing user's password or role from environment on restart. Incompatible existing state fails safely, without logging credentials. No public create-ADMIN operation.
- HR provisioning: require current ADMIN, then transactionally recheck its session/account and insert HR. Unique canonical email resolves races. Commit failure returns no account success; retry after uncertainty may encounter an existing account, not automatically reset its password. Candidate creation similarly requires HR; name/email are not deduplication keys.
- Administrative login: resolve account, verify password outside row locks, then lock/recheck account eligibility and verified credential version/hash before inserting session. If credentials changed meanwhile, fail/retry authentication rather than issue against stale proof. On every protected action, verify current session/account. Logout atomically persists revocation; protected mutations recheck access inside their transaction. Already-admitted operations may complete if revocation commits afterward; no retroactive cancellation claim.
- Invitation issue/reissue: lock assignment first; check current HR and assignment eligibility; revoke prior unrevoked invitations, insert new invitation, and commit before revealing token. Rollback retains former invitation. Concurrent reissues serialize; a later issuance may invalidate an earlier returned link. Lost-response/retry delivery details are Q3; do not claim exactly-once token delivery.
- Exchange: lock assignment, then invitation, check recognized credential, expiry/revocation and assignment state, and insert candidate session atomically. Q3 controls whether another exchange may insert another session. Commit failure sends no session cookie. A random token collision is retried internally with a new random value in a fresh valid transaction, with a bounded attempt count; no aborted transaction is reused.
- Candidate session resolution: check its own expiry/revocation and current assignment access. Source invitation revocation/expiry is checked at exchange, not blindly reapplied to existing sessions: otherwise reissuing a link would cancel active access contrary to BR-037. The source FK records provenance, not perpetual authorization. Assignment revocation invalidates all access through a live assignment predicate and coordinates cancellation under Interview ownership.
- Interview admission/revocation: assignment lock precedes execution locks per platform §5. Access module must not acquire assignment after execution or independently cancel execution. Persistent revocation/expiry during an active execution is integrated by the Interview owner; request denial alone is not proof of cancellation.

### Rule ownership and proof plan

All proof below is planned, not observed test evidence.

| Material rule | Owner | Enforcement paths | Proof plan |
|---|---|---|---|
| BR-020 | Bootstrap + ProvisionHR application operations | Startup, HR creation; absence of signup | Bootstrap rerun/concurrency; unauthorized enrollment denied |
| BR-021 | Fixed access policy + current account resolver | HTTP/non-HTTP ProvisionHR | ADMIN allow; HR/candidate deny; no writes on denial |
| BR-022 | Fixed access policy | All config/report authorization, replay | Valid ADMIN login followed by 403 on config; report consumer proof remains separate |
| BR-023 | HR policy, shared company scope | Configuration create/read/edit/validate/publish | HR B reads/edits HR A instance subject to version checks |
| BR-024/025/026 | AdministrativeSession + account eligibility; application authority | Login, current session, logout, every protected operation | Exact expiry boundary; revoked/disabled denial; already-issued credentials do not freeze role |
| BR-027/028 | Authority adapter + ACP operations | Principal resolution, new session, replay | Same SubjectID across sessions; HR replay allowed, current denial before lookup |
| BR-029 | Separate principal/session resolvers | Administrative and candidate entry points | Cookie substitution rejected; candidate never gets admin principal |
| BR-030 | Candidate identity + Interview assignment owner | Candidate creation/read; future assignment creation | One candidate with two real assignments; no implicit email dedup |
| BR-031 | Assignment-scoped invitation/session predicates + composite FK | Issue, exchange, session resolve, every candidate resource query | Session A cannot read B even for same candidate; cross-assignment FK fails |
| BR-032/033 | Invitation, exchange transaction, credential adapter | Exchange and restoration | Equality expiry, revoked/unknown token, assignment revoked, collision/rollback, distinct IDs |
| BR-034/035 | Access application operations; Interview owns initiation | Page load, exchange, reload, replacement session | No execution writes/model calls; valid reload reuses session |
| BR-036 | CandidateSession | Resolution and admission port | Exact 2-hour boundary; no sliding renewal |
| BR-037 | Assignment-serialized invitation replacement | Reissue, exchange old link, existing session resolve | Old invitation denied; active existing session not invalidated by reissue alone; rollback/concurrent issue |
| BR-038 | Live assignment access facts; Interview revocation coordinator | Exchange, protected reads, admission, active-run revocation | Revocation/exchange race; existing sessions denied; actual cancellation requires real Interview integration |
| BR-039 | Interview assignment/execution owner; access has no start operation | Reopen, reissue, initiation integration | Completed assignment not restarted; new interview uses a new assignment |

## 4. Component Breakdown

| Layer/component | Status | Responsibility |
|---|---|---|
| `domain/aggregate/access`, `domain/valueobject`, `domain/enum` | New | Account/candidate identity, role/status and immutable lifetime/scope predicates; no transport/crypto/ORM |
| `application/usecase/accessusecase`, `query/accessquery`, `dto` | New | Bootstrap/provision/login/logout/candidate/issue/exchange operations; trusted time, current authority, detached views |
| `application/repository/accessport`, `application/service` | New ports; Clock and ACP authority contract reused | Transactions, credential verification, session/assignment authority; no generic CRUD API |
| `infrastructure/repository/accessrepository` | New | Five separate models, mappings, locks, integrity constraints |
| `infrastructure/service/accesscredentials`, `accessauthority` | New | Password/token/CSRF implementation and real authority adapters; never accept client roles |
| `delivery/http/handler/accesshandler`, `delivery/http/middleware` | New | Strict bounded transport, cookies, origin/CSRF, metadata/error mapping through operations |
| `infrastructure/runtime`, `cmd/app` configuration | Modified | Additive schema, bootstrap configuration, shared resources and secured feature routing |
| `pkg/errs`, `translations`, `.mockery.yml`, tests and API docs | Modified/new at implementation | Classified localized errors, generated mocks, contracts and evidence |

Password hashing, token generation/digesting and CSRF use replaceable application ports because they cross the domain import boundary and need deterministic fault tests. Do not wrap pure role/lifetime predicates in I/O interfaces. Domain constructors/restorers own validity; repository restores corruption as technical failure. Handlers never decide roles on their own.

### Material structure decisions and assessment

| Choice | Reason and viable alternative/tradeoff | Authority and refinement boundary | Assessment outcome |
|---|---|---|---|
| Separate administrative and candidate identity/session models | A single users table with CANDIDATE role would introduce unneeded password/account lifecycle and ambiguous scope; separate records express approved invitation-only identity at the cost of two resolvers | Owner approval; BR-029..033; five tables binding target | Retained; shared crypto helpers allowed, shared authorization semantics not assumed |
| Fixed ADMIN/HR policy without role tables | Known roles have distinct purposes; configurable RBAC adds management rules/tables not requested | Owner role approval, BR-021..023 | Retained; combined roles/new permissions need business revision |
| Account/session roots plus assignment-serialized access changes | A whole-company aggregate would serialize unrelated users; independent invitation CRUD would race revocation and weaken assignment ownership | BR-026/031/037/038 and platform lock order | Retained; transaction coordination across module ports is required, not a new giant aggregate |
| Opaque persisted sessions rather than self-contained JWT authority | Current role/revocation must be read anyway; opaque credentials avoid stale authority claims and support restart, with one indexed database lookup per action | BR-024..029/036; internal adapter choice | Retained; no new Redis dependency |
| Separate Candidate aggregate and contact value | Person identity survives assignments, while email-based upsert would invent matching policy | BR-030, Q4 | Identity retained; contact normalization/matching deferred to Q4 |
| Existing ACP authority port | It already separates trusted principal and operation/resource authorization; replacing it adds reconciliation without needed behavior | Verified interface, ACP T2; SubjectID stable account UUID | Retained; no type rename/code repair required; real adapter is net-new |
| Three independent tables before assignment-dependent pair | Fake assignment storage would make a false integration claim; blocking admin behind interview would unnecessarily delay Q2 | Approved five-table final scope | Retained phased installation; T8/T11 depend on Q2/Q3 |
| Argon2id passwords, random opaque token digests | Passwords need adaptive hashes; high-entropy tokens can use fast digests. Using one hashing policy for both adds cost or weakens passwords | Internal crypto choice; external references below | Retained engineering proposal; benchmark/verification in T5 |
| Handler-local envelope parity initially | Task helpers are private; broad transport extraction expands scope | Existing envelope is binding, helper placement internal | Retained; extraction allowed only with Task/voice regressions |

Assessment gate: every approved rule has an owner/path/proof plan. Current role, lifetime, scope and table boundaries are retained with explicit alternatives. Q1 is resolved in §6; Q2–Q5 remain deferred with affected tasks in §9. No code behavior is asserted from diagrams or mock availability.

### Security adapter choices

Generate independent 32-byte CSPRNG invitation and session tokens; encode base64url without padding, persist SHA-256 digest only. Never use UUID, candidate ID or assignment ID as a credential. Digest/token namespaces and cookie names separate administrative from candidate access. Use Argon2id with individually random salt and encoded parameters; start assessment at OWASP's minimum 19 MiB, two iterations, parallelism one, then benchmark under bounded concurrency and cancellation before exposure. Malformed/unreasonably costly stored hashes fail safely without unbounded allocation. Unknown-account login follows equivalent verification work and generic rejection. Password acceptance and initial delivery follow §6.1; no email sender or forced-reset columns are invented.

Cookies are Secure, HttpOnly, SameSite=Lax, host-only with Path=/ and distinct `__Host-` names on HTTPS deployments. A cookie is a credential; session ID in JSON is metadata only. SameSite alone is insufficient: unsafe browser operations require exact configured Origin plus a session-bound CSRF header where authenticated. Select a signed HMAC CSRF value bound to credential kind and current session, derived using environment-supplied server secret; do not expose that secret or add a sixth table. Login/exchange use strict same-origin JSON/custom-header handling before a session exists. No permissive credentialed CORS. Bound login/exchange rate and hash concurrency before crypto work; administrative budgets and failure precedence are §6.2 and §6.5, not inherited from Task. Candidate exchange rate remains Q3-adjacent and uses the same hash-concurrency gate. Cookies/CSRF values and invitation/session credentials are excluded from logs and cacheable responses.

A link is a bearer credential, not independent proof that the holder is the named person. Candidate verification beyond possession remains Q4. Candidate frontend must remove the link token from visible URL/history before loading third-party resources, use no-referrer/no-store behavior, and avoid state-changing exchange on GET (including link preview requests). Q3 must settle repeated exchange/recovery before this frontend is mounted.

Technical references consulted for these adapter choices: [OWASP password storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html), [OWASP session management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html), and [OWASP CSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html). These are security baselines, not evidence that this repository implements them. The pinned Go/crypto/GORM APIs must be checked during implementation.

## 5. Flow Sequence

### Administrative access and Q2

1. Bounded startup installs administrative models, bootstraps initial ADMIN when applicable, and wires real ports. No secrets are logged.
2. Login applies origin/rate/input checks, verifies credentials, rechecks account eligibility and commits a new two-hour session before setting cookie.
3. ADMIN provisions HR through an operation that fixes the new role to HR; HR cannot provision accounts.
4. HR login resolves a stable account principal. ACP calls current authorization before any read, mutation or replay disclosure. ADMIN receives forbidden; candidate cannot resolve an administrative principal.
5. Logout persists revocation and clears the cookie. Later requests fail. Existing Task/voice routing and shutdown budgets remain unchanged.
6. Real authorizer/session evidence supplies the access part of ACP Q2; ACP T8/T9 must still prove actual secured configuration endpoints.

### Invitation access (after external assignment readiness)

1. HR selects/creates a candidate, then Interview prepares an assignment referencing that candidate and a pinned published revision.
2. HR requests invitation issuance; assignment lock coordinates replacement and access eligibility. Only the new credential is returned after commit.
3. Candidate browser opens the link without starting execution. Explicit POST exchange checks origin, invitation and assignment; it commits a scoped session before setting the candidate cookie.
4. Candidate page obtains safe session/assignment context. Reload uses the current cookie. Consent and sufficient validity for the execution deadline are checked by Interview before initiation.
5. Reissue blocks new exchanges using the old invitation, while existing session access remains governed by its session/assignment. Assignment revocation blocks existing and new access and coordinates execution cancellation.
6. A further interview for the same person creates a new assignment, not a reset of this one.

## 6. Request / Response

Accepted for `CORE-IAM-001` (T1 sections A–E) on 2026-11-01. Use the established JSON envelope, strict fields and UTC millisecond response timestamps. No password hashes, token digests, salary policy, HR reports or database details in responses. No session credential in response JSON; only HttpOnly cookies. Invitation issuance remains the deliberate exception for a new invitation credential shown once to authorized HR; never return it on read. Candidate-specific accepted inputs and exchange reuse stay Q3/Q4.

### 6.1 Administrative identity, password and bootstrap (approved A)

Product policy is constrained by approved non-goals: no mail delivery, no public signup, no account-recovery UI, and no forced-reset columns. Initial HR passwords are chosen by ADMIN at provisioning and delivered out of band. Bootstrap uses environment credentials only for the first applicable ADMIN. The service never emails, SMS-sends or prints passwords.

**Name:** trim outer Unicode whitespace (Zs plus ASCII space/tab); 1–120 Unicode code points afterwards; reject C0/C1/DEL controls. Store and return the trimmed value.

**Email:** trim the same outer whitespace, then Unicode lowercase the entire address. Require exactly one `@`, nonempty local and domain parts, LDH domain labels, total 3–254 ASCII characters after canonicalization, local part 1–64. Reject display names, spaces, consecutive dots and IDN/punycode in v1. Store and return only the canonical form. Login canonicalizes email the same way. Do not strip `+` tags. Candidate contact matching remains Q4.

**Password acceptance (bootstrap and HR provision):** 8–64 Unicode code points; never trim or Unicode-normalize; reject C0/C1/DEL; no composition, dictionary or breach checks in v1. Login treats password as an opaque secret: never trim; empty or missing is `INVALID_INPUT`; length other than 1–64 is `INVALID_INPUT`; otherwise verify exactly against the stored hash. Disabled, unknown and wrong-password logins share one generic authentication failure after dummy verification work.

**Bootstrap environment (validated at startup; never logged):** `ACCESS_BOOTSTRAP_ADMIN_NAME`, `ACCESS_BOOTSTRAP_ADMIN_EMAIL`, `ACCESS_BOOTSTRAP_ADMIN_PASSWORD` must all be present and satisfy the field rules above. `ACCESS_HTTP_ORIGIN` is an exact absolute origin (`https://host[:port]` or `http://localhost[:port]`), no path/query/trailing slash. `ACCESS_CSRF_SECRET` is ≥32 bytes of high-entropy material. Missing/invalid configuration fails startup without creating accounts.

### 6.2 Cookie, origin, CSRF and rate budgets (approved B)

HTTPS administrative cookie name `__Host-ia-admin` (Secure, HttpOnly, SameSite=Lax, Path=/, host-only, no Domain). HTTP localhost may use `ia-admin` with the same flags except `__Host-`/Secure. Candidate names `__Host-ia-candidate` / `ia-candidate` are reserved and never accepted on administrative routes. Session ID in JSON is metadata; the cookie is the credential. All access responses send `Cache-Control: no-store`.

Every administrative JSON request requires `Origin` equal to `ACCESS_HTTP_ORIGIN` and `Content-Type: application/json` on requests with bodies. No credentialed CORS. Login has no CSRF header. Authenticated unsafe methods require header `X-CSRF-Token` matching an HMAC-SHA256 value bound to credential kind `administrative` and the current session ID, keyed by `ACCESS_CSRF_SECRET`. GET `/admin/session` returns that value as `csrfToken` after authentication. CSRF rotation invalidates outstanding tokens; persisted sessions remain valid.

Login rate limit, before Argon2id: 5 attempts per canonical email per 15 minutes and 20 per client IP per 15 minutes. Exceeded returns 429 `IA_RATE_LIMITED` with `Retry-After` in seconds (maximum 900). Process-wide Argon2id concurrency is 2; waiters respect request cancellation. Candidate exchange later reuses the hash gate; its request-rate policy stays with Q3. Provision/logout are not login-rate-limited.

### 6.3 Administrative operation schemas (approved C)

Request bodies are limited to 1 MiB. Unknown fields, duplicate keys, non-objects, trailing documents, nulls where not allowed, and system-owned fields (`id`, `role`, `status`, `createdBy`, timestamps, session/cookie/CSRF values) are `INVALID_INPUT`. Success wraps the `data` objects below; listed members are required.

| Operation | Exact request | Response `data` | Status |
|---|---|---|---|
| POST `/admin/sessions` | exactly `email`, `password` | `account` `{id,name,email,role}` plus `session` `{id,expiresAt}`; Set-Cookie after commit | 201 |
| GET `/admin/session` | cookie only | `account` `{id,name,email,role,status}`, `session` `{id,expiresAt}`, `csrfToken` | 200 |
| DELETE `/admin/sessions/current` | cookie + `X-CSRF-Token`; empty object or absent body | `{}` after persisted revocation, then expired Set-Cookie | 200 |
| POST `/admin/hr-accounts` | exactly `name`, `email`, `password`; CSRF | `id`, `name`, `email`, `role` (`HR`), `status` (`ACTIVE`), `createdAt`, `updatedAt`; no password | 201 |

### 6.4 Duplicate accounts and existing bootstrap state (approved D)

Canonical administrative email is unique across all roles. Duplicate provision, including races and retry after uncertain commit, returns 409 `IA_ACCOUNT_CONFLICT` without existing IDs, roles or password reset.

Bootstrap: hash outside the DB transaction; serialize initialization. If no `administrative_accounts` rows exist, insert ADMIN with `created_by` null. If any row exists and canonical env email matches an ACTIVE ADMIN, skip and do not overwrite password, name or role. If the matching row is HR, DISABLED, or otherwise incompatible, fail startup. If other accounts exist and the env email is absent, skip (already initialized); do not create a second bootstrap ADMIN. Concurrent first inserts collapse to one ADMIN via uniqueness.

### 6.5 Public errors and precedence (approved E)

Planned codes for later `pkg/errs` and EN/ID catalogs, not T1 runtime additions: `IA_UNAUTHENTICATED` 401; `IA_FORBIDDEN` 403; `INVALID_INPUT` 400; `NOT_FOUND` 404; `IA_ACCOUNT_CONFLICT` 409; `IA_INVITATION_UNAVAILABLE` 410; `IA_RATE_LIMITED` 429; `IA_DEPENDENCY_UNAVAILABLE` 503; `INTERNAL_ERROR` 500.

| Priority | Condition | HTTP / code |
|---|---|---|
| 1 | Method, unknown route, shutdown admission | existing 405/404/503 `SHUTTING_DOWN` |
| 2 | Origin, content-type, size, JSON, unknown fields | 400 `INVALID_INPUT` |
| 3 | Login rate budget exceeded | 429 `IA_RATE_LIMITED` |
| 4 | Missing/invalid/expired/revoked administrative session, including candidate-cookie substitution | 401 `IA_UNAUTHENTICATED` |
| 5 | Invalid CSRF on an authenticated administrative request | 403 `IA_FORBIDDEN` |
| 6 | Valid session, insufficient role | 403 `IA_FORBIDDEN` |
| 7 | Field policy (name/email/password bounds) on reached input | 400 `INVALID_INPUT` |
| 8 | Duplicate canonical administrative email | 409 `IA_ACCOUNT_CONFLICT` |
| 9 | Login proof failure (unknown, disabled, mismatch, stale hash) | 401 `IA_UNAUTHENTICATED` |
| — | Missing/expired/revoked/malformed invitation without disclosing assignment | 410 `IA_INVITATION_UNAVAILABLE` |
| — | Store unavailable before known success, including logout/login commit uncertainty | 503 `IA_DEPENDENCY_UNAVAILABLE` |
| — | Corrupt storage or unexpected technical error | 500 `INTERNAL_ERROR` |

Order applies to checks actually reached. Login evaluates 1–3 and field presence before proof; it never returns 409. Protected operations authenticate then authorize before target disclosure. Technical failures terminate the current stage. Set no cookie or success payload before confirmed session commit. Logout commit uncertainty clears the local cookie defensively and returns 503 without claiming revocation succeeded. Indistinguishable invitation rejection is uniform 410; Q3 still owns exchange reuse and lost-response recovery. ACP routes keep ACP §6.5 codes.

#### Examples (accepted expectations)

Login request before canonicalization:

```json
{"email":"Sinta@Example.com","password":"correct-horse"}
```

Login success `data` (cookie omitted):

```json
{"account":{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"Sinta Wijaya","email":"sinta@example.com","role":"HR"},"session":{"id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","expiresAt":"2026-11-01T10:00:00.000Z"}}
```

HR provision request:

```json
{"name":" Sinta Wijaya ","email":"Sinta@Example.com","password":"correct-horse"}
```

HR provision success `data`:

```json
{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"Sinta Wijaya","email":"sinta@example.com","role":"HR","status":"ACTIVE","createdAt":"2026-11-01T08:00:00.000Z","updatedAt":"2026-11-01T08:00:00.000Z"}
```

Current session `data`:

```json
{"account":{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"Sinta Wijaya","email":"sinta@example.com","role":"HR","status":"ACTIVE"},"session":{"id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","expiresAt":"2026-11-01T10:00:00.000Z"},"csrfToken":"signed-csrf-material"}
```

Rejected cases: `Sinta@Example.com` provisioned twice → 409; env bootstrap on restart against the same ACTIVE ADMIN → no overwrite; matching env email on an HR row → startup failure; login `Sinta@Example.com` / wrong password → 401 with no cookie; origin mismatch → 400.

### Creation, restoration, and transition contracts

| Path | Accepted states / normalization | Errors/order and timing | Effects / compatibility | Proof plan |
|---|---|---|---|---|
| Bootstrap/provision | Stable new ID; fixed role; canonical email and accepted password §6.1; bootstrap creator null only | §6.5; unique conflict before success | Hash only; no reset on startup/retry | Concurrency, collision, rerun, no credential leakage |
| Login | Active ADMIN/HR credentials; password never trimmed/normalized; email §6.1 | Origin/rate/input then generic authentication result; store failure technical | New session and fresh token only after commit | Unknown/disabled/wrong credentials; failure/cancel; equality expiry |
| Restore identity/session | Recognized role/status/IDs; valid stored time order and references | Corruption technical; never silently convert unknown role into HR | No repair, clock reset or credential reissue on read | Round trip and corrupt persisted data |
| Current authority / logout | Current matching session/account; `now < expires_at`; unrevoked | Authentication then operation authorization before target disclosure; §6.5 | Logout revokes current session only; no role from client | Candidate cookie substitution, disabled account, replay, no side effects on denial |
| Candidate creation / assignment reuse | Existing identity can be referenced repeatedly; email not unique/upsert | Current HR before candidate disclosure; contact bounds Q4 | New candidate distinct from assignment; no historical-input rewrite | Same candidate two assignments; same email does not merge people |
| Invitation issue/reissue | Existing eligible assignment; trusted 24h expiry; prior invitation revoked | HR then assignment ownership/existence, serialize on assignment | Old link invalid; existing sessions/execution not automatically cancelled | Concurrent reissue, rollback, lost response Q3 |
| Invitation exchange | Exact secret proof, live invitation and assignment, Q3 reuse policy | Reject missing/invalid/expired/revoked proof as uniform 410 before session creation; technical failure distinct | New 2h access session only; no consent/execution/model call | Exact expiry, token collision, races, ambiguous response Q3 |
| Candidate session resolve | Live scoped session plus accessible assignment | Outside-session resource hidden as 404; unauthenticated as 401 | No new session/expiry extension on GET; invitation provenance not ongoing validity | Cross-assignment access, reload, source invitation reissued |
| Assignment revoke / admission | Actual Interview contract; lock assignment before execution | Serialize before acceptance; no admission from stale access facts | Live access removed; execution cancellation belongs to Interview | Real consumer race and cancellation integration Q2 |

## 7. Error Handling

Public access codes and overlap order are §6.5. These planned codes are not runtime catalog additions yet. Q3 still settles exchange reuse conflicts and commit-uncertain recovery before candidate exposure.

Authentication endpoints validate transport/origin/rate before proof; protected operations authenticate then authorize before target lookup/disclosure. Technical failures terminate the reached stage and are mapped before nested domain errors; do not mask store outage as bad password or assume failed commit is rollback.

For ACP routes preserve its already approved §6.5 order and codes: absent/invalid/candidate session → 401 `ACP_UNAUTHENTICATED`; valid ADMIN or invalid CSRF → 403 `ACP_FORBIDDEN`; outside HR scope → 404; replay/current-draft ordering unchanged. The shared adapter returns classified failures; each feature's delivery mapping owns its public catalog.

## 8. Constraints Checklist

These are implementation verification gates, not completed checks.

- [ ] BR-CORE-020..039 covered through public behavior at the owners above.
- [ ] Exactly five access tables; no invented assignment table, role tables, tenant tables or candidate password.
- [ ] ADMIN and HR remain distinct; stable account SubjectID, current HR authorization and shared company scope proven.
- [ ] Cookie/session/CSRF/credential protections proven with real adapter and browser-compatible HTTP tests.
- [ ] Candidate session scope, expiry, source provenance and assignment revocation proven with real assignment owner.
- [ ] Access exchange/reload has no consent/execution/provider side effects.
- [ ] Project architecture, declaration documentation, generated mocks, additive AutoMigrate and lifecycle budgets preserved.
- [ ] Active OpenAPI, endpoint docs, changelog, codes and EN/ID messages synchronized when implemented.
- [ ] No test/database acceptance, historical RED/GREEN, or Q2 completion inferred from this document.

## 9. Risks & Open Questions

| ID | Decision / owner | Affected tasks | Ready when |
|---|---|---|---|
| Q1 | **Resolved 2026-11-01:** T1 A–E; out-of-band ADMIN-chosen HR passwords; canonical email; bootstrap existing-state; cookies/CSRF/rate; error order in §6 | T1 Done; contract prerequisite for T4/T6/T8 | Approval recorded in decision-log D7; no mailer or recovery UI; implementation remains pending |
| Q2 | Interview owner: actual assignment table/access/locking/revocation/admission integration is absent | T8, T11, T12; full BR-030/038/039 proof | Real assignment persistence and transaction collaboration available; no in-memory fake counted as integration; consent/execution owner handles active cancellation |
| Q3 | Product/security: invitation reuse vs one-time exchange, concurrent/replacement candidate sessions, lost-response recovery; current five-column groups contain no consumed marker | T8 invitation final mapping, T11–T12 | Chosen policy is synchronized with schema/transactions, retries and browser tests; if extra columns are needed, record explicitly rather than silently inventing them |
| Q4 | Candidate-data/product owner: contact normalization, identity matching and independent verification expectations | T6 candidate fields, T8 candidate mapping, T11–T12 | Candidate registration/reuse contract approved without email-based identity guesses; bearer-link assurance stated |
| Q5 | Product/data owner: retention and FK-safe purge of contacts, credentials and assignment-bound references | T13 cleanup only | Explicit retention/purge ordering and rollback/data-loss consequences accepted; no physical purge before then |

Administrative foundation and ACP Q2 access delivery do not depend on candidate Q2/Q3/Q4 decisions; use the two-table administrative phase. Q1 is resolved. Proposed account disable/reset APIs remain outside scope, although current DISABLED state must always be denied. Multiple identities for an ADMIN who also needs HR duties are the consequence of the approved single-role model; combined-role access requires a later business decision.

## 10. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-01 | Initial Mode A capability-scoped assessed design; five-table target and staged readiness | Owner approved scope, properties, capability name and semantic BR blocks; satisfy ACP's real access dependency without inventing interview implementation |
| v1.1 | 2026-11-01 | Incorporate T1 A–E administrative contracts and resolve Q1 | Executable HTTP/schema/bootstrap contract for T4/T6/T8 without runtime implementation |
