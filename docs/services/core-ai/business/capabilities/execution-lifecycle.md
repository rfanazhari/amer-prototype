---
id: execution-lifecycle
service: core-ai
status: planned
rules: BR-CORE-050..BR-CORE-059
updated: 2026-11-02
implementation:
---

# Execution Lifecycle

## Purpose

Give every admitted agent run one stable identity and an honest ending. A repeated start finds the same run, a finished run stays finished, and later configuration edits do not rewrite the run that already began.

This capability states the shared rules. It does not define interview stages, question coverage, invoice fields, or the HR report. [Interview Runtime](./interview-runtime.md) supplies the interview's completion evidence. A later Document Analyzer capability must be able to satisfy the same lifecycle without adopting interview concepts.

## Ubiquitous language

**Agent Execution**, **Operation**, **Published Revision**, and **Model Session** stay defined in the [architecture design, section 4](../../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md#4-domain-and-platform-concepts).

| Term | Meaning |
|---|---|
| **Terminal outcome** | A committed ending of an execution: completed, failed, or cancelled. It is not reopened. |
| **Activation** | The moment an accepted execution actually begins its work. Acceptance of the run and activation of that work are separate events. |
| **Attempt** | One bounded try of a piece of work inside an execution. A newer attempt replaces the authority of an older one. It does not create a new execution. |

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| Submit an authorized start for an assignment | Candidate interview, on the candidate's request | Commit one execution bound to that assignment's published revision, or find the one already committed. |
| Repeat the same start | Candidate interview | Resolve the original execution. |
| Activate the execution | Interview runtime | Move the same execution from accepted to running, or record a failed activation attempt. |
| Commit the outcome required by the agent | Interview runtime | Allow completion only after that required progress is stored. |
| Cancel, fail beyond recovery, or revoke access | Candidate, runtime, or HR revocation | Commit the corresponding terminal outcome. |
| Publish a newer agent revision | HR, through configuration | Leave the execution bound to the revision it already has. |

## Rules

### BR-CORE-050 — An assignment has one execution

**Given** an interview assignment is admitted
**When** a start is accepted
**Then** that assignment has one execution, and another start does not create a second one.

**Why:** HR and the candidate must be able to point to one interview for one invitation. Two runs for the same assignment would make the outcome ambiguous.

### BR-CORE-051 — An accepted start exists before its work begins

**Given** a start is authorized and eligible
**When** the platform accepts it
**Then** the execution identity and its configuration binding are committed, and that acceptance does not by itself mean a voice session or other agent work has started.

**Why:** The candidate and HR need a durable answer that the interview exists even when connecting to the provider has not succeeded yet.

### BR-CORE-052 — The same start resolves the same execution

**Given** a start has already been committed
**When** the same logical start is repeated
**Then** the original execution is returned and no second execution is created.

**Why:** A lost response is a reason to look up the interview again, not a reason to hold the interview twice.

### BR-CORE-053 — A different start intent conflicts

**Given** a start identity already names one execution intent
**When** that identity is reused for a different intent
**Then** the request is rejected and does not replace the original execution.

**Why:** The caller must not receive confirmation for a different interview from the one originally requested.

### BR-CORE-054 — Work progresses through accepted, running, and one terminal outcome

**Given** an execution has been accepted
**When** its state changes
**Then** it becomes running only after activation succeeds, and from there it may complete, fail, or be cancelled. A completed, failed, or cancelled execution does not return to an earlier state.

**Why:** Everyone reading the interview must see one current answer, and a finished interview must not quietly become unfinished.

### BR-CORE-055 — Revocation or expiry during the interview cancels it

**Given** an execution is not yet terminal
**When** the assignment's access is revoked, or the candidate's authority to continue expires during the interview
**Then** the execution is cancelled and that cancellation is the committed outcome.

**Why:** Closing the candidate's right to be in the interview must also close the interview. Leaving it running after access was withdrawn would record an interview the company no longer authorizes.

### BR-CORE-056 — Completion waits for the required progress

**Given** an execution is running
**When** completion is requested
**Then** it becomes completed only after the progress the agent requires has been committed. Provider success alone does not complete it.

**Why:** A model finishing its connection is not evidence that the interview, or a later document task, produced the record the business agreed to keep.

### BR-CORE-057 — A superseded attempt cannot change newer progress

**Given** a newer attempt is the current authority for a piece of work
**When** an older attempt reports a late result
**Then** that late result does not overwrite the newer committed progress.

**Why:** A delayed provider event after reconnect must not erase the interview progress the candidate and HR are already relying on.

### BR-CORE-058 — Later publication does not rewrite a bound execution

**Given** an execution is bound to a published revision
**When** a newer revision of the same agent becomes active
**Then** the execution keeps the revision, inputs, and limits it was bound to.

**Why:** Questions or extraction rules changed after a run begins would make its result describe a different piece of work from the one that was started.

### BR-CORE-059 — The shared lifecycle carries no agent-specific checklist

**Given** a second kind of agent uses this lifecycle
**When** its execution is admitted, advanced, or finished
**Then** the lifecycle rules remain the identity, the state, the attempt authority, and the committed progress reference, without requiring interview stages or invoice fields.

**Why:** A shared ending is useful only when a non-interview agent can reach it without pretending to be an interview.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| The candidate starts, the response is lost, and the candidate starts again | One execution is returned. | `BR-CORE-050`, `BR-CORE-052` |
| The provider has not connected | The execution remains accepted and is not described as running. | `BR-CORE-051`, `BR-CORE-054` |
| Activation fails and is retried within its budget | The same execution is activated or explicitly failed. A second execution is not created. | `BR-CORE-054` |
| The candidate's browser disconnects after the execution is running | The execution stays running until recovery, completion, failure, or cancellation. Disconnect alone does not cancel it. | `BR-CORE-054` |
| HR revokes the assignment while the interview is running | The execution is cancelled. | `BR-CORE-055`, sibling `BR-CORE-038` |
| The provider reports success before the required transcript is stored | The execution does not become completed. | `BR-CORE-056` |
| An old voice attempt delivers a transcript after a newer attempt has committed progress | The old attempt's result is rejected. | `BR-CORE-057` |
| HR publishes a new revision while this execution is running | The running execution keeps its original revision. | `BR-CORE-058`, sibling `BR-CORE-013` |
| A document task later uses the same lifecycle | It does not need an interview stage to be admitted or completed. | `BR-CORE-059` |

## Examples

| Input | Expected outcome |
|---|---|
| Start for assignment `interview-demo-042` is accepted and the voice provider is not yet connected | Execution `interview-demo-042-run` exists as accepted, bound to the assignment's published revision. No voice session is implied. |
| The same start for `interview-demo-042` is submitted again | `interview-demo-042-run` is returned. A second execution is not created. |
| Activation of `interview-demo-042-run` succeeds | The execution is running. |
| The provider session for `interview-demo-042-run` ends before the required progress is stored | The execution is not completed. |
| Revision 3 is published while `interview-demo-042-run` is running on revision 2 | The run remains bound to revision 2. |
| A later invoice execution is admitted | It receives an execution identity and a terminal outcome under these rules, without an interview stage. |

## Known gaps

Not yet implemented — verify against code after implementation.

## Open questions

- **Platform owner:** The current `interview_executions` record stores the assignment and the pinned revision only. These rules describe the intended lifecycle. They do not claim that record already enforces it.
- **Product owner:** Confirm that each **Why** line captures the intended consequence. These lines interpret the approved design. They are not separate stakeholder statements.

## Related

- Sibling capability: [Agent Configuration and Publication](./agent-configuration-publication.md), `BR-CORE-013` and `BR-CORE-014` — an existing binding keeps its revision, and publication is not permission to execute.
- Sibling capability: [Identity and Access](./identity-access.md), `BR-CORE-034`, `BR-CORE-035`, `BR-CORE-038`, and `BR-CORE-039` — access does not start, reload, or restart the execution.
- Sibling capability: [Candidate Interview](./candidate-interview.md) — the candidate's start, refresh, and early stop are the interview triggers for these rules.
- Sibling capability: [Interview Runtime](./interview-runtime.md) — supplies the progress that completion requires and the attempts that must be fenced.
- Source document: [Architecture design, sections 6 and 9](../../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md) — kernel responsibilities and initiation semantics.
- Source document: [Scope freeze, sections 4 and 5](../../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) — the five-state lifecycle and INV-01..INV-09.
- Source document: [Technical Design v0.1, sections 3 and 5](../../../../design/core-ai/technical-design.md) — accepted transitions, binding, and the first terminal commit.
