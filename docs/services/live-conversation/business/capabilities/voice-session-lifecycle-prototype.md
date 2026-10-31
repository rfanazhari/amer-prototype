---
id: voice-session-lifecycle-prototype
service: live-conversation
status: planned
rules: BR-LIVE-POC-001..BR-LIVE-POC-004, BR-LIVE-POC-010..BR-LIVE-POC-014, BR-LIVE-POC-020..BR-LIVE-POC-022
updated: 2026-10-31
implementation:
---

# Voice session lifecycle prototype

## Purpose

A test candidate can open one live voice session, hear an interviewer start the conversation, continue it across later turns that still refer to an earlier answer, and stop it. The team uses that session to judge whether a live spoken conversation is feasible. Scoring, a mandated question bank, compensation, and a transcript kept after the session are outside this capability. Trial conversations use synthetic candidate information.

## Ubiquitous language

| Term | Meaning |
|---|---|
| **Voice session** | One live spoken conversation, from the candidate's manual start until that session ends or fails. |
| **Turn** | One side's spoken part. The interviewer speaks, or the candidate speaks, and the other side can respond in the same session. |
| **Interviewer** | The simple AI interviewer persona attached to the session. It carries the conversation; it does not apply an HR policy. |
| **Test candidate** | The person who starts, speaks in, and stops the prototype session. |

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| Start conversation | Test candidate | A new voice session is requested, and microphone access is required before speech is accepted. |
| Session ready and the first response is requested | Interviewer | The interviewer audibly greets the candidate and asks one opening question. |
| Candidate finishes a later turn | Interviewer | The next audible reply stays in the same session and refers to an earlier answer. |
| Stop conversation | Test candidate | The session ends and the microphone and connection are released. |
| Permission denied, setup fails, or the connection drops | Voice session | The failure is shown, and a new attempt waits for another manual start. |

## Rules

### BR-LIVE-POC-001 — A voice session starts only when the test candidate asks

**Given** the candidate is at the prototype and no voice session is active
**When** the candidate starts the conversation
**Then** a new voice session is opened for that request, and candidate speech is accepted only after microphone access is granted

**Why:** Without an explicit start, the prototype could open a microphone and a billed provider session the candidate did not ask for. A feasibility trial cannot count sessions nobody chose to begin.

### BR-LIVE-POC-002 — Stopping releases the session

**Given** a voice session is active
**When** the candidate stops the conversation
**Then** the session ends, and the microphone and the connection are released

**Why:** If the microphone stays open after the candidate is done, nobody can tell the session has ended. The next start would be mixed with a capture that was supposed to be finished.

### BR-LIVE-POC-003 — Later turns stay in the same session

**Given** a voice session is connected
**When** the candidate and the interviewer exchange further speech
**Then** those turns continue in that same session

**Why:** A new session on every turn would never show that one live conversation can continue. That continuation is the outcome this prototype exists to establish.

### BR-LIVE-POC-004 — A failed session is retried only by a new manual start

**Given** microphone permission was denied, setup failed, or the connection dropped
**When** that failure has been shown
**Then** the system waits for the candidate to start again and does not open the next session by itself

**Why:** An automatic retry would make a pass look like a session the candidate recovered on purpose. The team would also lose a clear view of which attempt actually failed.

### BR-LIVE-POC-010 — The session is spoken in Indonesian

**Given** a voice session is active
**When** the candidate and the interviewer speak
**Then** the conversation is Indonesian, and English technical terms are accepted

**Why:** An English-only session would miss the interview this prototype is meant to try. English terms stay valid because the subject matter is often named in English.

### BR-LIVE-POC-011 — The interviewer opens with a greeting and one question

**Given** a voice session is ready
**When** the first response is requested
**Then** the interviewer audibly greets the candidate in Indonesian and asks one opening question before the candidate's first answer

**Why:** A ready connection that stays silent is not an interview. The candidate has to hear that the conversation has started. This opening is a base rule of the prototype, not a later check that can be skipped.

### BR-LIVE-POC-012 — A later reply refers to an earlier answer

**Given** the candidate has already answered in this voice session
**When** the candidate finishes a later turn
**Then** the interviewer's next audible reply uses this session's context and refers to an earlier answer from this session

**Why:** A reply that ignores what the candidate already said is a new prompt, not a continuing conversation. One session has to carry an earlier answer into the next spoken reply. This reference is a base rule of the prototype. A fixed HR script is not required for it.

### BR-LIVE-POC-013 — An answer is not scored

**Given** the candidate has spoken in the session
**When** that turn ends
**Then** the session continues without recording a score, pass, or fail for the answer

**Why:** A stored grade would make a connectivity trial look like a candidate assessment. A missing grade would then be read as a product failure even when the conversation itself continued.

### BR-LIVE-POC-014 — The next reply is not taken from a mandated question bank

**Given** the interviewer is about to reply
**When** the session continues
**Then** the reply is a conversational continuation, and no mandated HR question list or backend interview policy selects it

**Why:** Mandatory questions, adaptive policy, and compensation belong to the later interview workflow. Requiring them here would hold a live session hostage to a product this prototype is not deciding.

### BR-LIVE-POC-020 — The long-lived provider key stays on the server

**Given** the browser needs a voice session
**When** the session is set up
**Then** the long-lived provider API key is used only on the server and is not sent to the browser

**Why:** That key is a billed credential that outlives one session. If the page can read it, anyone who opens the prototype can spend the account, and the trial is no longer a closed local test.

### BR-LIVE-POC-021 — Session failure is visible

**Given** the candidate tried to start, or a voice session is underway
**When** microphone permission is denied, setup fails, or the connection drops
**Then** the candidate sees an error or disconnected state and a useful failure message

**Why:** A silent failure leaves no evidence. The team cannot decide whether permission, setup, or the live connection was the boundary that failed.

### BR-LIVE-POC-022 — The session does not keep a durable transcript

**Given** a voice session is exchanging speech
**When** the session ends or fails
**Then** the prototype is not required to keep a transcript after the session

**Why:** A transcript kept after the session would turn a connectivity trial into retained speech. This prototype has not accepted that record. The words are needed only while the session is live.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| Microphone permission is denied | No candidate speech is accepted. An error is shown. The next attempt is a new manual start. | `BR-LIVE-POC-001`, `BR-LIVE-POC-004`, `BR-LIVE-POC-021` |
| Provider setup fails | An error is shown. The long-lived provider key is not in the browser. Retry is a new manual start. | `BR-LIVE-POC-004`, `BR-LIVE-POC-020`, `BR-LIVE-POC-021` |
| The connection is ready, and the first response has not been requested | The interviewer has not greeted the candidate yet. Readiness alone is not the opening. | `BR-LIVE-POC-011` |
| The candidate answers with an English technical term inside Indonesian | The term is accepted as part of the Indonesian session. | `BR-LIVE-POC-010` |
| The candidate speaks a second and third turn | Both stay in the original session. The next audible reply refers to an earlier answer. No score is stored. | `BR-LIVE-POC-003`, `BR-LIVE-POC-012`, `BR-LIVE-POC-013` |
| The connection drops during a turn | The drop is shown. That session does not continue. A new attempt is manual. | `BR-LIVE-POC-003`, `BR-LIVE-POC-004`, `BR-LIVE-POC-021` |
| The candidate stops, then starts again | The first microphone and connection are released. The new start is a new session. | `BR-LIVE-POC-002`, `BR-LIVE-POC-001` |
| A reply would have to come from a fixed HR question list | The session does not select from that list and does not score the answer. | `BR-LIVE-POC-014`, `BR-LIVE-POC-013` |
| The session ends | No transcript is required after the session. | `BR-LIVE-POC-022` |

## Examples

| Input | Expected outcome |
|---|---|
| The candidate starts the conversation, allows the microphone, the session becomes ready, and the first response is requested. | An audible Indonesian greeting and one opening question play before the candidate answers. |
| The candidate says "Saya memakai RabbitMQ untuk antrian", then answers a follow-up in the same session. | The interviewer's next audible reply refers to RabbitMQ or to that earlier answer. The session is not restarted, and no score is stored. |
| The candidate stops the conversation. | The microphone and the connection are released. |
| The candidate denies microphone permission. | An error is shown. A later attempt requires a new manual start. |
| Provider setup fails before the session is ready. | An error is shown. The long-lived API key is not available to the browser. |

## Known gaps

Not yet implemented — verify against code after implementation.

## Open questions

None.

## Related

- Existing flow reference: [`AI_Interview_Agent_Prototype_v0_FSD.md`](../../AI_Interview_Agent_Prototype_v0_FSD.md) — prototype objective, user flow, and acceptance evidence for one live voice session.
- Sibling capabilities: none in this service yet. Task tracking ([`task-tracking`](../../../amer-prototype/business/capabilities/task-tracking.md)) uses `BR-AMER` and is a different capability.
- Source document: [`ADR-001_Realtime_Voice_Integration_v0_Updated.md`](../../ADR-001_Realtime_Voice_Integration_v0_Updated.md) — session lifecycle and the base opening and contextual-reply behaviour. Transport, model choice, and turn-detection settings stay in that ADR.
- Naming mark: the capability id matches `*prototype`.
- The FSD and ADR also cite "AI Interview Agent — MVP Draft Summary v0.1". That summary is not in this repository. Its interview workflow is a later capability, not a rule of this one.
