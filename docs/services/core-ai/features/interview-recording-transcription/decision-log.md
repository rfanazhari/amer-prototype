# Decision Log: Interview Recording and Transcription

Business-driven technical decisions for the additional feature. These are planning decisions, not proof of delivered behavior.

## D1: The interview finishes before recording processing

- **Business driver:** The owner accepted asynchronous processing after COMPLETED on 2026-11-03. Candidate completion must not depend on the later account (BR-CORE-047 and reporting D2).
- **Technical decision:** Commit completion and durable processing intent together. Background workers wait for the recording, download it and transcribe it. A later failure changes the transcription job, never the interview outcome.
- **What this means for the product:** The candidate can leave immediately after completion. HR's evidence and report can become available later.
- **Alternatives considered:** Waiting for transcription in the completion request would delay the finished screen. Starting only an in-memory goroutine would lose work after a restart.
- **Trade-offs accepted:** Completed does not mean the transcript is ready. Older capability wording requiring a final transcript before completion needs reconciliation in T2; this plan does not claim that original acceptance criterion is met.
- **Revisit when:** Product explicitly requires candidate completion to wait for a verified transcript.
- **Status:** active

## D2: Stored audio replaces the earlier primary transcript source

- **Business driver:** The owner selected recording download and post-interview transcription as the primary source on 2026-11-03, replacing Interview Runtime todo-gap G14's previous sideband transcript collection direction.
- **Technical decision:** Request recording storage for eligible new sessions, retrieve every session's recording after completion and finalize one ordered evidence revision. Sideband remains available for controls/errors and recording-close confirmation, rather than a second canonical transcript pipeline.
- **What this means for the product:** Candidate and interviewer speech can both appear in the final record, including sessions before a reconnect. Missing audio is visible as a technical gap.
- **Alternatives considered:** Keeping sideband transcript events as primary would retain G14 but would not implement the owner's chosen source. Running two independent primary transcripts would leave conflicting evidence to reconcile.
- **Trade-offs accepted:** Provider storage permission, candidate notice, retention and transcription timing must be settled before rollout. Old sessions cannot be made recorded retroactively. No successful recording download is claimed yet.
- **Revisit when:** Stored recording cannot preserve the required interview evidence or the approved data policy prohibits it.
- **Status:** active, under review: the [per-stage proposal](proposal-per-stage-live-transcript.md) (D5) would make live sideband events the primary source and keep recordings for audit and recovery. Nothing changes until the spike (`CORE-IRTREC-012`) result and an owner decision. Supersedes G14's source direction only. It does not supersede live control/error-reading requirements or claim streaming-finality Q1 is solved.

## D3: Download is a backend processing step

- **Business driver:** The owner selected “backend for transcription only” on 2026-11-03.
- **Technical decision:** Only the backend calls the provider recording endpoint. No HR or candidate recording-download route or button is added.
- **What this means for the product:** This feature prepares evidence for reports; it does not introduce an audio-library workflow for HR.
- **Alternatives considered:** Providing HR downloads would need a separate access and retention contract beyond this request.
- **Trade-offs accepted:** HR cannot replay or export audio through this feature. Recording and temporary text retention remain explicit rollout decisions, not permanent storage by default.
- **Revisit when:** Product requests authorized audio replay/export.
- **Status:** active
