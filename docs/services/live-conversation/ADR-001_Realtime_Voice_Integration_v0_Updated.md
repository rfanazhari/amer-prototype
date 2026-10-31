# ADR-001 — Realtime Voice Integration Strategy

| Field | Value |
|---|---|
| Status | Proposed — pending prototype validation |
| Date | 2026-10-31 |
| Project | AI Interview Agent |
| Scope | Prototype v0 — Live Voice Integration |
| Related specification | FSD — AI Interview Agent: Prototype v0 |

## 1. Context

The AI Interview Agent initiative requires live, bidirectional voice conversation. The first prototype must establish whether this capability can be integrated with an **existing Golang backend** and a browser application.

At this stage, the decision covers **connectivity, media flow, and autonomous conversation lifecycle**, rather than interview business orchestration. The selected approach must be straightforward to test locally and protect the long-lived OpenAI API key.

## 2. Decision (Proposed)

Adopt **browser-native WebRTC** for media transport between the browser and **OpenAI Realtime API**. Use the **Golang backend for server-mediated session initialization and SDP negotiation**.

Proposed connection flow:

1. Browser captures microphone audio and creates a WebRTC offer.
2. Browser sends the offer to Go over HTTP(S).
3. Go makes the authenticated provider-facing setup request and returns the SDP answer.
4. Browser establishes WebRTC and exchanges audio directly with the Realtime service.
5. Once the connection is ready, the browser triggers the first AI response using the configured interviewer instructions.
6. Realtime API detects candidate speech turns, maintains conversation context, and generates subsequent spoken responses automatically.
7. Browser ends the session by closing the peer connection and local media resources.

The provider API key remains on the Go server. The browser may use a WebRTC data channel for realtime events and local diagnostics.

### 2.1 Conversation Lifecycle

1. **Session ready:** the Realtime session is configured with a concise AI-interviewer persona, Indonesian-language preference, and automatic turn detection/response behavior.
2. **AI starts:** the browser explicitly requests the first model response after the session is ready. A successful WebRTC connection alone does not guarantee that the AI begins speaking.
3. **Candidate responds:** browser microphone audio streams over WebRTC to the Realtime service.
4. **Turn detection:** the Realtime service detects the end of the candidate's turn using its configured VAD/turn-detection behavior.
5. **Contextual continuation:** the Realtime model processes the candidate's utterance together with the conversation context retained in that active session, then generates its next spoken response.
6. **Repeat / finish:** steps 3–5 repeat until the candidate stops the session or the connection ends.

For v0, **Go does not evaluate each candidate answer or trigger each follow-up**. The Realtime service owns turn detection and normal automatic reply generation. This is a prototype-level behavior, not a guarantee that the AI follows mandatory interview questions or stage policies.

### 2.2 Responsibility Boundaries

| Component | Responsibility in v0 |
|---|---|
| Go backend | Authenticated provider setup, SDP exchange, initial session configuration, and protection of the long-lived provider API key |
| Browser | Microphone permission and capture, WebRTC connectivity, audio playback, initial AI-response trigger, manual start/stop, and basic connection diagnostics |
| OpenAI Realtime API | Turn detection (VAD), speech understanding and generation, automatic follow-up responses, and conversation context within the active session |

The initial model/voice selection and turn-detection settings are **prototype configuration**, subject to experimental validation; this ADR does not lock a particular model version.

## 3. Rationale

- **Direct audio path:** avoids adding an application-level audio relay through Go.
- **Existing backend integration:** exercises Go's ability to initialize and authorize the provider connection.
- **Browser-native capabilities:** leverages the user's microphone, playback, and WebRTC support.
- **Narrow validation boundary:** tests connection setup, media transfer, autonomous multi-turn conversation, and basic teardown without introducing interview business orchestration.

## 4. Alternatives Considered

| Option | Assessment |
|---|---|
| Browser WebRTC + Go-mediated setup (**chosen for trial**) | Direct browser-to-provider audio path; Go handles authenticated setup |
| Browser → Go WebSocket audio relay → provider | More server involvement in streaming; not necessary to prove the initial integration boundary |
| Server-side realtime audio client | Useful for other runtime scenarios but does not directly validate the intended browser voice experience |

## 5. Consequences and Trade-offs

**Positive:** a small end-to-end system; long-lived credentials remain server-side; browser handles native microphone/audio concerns.

**Trade-offs:** browser network/device permissions affect reliability; setup and session negotiation require provider-specific integration; Go does not automatically receive every media/conversation event when audio flows directly browser-to-provider. Automatic conversation behavior depends on the selected model and Realtime turn-detection configuration; conversational consistency must be validated experimentally. Backend-side orchestration will need a separate design/decision if required in subsequent versions.

## 6. Validation Plan

Use the accompanying FSD to validate:

- Successful Go-mediated Realtime session setup.
- WebRTC connection and browser microphone access.
- Indonesian speech input and audible AI speech output.
- AI initiates the interview with an audible greeting and opening question after the client explicitly triggers the first response.
- Realtime turn detection automatically leads to an AI response after the candidate finishes speaking, without Go manually processing each answer.
- At least three candidate/AI exchanges in the same session, with a contextual follow-up referencing an earlier answer (e.g., RabbitMQ).
- Normal manual termination and visible error handling.

Record the results and any relevant provider/API constraints. **Do not mark this ADR Accepted solely because the architecture is plausible; acceptance depends on a working prototype.**

## 7. Decision Outcome

**Pending.** After the v0 experiment, update this section with test date, results, known limitations, and final status (`Accepted`, `Revised`, or `Rejected`).

## 8. Reference

- OpenAI Realtime WebRTC guide: https://platform.openai.com/docs/guides/realtime-webrtc
- OpenAI Realtime conversation and turn detection guide: https://platform.openai.com/docs/guides/realtime-vad
- Internal: FSD — AI Interview Agent: Prototype v0
- Internal: AI Interview Agent — MVP Draft Summary v0.1 (broader product context)
