---
id: candidate-interview
service: core-ai
status: planned
rules: BR-CORE-040..BR-CORE-047
updated: 2026-11-02
implementation:
---

# Candidate Interview

## Purpose

Let a candidate who already holds an assignment-scoped session review the interview notice, accept it, enter one live interview, follow its progress, and leave. The screen is the candidate's. This phase does not add an HR or admin screen.

Opening the invitation and holding a session are owned by [Identity and Access](./identity-access.md). Whether the interview may start, how it progresses, and how it becomes terminal are owned by [Execution Lifecycle](./execution-lifecycle.md) and [Interview Runtime](./interview-runtime.md). The HR report is owned by [Interview Reporting](./interview-reporting.md) and is not part of the candidate's screen.

## Ubiquitous language

**Agent Execution**, **Published Revision**, and **Candidate session** stay defined by the sibling capabilities and the [architecture design, section 4](../../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md#4-domain-and-platform-concepts).

| Term | Meaning |
|---|---|
| **Notice** | The statement shown to the candidate before an interview starts, identified by the version the candidate is asked to accept. |
| **Consent** | The candidate's explicit acceptance of one notice version, recorded with the time of that acceptance. |
| **Interview controller** | The single browser tab allowed to advance the live interview. Other tabs of the same session may look, and taking control ends the previous tab's authority. |

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| Open the interview after a session exists | Candidate | See the notice, consent state, current interview, and available actions. |
| Accept the notice | Candidate | Record consent for that notice version. |
| Start or rejoin the interview | Candidate | Enter the one execution for this assignment, after consent and a session that still covers the interview. |
| Refresh the interview page | Candidate | Return to the same session and the same interview. |
| Use a second browser tab | Candidate | Read status, or take control and end the previous tab's authority. |
| Confirm an early stop | Candidate | Leave the interview as a cancellation. |
| Reach a terminal interview | Candidate | See that outcome without waiting for the HR report. |

## Rules

### BR-CORE-040 — Consent precedes the start of the interview

**Given** a candidate holds a valid session for one assignment and has not accepted the current notice
**When** the candidate asks to start the interview
**Then** the interview does not start, and the candidate remains on the notice until that notice version is accepted.

**Why:** An invitation proves access to an assignment. It does not prove the candidate agreed to be interviewed under the notice the company must be able to point back to.

### BR-CORE-041 — A start request enters the assignment's one interview

**Given** the candidate has accepted the current notice and the session still covers the interview's deadline
**When** the candidate asks to start, including after a lost response or a repeated click
**Then** the candidate enters the single interview already committed for that assignment, or the interview committed by that request.

**Why:** A double click or a timeout must not open a second interview for the same invitation, and the candidate must not be told to wait while an accepted start is thrown away.

### BR-CORE-042 — The candidate session must cover the whole interview

**Given** a candidate session will expire before the interview's hard deadline
**When** the candidate asks to start or to continue
**Then** the request is refused until the candidate enters again through an invitation that is still valid.

**Why:** The company cannot stand behind an interview that continues after the access which authorized the candidate has already ended.

### BR-CORE-043 — The candidate sees only this interview

**Given** a candidate is viewing the interview for the assigned session
**When** the screen or its status is shown
**Then** it shows the notice, consent, this interview's state, the actions now available, and progress the candidate is allowed to follow, and it does not reveal the salary ceiling, another candidate, administrative configuration, or the HR report.

**Why:** Participation in one interview must not become a window onto hiring policy or anyone else's process.

### BR-CORE-044 — One tab controls the live interview

**Given** the interview is underway in one browser tab
**When** the same session opens another tab
**Then** the other tab can read status, and an explicit takeover makes the new tab the controller and ends the previous tab's authority to advance the interview.

**Why:** Two tabs answering at once would produce an interview neither the candidate nor HR can treat as one conversation.

### BR-CORE-045 — The candidate can leave before the interview finishes

**Given** the interview is not yet terminal
**When** the candidate confirms an early stop
**Then** the interview ends as a cancellation, local capture stops, and the screen shows that committed cancellation.

**Why:** The candidate must be able to withdraw from a live interview without the screen presenting that withdrawal as a completed interview.

### BR-CORE-046 — Refreshing the page returns to the same interview

**Given** the candidate has a valid session and an interview already exists for the assignment
**When** the candidate refreshes or reopens the interview page
**Then** the page shows that same interview and does not start another one.

**Why:** Ordinary recovery from a reload is how a candidate gets back to an interview already underway, not a reason to begin a different one.

### BR-CORE-047 — The candidate's final screen does not wait for the HR report

**Given** the interview has reached a terminal outcome
**When** the candidate is shown the final screen
**Then** the screen shows that interview outcome and does not wait until the HR report exists.

**Why:** The candidate's part ends with the interview. A delay in HR's later write-up must not keep the candidate on a waiting screen.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| The candidate opens the interview link and exchanges it | Access is established and the interview does not start. | sibling `BR-CORE-034` |
| The candidate accepts notice version `notice-2026-11-02` and the start response is lost | The same start returns the interview already committed for the assignment. | `BR-CORE-041` |
| The session has 30 minutes left and the interview's hard deadline is 15 minutes from start | The start is allowed only when those 30 minutes still cover that deadline; otherwise the candidate re-enters through a valid invitation. | `BR-CORE-042` |
| A second tab asks for the interview while the first tab is speaking | The second tab sees status. It advances the interview only after an explicit takeover. | `BR-CORE-044` |
| The candidate confirms an early stop during role exploration | The interview is cancelled. It is not presented as completed. | `BR-CORE-045` |
| The candidate refreshes after the interview was cancelled | The same cancelled interview is shown. No new interview is created. | `BR-CORE-046`, sibling `BR-CORE-035`, sibling `BR-CORE-039` |
| The interview is complete and the HR report is still being written | The candidate's final screen shows the completed interview. | `BR-CORE-047` |
| HR revokes the assignment during the interview | Candidate access ends, and the nonterminal interview is cancelled by the execution lifecycle. | sibling `BR-CORE-038`, `BR-CORE-055` |

## Examples

| Input | Expected outcome |
|---|---|
| Candidate Andi holds a session for assignment `interview-demo-042` and has not accepted the notice | The interview page shows the notice. Start does not create an interview. |
| Andi accepts notice `notice-2026-11-02` and starts once | One interview for `interview-demo-042` is entered. |
| Andi starts again with the same request after the first response timed out | The same interview is returned. |
| Andi's session was created at 2026-11-02T08:00:00.000Z and he tries to start at 2026-11-02T09:50:00.000Z, when the remaining session no longer covers a 15-minute interview | Start is refused until he enters again through a valid invitation. |
| Andi confirms early stop while the interview is in progress | The interview becomes cancelled and the page shows that cancellation. |
| Andi's interview is completed while the report for `interview-demo-042` is still pending | Andi sees the completed interview, not a screen waiting on the report. |

## Known gaps

Not yet implemented — verify against code after implementation.

## Open questions

- **Product owner:** What text and version identifier are shown as the notice the candidate must accept? The sources require a notice version and an explicit acceptance, and do not record the notice copy.
- **Product and data owner:** What happens to the recorded interview when the candidate withdraws consent after it has started? Consent-withdrawal handling is still unresolved in the platform technical design.
- **Product owner:** Confirm that each **Why** line captures the intended consequence. These lines interpret the approved design. They are not separate stakeholder statements.

## Related

- Sibling capability: [Identity and Access](./identity-access.md), `BR-CORE-034`..`BR-CORE-039` — a session is not an interview, a reload is not a new interview, and revocation removes access.
- Sibling capability: [Execution Lifecycle](./execution-lifecycle.md), `BR-CORE-050`..`BR-CORE-059` — the start request commits or finds one execution and a terminal outcome stays closed.
- Sibling capability: [Interview Runtime](./interview-runtime.md), `BR-CORE-060`..`BR-CORE-071` — what the candidate hears and how the interview progresses.
- Sibling capability: [Interview Reporting](./interview-reporting.md), `BR-CORE-080`..`BR-CORE-086` — the HR report is outside the candidate's final screen.
- Source document: [Technical Design v0.1, sections 7, 9, and 10](../../../../design/core-ai/technical-design.md) — candidate journey, candidate API direction, and session limits.
- Source document: [Scope freeze, Journey A](../../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md#journey-a--hr-interviewer) — the candidate steps from notice through the final screen.
