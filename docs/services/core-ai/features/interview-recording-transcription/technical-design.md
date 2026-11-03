# Technical Design: Interview Recording and Transcription

Date: 2026-11-03. Mode B: an additive extension to implemented Interview Runtime. Planning only; no recording download or transcription pipeline is delivered by this document.

## 1. Overview

After an interview commits `COMPLETED`, the backend asynchronously obtains its stored OpenAI Live recordings, transcribes them, and publishes one finalized evidence revision. The candidate does not wait for recording finalization, download, transcription, or reporting. The owner selected backend-only downloads and recording-based transcription as the primary source, replacing the earlier Interview Runtime G14 sideband-transcript direction (2026-11-03 chat).

### Goals and boundaries

- Enable provider recording for new eligible sessions; preserve every local/provider session and attempt association across reconnects.
- Recover work after a crash, process duplicates safely, and distinguish unavailable audio from a candidate omission.
- Preserve speaker, session, attempt, chronological order, and immutable entry/revision identity in final evidence.
- Keep transcript failure separate from the already completed interview and from report generation.
- Provide a stable read boundary for Interview Reporting T14. Report writing remains IRP T13; this feature does not resolve reporting Q2.

No HR/candidate audio-download endpoint or UI, browser recorder, live caption requirement, score, hiring verdict, automatic coverage inference, provider transcript-event classifier, or report runtime registration is added. Failed/cancelled executions are not transcribed by this initial flow. Their recording cleanup must still follow the approved policy.

Proposed rollout switch: `INTERVIEW_RECORDING_ENABLED=false` by default. Disabled mode retains current voice/completion behavior. Enabled mode requires the approved notice, storage limits/location/retention and broker configuration; validate local configuration within the existing startup budget and reject incomplete configuration before listening. Project recording permission is established by T1/rollout proof rather than an unbounded startup network probe. Exact storage setting names follow the Q2 adapter choice. No credentials appear in example values.

### Verified context and authority

- Root `AGENTS.md` and `CLAUDE.md` require four root layers, documented declarations, injected UTC millisecond time, manual migrations, and bounded shutdown. They override the bundled standard's prohibition on delivery tests and its domain repository placement: retain application ports and existing handler-test conventions.
- `go.mod`: Go 1.26.10, GORM 1.25.12, clientmanager v0.10.1, logmanager v1.44.1, sqlmock v1.5.2, amqp091-go v1.13.0. No new OpenAI SDK is required by this plan.
- `infrastructure/service/interviewvoice/driver.go`: `Driver.Create` already uses `/v1/live/sessions`, retains provider identity, and does not send `store`. `Driver.Close` in `driver_sideband.go` sends `session.close` and immediately releases the socket; it does not confirm recording finalization.
- `delivery/http/candidateinterview/page.html`: microphone and remote playback use WebRTC; no recording/download implementation exists.
- `application/repository/interviewport/store.go` and migration `000010`: session/attempt correlation and `interview_transcript_entries` already exist. There is no need to invent a missing transcript table.
- `interviewrepository.writeEntries` chooses the latest session of a stage and writes revision 1. That mapping is unsuitable for importing old/replaced sessions or corrections.
- `interviewlifecycle.Lifecycle.WithProgress` fences active work; it must not be weakened to permit post-terminal transcription. `Lifecycle.Complete` delegates to the generic execution completion transaction. The assignment `completed_at` marker required by report admission is a separate reporting integration concern.
- Schema release is 13. `outbox_messages` accepts only `interview_report`; report outbox claims are not filtered by kind. Do not insert transcription work into that table or route it through the report consumer.

### Provider facts versus planned proof

Official documentation inspected on 2026-11-03:

1. [Managing GPT-Live sessions](https://developers.openai.com/api/docs/guides/live-conversations): recording storage is selected with `session.store: true` at creation; the project must support storage. ZDR prevents storage. A completed recording downloads as stereo WAV, input left and output right. Storage finalization can extend close time.
2. [Download recording](https://developers.openai.com/api/reference/go/resources/live/subresources/sessions/methods/download_recording): backend fetches `/v1/live/sessions/{session_id}/content` using stored session identity.
3. [File transcription](https://developers.openai.com/api/docs/guides/speech-to-text): file upload through `/v1/audio/transcriptions`, maximum 25 MB; the current starting model is `gpt-transcribe`.

These documents establish capabilities, not account access, measured readiness times, successful downloads, or the timestamp fidelity needed here. T1 must capture a sanitized live result before the dependent adapters and schema are finalized. Do not guess a readiness status code or timestamp response field. Stereo channel identity avoids inferring candidate/model from anonymous speaker labels; segment alignment still needs proof.

## 2. API / Interface Contract Changes

Public candidate/admin payloads, status codes, error precedence and routes remain unchanged. No public download endpoint is planned. Review existing consent notice wording before enabling provider storage; if it changes, issue a new notice version through the existing consent mechanism rather than editing an already accepted notice silently.

Proposed application ports (names are internal choices):

- `RecordingSource.Download(ctx, providerSessionID)`: returns a bounded stream plus verified media metadata; distinguishes pending, unavailable, permanent rejection and transient failure. The caller owns closing the stream.
- `AudioTranscriber.Transcribe(ctx, monoChunk)`: returns segments with text and verified timing, never a competence judgement. Request includes language/context only when appropriate; excludes the salary ceiling and unrelated configuration.
- `TranscriptionStore`: admit, claim with lease/token, save checkpoints, commit a complete evidence revision, read finalized evidence, and reconcile overdue work. Transaction contexts hide GORM.
- `FinalizedTranscriptReader`: execution-scoped immutable revision and ordered entries with body, speaker, session, attempt and optional question association. It returns no readable final revision while the initial manifest is incomplete.

Mock interfaces with Mockery; SQL expectations use sqlmock. Production adapters must preserve the same failure and stream-ownership contracts.

## 3. Data Model & Schema Changes

Proposed migration pair: `migrations/pgsql/000014_interview_recording_transcription.up.sql` and `.down.sql`. Version 14 is **proposed, not reserved**: recheck the migration head before implementation. Preserve versions 1–13 and their checksums. Update release guard, grants, checksum inventory, runner target limits, isolated verification scripts and migration runbook together when implementing.

Proposed relational additions (finalize with T1/T2; no audio blob in PostgreSQL):

- Add nullable `recording_enabled` to `interview_voice_sessions`: true/false is the selection persisted when a new provider session is created; null identifies legacy sessions whose recording configuration is unknown. Do not backfill null to true. Persist provider-finalization observations with the corresponding source, independently of local `closed_at`.
- `interview_transcription_jobs`: UUID PK; `execution_id` FK interview execution and unique; state `pending|processing|retry_wait|succeeded|failed|unavailable`; `completed_at` copied from authoritative completion; `deadline_at`; nonnegative `attempt_count`, `fencing_token`; nullable lease/eligibility instants; nullable safe `failure_reason`; nullable positive `published_revision`; created/updated timestamps. Check lease fields against processing state and revision against succeeded state. Index eligible nonterminal jobs. Retry never updates execution state.
- `interview_transcription_sources`: UUID PK; job/session FKs; unique `(job_id, session_id)`; provider session identity; explicit attempt identity; deterministic `source_order`; state `pending|downloaded|transcribed|unavailable`; nonnegative `download_attempt_count`; nullable next-eligible instant, safe failure reason and provider-finalized instant; nullable digest, byte count, media format, protected audio-object key and expiry; created/updated timestamps. Positive ordering and nonnegative byte counts; unique `(job_id, source_order)`. Never store signed URLs or credentials. Reject a session from another execution inside the locked admission transaction; add a composite FK if the final schema can enforce that association cleanly.
- `interview_transcription_entries`: mapping from new canonical `interview_transcript_entries.id` (unique FK) to job/source/revision, `segment_key`, source-relative start/end milliseconds and channel. Unique `(source_id, revision, segment_key)`; checks enforce positive revision, valid channel, and `0 <= start_ms < end_ms`. Index `(job_id, revision, source_id, start_ms)` for batched ordered reads. Source identity must match the job and canonical row session/attempt in the same commit.
- `interview_transcription_chunks`: UUID PK; source FK; channel and positive chunk ordinal; start/end offsets; state `pending|processing|succeeded|failed`; nonnegative attempt count; nullable next-eligible instant, safe failure reason and protected result-object key/digest. Unique `(source_id, channel, chunk_ordinal)`; valid channel and offset checks. Chunk output remains private staging until final publication. Preserve checkpoints and attempt counts across lease recovery; no provider payload in these metadata rows. The approved storage policy covers staged text as well as audio.
- `interview_transcription_outbox`: UUID PK; job FK; unique dispatch identity; created/claimed/confirmed instants. Dedicated table avoids changing the report outbox constraint and accidentally delivering transcription jobs to the report consumer.

Reuse the existing transcript table for final bodies, speaker, ordinal, revision and supersession links. No stage is inferred from `SessionModel.Stage`: current code can keep the same session while stages advance. Stage/question attribution is optional unless supported by separate committed facts; recording alone must not mark a question answered. A text-only response without reliable timing cannot satisfy the ordered-record contract and remains blocked by Q1.

GORM owns CRUD, explicit `Select` persists zero counts and null failures. No automatic associations, soft deletion, or per-entry lazy queries. Use a bounded bulk load for job, sources and entries. Raw SQL is justified only for bounded eligible-job/outbox claims using `FOR UPDATE SKIP LOCKED`. Claims commit before network I/O. Lease/token comparisons fence all checkpoint and final writes.

### Concepts, ownership and path contracts

`Transcription` is a separate post-completion aggregate keyed by execution, owning source completeness, immutable evidence revisions, job failure and publication eligibility. A recording source is an owned entity; a segment identity/timing is a value. Existing live progress/handoffs are not rewritten. This boundary is justified by completion already being committed: applying live-progress mutation rules would either reject valid asynchronous work or weaken terminal fencing.

| Rule | Owner and enforcement | Planned proof |
| --- | --- | --- |
| Candidate completion precedes provider work | Admission records intent in the same completion transaction; worker performs network I/O only after commit | Blocked provider does not delay completion response; rollback leaves no job/outbox |
| One job and fixed manifest per completed execution | Admission locks execution then job/sessions; unique execution identity; same completion replays | Concurrent admission, restart and duplicate messages create no second job |
| No fabricated completeness | Aggregate requires each expected source transcribed before publication | Missing reconnect recording leaves evidence unavailable, not a final partial account |
| No stale writer or duplicated segments | Lease token and unique source/revision/segment keys; atomic publication | Expired worker cannot publish; repeat import is idempotent |
| Revision identity and content immutable | Post-completion import port creates canonical rows and revision together | Existing references remain unchanged; failed commit exposes no new revision |
| Lost audio is not a candidate omission | Aggregate records safe unavailable reason; no coverage/lifecycle mutation | Column-for-column execution, coverage and handoff isolation |

Creation accepts only a committed COMPLETED interview with persisted provider-linked sessions created with recording enabled. Restoration preserves existing job state and identities without provider calls. Legacy sessions lacking recording configuration are `unavailable`, never retroactively assumed recorded. Transitions follow `pending -> processing -> retry_wait -> processing` or a terminal outcome; a terminal initial job is not silently regenerated. Future reprocessing uses a separately approved revision command.

No reference/master data or seeder is needed. Backfill policy: no automatic transcription of historical completed interviews. Recovery scans only the explicit rollout eligibility boundary (Q2), not every historical row. Down migration is disposable-database-only and must not remove old transcript evidence; production rollback stops workers and keeps new tables/evidence for forward repair.

## 4. Component Breakdown and Assessment

- **Domain/new** `domain/aggregate/interviewtranscription`: job/source completeness and revision rules. Retained decision: a separate post-completion boundary preserves live progress invariants; adding generic lifecycle states for transcription would reopen completed interviews.
- **Application/new** `application/usecase/interviewtranscriptionusecase`, `application/repository/interviewtranscriptionport`, provider/read ports under `application/service`: admission, reconciliation, worker and detached evidence results. Plain operation structs with injected Clock and ports. Retained DIP: provider errors and binary stream behavior can be tested without credentials.
- **Application/modified** interview completion integration: invoke admission in the existing shared transaction, never generic execution domain code. A scan repairs missed eligible intent after a restart; it is a safety net, not the only handoff.
- **Infrastructure/modified** `interviewvoice`: startup storage flag and recording-aware close confirmation. One sideband reader owns receive; close completion cannot compete with another reader. Preserve controls/error handling still needed by IRT T21/G13. Async finalization must not extend the candidate HTTP request or the process shutdown budget.
- **Infrastructure/new** `interviewrecording` and `audiotranscription` service adapters; transcription repository; broker adapter. Prefer existing clientmanager/logmanager; T1 verifies bounded binary/multipart I/O and payload-log suppression with their actual versions. If unsuitable, record a scoped adapter/library exception before adopting it.
- **Delivery/reused** current candidate completion route and page. No audio field, URL, transcript or new user action is exposed.
- **Runtime/modified** compose owned workers, publisher and reconciler only after rollout prerequisites. Cancel I/O, release sockets/temp files and preserve durable checkpoints within the existing 5-second shutdown budget (4-second drain).

Assessment: retain four layers and provider/session identity; revise G14's canonical source; introduce a separate terminal-safe import boundary; defer media alignment, storage policy and new broker topology until evidence/owner decisions. Sharing report job tables/budgets is rejected because their lifecycle, consumer and schema are report-specific. Existing sqlmock broker tests do not establish binary recording or account availability.

## 5. Flow Sequence

1. At rollout, validate recording eligibility/consent and project storage support; create new voice sessions with `session.store: true`, persisting that selection in `interview_voice_sessions.recording_enabled`.
2. The existing completion operation commits COMPLETED and one transcription job/source manifest/outbox in a shared transaction. Provider download is absent from this transaction. A failed database admission rolls back this completion attempt; previously committed completion cannot be undone by later worker failure.
3. Candidate receives the ordinary completed response. A background finalizer requests provider close and observes recording finalization. Reconcile a lost close observation through the verified provider readiness/download behavior; a local `closed_at` alone is insufficient proof.
4. Dedicated outbox publisher uses RabbitMQ persistent delivery and confirms. Reconciler recovers missed admission, unconfirmed publication and expired leases. Q3 governs the independent queue and malformed-message policy before registration.
5. Worker claims a token, then downloads each manifest source once ready. Pending recording schedules another bounded check; unavailable recording records a technical gap and prevents full publication. Do not omit an older session because a newer attempt exists.
6. Validate the WAV, split input/output channels, and transcribe bounded mono chunks. Keep original offsets; merge overlap deterministically and order by session chronology, source offset, channel and stable segment identity. Do not merge both speakers into unlabelled text. T1 must prove timing/overlap treatment before finalizing this contract.
7. In one short transaction, lock transcription job, verify current token and complete manifest, insert immutable canonical entries/mappings, publish the revision and mark job succeeded. Readers see all or none. No network calls in the transaction.
8. Reporting T14 can read the final revision after its adapter is implemented. Report admission/IRP T15 remains separately owned; preserve its 30-minute evidence wait and two writer attempts. A transcription finishing after report expiry does not silently retry or reopen that report.

## 6. Input / Output and Budgets

Broker body contains only `jobKind: interview_transcription`, `jobId` and `messageId`; identities must match broker metadata. No audio, transcript, instructions, credentials or salary material in broker messages/logs.

Recording request uses the backend key and persisted provider ID, never a client-supplied URL. Final evidence read contains execution/revision and ordered entries `{id, revision, sessionId, attemptId, speaker, body, startMs, endMs}`; `questionId` is nullable and is not invented. Exact provider multipart fields and segment response schema are T1 deliverables.

Proposed technical defaults to verify under T1: one job worker, one source/chunk at a time; 120-second provider call timeout, 150-second lease, 30-second pending-readiness checks, maximum three transient call attempts per source/chunk with 5/10-second delays, and a 30-minute job deadline from completion. Persist per-source/chunk progress so restarts do not reset attempts. These are transcription budgets, never the report writer's two-attempt budget. Q2 must set a total recording byte/duration cap, temporary storage capacity and retention before enablement; enforce the documented 25 MB upload limit per chunk regardless of configuration. Deadline expiry becomes safe failure, never execution failure.

## 7. Error Handling

Use typed internal errors for recording pending, recording unavailable, unsupported media, provider rejection, transient provider failure, invalid transcription, lost lease and storage failure. Map only safe reason codes to job state. No new public error catalog is needed without a public route. Suppress HTTP request/response body capture, multipart/audio payloads, transcript text, signed object URLs and authorization headers in clientmanager/logmanager.

ACK only after durable job disposition; do not ACK database failures. Duplicate terminal/active deliveries perform no provider work. Retries may repeat a provider call after an ambiguous network outcome; promise idempotent storage, not exactly-once provider billing. Malformed messages use only the approved Q3 isolation contract. A cleanup failure is retryable operational work and does not erase a committed evidence revision.

## 8. Constraints Checklist

- [ ] Automatic backend-only download after COMPLETED and provider readiness; candidate response remains independent.
- [ ] Recording is the primary source; raw sideband transcript collection is not a second canonical pipeline.
- [ ] Multi-session coverage, stereo speaker identity, ordering and bounded resources have real proof.
- [ ] Manual migration, terminal-safe writes, lease fencing and atomic final publication are verified on PostgreSQL.
- [ ] SQL mocks use sqlmock; interface mocks use Mockery; provider mocks do not count as live acceptance.
- [ ] Recording policy, notice and project prerequisites are resolved before enablement.
- [ ] No report Q2/T13 completion claim and no runtime registration of report routes/consumer.

## 9. Risks & Open Questions

- **Q1 — provider feasibility (provider/engineering owner; T1):** prove project storage, recording-aware close, pending/permanent-error mapping, stereo WAV shape and a transcription contract with reliable segment timing. `gpt-transcribe` is a candidate, not an approved report writer model. Blocks final adapter/schema contracts T3–T8 until evidence is synchronized. A text-only response is insufficient; choose a supported alignment method or explicitly revise scope before implementation.
- **Q2 — data and rollout policy (product/platform owner; T2):** set notice/version, provider persistence permission, temporary/durable storage location, byte/duration caps, retention/deletion and eligibility cutoff. Storage may be disabled by project/ZDR settings. Blocks storage activation, schema policy fields and production registration. Do not invent a retention period or enable recording in historical sessions.
- **Q3 — dispatch isolation (platform owner; T2):** approve the transcription queue/exchange/routing and malformed-message isolation. Reuse delivery guarantees, not the report queue/DLX ownership. Blocks broker T8 and runtime T9; no new topology is declared by this document.
- **Q4 — business-document reconciliation (runtime/product owner; T2):** existing BR-CORE-065/070 text requires transcript before handoff/completion, while G9 and this owner direction finalize it afterward. Update the capability/design authority to reflect the accepted asynchronous meaning; do not claim H-09 as originally written is satisfied by post-completion work. Blocks enabling the conflicting path, not T1 investigation.
- **Reporting handoff (IRP T13/T14 owners; T10):** current writer `EvidenceEntry` has ID/revision/question only, with no body; this alone cannot supply a narrative writer. Synchronize the body-read/revision pin contract and the actual assignment completion marker in reporting before those tasks run. Existing reporting design statements that no transcript table exists are stale. Q2 for report model/schema remains open independently.

## 10. Revision History

| Version | Date | Change | Reason |
| --- | --- | --- | --- |
| v1 | 2026-11-03 | Initial additive design; backend-only asynchronous recording transcription | Owner selected recording as canonical source, replacing G14; current implementation has no download/transcription pipeline |
