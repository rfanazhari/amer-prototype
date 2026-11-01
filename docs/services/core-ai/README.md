# Core AI

Core AI is one Go modular monolith. The baseline already in the service is Identity and Access and Agent Configuration and Publication; startup accepts clean schema version 8. This phase continues the candidate interview after the candidate already has a session. The candidate UI is in scope. This phase has no admin UI. Document Analyzer is outside this phase.

## Done

### Identity and Access

ADMIN provisions HR, and an invited candidate reaches one assignment without a permanent account. Rules `BR-CORE-020`–`BR-CORE-039`: [Identity and Access](business/capabilities/identity-access.md). Tripack: [identity-access](identity-access/task-plan.md).

The task plan records T1–T14, T16, and T18–T20 Done, including the published session, HR-account, candidate, invitation, exchange, and candidate-session routes. Retention (T15) stays blocked, T17 is still marked in progress, and the creator-scoped candidate list (T21) is in progress with its public contract outstanding.

### Agent Configuration and Publication

HR configures and publishes the HR Interviewer and Document Analyzer templates, then pins an interview assignment to the current published revision. Rules `BR-CORE-001`–`BR-CORE-014`: [Agent Configuration and Publication](business/capabilities/agent-configuration-publication.md). Tripack: [agent-configuration-publication](agent-configuration-publication/task-plan.md).

T1–T14 are recorded Done and startup accepts clean version 8 (`000008_execution_lifecycle` adds the shared execution tables), including the published configuration routes, instance list, and assignment create. There is no admin UI.

## Next

The four capabilities below are planned. None is implemented.

1. **Execution lifecycle.** One admitted run keeps a stable identity and an honest ending; a repeated start finds the same run. Rules `BR-CORE-050`–`BR-CORE-059`: [Execution Lifecycle](business/capabilities/execution-lifecycle.md). Tripack: [execution-lifecycle](execution-lifecycle/task-plan.md). Migration `000008_execution_lifecycle` raises the clean floor to 8. It needs the current version-7 execution pin, and nothing from the three capabilities below. Candidate start and interview runtime both need one real execution, so this is first. T1–T9 are executable; T10 stays blocked until a real revocation or in-interview expiry caller exists, and T11 stays blocked on a future Document Analyzer.

2. **Candidate interview.** A candidate who already holds a session reviews the notice, accepts it, enters one live interview, follows its progress, and can leave. The candidate UI is in this phase. There is no admin or HR UI. Rules `BR-CORE-040`–`BR-CORE-047`: [Candidate Interview](business/capabilities/candidate-interview.md). Tripack: [candidate-interview](candidate-interview/task-plan.md). Migration `000009_candidate_interview` adds `interview_consents` and `interview_controllers` and raises the floor to 9, after clean version 8. The notice text blocks T1; real start, rejoin, and cancel wait on execution lifecycle, and progress and advance fencing wait on interview runtime.

3. **Interview runtime.** The live interview asks the published questions, records gaps, advances only on a committed handoff, and finishes only when the required record is stored. Rules `BR-CORE-060`–`BR-CORE-071`: [Interview Runtime](business/capabilities/interview-runtime.md). Tripack: [interview-runtime](interview-runtime/task-plan.md). Migration `000010_interview_runtime` raises the floor to 10, after clean version 9. Activation and session HTTP wait on execution lifecycle. Transcript ingest and one deliberate reconnect stay blocked until provider event finality is written from reviewed evidence; live H-01 waits on a sanitized live record, and a fake provider may proceed earlier.

4. **Interview reporting.** HR receives a written account of a finished interview that does not hire, reject, or rank the candidate, and a report failure must not change a completed interview. This is an HR API only, with no new admin screen. Rules `BR-CORE-080`–`BR-CORE-086`: [Interview Reporting](business/capabilities/interview-reporting.md). Tripack: [interview-reporting](interview-reporting/task-plan.md). Migration `000011_interview_reporting` raises the floor to 11, after clean version 10 and a completed interview from the runtime. Dead-letter topology, the report model contract, the finalized-evidence adapter, and process registration are blocked. Schema and completion-plus-outbox work can follow the runtime.

## Suggested build order

Build execution lifecycle first (`000008_execution_lifecycle`, clean floor 8). Candidate interview comes next (`000009_candidate_interview`, floor 9): consent and the page can overlap once the lifecycle port exists, and live start follows the real lifecycle. Interview runtime follows (`000010_interview_runtime`, floor 10), then interview reporting (`000011_interview_reporting`, floor 11). The migration numbers stay in that order: 000008, then 000009, then 000010, then 000011. The repository floor is clean version 8 after `000008`; it rises again only when each later migration is actually added.
