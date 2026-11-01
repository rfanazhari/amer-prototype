---
id: interview-reporting
service: core-ai
status: planned
rules: BR-CORE-080..BR-CORE-086
updated: 2026-11-02
implementation:
---

# Interview Reporting

## Purpose

Give HR a separate written account of a finished interview: what was covered, what was missing, and what remains uncertain, traced to the interview record. The account does not hire, reject, or rank the candidate, and a failure to produce it does not undo the interview.

This phase stores and exposes that account to HR through the service. It does not add an HR or admin screen. The candidate's final screen is owned by [Candidate Interview](./candidate-interview.md) and does not wait for this account.

## Ubiquitous language

**Canonical transcript**, **Coverage**, and **Terminal outcome** stay defined by [Interview Runtime](./interview-runtime.md) and [Execution Lifecycle](./execution-lifecycle.md).

| Term | Meaning |
|---|---|
| **Interview report** | HR's evidence-linked account of one finished interview, produced after the interview has ended. |
| **Report generation** | One attempt to produce or replace that account. It has its own outcome and does not change the interview's outcome. |

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| The interview is completed with its required record | Interview runtime | Make a report generation possible for that interview. |
| Produce the report | Reporting | Write observations, gaps, and uncertainty linked to the interview record. |
| The report generation fails | Reporting | Leave the completed interview unchanged. |
| Ask for another report | HR | Start a new generation only when the interview is complete, its evidence is available, and no generation is currently active. |
| The candidate opens the final screen | Candidate | See the interview outcome without the report. |

## Rules

### BR-CORE-080 — A report is a separate outcome of a completed interview

**Given** an interview has completed and its required progress is stored
**When** HR's account of that interview is produced
**Then** the report is a new outcome of its own, created from the committed interview record, and the interview's completed state remains the interview's outcome.

**Why:** HR needs a write-up after the conversation. That write-up must not be confused with the interview itself, or the interview's result would change whenever the write-up is attempted.

### BR-CORE-081 — A report that fails leaves the interview completed

**Given** an interview is completed
**When** its report generation fails or is still unfinished
**Then** the interview stays completed. The failure is recorded on the report generation.

**Why:** A problem in the later write-up must not turn a finished interview into a failed one, and it must not force the candidate to repeat the conversation.

### BR-CORE-082 — The report says what was observed, what was missing, and what is uncertain

**Given** a report is produced from a completed interview
**When** HR reads it
**Then** it includes the observations, coverage gaps, and uncertainties supported by the interview record.

**Why:** HR needs the limits of the conversation to be visible. A report that only sounds conclusive hides the questions that were omitted or unclear.

### BR-CORE-083 — A report statement cites the interview record

**Given** a report states something the candidate said or a question the interview covered
**When** that statement is stored
**Then** it refers to finalized interview evidence. A statement with no such evidence is not presented as an answer or as demonstrated competence.

**Why:** HR must be able to trace a finding back to the interview. An unsupported finding would let the write-up invent a conversation that did not happen.

### BR-CORE-084 — The report does not hire, reject, or rank

**Given** a report is produced
**When** its findings are written
**Then** it does not recommend hiring, rejection, or a rank for the candidate.

**Why:** The interview informs HR. The decision to hire remains a human decision and must not be presented as the platform's verdict.

### BR-CORE-085 — Another report is an explicit new generation

**Given** a completed interview has evidence available
**When** HR requests another report
**Then** a new generation starts only when no generation for that interview is currently active, and it does not reopen or replace the interview's completed outcome.

**Why:** Repeating a write-up must be a deliberate second account. An accidental repeat must not start two write-ups at once or make the finished interview look unfinished.

### BR-CORE-086 — The candidate does not receive the HR report

**Given** a report exists for the candidate's interview
**When** the candidate views the finished interview
**Then** the candidate's screen does not include the report.

**Why:** The report is HR's internal account of the interview, including gaps and compensation context. It is not a result to hand back to the candidate as part of finishing the call.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| The interview is still running | No report is produced as the interview's outcome. | `BR-CORE-080` |
| The interview failed or was cancelled before completion | A completion report is not produced for it. | `BR-CORE-080` |
| Report generation fails after the interview completed | The interview remains completed and the generation shows the failure. | `BR-CORE-081` |
| The report is still being written | The candidate's final screen shows the completed interview. | `BR-CORE-086`, sibling `BR-CORE-047` |
| A finding has no finalized transcript entry | The finding is not stated as something the candidate answered. | `BR-CORE-083` |
| The report describes a compensation expectation | It may include that expectation and its uncertainty. It does not disclose the salary ceiling and does not recommend hire or reject. | `BR-CORE-082`, `BR-CORE-084`, sibling `BR-CORE-071` |
| HR requests a second report while the first generation is still active | The second generation does not start. | `BR-CORE-085` |
| HR requests a second report after the first generation finished | A new generation starts and the interview stays completed. | `BR-CORE-085` |

## Examples

| Input | Expected outcome |
|---|---|
| Interview `interview-demo-042-run` is completed with coverage for `background` and an omission for `compensation` | A report generation may start from that record. The interview stays completed. |
| The report generation for `interview-demo-042-run` fails | The interview remains completed. The generation records the failure. |
| The report says Andi described backend experience | The statement cites the finalized `background` transcript entry. |
| The report has no transcript entry for `compensation` | The report shows the omission. It does not treat `compensation` as answered. |
| HR asks for a new report while one generation is active | The request does not start a second generation. |
| Andi opens the finished interview while the report exists | Andi sees the completed interview and does not see the report. |

## Known gaps

Not yet implemented — verify against code after implementation.

## Open questions

- **Product owner:** This phase exposes the report to HR through the service and does not add an admin screen. The reader, beyond the current HR role, is the HR colleague already allowed to operate interviews. A later screen is a separate decision.
- **Product owner:** Confirm that each **Why** line captures the intended consequence. These lines interpret the approved design. They are not separate stakeholder statements.

## Related

- Sibling capability: [Interview Runtime](./interview-runtime.md), `BR-CORE-067`..`BR-CORE-071` — the finalized transcript, evidence, and confidential salary ceiling the report must respect.
- Sibling capability: [Execution Lifecycle](./execution-lifecycle.md), `BR-CORE-056` — the interview completes on its own required progress, before this report exists.
- Sibling capability: [Candidate Interview](./candidate-interview.md), `BR-CORE-047` — the candidate's final screen does not wait for or show the report.
- Sibling capability: [Identity and Access](./identity-access.md), `BR-CORE-022` — ADMIN account administration does not grant HR report authority.
- Source document: [Architecture design, section 10](../../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md#completion-and-reporting) — the report is downstream of the interview.
- Source document: [Scope freeze, H-10..H-12](../../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) — separate report lifecycle, no hiring verdict, and confidential salary.
- Source document: [Technical Design v0.1, sections 6 and 7](../../../../design/core-ai/technical-design.md) — report work follows interview completion and does not change that completion.
