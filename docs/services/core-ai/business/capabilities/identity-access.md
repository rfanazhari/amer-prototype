---
id: identity-access
service: core-ai
status: planned
rules: BR-CORE-020..BR-CORE-039
updated: 2026-11-01
implementation:
---

# Identity and Access

## Purpose

Enable the company's ADMIN to provision HR access and enable HR colleagues to manage shared agent configuration. Give each invited candidate access to the assigned interview without requiring a permanent login account, while preserving one candidate identity across multiple interview assignments.

This capability establishes identity and access. Interview assignment preparation, consent, execution, provider connections and reporting have separate business owners; obtaining access does not execute those workflows.

## Ubiquitous language

Agent concepts follow [Agent Configuration and Publication](./agent-configuration-publication.md). Assignment and execution boundaries follow the [platform design, sections 3 and 7](../../../../design/core-ai/technical-design.md).

| Term | Meaning |
|---|---|
| **Administrative account** | A stable company user identity with one role, ADMIN or HR, and credentials for administrative login. |
| **ADMIN** | The role responsible for provisioning HR accounts; it does not confer HR configuration or reporting authority. |
| **HR** | The role responsible for shared company agent configuration and authorized interview operations. |
| **Administrative session** | A time-limited authenticated access period for an administrative account; it does not independently grant permissions. |
| **Candidate** | A person identified across interview assignments without a permanent application login account. |
| **Invitation** | A time-limited credential granting access to one interview assignment. |
| **Candidate session** | A time-limited access period established after invitation verification and restricted to that invitation's assignment. |

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| Initialize the first administrative account | Deployment operator | Bootstrap ADMIN from environment-supplied credentials. |
| Add an HR account | Authenticated ADMIN | Provision HR administrative login. |
| Log in or log out | ADMIN or HR | Establish or revoke an administrative session. |
| Access configuration, including publication replay | HR | Evaluate current HR authority before access. |
| Record a candidate | HR | Establish a reusable candidate identity. |
| Issue or replace an interview invitation | HR | Establish access to an existing assignment. |
| Exchange an invitation | Candidate holding the link | Establish an assignment-scoped candidate session. |
| Reload the interview page | Authenticated candidate | Reuse valid access without creating an interview execution. |
| Revoke an assignment | HR through the Interview capability | Remove assignment access; execution cancellation remains owned by Interview. |

## Rules

### BR-CORE-020 — Administrative access is provisioned

**Given** the company requires administrative access
**When** the first account is initialized or an HR account is added
**Then** access is established through deployment bootstrap or authorized ADMIN provisioning, rather than public signup.

**Why:** The company must control who receives internal operating authority instead of exposing administrative enrollment to arbitrary visitors.

### BR-CORE-021 — ADMIN provisions HR accounts

**Given** an authenticated actor requests creation of an HR account
**When** the platform checks account-management authority
**Then** only current ADMIN authority permits the operation.

**Why:** Allowing HR operators or interview participants to create privileged users would bypass the company's access owner.

### BR-CORE-022 — ADMIN does not inherit HR business authority

**Given** an authenticated account has the ADMIN role
**When** it requests configuration management or internal HR reports
**Then** ADMIN alone does not authorize access.

**Why:** Managing user access must not automatically expose interview policy or confidential HR outcomes.

### BR-CORE-023 — HR colleagues share company configuration

**Given** an authenticated account has current HR authority
**When** it accesses agent configuration within the single company's scope
**Then** another HR's authorship does not exclude it from otherwise permitted configuration operations.

**Why:** Configuration must remain available for collaboration and handover rather than become dependent on one employee's continued presence.

### BR-CORE-024 — Administrative sessions expire absolutely

**Given** an administrative session was established
**When** two hours have elapsed since its creation
**Then** the session no longer authorizes access, including at the exact expiry instant.

**Why:** Continued activity must not extend an unattended or compromised administrative access period indefinitely.

### BR-CORE-025 — Administrative logout revokes the current session

**Given** an account has an administrative session
**When** the current-session logout operation succeeds
**Then** that session cannot authorize subsequent requests.

**Why:** Closing a browser view is insufficient when a previously issued credential could still be replayed.

### BR-CORE-026 — Current account eligibility governs administrative access

**Given** a previously established administrative session
**When** the platform evaluates a protected action
**Then** an expired or revoked session, disabled account, or insufficient current role cannot authorize it.

**Why:** Previously granted access must not remain effective after its underlying authority has been removed.

### BR-CORE-027 — Administrative subject identity survives login changes

**Given** an administrative account establishes another session
**When** the platform identifies the actor for an operation
**Then** it uses the same stable account identity rather than the new session identity.

**Why:** A network retry after reauthentication must resolve the original subject-scoped publication instead of creating a different command scope.

### BR-CORE-028 — Publication replay still requires current authority

**Given** an earlier publication has an outcome recorded for an account
**When** that account requests the outcome again
**Then** current HR authority is required before disclosing the result.

**Why:** Deduplication must not become an alternative route for retrieving information after access has been lost.

### BR-CORE-029 — Candidate access grants no administrative authority

**Given** an actor has a valid candidate session
**When** it requests account management, agent configuration, or an internal HR report
**Then** candidate access does not authorize the operation.

**Why:** Interview participation must not permit changing assessment policy or inspecting internal evaluation material.

### BR-CORE-030 — A candidate identity can have multiple assignments

**Given** HR selects an existing candidate
**When** another interview is assigned to that person
**Then** the new assignment can reference the existing candidate identity without requiring a new candidate account.

**Why:** Repeated interviews must retain a coherent person identity while preserving distinct interview histories.

### BR-CORE-031 — An invitation grants access to one assignment

**Given** HR issues an invitation for an existing interview assignment
**When** the invitation grants candidate access
**Then** its authority is restricted to that assignment and its assigned candidate.

**Why:** Possession of one invitation must not expose other interviews, even those involving the same person.

### BR-CORE-032 — An invitation must be currently valid at exchange

**Given** a candidate presents an invitation
**When** the platform exchanges it for access
**Then** only a recognized, unrevoked invitation before its 24-hour expiry can establish a session for an accessible assignment.

**Why:** A leaked, withdrawn or outdated invitation must not keep creating new access periods.

### BR-CORE-033 — Invitation verification establishes a distinct access session

**Given** an invitation is accepted for exchange
**When** access is established successfully
**Then** the candidate receives a uniquely identified session bound to that invitation's assignment.

**Why:** Session validity and revocation need their own identity rather than treating the reusable interview reference as a credential.

### BR-CORE-034 — Obtaining access does not start the interview

**Given** a candidate opens or exchanges an invitation
**When** the candidate receives access
**Then** the platform does not start an interview execution as a side effect.

**Why:** Candidates need to review interview context and complete prerequisites such as consent before an interview begins.

### BR-CORE-035 — Reloading does not create another interview

**Given** a candidate has a valid session for an assignment
**When** the interview page is refreshed
**Then** the existing session provides access to the same assignment without creating a new execution.

**Why:** Normal navigation and connection recovery must not produce duplicate interview attempts or conflicting results.

### BR-CORE-036 — Candidate sessions expire absolutely

**Given** a candidate session was established
**When** two hours have elapsed since its creation
**Then** the session no longer authorizes access, including at the exact expiry instant.

**Why:** An interview link must not turn into an indefinitely valid application credential.

### BR-CORE-037 — Reissuing an invitation preserves the interview identity

**Given** an assignment already has an invitation
**When** HR issues a replacement
**Then** the former invitation is revoked and the new invitation still targets the same assignment, without automatically cancelling an active interview.

**Why:** Replacing an access link must not erase progress or silently create a different interview.

### BR-CORE-038 — Assignment revocation removes candidate access

**Given** a candidate has invitation or session access to an assignment
**When** the assignment is revoked
**Then** the platform stops authorizing access to that assignment.

**Why:** Revoking the interview must close existing access as well as prevent entry through links; execution cancellation is coordinated by the Interview capability.

### BR-CORE-039 — A completed interview is not restarted by an access operation

**Given** an assignment has a completed interview
**When** a link is reopened, a session is replaced, or an invitation is reissued
**Then** those access operations do not restart its execution; a new interview requires a new assignment.

**Why:** Historical outcomes must remain attributable to a distinct interview rather than being overwritten by another sitting.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| ADMIN knows a configuration UUID | Deny configuration operations despite valid administrative login. | `BR-CORE-022` |
| HR B opens an instance created by HR A | Allow permitted operations within shared company scope. | `BR-CORE-023` |
| Session expiry equals trusted current time | Deny access; activity does not extend expiry. | `BR-CORE-024`, `BR-CORE-036` |
| HR retries an old publication after losing authority | Deny before returning the recorded outcome. | `BR-CORE-028` |
| Same candidate has screening and technical interviews | Retain one candidate identity and two distinct assignments. | `BR-CORE-030` |
| Session for assignment A requests assignment B | Deny, even when both assignments concern the same candidate. | `BR-CORE-031` |
| Invitation expires at the instant of exchange | Do not create a session. | `BR-CORE-032` |
| HR replaces a link during an active interview | Revoke the old invitation without automatically cancelling execution. | `BR-CORE-037` |
| Candidate refreshes a completed interview page | Do not start another execution. | `BR-CORE-035`, `BR-CORE-039` |

## Examples

| Input | Expected outcome |
|---|---|
| ADMIN Rina adds HR Sinta | HR account created through authorized provisioning. |
| ADMIN Rina attempts to publish Sinta's configuration | Denied; ADMIN is not a configuration role. |
| HR Budi edits Sinta's configuration with its current draft version | Permitted by shared company access; ordinary draft/version rules still apply. |
| Sinta logs in again and retries publication key `publish-007` | Same account scope; existing publication contract resolves the original outcome. |
| Candidate Andi has assignments `screening-01` and `technical-02` | Separate invitation/session scopes sharing one candidate identity. |
| Administrative session created 2026-11-01T08:00:00.000Z, used at 10:00:00.000Z | Expired. |
| Invitation issued 2026-11-01T08:00:00.000Z, exchanged at 2026-11-02T08:00:00.000Z | Expired; no new session. |

## Known gaps

Not yet implemented — verify against code after implementation.

## Open questions

- Resolved 2026-11-01 (T1 / CORE-IAM-001): Administrative email is canonicalized by trimming outer whitespace and Unicode lowercasing the entire address. Initial HR passwords are chosen by ADMIN at provisioning and delivered out of band; bootstrap uses environment credentials once. Self-service reset, account edits, role changes, and disable/reactivate endpoints are not approved by adding a status column.
- Resolved 2026-11-01 (T10 / CORE-IAM-010): Candidate email uses the same canonicalization as administrative email and is never unique or used for upsert. Reuse is by known candidate identity on an Interview assignment. A valid invitation may be exchanged repeatedly until expiry or revocation; each exchange may create another session; a lost response is not exactly-once cookie recovery. The bearer link proves possession, not independently verified identity. Access reads Interview assignment facts after an assignment-first lock and never creates an assignment table.
- Interview owner: supply the runtime `interview_assignments` table, adapter and admission/cancellation consumers. Consent and execution tables belong to that capability. Access contracts for those facts are approved in T10.
- Product/data owner: decide candidate/contact/session retention and purge ordering before cleanup is implemented.

## Related

- Sibling capabilities: [Agent Configuration and Publication](./agent-configuration-publication.md), especially `BR-CORE-003`.
- Source document: [Identity and Access technical design](../../identity-access/technical-design.md).
- Source document: [Platform design, sections 9–10 and TD-21](../../../../design/core-ai/technical-design.md).
- Existing flow reference: `application/service/agentconfiguration.go` — authority integration port already present; no real authentication adapter is supplied.
- Source decisions: owner approvals in this conversation on 2026-11-01 cover ADMIN/HR separation, one company, invitation-based repeat candidate interviews, five access tables, and the `identity-access` name/block allocation. Rules use `BR-CORE-020..029` for administrative access and `BR-CORE-030..039` for candidate access; `040..049` remains unused within the approved candidate block.
