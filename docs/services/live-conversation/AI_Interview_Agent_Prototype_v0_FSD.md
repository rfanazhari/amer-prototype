# FSD — AI Interview Agent: Prototype v0 (Live Voice Integration)

| Field | Value |
|---|---|
| Document | Functional Specification Document (FSD) |
| Version | 0.1 |
| Status | Draft — implementation baseline |
| Project | AI Interview Agent / Internal AI Agent Challenge |
| Scope | Prototype v0 — technical integration feasibility |
| Related decision | ADR-001 — Realtime Voice Integration Strategy |

## 1. Objective

Validate whether the existing **Golang backend** can be integrated with **OpenAI Realtime API** to enable a **live, bidirectional voice conversation** in a browser.

**Success is technical feasibility**, not validation of a complete HR interview workflow.

## 2. Scope

A user opens a minimal local web page, starts a voice session, speaks in Indonesian, hears AI responses, continues for multiple turns, and ends the session.

The AI uses a simple **interviewer persona** via initial session instructions. Questions may be conversational; they are not yet enforced by backend business rules.

## 3. Actors and Components

| Component | Responsibility |
|---|---|
| User (test candidate) | Starts/stops conversation and speaks through microphone |
| Browser (Vanilla HTML/JavaScript) | Minimal UI, microphone capture, audio playback, WebRTC connection |
| Existing Golang backend | Authenticated session setup / SDP exchange; keeps provider API key server-side |
| OpenAI Realtime API | Realtime speech input, conversation processing, and audio response |

## 4. Functional Requirements

| ID | Requirement | Expected behavior |
|---|---|---|
| F-01 | Initialize session | Browser requests setup; Go handles provider-facing connection initialization without exposing the long-lived API key |
| F-02 | Start voice conversation | User starts a session and grants microphone access; browser establishes WebRTC connection |
| F-03 | Receive user speech | User speaks in Indonesian and the AI responds based on the utterance |
| F-04 | Play AI response | Generated audio is played through the browser |
| F-05 | Support multi-turn dialogue | Conversation continues for several back-and-forth turns in one session |
| F-06 | Stop conversation | User stops the session; browser closes microphone tracks, data channel (if used), and peer connection |
| F-07 | Display basic state and errors | UI shows connecting/connected/disconnected/error and a useful failure message |

## 5. User Flow

1. User opens the prototype page on localhost.
2. User clicks **Start Conversation**.
3. Browser requests microphone permission and starts session initialization through Go.
4. Browser negotiates WebRTC with the Realtime service using the setup result.
5. User and AI exchange audio over multiple turns.
6. User clicks **Stop Conversation**; session resources close.

**Error path:** on permission denial, setup failure, or connection drop, show an error/disconnected state and allow a new manual attempt.

## 6. UI Minimum

- **Start Conversation** / **Stop Conversation** buttons.
- Connection status: `Idle`, `Connecting`, `Connected`, `Disconnected`, `Error`.
- Optional debugging panel for connection events (no durable transcript required).

## 7. Prototype Configuration

| Item | Baseline |
|---|---|
| Backend | Existing Golang boilerplate |
| Frontend | Vanilla HTML + JavaScript |
| Transport | Browser WebRTC |
| AI service | OpenAI Realtime API |
| Model | Realtime-capable model; exact model ID to be validated at implementation time |
| Language | Indonesian; English technical terms allowed |
| Persona | Simple AI interviewer instructions |
| Secrets | OpenAI API key stored server-side in environment/config |
| Environment | Localhost, browser with WebRTC and microphone permission |

## 8. Acceptance Criteria / Definition of Done

- [ ] Existing Go backend successfully performs Realtime session setup.
- [ ] Browser connects with WebRTC and accesses the microphone with permission.
- [ ] AI understands a spoken Indonesian prompt sufficiently to give a relevant answer.
- [ ] AI audio is audible through the browser.
- [ ] At least **three consecutive conversation turns** work in the same session, without restarting it.
- [ ] Manual Stop ends the connection and releases microphone access.
- [ ] A failed setup or denied microphone permission is visibly reported rather than silently failing.

**Evidence to capture:** short demo recording or manual test notes, connection result, observed errors, and chosen model/configuration. Any latency observations are diagnostic, not a formal performance target for v0.

## 9. Assumptions and Constraints

- An OpenAI account/API key with access and billing for an appropriate Realtime model is available.
- A browser with microphone access is available; `localhost` is used for local testing.
- Go mediates session setup; **the browser sends audio directly to the Realtime service after WebRTC negotiation**.
- Prototype test conversations use synthetic/non-sensitive candidate information.
- Voice model availability, request details, and session configuration will be checked against provider documentation during implementation.

## 10. Exit Decision

**Pass:** all core acceptance criteria succeed; integration feasibility is demonstrated and ADR-001 may be marked *Accepted* based on observed results.

**Fail / revisit:** document which boundary fails (setup, permission, negotiation, media, AI response, or teardown) before choosing an alternative architecture.

## 11. Relationship to MVP

The broader MVP covers HR-defined questions, adaptive follow-ups, orchestration, compensation policy, and evidence-based reporting. Those goals remain documented in the **AI Interview Agent — MVP Draft Summary v0.1**. **Prototype v0 tests only the foundational live voice connection.**
