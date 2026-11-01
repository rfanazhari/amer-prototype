# Shared AI Agent Execution Platform — Consolidated Architecture & MVP Scope Freeze v0.1

**Status:** Consolidated baseline / MVP scope frozen; ready to initiate Technical Design subject to readiness gates  
**Date:** 2026-10-31  
**Implementation target:** Go modular-monolith candidate; existing Go + HTML + OpenAI Realtime + WebRTC PoC reuse  
**Scope:** Shared AI Agent Execution Platform, HR Interviewer showcase, Document Analyzer reusable-runtime proof  
**Next phase:** Technical Design (not included in this document)

> **Source of truth and status discipline.** This document consolidates `shared_ai_agent_platform_architecture_design_v0.1.md` and the explicit Step 1–5 confirmations (D1–D3, C1–C4, S1–S2, F1–F2, G1–G2). **Accepted** means expressly approved in the source or follow-up. **Proposed** means recommended but not explicitly voted through as an independent ADR. **Frozen MVP scope** means selected feature depth/acceptance gate, not that its implementation is complete. Where a detail is newly synthesized, it is labeled a recommendation or a testable elaboration. The earlier HR Interviewer draft is background context only, not an override of this platform architecture.

## 1. Executive summary

Build the **smallest coherent template-based, no-code Shared AI Agent Execution Platform** that proves runtime reusability across two distinct execution styles. Non-developers choose a **predefined Agent Template**, configure an **Agent Instance** through constrained fields, **validate and publish** an immutable revision, **initiate execution**, **monitor authoritative status**, and inspect domain-specific results.

- **HR Interviewer (showcase):** autonomous realtime voice interview, mandatory HR questions plus adaptive follow-ups, staged sessions, target **5–10 minutes**, transcript/evidence, bounded checkpoint recovery, and **separate** evidence-backed HR report. No autonomous hiring decisions.
- **Document Analyzer (architecture proof):** runnable, configurable **single-invoice** extraction with supported required fields, validation, bounded retry, structured output, and shared execution integrity; **not** a second full product.
- **Shared execution foundation:** stable execution identity, committed lifecycle, operation/attempt tracking, stable published-configuration binding, traceability, checkpoint references, and recovery/fencing semantics. Different agents define their own progress and completion contracts.
- **Delivery:** risk-first vertical slices with early verification of existing realtime PoC compatibility; **synthetic data only** for MVP demonstration; **every Must Have acceptance criterion is mandatory** for the MVP-complete claim.

**Architecture principle:** Control Plane governs what may be configured/published/admitted; Execution Plane governs execution identity and committed outcomes. The Minimal Execution Kernel enforces execution integrity, **not** business reasoning, HR stages, invoice fields, or a workflow DSL.

## 2. Decision register and traceability

### 2.1 Original architecture decisions

| ID | Decision | Consolidated status | Clarification |
|---|---|---|---|
| ADR-01 | Shared execution model, mode-specific strategies | **Accepted — retained** | No universal blocking loop or duplicated lifecycle implementations. |
| ADR-02 | Hybrid stage transitions | **Accepted — retained** | Model proposes → HR workflow validates → runtime commits; handoff prerequisite applies. |
| ADR-03 | Realtime model session per interview stage | **Accepted — retained** | Replacement attempts possible after disconnect. |
| ADR-04 | Hybrid cross-stage carryover | **Accepted — retained** | Required structured context; optional narrative summary. |
| ADR-05 | Eager handoff before next-stage activation | **Accepted — retained** | Source handoff committed first. |
| ADR-06 | Conditional fallback if narrative summary missing | **Accepted — retained** | Never waive mandatory structured evidence. |
| ADR-07 | Resume from committed checkpoint | **Accepted — retained** | Do not restore hidden model state. |
| ADR-08 | Backend canonical transcript authority | **Accepted — retained** | Normalized provider events; transcript is not automatically verified evidence. |
| ADR-09 | Post-execution HR reporting | **Accepted — retained** | Independent downstream lifecycle. |
| ADR-10 | Separate execution and post-processing | **Accepted — retained** | Document primary extraction result belongs inside its execution. |
| ADR-11 | Minimal Execution Kernel | **Accepted — retained** | Integrity primitives only. |
| ADR-12 | Shared Agent Execution Platform | **Accepted — retained** | Platform journey, not merely a library. |
| ADR-13 | Draft → Validate → Publish | **Accepted — retained** | Specific-draft validation; no implicit mutation of published versions. |
| ADR-14 | Unified Execution Request Model | **Accepted — retained** | Trigger-specific application admission; common execution request semantics. |
| ADR-15 | Logical Control Plane / Execution Plane separation | **Proposed — consistent with consolidated ownership** | Explicitly used as an architectural working boundary; no separate formal ADR acceptance recorded. |
| ADR-16 | Immutable published revisions and execution snapshots | **Accepted in substance through D2** | Frozen behavioral configuration; stable binding and permitted live safety overrides. Storage mechanics deferred. |
| ADR-17 | Modular monolith for MVP | **Proposed — preferred deployment direction** | Strong default, not separately confirmed as an Accepted ADR; no microservices required. |
| ADR-18 | Idempotent initiation, domain-specific business uniqueness | **Partially accepted through D2** | Logical request deduplication **Accepted**; specific domain uniqueness rules remain application policy to resolve in Technical Design. |
| ADR-19 | Logically decoupled creation and activation | **Accepted through D1** | Does not require queue, worker, or distributed infrastructure. |

### 2.2 Explicit follow-up decisions

| ID | Accepted decision | Consequence |
|---|---|---|
| D1 | Logically decoupled creation and activation | Accepted request ≠ active provider workload. |
| D2 | Immutable configuration and minimal request safety | Stable binding, idempotent publish/initiate, live authorization/safety enforcement. |
| D3 | Runnable Document Analyzer thin slice | Reusability must be tested with actual execution. |
| C1 | Five-state minimum execution lifecycle | CREATED, RUNNING, COMPLETED, FAILED, CANCELLED; WAITING_INPUT deferred. |
| C2 | Activation and termination semantics | Bounded attempts/deadlines; explicit failure if activation cannot progress. |
| C3 | Handoff ordering and terminal correctness | Commit handoff before next stage activation; commit required output before COMPLETED. |
| C4 | Boundary and invariant ownership | Kernel integrity; workflow domain; driver provider; application admission. |
| S1 | HR end-to-end showcase | Realtime, staged, adaptive, checkpoint recovery, transcript, report. |
| S2 | Configurable Document Analyzer thin slice | Configurable supported invoice fields, extraction, validation, result. |
| F1 | Bounded recovery acceptance | One HR disconnect/reconnect; one Document retry; duplicate initiation; terminal protection. |
| F2 | Risk-first delivery | Shared foundation → Document proof → HR integration; voice feasibility early. |
| G1 | Synthetic demo data | No real candidate records needed for demo; basic access/safety still mandatory. |
| G2 | All Must Have mandatory | No waived Must Have for an unqualified MVP-complete claim. |

**Decision integrity:** ADR-15 and ADR-17 should not silently be relabeled Accepted. ADR-18 has a confirmed platform idempotency portion and a still-to-be-specified domain-uniqueness portion. No previously Accepted ADR is reversed.

## 3. Consolidated logical architecture

```text
HR/Admin UI        Candidate access         Document UI / API
      \                   |                    /
       ---- Application entry / domain admission ----
                    | actor, authority, eligibility
           CONTROL PLANE (logical)
       Templates | Instances | Draft/Validation
       Publish | Active Revision | Monitoring reads
                    |
          Authorized Execution Request
                    |
           EXECUTION PLANE (logical)
         Activation / execution coordination
         +----------------------------------+
         | Minimal Execution Kernel         |
         | identity; committed lifecycle;   |
         | operations; progress references; |
         | recovery/fencing; events         |
         +----------------------------------+
                    |
          Agent-specific execution behavior
         /                                 \
   HR Voice Workflow                  Document Workflow
   stage/coverage/evidence            extract/validate/result
   Realtime driver                    Task driver
         \                                 /
             Shared persistence/providers

Post-processing (HR report): downstream of completed interview;
separate lifecycle, not an execution-completion prerequisite.
```

**Logical boundaries ≠ physical deployments.** The proposed physical shape is one Go backend, existing HTML frontend where practical, shared persistence, and external model provider integrations. No separate service per plane, agent, instance, or stage.

### 3.1 Component ownership matrix

| Concern | Authoritative owner | Must not own |
|---|---|---|
| Template catalog, supported configuration, template versions | Control Plane / template developer | Active execution state |
| Instance draft, validation of exact revision, publish, active revision pointer | Control Plane | Domain progression or session state |
| Actor/delegation, invitation/access, domain eligibility, business uniqueness | Application/admission boundary | Kernel transition commits |
| Execution identity, lifecycle, committed progress refs, operation attempts, meaningful state events | Minimal Execution Kernel | HR stage IDs, invoice extraction fields, model reasoning |
| Activation sequencing and mode-specific coordination | Execution Plane coordinator | Authorization policy decisions, arbitrary domain completion |
| Model session, WebRTC/transport, provider calls and their attempts | Driver/capability | Authoritative execution identity or final business verdict |
| Stage transitions, question coverage, field validation, domain completion criteria | Agent-specific workflow | Shared lifecycle persistence rules |
| HR invitation, compensation policy, business artifacts, downstream HR report | Domain application/post-processing | Kernel generic execution integrity |
| Execution monitoring views | Control Plane **read side** | Independent/parallel lifecycle source of truth |

**Ownership rule:** any layer may *request* a state transition, but only the authoritative owner may *commit* it, subject to workflow-supplied domain prerequisites. Monitor reads are projections of the execution authority, not a new execution state machine.

## 4. Canonical configuration and execution model

### 4.1 Template → Instance → Published Revision → Execution

1. **Template:** developer-defined/versioned blueprint of allowable configuration, capabilities, validation, and execution behavior.
2. **Instance:** reusable logical configured agent; never a deployment, process, or provider session.
3. **Draft:** mutable working settings that can coexist with an active published revision.
4. **Validate:** deterministic structural/semantic/readiness checks of a **specific draft revision**; errors block publishing, warnings may be surfaced.
5. **Publish:** produces a **new immutable revision** only if its exact draft revision remains valid; repeated logical publish does not create duplicates.
6. **Active revision:** default selection for **new** executions, not a mechanism to migrate existing executions.
7. **Execution:** an admitted, stable-identity attempt bound to one effective behavioral configuration and relevant stable input references.

Snapshot behavior freezes mandatory questions, stage policies, extraction configuration, and operational limits as appropriate. Sensitive CV/salary/document content need not be copied indiscriminately: a stable versioned reference with proper access control is preferable where feasible. **Live authorization/safety revocation remains effective** even after the configuration is frozen.

### 4.2 Unified initiation model

**Trigger** (run button, candidate link, API) → **application authorization and domain eligibility** → **resolve published revision and execution input** → **deduplication/admission** → **commit or identify execution** → **attempt activation**.

- Retrying the **same logical request** finds the same execution; *intentional new run* can create a new execution if domain policy allows.
- Candidate page loads/reconnects alone do **not** create duplicate interviews.
- An HR assignment may impose one active execution; an invoice can legitimately be reprocessed. This business uniqueness is **not** a universal kernel rule.
- Rejected authorization/admission need not create an execution.
- Accepted initiation means execution creation is committed, **not** that provider work already started.

### 4.3 Minimum authoritative lifecycle

```text
CREATED --successful activation--> RUNNING --domain output finalized--> COMPLETED
   |                                   | 
   +--unrecoverable / exhausted--------+----> FAILED
   +--authorized cancellation----------+----> CANCELLED

Terminal states: COMPLETED / FAILED / CANCELLED — never reopen.
```

| State | Meaning |
|---|---|
| CREATED | Execution identity and configuration binding committed; activation not yet confirmed. |
| RUNNING | Activated execution not yet terminal; temporary waiting, reconnection, or operation retry may happen within it. |
| COMPLETED | Workflow completion contract satisfied and required progress/output committed. |
| FAILED | Cannot satisfy completion under bounded failure/recovery policy. |
| CANCELLED | Authorized cancellation has been durably committed. |

`WAITING_INPUT`, `ACTIVATING`, `RECOVERING`, and provider connectivity are **not required top-level states** for v0.1; represent them as operation/session conditions where necessary. A candidate's silence does not automatically require a new execution status. Specific timeout thresholds are Technical Design/operational policy decisions, not silently selected here.

### 4.4 Activation, recovery, and completion semantics

- Creation and activation are **logically decoupled** even if an MVP request initiates both synchronously. Queue/scheduler infrastructure is not implied.
- Failed activation is an **attempt outcome**, not a second execution; bounded retries either activate the same execution or lead to explicit FAILED.
- Realtime provider sessions and operation attempts are **subordinate** to execution identity. Superseded attempts cannot commit newer progress.
- Recovery resumes only from **verifiable committed progress**, never imagined intermediate state or hidden model memory.
- Provider success alone is **not** sufficient for `COMPLETED`: the domain-required output/progress must already be committed.
- External tool side effects are **not guaranteed exactly once** by idempotent platform commands. Any side-effectful capability requires its own safety/deduplication assessment.
- Post-execution HR reporting may independently be pending, successful, or failed without changing an already COMPLETED interview.

### 4.5 HR stage transition ordering — consolidated clarification

1. Realtime model **proposes** stage transition.
2. HR workflow checks mandatory coverage, time/policy rules, and transition eligibility.
3. Source-stage required transcript/coverage/evidence is finalized and **handoff committed**.
4. Authorized stage advancement is committed against that valid handoff.
5. A new realtime provider session is activated for the next stage.

No next-stage activation may precede a durable valid source handoff. Optional narrative summary failure may fall back to committed structured evidence; required missing evidence may not be invented. This clarifies sequencing while preserving ADR-02 and ADR-05.

## 5. Core execution invariants — frozen conceptual baseline

| ID | Invariant | Consequence / test |
|---|---|---|
| INV-01 | Execution identity stable across reconnects/retries | New provider session ≠ new execution. |
| INV-02 | Only valid transitions; terminal state cannot reopen | Late commands cannot mutate terminal outcome. |
| INV-03 | One authoritative committed state per execution revision | Concurrent updates cannot create contradictory state. |
| INV-04 | Repeated logical commands cannot duplicate committed outcomes | Idempotent publish/initiation and relevant state changes. |
| INV-05 | Recovery only from verifiable committed progress | No invented completion or uncommitted replay as truth. |
| INV-06 | Superseded session/operation attempts fenced | Stale events cannot overwrite newer progress. |
| INV-07 | Terminal outcome refers to committed required output/progress | Domain completion contract precedes `COMPLETED`. |
| INV-08 | Relevant state changes traceable to operation/actor | Minimum observability and attribution. |
| INV-09 | Stable effective behavioral configuration binding | Later publish/draft edits cannot change an existing execution. |

These are conceptual rules, **not** package, persistence, or concurrency-control design prescriptions.

## 6. MVP success definition and explicit feature scope

**MVP success statement:** A non-developer can configure, validate, publish, start, monitor, and inspect outcomes of **two predefined agents** without code changes; both agents actually use the same authoritative execution mechanics while retaining domain-specific workflows.

### 6.1 Feature priority matrix

| Capability | Must Have | Should Have | Deferred / Out of Scope |
|---|---|---|---|
| Templates/instances | Two predefined templates, constrained drafts | Clear validation guidance | User-authored agent builder, unlimited prompts/code |
| Publishing | Exact-draft validate/publish, immutable revision, active revision | Revision history read view | Sophisticated rollback/migration/editor collaboration |
| Initiation | Authorized admission, stable binding, deduplication | Clear rejection feedback | General scheduler, arbitrary webhooks |
| Shared execution | Five states, operations, committed progress, bounded recovery/fencing, cancellation consistency | Simple event timeline | Workflow DSL, event sourcing, distributed engine |
| Monitoring | Execution status, meaningful progress/outcome, result visibility | Human-readable incident summary | Distributed analytics suite |
| HR | Working realtime voice, staged hybrid questions, committed handoff, canonical transcript/evidence, basic reconnect, HR report | Candidate-friendly device/incident messaging | Automated hiring/rejection/ranking, voice personality scoring |
| Document | Single-invoice configurable extraction, required-field validation, structured result, basic retry | Helpful per-field evidence display | Batch processing, multiple document types, custom schema builder |
| Privacy/safety | Synthetic data for demo, scoped access, confidential salary guard, no autonomous hiring decision | Clear demo-data labeling | Enterprise governance suite |
| Physical architecture | Reuse PoC, simple deployment | Basic operations instrumentation | Microservices, HA/distributed runtime, per-agent deployments |

**Should Have must never implicitly become mandatory.** All listed **Must Have** criteria remain a strict gate (G2), not a license to drop correctness requirements for demo polish.

### 6.2 HR Interviewer — showcase depth

**Inputs/configuration:** one representative Backend Engineer role; synthetic CV/JD and applicant; HR-defined mandatory questions, competency topics, supported stage/time policy, restricted compensation discussion; candidate invitation/access and required notice/consent UX.

**Workflow:** realtime voice, 5–10-minute target, mandatory coverage with relevant adaptive follow-up, role/background/technical/compensation/closing stages according to predefined policy, one provider session per activated stage, workflow-authorized transitions and committed handoffs. For incomplete coverage, mark omissions/gaps explicitly rather than claiming success.

**Output:** backend canonical transcript and structured coverage/evidence; after execution completion, independently generated evidence-linked HR report. Do not produce an automatic hire/reject verdict. Salary ceiling remains server-side and confidential.

**Recovery scope:** one deliberate disconnect/reconnect from a committed checkpoint, new session attempt under same execution, and protection against duplicate or stale progress. This is **not** a guarantee against every realtime interruption.

### 6.3 Document Analyzer — runnable proof depth

**Inputs/configuration:** one invoice per execution, synthetic invoice, selectable **required fields from the predefined supported set**: invoice number, vendor, date, total, currency. Not arbitrary schema authoring.

**Workflow:** authorized execution → bounded model extraction → deterministic required-field validation where feasible → terminal result commit; track meaningful operation attempts and a simulated/controlled retry case.

**Output:** structured extracted fields, validation states (`valid`, `missing`, `invalid` where applicable), and supporting field evidence references when supported by the available document representation. Unknown values remain missing/uncertain, never invented.

**Completion:** a task-specific explicit contract. A validated result with flagged missing optional fields may be successful under configured rules; unmet required criteria may lead to a non-successful result or FAILED under a consistent predefined policy. The exact rules must be specified in Technical Design **without weakening terminal correctness**.

**No product expansion:** no batch, generalized OCR platform, multiple document categories, or full document management application.

## 7. Mandatory measurable acceptance criteria

**Gate:** every P/H/D item below must pass before an unqualified **MVP complete** declaration. Tests may be automated/manual and need not all be shown live; evidence should be retained. These are frozen MVP acceptance targets, **not claims that the implementation already passes**.

### 7.1 Shared platform (P-01–P-12)

| ID | Pass condition |
|---|---|
| P-01 | Both predefined Agent Templates selectable. |
| P-02 | User creates/configures both Agent Instances using supported fields, no source edits. |
| P-03 | Invalid draft blocked from publishing with intelligible reason. |
| P-04 | Published revision immutable; valid publish bound to exact validated draft revision. |
| P-05 | Existing execution configuration unchanged after later draft edit/publish. |
| P-06 | Both agents initiated through common authorized admission semantics. |
| P-07 | Retry of same logical initiation identifies one execution, not two. |
| P-08 | Valid lifecycle transitions; terminal executions cannot reopen. |
| P-09 | Meaningful operation attempts, outcomes, and failures traceable. |
| P-10 | No `COMPLETED` without committed domain-required output/progress. |
| P-11 | User sees execution identity, authoritative state, progress summary and output availability. |
| P-12 | Architectural review and runnable behavior show Kernel requires no HR-stage/invoice-field logic. |

### 7.2 HR Interviewer (H-01–H-12)

| ID | Pass condition |
|---|---|
| H-01 | Candidate completes live voice interaction via reused WebRTC path. |
| H-02 | Required questions asked or each omission explicitly recorded; ≥1 context-relevant adaptive follow-up. |
| H-03 | Stage advancement rejected when workflow authorization invalid. |
| H-04 | Activated stages use separate realtime model sessions. |
| H-05 | Next stage activation cannot precede committed required handoff. |
| H-06 | Backend maintains authoritative normalized transcript and committed progress references. |
| H-07 | Coverage/report statements linked to transcript evidence; missing evidence not treated as answered. |
| H-08 | One disconnect/reconnect resumes from committed checkpoint, preserves execution identity, avoids duplicate committed progress. |
| H-09 | Interview only completes after required transcript/progress finalization. |
| H-10 | Separate HR report lifecycle; failed report does not turn completed interview into failed execution. |
| H-11 | No autonomous candidate hire/reject decision. |
| H-12 | Confidential internal salary ceiling not disclosed in normal or adversarial test scenario. |

**Duration target:** 5–10 minutes for representative interview; track actual duration and explicit coverage omissions. Target duration does not override correctness or justify fabricated evidence.

### 7.3 Document Analyzer (D-01–D-08)

| ID | Pass condition |
|---|---|
| D-01 | User selects required fields from predefined invoice field set. |
| D-02 | Published configuration drives one invoice execution. |
| D-03 | Result represents invoice number, vendor, date, total, currency as appropriate. |
| D-04 | Required-field outcome distinguishes valid/missing/invalid. |
| D-05 | Missing or ambiguous fields are explicit, not fabricated. |
| D-06 | Committed extraction result precedes successful terminal completion. |
| D-07 | One controlled extraction/validation failure follows bounded retry/fail policy. |
| D-08 | Shared lifecycle, binding and operation-tracking are demonstrably reused. |

### 7.4 Minimum negative-path acceptance set

1. Invalid configuration publish rejected without creating a published revision.
2. New publish during running execution leaves prior binding intact.
3. Duplicate initiation returns/resolves the existing execution.
4. Activation fails before RUNNING: bounded retry or explicit FAILED, same execution identity.
5. HR transition blocked when required handoff is absent.
6. HR reconnect uses checkpoint; superseded session cannot overwrite committed progress.
7. Document operation failure retries within bounds then either commits valid result or fails honestly.
8. Post-terminal mutation rejected; report failure leaves completed interview intact.

## 8. End-to-end journeys and demo narrative

### Journey A — HR Interviewer

1. HR chooses the HR Template and creates a configured Instance using synthetic CV/JD, mandatory questions, stage and compensation policy.
2. Validate draft and publish an immutable active revision.
3. Candidate accesses narrowly scoped invitation; notice/consent and mic readiness as applicable.
4. Authorized initiation commits an execution with stable revision/input binding; activation starts a realtime stage session.
5. Interview asks mandatory questions and a relevant adaptive follow-up; model proposes, workflow authorizes, and runtime commits progression only after valid handoff.
6. Demonstrate one reconnect into the same execution using committed checkpoint and fresh realtime session.
7. Close the interview; commit finalized transcript and coverage before COMPLETED.
8. Downstream HR report generates evidence-backed notes, gaps, and compensation expectations without hiring verdict; HR monitors interview and separate report status.

### Journey B — Document Analyzer

1. User selects Document Template and configures required invoice fields.
2. Validate and publish the revision.
3. Initiate authorized extraction for one synthetic invoice.
4. Execute bounded model extraction plus required-field validation, through the same shared execution integrity model.
5. Demonstrate a controlled retry/failure path with honest final outcome.
6. Commit structured extraction result, then `COMPLETED` if domain contract satisfied; show missing/invalid fields transparently.
7. Inspect shared execution status/operations and document-specific output.

### Demo proof narrative

**Showcase:** voice-based HR interview demonstrates user/business value.  
**Architecture proof:** invoice task uses the same lifecycle and configuration rules with a different workflow.  
**Correctness proof:** duplicated initiation, revised configuration, disconnect/retry, and terminal protection do not corrupt authoritative execution state.

## 9. Delivery plan — risk-first, no time estimates implied

| Milestone | Outcome | Exit gate |
|---|---|---|
| M0 — Baseline freeze | This architecture and MVP scope record | Explicit decisions and acceptance matrix frozen. |
| M1 — Voice feasibility | Validate actual PoC reuse constraints early | Per-stage session activation, provider event behavior, and reconnect assumptions shown feasible or risks documented. |
| M2 — Shared foundation | Config/publish/initiation/lifecycle/monitoring minimum | A bounded execution follows the committed lifecycle with stable config. |
| M3 — Document thin slice | First runnable second-style agent | D-01–D-08 and core P scenarios pass. |
| M4 — HR integration | Voice workflow attached to shared runtime | Live hybrid interview and authorized stage progression work. |
| M5 — HR recovery/reporting | Checkpoint/transcript/report outcomes | H-05–H-12 and relevant negative-path checks pass. |
| M6 — Integrated demo | Two-agent reusable platform proof | **All** P-01–P-12, H-01–H-12, D-01–D-08 pass; deviations tracked and prevent unqualified complete claim. |

**Dependency correction:** while the shared foundation/Document slice is built first, voice feasibility must be validated from M1, **not** discovered late at M4. Do not spend heavily on admin UI before knowing the realtime PoC can satisfy accepted session/transition semantics.

## 10. Architecture and delivery risks

| Risk | Impact | Mitigation / guardrail |
|---|---|---|
| Activation committed but never progresses | Stuck execution | Bounded attempt/deadline policy and terminal failure outcome. |
| Realtime session transition handoff gaps | Lost or contradictory conversation progress | Commit required source handoff before new stage activation. |
| Provider late events after reconnect | Progress corruption | Attempt fencing and authoritative committed revisions. |
| Transcript mistaken for verified competence | Unsupported reporting | Evidence provenance, uncertainty/gaps, human oversight. |
| Mutable configuration affecting active runs | Non-reproducible behavior | Immutable published revision and effective binding. |
| HR concepts creep into Kernel | False reusability | Require runnable Document reuse and P-12 review. |
| Document agent too trivial or independent | Weak platform proof | Use actual shared admission/lifecycle/operations, test one retry. |
| Too much abstraction and dashboard work | Missed demo | Fixed templates, one role, one invoice type, risk-first vertical slices. |
| Voice PoC incompatible with stage-per-session assumption | Showcase blocked | Early M1 feasibility gate and explicit integration risk assessment. |
| Sensitive candidate and compensation information | Privacy/confidentiality harm | Synthetic demo data, scoped access, restricted salary data, no hiring verdict. |
| Report lifecycle conflated with execution | Incorrect status | Separate downstream report status; do not reopen completed interview. |

## 11. Explicit out of scope

- Arbitrary user-authored agent code, prompt/tool freedom, user-defined agent types.
- General workflow DSL, visual flow editor, dynamic DAG execution, multi-agent orchestration or planner agents.
- Distributed scheduling, general cron/recurring trigger platform, generic webhook or external connector marketplace.
- Microservices, individual instance deployments, distributed queues/compute, enterprise HA or horizontal scalability guarantees.
- Event-sourcing platform, sophisticated analytics/distributed tracing, full operational dashboard suite.
- Collaborative editing, automated template migrations, comprehensive revision rollback/version diff UX.
- Enterprise multi-tenant organization management, complex RBAC/approvals/governance suite.
- HR hiring/rejection automation, opaque candidate rankings, accent/emotion/personality scoring, complex ATS product.
- Document batch processing, multi-document support, custom schema authoring, advanced OCR-correction suite or full document-management product.
- Generic RAG/vector database, model marketplace or generalized tool registry without validated requirements.

**Not out of scope:** elementary authorization, access scoping, synthetic-data hygiene, confidential policy protection, minimal consent/notice behavior for the simulated HR journey, correct terminal states, and enough monitoring to evaluate completion/failures.

## 12. Technical Design readiness — checklist

### Architecture baseline

- [x] Original Accepted ADRs preserved; D1–D3, C1–C4 consolidated.
- [x] Shared lifecycle, binding, handoff order, and completion rules accepted.
- [x] Authority boundaries defined at the conceptual level.
- [x] Logical vs physical deployment boundaries distinguished.
- [ ] ADR-15 and ADR-17 formal acceptance **if** the team requires ADR-level ratification rather than working recommendations. This is not a blocker to studying Technical Design alternatives.

### MVP scope and proof

- [x] S1/S2 frozen: complete HR showcase, configurable runnable Document thin slice.
- [x] F1/F2 frozen: bounded recovery, risk-first sequence.
- [x] G1/G2 frozen: synthetic data, strict acceptance gate.
- [x] Must Have/Should Have/Out-of-Scope enumerated.
- [ ] P/H/D acceptance gates reviewed as a team and mapped to repeatable evidence/tests (before declaring MVP complete).

### Feasibility and policy gates

- [ ] Inspect and validate PoC integration against session-per-stage, canonical events, and checkpoint assumptions (M1).
- [ ] Set finite activation/retry/deadline and authorization/cancellation policy values during Technical Design (do not invent them here).
- [ ] Specify Document required-field completion/failure behavior consistently during Technical Design.
- [ ] Confirm synthetic CV, JD, candidate, salary, and invoice fixtures; avoid accidental real personal data.
- [ ] Define minimal demo access, notice/consent handling, and retention boundaries before running actual demonstrations.
- [ ] Ensure operational visibility is sufficient to verify failures, config bindings, and report/result outcomes.

**Readiness verdict:** Architecture consolidation and **MVP scope selection are finished**. Technical Design may begin using this baseline, with feasibility and operational details above treated as explicit readiness checks, **not** silent changes to already Accepted decisions. MVP implementation is **not yet complete** and its strict acceptance gate remains to be demonstrated.

## 13. Deliberately deferred Technical Design questions

These are **not** permissions to enlarge scope; they are where implementation-specific decisions belong:

- Go package/interface boundaries and adapter structure.
- Database and artifact persistence schemas, transaction boundaries, and operational locking/fencing mechanisms.
- API contracts, idempotency-key representation, authorization wiring and access-token mechanics.
- Provider/model selection, exact WebRTC integration, transcript normalization and operation event details.
- Checkpoint representations, configuration version-reference storage, retry/deadline numbers, failure codes.
- Deployment topology, logging/metrics details, UI component structure, test harness design.

If any future Technical Design discovery threatens an Accepted invariant, record a **proposed ADR revision** and obtain explicit approval rather than silently weakening it.

---

## 14. Source provenance

1. **Primary architecture design:** `shared_ai_agent_platform_architecture_design_v0.1.md` (2026-10-31), especially ADR register, logical boundaries, execution model, configuration lifecycle, initiation semantics, HR/Document agents, risks and open questions.
2. **Explicit follow-up confirmations in the project conversation:** Step 1 D1–D3; Step 2 C1–C4; Step 3 S1–S2; Step 4 F1–F2; Step 5 G1–G2.
3. **Earlier HR Interviewer MVP summary:** contextual background only; the later shared platform baseline takes precedence.

**Working principle:** *Build the smallest coherent platform that proves the architecture, not the largest platform that demonstrates features.*
