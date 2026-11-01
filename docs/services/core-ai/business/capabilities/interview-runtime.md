---
id: interview-runtime
service: core-ai
status: planned
rules: BR-CORE-060..BR-CORE-071
updated: 2026-11-02
implementation:
---

# Interview Runtime

## Purpose

Conduct the live HR interview the candidate entered: ask the published questions, record honest gaps, move forward only with a committed handoff, recover one lost connection, and finish only when the required record is stored.

The candidate's screen and consent belong to [Candidate Interview](./candidate-interview.md). The execution's identity and terminal rules belong to [Execution Lifecycle](./execution-lifecycle.md). The later HR write-up belongs to [Interview Reporting](./interview-reporting.md).

## Ubiquitous language

The stage names and time budgets come from the published HR Interviewer revision. This capability does not redefine **Published Revision** or **Agent Execution**.

| Term | Meaning |
|---|---|
| **Coverage** | The record of each required question as pending, asked, answered, or omitted. Answered means a response was recorded, not that competence was shown. |
| **Omission** | An explicit gap: the candidate skipped, no answer followed one clarification, or the stage's soft time budget expired. A lost recording is not an omission. |
| **Handoff** | The committed record of a finished stage, including its coverage, gaps, and the evidence the next stage is allowed to rely on. |
| **Canonical transcript** | The backend's ordered record of the interview, with speaker, session, and attempt. An unfinished fragment is not yet evidence. |

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| Activate the accepted interview | Interview runtime | Open the first voice session under the bound revision. |
| Candidate speaks, skips, or stays silent | Candidate | Update coverage only from a finalized response or an allowed omission. |
| Model proposes the next stage | Voice model | Ask the workflow to authorize the move. |
| Workflow authorizes the move | Interview runtime | Store the source handoff, then open a new voice session for the next stage. |
| Connection drops | Candidate or provider | Resume the same interview from the last committed checkpoint. |
| Required progress is stored | Interview runtime | Allow the execution to complete. |
| Required final evidence is missing at the deadline | Interview runtime | Fail the interview and keep the progress already committed. |

## Rules

### BR-CORE-060 — The live interview uses the revision it was bound to

**Given** an interview execution is bound to one published HR Interviewer revision
**When** the interview is conducted
**Then** its questions, stage order, time budgets, follow-up limit, and compensation mode are those of that revision.

**Why:** The candidate and HR must be able to say which questions governed the conversation. A newer draft must not change an interview that has already started.

### BR-CORE-061 — Required questions are asked or explicitly omitted

**Given** the bound revision lists mandatory questions
**When** the interview progresses
**Then** each required question is asked, or its omission is recorded with a reason. The representative interview also includes at least one follow-up that is relevant to what the candidate said, within the revision's follow-up limit.

**Why:** HR cannot treat silence or a skipped topic as if the question had been covered, and a follow-up that ignores the candidate is not evidence of an adaptive interview.

### BR-CORE-062 — A recorded answer is not a judgement of competence

**Given** a candidate response is finalized
**When** coverage is recorded
**Then** the question is answered because the response was recorded. The interview does not mark the candidate competent, suitable, or unsuitable.

**Why:** The interview collects what was said. Hiring judgement belongs to HR after the interview, not to the conversation record.

### BR-CORE-063 — Lost audio is not recorded as an omission

**Given** a required question was not captured because the recording or connection failed
**When** coverage is assessed
**Then** the gap stays a technical failure. It is not stored as a permitted omission and it does not authorize the next stage.

**Why:** A missing recording must not be rewritten as the candidate having skipped the question.

### BR-CORE-064 — The workflow decides whether a stage may change

**Given** the voice model proposes a move to the next stage
**When** the proposal is evaluated
**Then** the move proceeds only when coverage, time, and policy for the source stage allow it. An unauthorized proposal does not change the stage.

**Why:** The model may suggest that the conversation is ready to move. HR's published rules decide whether it is.

### BR-CORE-065 — The next stage starts only after the current handoff is stored

**Given** a stage change is authorized
**When** the next stage is activated
**Then** the source stage's required transcript, coverage, and handoff are already committed. A missing required handoff blocks activation. A failed narrative summary may fall back to that committed evidence.

**Why:** The next part of the interview has to rest on a record HR can inspect. A summary that failed to generate must not erase the evidence that did exist, and a missing record must not be invented.

### BR-CORE-066 — Each active stage has its own voice session

**Given** a stage has been activated
**When** the candidate speaks in that stage
**Then** the stage uses a voice session of its own. Replacing a session after a fault does not create another interview.

**Why:** A stage boundary has to be a real break in the provider conversation, while the interview the candidate is in remains the same one.

### BR-CORE-067 — The backend transcript is the record the interview relies on

**Given** provider speech events arrive
**When** the interview stores what was said
**Then** the backend keeps one ordered transcript with speaker, session, and attempt. An unfinished fragment is not evidence. A correction replaces the affected evidence by a new revision of the record.

**Why:** The candidate's screen and the provider's stream can disagree or arrive late. HR needs one record that says what the interview will stand on.

### BR-CORE-068 — Evidence is cited, and a missing record is not treated as an answer

**Given** coverage or a later report statement refers to what the candidate said
**When** that statement is stored
**Then** it points at a finalized transcript entry. A question with no such entry remains unanswered or omitted. It is not treated as answered.

**Why:** An interview finding that cannot be traced to what was actually recorded is a claim HR cannot defend.

### BR-CORE-069 — One lost connection resumes the same interview

**Given** the interview has a committed checkpoint and the connection drops once
**When** the candidate reconnects within the recovery window
**Then** the same execution resumes from that checkpoint in a new voice session. Speech that was never committed may need to be repeated. Hidden provider memory is not restored, and a late event from the old session does not overwrite the new progress.

**Why:** A dropped call is a reason to continue the interview the candidate already started, not a reason to lose the committed record or to begin a second interview.

### BR-CORE-070 — The interview finishes only with its required record

**Given** the interview reaches its end or its hard deadline
**When** completion is decided
**Then** the interview completes only after the required coverage and finalized progress are stored. If that final evidence is missing, the interview fails and the progress already committed remains available.

**Why:** HR must not receive a completed interview whose ending has no record, and a failure at the end must not discard the part that was already stored.

### BR-CORE-071 — The salary ceiling stays out of the conversation

**Given** the bound revision permits a compensation discussion and holds a confidential salary ceiling
**When** the interview is conducted or its candidate-visible record is produced
**Then** the ceiling is enforced on the server and does not appear in the candidate's experience or in the model context.

**Why:** The ceiling is an internal limit on what the interview may discuss. Disclosing it would turn a compensation conversation into a revelation of hiring policy.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| The candidate skips a question after one clarification | Record an omission with that reason. Do not mark the question answered. | `BR-CORE-061`, `BR-CORE-062` |
| The stage's soft budget expires with a question still pending | Record the omission because the budget expired. | `BR-CORE-061` |
| Audio for a required answer is lost | Keep the gap as a technical failure and do not advance because of it. | `BR-CORE-063` |
| The model proposes closing before required coverage exists | Reject the stage change. | `BR-CORE-064`, `BR-CORE-065` |
| The optional stage summary fails and the structured handoff exists | Continue from the structured handoff. | `BR-CORE-065` |
| The candidate reconnects after one drop | Resume the same execution from the last checkpoint in a new voice session. | `BR-CORE-066`, `BR-CORE-069` |
| A transcript fragment is still partial | Do not use it as evidence or as an answered question. | `BR-CORE-067`, `BR-CORE-068` |
| The hard deadline arrives without the final record | Fail the interview and retain the committed progress. | `BR-CORE-070` |
| The candidate asks the interview to reveal the salary ceiling | The ceiling remains server-side. | `BR-CORE-071` |

## Examples

| Input | Expected outcome |
|---|---|
| Interview `interview-demo-042-run` is bound to revision 2 with questions `background`, `screening`, and `compensation` | Those questions and revision 2's stage budgets govern the live interview. |
| The candidate answers `background` and the model asks one follow-up about that answer | `background` is answered, and the follow-up is recorded within the revision's limit. |
| The model proposes `compensation` while `screening` has no committed handoff | The stage does not change. |
| The connection drops once after the `background` checkpoint | The same run resumes at that checkpoint in a new voice session. |
| The final transcript required at closing is missing when the 15-minute deadline ends | The interview fails. Progress committed before that point remains. |
| Revision 2 allows compensation discussion and refers to salary ceiling `22222222-2222-4222-8222-222222222222` | The interview can discuss expectations. The ceiling value is not spoken or shown. |

## Known gaps

Not yet implemented — verify against code after implementation.

## Open questions

- **Platform and provider owner:** Which provider events count as finalized speech? The technical design leaves the event mapping unresolved, and transcript rules cannot be proven until that mapping is chosen.
- **Product owner:** The accepted recovery scope is one deliberate disconnect and reconnect. Further kinds of realtime interruption are outside this capability until a later decision extends them.
- **Product owner:** Confirm that each **Why** line captures the intended consequence. These lines interpret the approved design. They are not separate stakeholder statements.

## Related

- Sibling capability: [Candidate Interview](./candidate-interview.md) — consent, the candidate screen, takeover, and early stop.
- Sibling capability: [Execution Lifecycle](./execution-lifecycle.md) — the interview's identity, activation, completion, and attempt fencing.
- Sibling capability: [Interview Reporting](./interview-reporting.md) — uses the finalized record and does not decide when the interview completes.
- Sibling capability: [Agent Configuration and Publication](./agent-configuration-publication.md) — the bound revision supplies questions, stages, and compensation mode.
- Source document: [Architecture design, section 10](../../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md#10-reference-agent-a--hr-interviewer) — HR interview behavior, handoff, transcript, and recovery.
- Source document: [Scope freeze, sections 6.2 and 7.2](../../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) — showcase depth and H-01..H-09, H-11, and H-12.
- Source document: [Technical Design v0.1, section 7](../../../../design/core-ai/technical-design.md) — stage order, omissions, handoff, and completion.
