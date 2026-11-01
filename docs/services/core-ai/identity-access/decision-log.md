# Decision Log: Identity and Access

Technical decisions driven by the owner's business choices, approved in this conversation on 2026-11-01. Entries explain intended product behavior; none establishes implementation or test acceptance. Technical library and helper choices remain in technical-design.md.

T1 (`CORE-IAM-001`) A–E administrative contracts are recorded in D7. Q1 is resolved. T10 (`CORE-IAM-010`) F–I candidate contracts are recorded in D9; Q2–Q4 are resolved as access contracts. Q5 remains open. T21 (`CORE-IAM-021`) creator-scoped candidate listing is recorded in D10. D8 records the later owner-approved shift to manual migrations (Q6 resolved); its implementation is planned in T16–T20. T1–T9 Done statuses remain as recorded in task-plan.md; the migration review does not rerun their acceptance. Real Interview assignment persistence remains a T14 runtime obligation.

## D1: Account administration and HR operations have separate roles

- **Business driver:** The owner selected ADMIN for user management, initially adding HR, and HR for configuration and interview work.
- **Technical decision:** Administrative accounts have one fixed role, ADMIN or HR. ADMIN provisions HR but does not inherit configuration or internal report access. HR authority is checked for every configuration operation, including publication replay.
- **What this means for the product:** Logging in as ADMIN does not let someone publish agent settings. HR users cannot add other HR accounts. The `/admin` URL prefix is used by both roles and does not grant permissions.
- **Alternatives considered:** A superuser ADMIN would contradict the selected responsibilities; configurable or combined roles introduce access decisions that were not requested.
- **Trade-offs accepted:** A person needing both responsibilities would require separate identities under this first model. Reset, disable/reactivate and role-change interfaces require their own product decisions.
- **Revisit when:** The company requests combined duties, delegated account management, or additional business roles.
- **Status:** active; refines the earlier combined admin/HR wording in platform section 10 and ACP D4, as recorded by platform TD-21 and ACP D6.

## D2: HR colleagues share configuration within one company

- **Business driver:** The owner selected one company as the initial application target and approved shared HR configuration.
- **Technical decision:** Use shared company scope for permitted HR configuration operations. Do not add organization/workspace tables or creator-only restrictions. Preserve authorship separately from access authority.
- **What this means for the product:** One HR can continue another HR's configuration work; normal draft-version rules still prevent silent overwrites.
- **Alternatives considered:** Creator-only ownership would hinder handover; multiple-company membership and workspace management solve a later product need.
- **Trade-offs accepted:** All HR users in this installation share the approved configuration scope. Supporting several companies later will require explicit ownership, data migration and isolation rules.
- **Revisit when:** More than one company uses an installation, or departments need confidential configuration boundaries.
- **Status:** active.

## D3: Candidates use invitations and retain identity across interviews

- **Business driver:** The owner approved invitation-based entry without permanent candidate login accounts and confirmed the same person can interview repeatedly.
- **Technical decision:** Store candidate identity separately from administrative accounts. Each new interview assignment can reference the same candidate, with its own invitation and access scope.
- **What this means for the product:** Candidates do not need passwords or a permanent dashboard. An invitation opens only its assigned interview, even when the person has other assignments.
- **Alternatives considered:** A candidate password account creates signup/recovery responsibilities; a new person record for every invitation fragments interview history.
- **Trade-offs accepted:** The link proves possession rather than independently verifying the person's identity. Candidate matching, further verification and repeated exchange behavior still need explicit decisions.
- **Revisit when:** Candidates need a permanent dashboard, verified identity, or self-service interview history.
- **Status:** active.

## D4: Receiving access does not begin an interview

- **Business driver:** The owner approved a distinct candidate access session after link verification and access before consent/start.
- **Technical decision:** Keep assignment identity, candidate session identity and execution identity separate. Page loads, exchange, refresh and replacement access never create or restart execution.
- **What this means for the product:** A candidate can inspect the interview context first. Refreshing the page does not create a second interview; a new interview after completion uses a new assignment.
- **Alternatives considered:** Automatically starting on link access risks starting from a preview request and bypassing prerequisites; treating each login as a new interview duplicates history.
- **Trade-offs accepted:** The Interview capability must separately implement consent, initiation and recovery. Completing this access module alone does not deliver the interview feature.
- **Revisit when:** The start experience changes, while retaining explicit consent and stable execution history.
- **Status:** active.

## D5: Five access tables support two distinct access channels

- **Business driver:** The owner approved properties for administrative accounts/sessions, candidates, invitations and candidate sessions, with assignments owned separately by Interview.
- **Technical decision:** Plan exactly those five access tables and no candidate password, role-management or organization tables. Deliver the two administrative tables first; candidate invitation/session storage waits for real assignment references and unresolved exchange policy.
- **What this means for the product:** Secure HR configuration access can be delivered before candidate interview integration. The schema remains understandable without implying that candidate or assignment functionality already exists.
- **Alternatives considered:** A universal user/session record obscures distinct candidate authority; implementing a placeholder assignment table would create a misleading integration claim; blocking all access work behind Interview would delay HR unnecessarily.
- **Trade-offs accepted:** Candidate access requires a separate dependency handoff. Extra properties needed by a later exchange policy must be assessed and recorded explicitly rather than silently added under this approval.
- **Revisit when:** The company requests permanent candidate accounts, multiple companies or a different invitation lifecycle.
- **Status:** active.

## D6: Replacing an invitation is different from revoking an interview

- **Business driver:** Platform section 10 preserves an active interview during invitation replacement, while assignment revocation removes access and coordinates cancellation.
- **Technical decision:** Check invitation validity at exchange, then check candidate session and assignment validity for ongoing access. Reissue revokes the old invitation credential, not automatically every session originating from it.
- **What this means for the product:** HR can replace an entry link without unexpectedly interrupting an interview already underway. Revoking the assignment remains the action that removes ongoing interview access.
- **Alternatives considered:** Checking the source invitation's revoked flag on every session action would interrupt active sessions on reissue; ignoring assignment revocation would leave withdrawn access usable.
- **Trade-offs accepted:** Invitation provenance and current session authority have different lifecycles. Actual execution cancellation needs the Interview capability and cannot be claimed from request denial alone.
- **Revisit when:** Product requests that replacing an invitation also eject current devices, with explicit cancellation and recovery behavior.
- **Status:** active.

## D7: Administrative input, provisioning and security contracts are approved

- **Ticket:** `CORE-IAM-001` / T1.
- **Confirming actor/date:** user in this conversation, 2026-11-01, requesting exclusive T1 execution and a Done checklist; no separate external sign-off is claimed.
- **Business driver:** BR-CORE-020..028 and platform TD-21 require bootstrap ADMIN, ADMIN-only HR provisioning, hashed passwords, unique login identity, revocable sessions and CSRF/origin protection without public signup or mail infrastructure.
- **Approved contract:** sections A–E now canonical in [technical-design §6](technical-design.md#6-request--response); [approval record](t1-contract.md) maps each section.
- **Product consequences:** Initial HR passwords are chosen by ADMIN at provisioning and delivered out-of-band. Administrative email is unique after trim and Unicode lowercase. Bootstrap never overwrites an existing ACTIVE ADMIN and fails on incompatible existing state. No mailer, recovery UI or first-login reset column.
- **Engineering/security consequences:** `__Host-ia-admin` / `ia-admin` cookies; `X-CSRF-Token`; login origin + JSON pre-session protection; 5/email and 20/IP per 15 minutes before Argon2id; process hash concurrency 2; error order in §6.5; uniform 410 for indistinguishable invitation rejection.
- **Alternatives considered:** inventing an email-delivery or forced-reset flow contradicts approved non-goals; leaving Q1 open blocks T4/T6/T8; inheriting Task rate limits would under-specify credential hashing.
- **Trade-offs accepted:** operators must communicate first passwords; v1 email is LDH ASCII only; login of a DISABLED account is indistinguishable from unknown credentials; CSRF key rotation invalidates outstanding tokens without dropping persisted sessions.
- **Assessment:** retain account/session aggregates and application-owned uniqueness/bootstrap; delivery owns cookies/origin/CSRF/rate; T4 maps canonical email; T5/T8 enforce crypto and HTTP. No Go/SQL/HTTP implementation is claimed.
- **Closure (historical T1 scope):** Q1 resolved; technical-design/tactical-strategy/task-plan/capability synchronized and T1 Done. T2–T15 were not executed at that closure; Q2–Q5 remained. Later T2–T9 completion is preserved in task-plan.md; D8 adds migration follow-up without reopening T1.
- **Revisit when:** product requests mail delivery, self-service reset, IDN emails, or combined-role accounts; approve and synchronize before dependent implementation.
- **Status:** active.

## D8: Database changes must be reviewed and applied before the application starts

- **Business driver:** On 2026-11-01 the owner stated that the project has grown and explicitly requested changing from AutoMigrate to manual migrations. The intended operational change is deliberate, reviewable database evolution as more capabilities share the service.
- **Technical decision:** Plan ordered SQL files and a separate command to apply them before starting the application. Cover both the existing Task table and the two administrative tables in one service-wide sequence; future candidate and Interview changes use the same sequence. The application checks that the database is compatible and stops safely if it is unprepared.
- **What this means for the product:** Existing Task history, account identities and live session validity must survive the transition. Operators gain an explicit database preparation step. An unprepared database no longer becomes ready merely by starting the service.
- **Alternatives considered:** Retaining startup AutoMigrate would leave the schema change attached to every application start; keeping unused SQL files alongside it would leave two competing ways to change the database. Recreating the database would discard existing history and is not the adoption plan.
- **Trade-offs accepted:** Implementation must establish the real existing schema, preserve data, rehearse backup/recovery and change local/test startup procedures. Older binaries that still run AutoMigrate are not routine rollback targets. The new command is not permission to modify an unspecified live database.
- **Approval boundary:** The owner approved the schema-policy change and requested its plan; no further policy confirmation is required. Tooling details and artifact paths are assessed engineering choices in technical-design §3.1–3.4, not additional product decisions. No migration has been applied by this revision.
- **Prior decision preserved:** D5's five access business tables and two-phase delivery remain active. Tool-owned migration metadata is separate. D7/T1 password, bootstrap and security contracts remain active; schema migration never inserts credentials or resets an existing account.
- **Revisit when:** Database ownership splits into separate services, release compatibility needs a broader rollout window, or operational recovery requirements change.
- **Status:** active; supersedes the former AutoMigrate schema policy from the original boilerplate requirement/design, not D1–D7 business decisions.

## D9: Candidate contact, exchange reuse and assignment collaboration contracts are approved

- **Ticket:** `CORE-IAM-010` / T10.
- **Confirming actor/date:** user in this conversation, 2026-11-01, requesting exclusive T10–T12 execution and a Done checklist; no separate external sign-off is claimed.
- **Business driver:** BR-CORE-029..039 require reusable candidate identity, assignment-scoped invitations, access before consent/start, and assignment-first revocation without inventing Interview storage.
- **Approved contract:** sections F–I now canonical in [technical-design §6.6–6.9](technical-design.md#66-candidate-contact-and-matching-approved-f); [approval record](t10-contract.md) maps each section.
- **Product consequences:** Candidate email uses administrative canonicalization and stays nonunique contact. A valid invitation may be exchanged repeatedly until expiry or revocation; each exchange may create another two-hour session. Lost HTTP responses are not exactly-once cookie recovery. The bearer link proves possession, not independently verified identity.
- **Engineering/security consequences:** Access never creates `interview_assignments`. Invitation/session FKs wait for Interview's migration. Assignment lock precedes access writes. Terminal completion does not deny access and does not start or cancel execution. Five-table properties stand; extra `consumed_at` was assessed and rejected.
- **Alternatives considered:** one-time consumed invitations would add a column contrary to D5; email upsert would invent matching; a placeholder assignment table would create a false integration claim.
- **Trade-offs accepted:** retry after a lost exchange may create a second session; operators must keep invitation links confidential; T11 stage B and T14 still need the real Interview table and consumers.
- **Assessment:** retain Candidate/Invitation/CandidateSession ownership in access; keep assignment facts as an Interview-owned port; T12 implements operations against mocks; T11 sqlmock covers Access SQL only.
- **Closure:** Q2/Q3/Q4 resolved as executable access contracts; Q5 retention remains open; no candidate HTTP or Interview runtime is claimed.
- **Revisit when:** product requests one-time invitations, email-based matching, device ejection on reissue, or Access-owned assignment storage.
- **Status:** active.

## D10: An HR lists only the candidates they created

- **Ticket:** `CORE-IAM-021` / T21.
- **Confirming actor/date:** approved HR list plan, 2026-11-02. No separate external sign-off is claimed.
- **Business driver:** HR needs the people they recorded, without email search and without another HR's candidates appearing in that collection.
- **Technical decision:** GET `/admin/candidates` uses the same origin and administrative-cookie rules as GET `/admin/candidates/{id}`, with no CSRF. Rows are those whose `created_by` equals the logged-in account, ordered by `created_at` descending then `id` descending. `data` is an array of `id`, `name`, `email`, `createdAt`, and `updatedAt`. An empty result is 200 with `[]`. ADMIN is 403 `IA_FORBIDDEN` before the read. A missing or candidate-cookie session is 401 `IA_UNAUTHENTICATED`. GET by id still returns any known candidate to any current HR. POST create is unchanged. Other methods on the exact path stay 405.
- **What this means for the product:** Each HR sees their own candidate list. Knowing an id is still enough for any current HR to open that candidate. Email does not find or merge people.
- **Alternatives considered:** Email search would invent matching that D9 forbids. Listing every candidate would ignore the existing creator column. Adding pagination or a new ownership column was not requested.
- **Trade-offs accepted:** The collection cannot browse another HR's candidates. A known id remains readable by any current HR. There is no page size or search box on this route.
- **Revisit when:** product requests cross-HR candidate browse, email lookup, or paged results.
- **Status:** active.
