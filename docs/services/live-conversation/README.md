# Live conversation

Prototype v0 checks whether the existing Go service and a browser can hold one live spoken interview. Success is a continuing voice session, not an HR interview product.

The documents below are the source. This page is only the map.

| Topic | Document |
| --- | --- |
| Concept | [AI Interview Agent Prototype v0 FSD](AI_Interview_Agent_Prototype_v0_FSD.md) |
| Decision | [ADR-001 Realtime voice integration](ADR-001_Realtime_Voice_Integration_v0_Updated.md) |
| Capability | [Voice session lifecycle prototype](business/capabilities/voice-session-lifecycle-prototype.md) |
| Technical design | [technical-design.md](voice-session-lifecycle-prototype/technical-design.md) |
| Tactical strategy | [tactical-strategy.md](voice-session-lifecycle-prototype/tactical-strategy.md) |
| Task plan | [task-plan.md](voice-session-lifecycle-prototype/task-plan.md) |
| Decision log | [decision-log.md](voice-session-lifecycle-prototype/decision-log.md) |
| Implementation gaps | [implementation-gaps.md](voice-session-lifecycle-prototype/implementation-gaps.md) |
| RCA: invalid offer | [rca-invalid-offer-sdp-eof.md](voice-session-lifecycle-prototype/rca-invalid-offer-sdp-eof.md) |

Status: planned. No voice implementation is in the service yet. Task Tracking is a separate capability.

## 1. Concept

Source: [FSD](AI_Interview_Agent_Prototype_v0_FSD.md). Draft, implementation baseline.

A test candidate opens a local page, starts one voice session, speaks Indonesian, hears the interviewer, continues for several turns, and stops. The interviewer is a simple persona. Questions are conversational. The browser captures the microphone, plays audio, and keeps the WebRTC connection. Go sets up the provider session and keeps the long-lived API key on the server. Audio after setup goes from the browser to the provider.

In scope: start, microphone permission, spoken Indonesian with English technical terms allowed, audible replies, at least three turns in the same session, stop, and a visible error when permission, setup, or the connection fails.

Out of scope: HR question banks, adaptive policy, compensation, scoring, and a transcript kept after the session. Trial speech uses synthetic candidate information.

Pass means the acceptance list in the FSD succeeds and ADR-001 can then be marked Accepted from observed results. A failure is recorded against the boundary that broke: setup, permission, negotiation, media, the spoken reply, or teardown.

## 2. ADR

Source: [ADR-001](ADR-001_Realtime_Voice_Integration_v0_Updated.md). Proposed, pending a working prototype.

The browser uses WebRTC. Go performs the authenticated setup and SDP exchange. The provider detects turns and produces later spoken replies. Go does not judge each answer or choose each follow-up.

The browser requests the first interviewer response only after the session is ready. A connected peer alone is not the greeting. Stop closes the peer connection and the local microphone.

The long-lived provider key stays on the server. Model, voice, and turn-detection values are prototype configuration, not a locked product choice. Transport for this prototype stays with this ADR.

## 3. Capabilities

Source: [voice session lifecycle prototype](business/capabilities/voice-session-lifecycle-prototype.md). Rules `BR-LIVE-POC-001`–`004`, `010`–`014`, and `020`–`022`.

One voice session runs from an explicit Start until that session ends or fails.

| Rules | Agreement |
| --- | --- |
| `001`, `002`, `004` | Start is manual. Stop releases the microphone and the connection. A failed attempt waits for another Start. |
| `003`, `012` | Later turns stay in the same session, and a later reply refers to an earlier answer. |
| `010`, `011` | Speech is Indonesian. After the session is ready, the first requested response greets the candidate and asks one opening question. |
| `013`, `014`, `022` | No score, no mandated question bank, and no transcript kept after the session. |
| `020`, `021` | The long-lived provider key stays on the server. Permission, setup, and connection failures are visible. |

Sibling capabilities: none in this service. Task Tracking uses `BR-AMER` and is not part of this flow.

## 4. Tripack

Planning for the capability above. The design is v1, dated 2026-10-31, and is not implemented.

### Technical

Source: [technical-design.md](voice-session-lifecycle-prototype/technical-design.md).

The Go process gains two routes and no database change:

- `GET /voice-session` serves one vanilla page.
- `POST /voice-sessions` accepts the browser SDP offer and returns the provider SDP answer in the standard JSON envelope.

The page asks for the microphone before setup. One peer connection carries later turns. When its data channel opens, the page sends one `response.create`. Provider turn detection creates the later replies. Setup failure, denied permission, and a dropped connection are shown, and nothing starts a replacement session.

`Negotiate` owns the interviewer instructions: Indonesian, one opening question, contextual replies, no score, and no HR question list. The provider adapter sends the server key to `POST /v1/realtime/calls`. The public response contains the SDP answer only. A missing key or a provider failure is `503 SESSION_UNAVAILABLE`. Task routes and health stay as they are.

No voice aggregate is added. Go does not see later turns, so it does not keep a session state machine. No score and no transcript are stored.

Business-driven consequences are recorded in the [decision log](voice-session-lifecycle-prototype/decision-log.md): server-held key, no per-turn orchestration, no retained transcript or score, no automatic retry, and no sign-in while the trial stays local.

### Tactical

Source: [tactical-strategy.md](voice-session-lifecycle-prototype/tactical-strategy.md).

The change is additive. Work order is the setup use case, then the provider adapter, HTTP contract, and page, then process composition. One branch is enough. Each task commit uses its ticket ID from the [task plan](voice-session-lifecycle-prototype/task-plan.md), for example `LIVE-POC-T1: <why>`.

Automated tests use a local provider double and the existing Makefile checks. They do not call the live provider. The spoken proof — greeting, three turns, contextual reply, stop, and visible failure — is a manual check that waits on a local API key, a browser, and a microphone.

Rollback removes the voice route and page. There is no voice migration to undo.
