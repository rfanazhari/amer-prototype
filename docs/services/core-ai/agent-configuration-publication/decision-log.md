# Decision Log: Agent Configuration and Publication

These entries explain technical consequences of the approved capability and platform decisions. They do not claim implementation or new owner approval of unresolved details. Technical library and package choices remain in [technical-design.md](technical-design.md).

T1 (`CORE-ACP-001`) A–E approval is recorded in D5 below. Q1/Q3 are resolved. The administrative access prerequisite is satisfied by Identity and Access D1/D6/D7 and CORE-IAM-009; configuration-route proof remains ACP T8/T9. Schema installation follows Identity and Access D8, recorded here as D7. T10 (`CORE-ACP-010`) records historical deployment compatibility in D8 and resolves Q4. Q5 remains open.

## D1: Published settings remain reliable while users prepare changes

- **Business driver:** BR-CORE-005/009/013 and the approved platform configuration lifecycle require stable published settings and unchanged existing assignments/executions.
- **Technical decision:** Keep an editable working draft separate from permanent published versions. New publication changes the default selection; it does not edit earlier versions.
- **What this means for the product:** HR can prepare different questions without changing an interview assignment already prepared under earlier settings.
- **Alternatives considered:** Updating one shared configuration in place is simpler to store, but would change the meaning of existing references and violate the accepted behavior.
- **Trade-offs accepted:** More stored versions and an obligation to keep their meaning supported. The exact deployment policy remains open.
- **Revisit when:** Product introduces an explicit migration or retirement policy for previously published versions; existing commitments must remain accounted for.
- **Status:** active

## D2: Retrying publication cannot silently create or reactivate another version

- **Business driver:** BR-CORE-010/011/012 requires one consistent publication outcome despite repeated clicks, lost responses, or conflicting requests.
- **Technical decision:** Save the request identity and original outcome together with the new version and active selection. A matching retry reads that saved outcome; it does not publish again.
- **What this means for the product:** A user can safely retry after a timeout without creating extra versions or replacing a newer active selection with an older one.
- **Alternatives considered:** Disabling the publish button reduces repeated clicks but cannot handle a lost response or requests from another client. Keeping request results only in memory loses protection on restart.
- **Trade-offs accepted:** Persistent request records and careful handling of concurrent attempts; the public request-key contract is finalized in design §6.4 through D5.
- **Revisit when:** Request-record retention is introduced or publication semantics change; retained versions and safe retries must remain consistent.
- **Status:** active

## D3: Publication verifies settings, not the availability of a model provider

- **Business driver:** BR-CORE-006/014 separates repeatable configuration checks from execution-time availability and authority.
- **Technical decision:** Validate supported fields, their relationships, and stable references without making a model call. Check current availability and execution permission when an execution is admitted or activated.
- **What this means for the product:** Users can prepare and publish valid agents during a temporary model-provider outage, but publishing does not guarantee an immediate run.
- **Alternatives considered:** Testing a live model during every publication could detect a current outage, but would add cost and variable results without guaranteeing later availability.
- **Trade-offs accepted:** Some execution failures can only be discovered when a run is requested. Product messaging must distinguish published settings from an active execution.
- **Revisit when:** A separately defined live-readiness or preview feature is requested; it must not silently replace deterministic publication validation.
- **Status:** active

## D4: Interview access does not confer configuration authority

- **Business driver:** BR-CORE-003 and the platform access decision restrict configuration management to authorized HR/admin actors.
- **Technical decision:** Require trusted administrative authority in every configuration operation, including reads and repeated publication requests. Mount routes only when the real admin access dependency is available.
- **What this means for the product:** A candidate cannot change interview questions or retrieve administrative configuration using an interview session.
- **Alternatives considered:** Hiding buttons or relying on an admin URL alone would not protect direct requests. A temporary unrestricted endpoint would break the business rule even for a synthetic-data demonstration.
- **Trade-offs accepted:** Endpoint delivery depends on admin-session work that is not yet implemented in this repository; domain and persistence work can proceed independently.
- **Revisit when:** Product introduces more administrative roles or delegated configuration access, with explicit scope and disclosure rules.
- **Status:** superseded by D6 (2026-11-01, owner separated ADMIN account management from HR configuration authority)

## D5: T1 template, publication, and public error contracts are approved

- **Ticket:** `CORE-ACP-001` / T1.
- **Confirming actor/date:** user in this conversation, 2026-11-01; no separate external sign-off is claimed.
- **Exact confirmation:** “approve. - Setujui bagian A–E sebagai kontrak T1”.
- **Business driver:** BR-CORE-001/002/006/007/008/011/012 require concrete supported choices, understandable validation and repeatable publication outcomes before implementation.
- **Approved contract:** all sections A–E, without requested revision, now canonical in [technical-design §6](technical-design.md#6-request--response); [approval record](t1-contract-proposal.md) maps each section.
- **Product consequences:** editable incomplete drafts; concrete version-1 HR and Document schemas; HR hard 300–600 second total with 30-second question capacity and spare follow-up budget; salary reference remains write-only; Document selects from the five predefined fields. Those exact limits refine earlier source targets through this approval.
- **Warning decision (Q3):** informational warnings need no acknowledgment; only v1 warning is `CURRENCY_NOT_REQUIRED` for total without currency; blocking errors still prevent publication.
- **Publication/API decision (Q1):** subject-scoped persistent keys; fixed normalized intent fingerprint; same-intent replay returns the original revision without changing active selection; different intent conflicts; validation returns 201 with findings; blocked publication returns 422. Exact normalization, statuses/codes and overlapping error precedence are in §6.1–6.5.
- **Alternatives considered:** leaving Q1/Q3 open prevents dependent implementation; enforcing all warnings as errors, guessing API rules, or using current mutable configuration for replay would change the approved behavior. All A–E choices were submitted for explicit review before acceptance.
- **Trade-offs accepted:** confidential policy references must be resubmitted for full replacement; creation is non-idempotent and names are not deduplication identities; no key expiry/purge is introduced; different admins have separate request scopes. Published records still need compatible historical interpretation under Q4.
- **Assessment:** retain instance-owned versions/selection, concrete template rules and application-owned authority/intent with atomic persistence. T2/T3 interfaces are unchanged; T5 must supply production configurations, severity and approved public bounds. No Go/SQL/HTTP implementation or mock test is claimed.
- **Closure:** Q1/Q3 resolved; technical-design/tactical-strategy/task-plan synchronized and T1 Done. At that closure T4–T11 were not executed and Q2/Q4/Q5 remained. Later alignment records the access prerequisite as satisfied and moves schema installation to D7; Q4/Q5 remain.
- **Revisit when:** product changes accepted template behavior, warning acknowledgment, normalization, confidentiality, identity retention/scope or error precedence; approve and synchronize before dependent implementation.
- **Status:** active.

## D6: HR owns configuration; ADMIN manages HR accounts

- **Business driver:** The owner explicitly selected ADMIN for user management and HR for configuration and interview work, then approved one-company scope and the five-table access proposal on 2026-11-01.
- **Technical decision:** Authenticate ADMIN and HR through administrative sessions, but permit configuration operations only for current HR authority. Keep candidate sessions separate. Use the stable account ID for publication identity across sessions.
- **What this means for the product:** HR colleagues share company configuration. ADMIN can add HR users but cannot inspect, edit or publish agent configuration merely by administering accounts. Interview invitations grant no administrative access.
- **Alternatives considered:** A superuser ADMIN would contradict the selected responsibilities; creator-only configuration would obstruct the selected shared company scope; organization/workspace management is unnecessary for the current single-company target.
- **Trade-offs accepted:** Account management and business operation require separate identities/roles under the approved single-role account model. Candidate and administrative sessions have separate authority checks. No role-management tables are introduced.
- **Revisit when:** The owner requests combined roles, delegated configuration ownership, or multiple companies.
- **Companion plan:** [Identity and Access D1–D7](../identity-access/decision-log.md). CORE-IAM-009 and [q2-access-evidence.md](q2-access-evidence.md) prove the administrative adapter. ACP T9 records the same HR allow, ADMIN deny, and candidate/expired/revoked denial on the configuration routes. D9 does not change this authority split.
- **Status:** active; supersedes D4's combined-role scope while preserving candidate exclusion. D5's payload, confidentiality, fingerprint and error contracts remain active.

## D7: Configuration tables are installed by an explicit database migration

- **Business driver:** On 2026-11-01 the owner required service-wide manual migrations because the project had grown (Identity and Access D8). HR configuration must follow that same operational rule.
- **Technical decision:** The five configuration tables are the next SQL version after the current clean version 2. Starting the service does not create them. They are placed before the unpublished candidate and assignment-access migrations so HR configuration can be installed without the missing Interview assignment table.
- **What this means for the product:** Operators prepare the database, then start the service. An unprepared database does not become a configuration editor merely by booting. Previously published Task history and administrative accounts remain in place.
- **Alternatives considered:** Creating the tables automatically on startup would contradict the approved operational change. Putting configuration tables after the assignment-access migration would make HR configuration wait for an Interview table that does not exist yet.
- **Trade-offs accepted:** Release procedure gains a migration step and a new accepted database version. If candidate or assignment SQL has already been applied in an environment, that environment needs a reviewed forward plan before configuration tables can be inserted at version 3.
- **Revisit when:** Database ownership splits into separate services, or Interview's assignment migration is applied first and the version order must be replanned.
- **Status:** active; applies Identity and Access D8 to this capability. D1–D6 remain active.

## D8: Historical template deployment compatibility is approved

- **Ticket:** `CORE-ACP-010` / T10.
- **Confirming actor/date:** user in this conversation, 2026-11-01; no separate external sign-off is claimed.
- **Business driver:** BR-CORE-009/013 and platform lifecycle require published revisions and future bindings to remain readable without silently changing meaning when validators evolve.
- **Approved contract:** retain-and-restrict deployment, restoration without current-creation reinterpretation, and explicit rejection of unsupported stored behavior versions; canonical in [technical-design §6.6](technical-design.md#66-historical-deployment-compatibility-approved-t10) and [t10-contract.md](t10-contract.md).
- **Product consequences:** Operators must deploy a binary whose restoration catalog covers every stored template/behavior tuple before serving configuration traffic. Rolling back a binary without restoration support for persisted behavior versions is refused at startup rather than corrupting reads. Historical revisions restore without current-creation-policy reinterpretation; current assessors are not rerun on restore.
- **Alternatives considered:** silently revalidating stored rows against today's publication rules would make historical revisions unreadable; unrestricted startup would allow partial interpretation of unsupported behavior tokens.
- **Trade-offs accepted:** Each new behavior version requires an explicit restoration registration and coordinated rollout; sparse historical fixtures remain valid stored shapes even when current assessors would block new publication.
- **Closure:** Q4 resolved as executable deployment policy; representative fixtures and sqlmock/adapter tests recorded in [t10-deployment.md](t10-deployment.md). T11 external consumer proof remains open.
- **Revisit when:** product retires a behavior version while stored rows or bindings still reference it; approve retirement/migration before removing restoration support.
- **Status:** active.
