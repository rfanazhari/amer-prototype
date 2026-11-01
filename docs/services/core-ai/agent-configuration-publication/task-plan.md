# Task Plan: Agent Configuration and Publication

Planning baseline: 2026-11-01, Mode A. T2 and T3 are implemented; all other implementation tasks remain Todo. No code was written or verified for this capability during document generation.

## Ticket Identity

Capability group: **ACP — Agent Configuration and Publication**. Ticket prefix: `CORE-ACP` (`CORE` identifies the service group). These are repository planning IDs; no external tracker tickets have been created.

Each task has a stable ticket ID in the form `CORE-ACP-NNN`, mapped to its existing local task ID: `T1` → `CORE-ACP-001`, through `T11` → `CORE-ACP-011`. Keep both identities on revisions, including obsolete tasks; allocate new numbers without reusing earlier IDs. Other capability groups use their own distinct prefix under `CORE`.

## Legend

- IDs T1–T11 are stable; never renumber them on revision.
- Complexity: S (<2h), M (2–4h), L (>4h); rough estimates requiring engineer validation.
- Status: `[ ]` Todo, `[~]` In Progress, `[x]` Done, `~~obsolete~~` superseded work.
- `ref` identifies sections of [technical-design.md](technical-design.md); DoD is required observable evidence.
- A decision-blocked task is not executable until its stated readiness condition is met. Ordinary dependency blocks also apply.
- At closure synchronize affected design/strategy/status. Material changes to contracts or transactions need a resolved decision before dependent tasks; local structural refinements may proceed with preserved-contract evidence.

## Tasks

### Independent foundation work

- [x] **T2. Define application transaction, authorization and reference boundaries** — Application — M
  - Ticket ID: `CORE-ACP-002`
  - depends: none — ref: design §2–5, §9 Q2.
  - Evidence: application contracts and six Mockery-generated mocks; `TestCommitUncertainError` RED `39d7c05` then GREEN. See [T2 boundary assessment](t2-boundaries.md).
  - DoD: purpose-specific read/mutation/replay contracts describe rollback, ambiguous commit, trusted principal, and immutable reference behavior without ORM handles; authority cannot be supplied as a client role; interfaces are documented and generated mocks registered. Exact Q1-dependent DTO fields remain deferred, not guessed. This does not implement or declare real admin authentication complete.

- [x] **T3. Implement instance version and ownership mechanics** — Domain — M
  - Ticket ID: `CORE-ACP-003`
  - depends: none — ref: design §3–4, §6.
  - Evidence: public domain version/ownership/restoration tests; RED `2c5d335` then GREEN; architecture/declaration checks pass. See [T3 domain assessment](t3-domain.md).
  - DoD: public tests prove version changes, stale replacement rejection, validation version/ownership matching, active selection protection, immutable revision snapshots, separate restoration, and copied mutable values. Use minimal test configuration fixtures; production template field validation is T5. Architecture/declaration checks pass; no JSON/hashing or outward imports in domain.

T2 and T3 are independent work streams, not instructions to spawn agents. Changes to their interface shape must be synchronized before integration.

### Contract decisions before dependent implementation

- [ ] **T1. Resolve template, publication, and public error contracts** — Product/API/domain design — M
  - Ticket ID: `CORE-ACP-001`
  - depends: none — ref: design §2–3, §6–7, §9 Q1/Q3.
  - blocked by: product/template owners and API lead decisions on exact supported configuration, initial draft, limits/normalization, warning acknowledgment, request-key scope/fingerprint, validation statuses, and overlapping errors.
  - ready when: the decision owners provide or confirm concrete examples and choices.
  - DoD: design records complete accepted input/output schemas, valid/incomplete/invalid examples for both templates, warning behavior, same/different-intent examples, and error ordering. Preserve source-approved bounds. No question is converted into an invented business rule. Synchronize all three planning documents before starting dependents.

### Sequential implementation after contracts are resolved

- [ ] **T4. Specify and register additive configuration schema** — Infrastructure/runtime — M
  - Ticket ID: `CORE-ACP-004`
  - depends: T1, T3 — ref: design §3–4.
  - decision gate: Q1 resolved in T1.
  - DoD: five versioned record mappings, all ownership/uniqueness constraints and explicit zero/null update mappings are reviewable; runtime AutoMigrate stays additive and bounded; isolated database test proves installation both empty and with Task rows. No publication adapter behavior or live database migration is assumed from schema declarations.

- [ ] **T5. Implement both concrete catalogs and deterministic validators** — Domain/application reference resolution — L
  - Ticket ID: `CORE-ACP-005`
  - depends: T1, T3 — ref: design §3–4, §6.
  - decision gate: Q1/Q3 resolved in T1.
  - DoD: both versioned templates discoverable; full supported field/boundary matrix tested; editable incomplete drafts yield correct findings; warning policy tested; exact reference versions preserved; no model/provider calls. BR-CORE-001/002/006/007/014 local proof captured.

- [ ] **T6. Implement configuration commands and detached queries** — Application — L
  - Ticket ID: `CORE-ACP-006`
  - depends: T1, T2, T3, T5 — ref: design §2–7.
  - decision gate: Q1/Q3 resolved; real Q2 adapter not needed for mocked application tests.
  - DoD: create, replace, validate, publish and read operations use authority checks; exact-version validation recording rejects an edit race; replay precedes current-draft eligibility; mismatched intent conflicts; trusted UTC time and detached outputs used. Generated mock tests cover denied access, reference failures, no result before commit, ambiguous-commit reconciliation and old-command replay after later publication. No assumption that mocks establish SQL or real authentication correctness.

- [ ] **T7. Implement PostgreSQL publication transactions and repository proof** — Infrastructure — L
  - Ticket ID: `CORE-ACP-007`
  - depends: T1, T2, T3, T4, T5, T6 — ref: design §3, §5–7.
  - decision gate: Q1/Q3 resolved.
  - DoD: real PostgreSQL tests cover exact-version replacement/validation/publication races, all write/commit rollback points, duplicate keys with same/different intent, cross-instance constraint checks, replay after new active revision, corrupted restoration, and null/zero mapping. Unique-constraint loser resolves in a fresh transaction. Restart retains revisions. Immutable read-by-ID supports future binding; do not claim full BR-CORE-013 consumer proof. Use isolated Docker or explicitly authorized integration database.

T4 and T5 may proceed independently once their shared prerequisites are complete. T6 can use generated repository mocks while T4 is prepared; T7 integrates the finalized application contract.

### Blocked exposure and acceptance

- [ ] **T8. Add secured HTTP endpoints and synchronized API documentation** — Delivery/runtime/contracts — L
  - Ticket ID: `CORE-ACP-008`
  - depends: T1, T2, T5, T6, T7 — ref: design §2, §4, §6–7, §9 Q2.
  - blocked by: absent real admin-session/authorization integration (platform/security owner).
  - ready when: companion access implementation establishes trusted principal, scoped authorization, session expiry/revocation and CSRF controls with evidence. No mounted permissive placeholder is acceptable.
  - DoD: all seven endpoint directions implemented using operation interfaces; strict input, confidentiality, safe EN/ID errors, correct new/replayed response statuses and exact precedence tested; update OpenAPI, new endpoint documentation and changelog together. Candidate/unauthenticated requests fail. Composition preserves existing routes and lifecycle budgets.

- [ ] **T9. Verify the complete backend configuration flow and regressions** — Integration/verification — L
  - Ticket ID: `CORE-ACP-009`
  - depends: T4, T5, T6, T7, T8 — ref: design §3 proof matrix, §8–9.
  - blocked by: Q2 until T8 is ready; Q1/Q3 inherited from prior tasks.
  - ready when: both template contracts and real secure endpoint composition are available.
  - DoD: each template completes create/replace/validate/publish/read through real HTTP/PostgreSQL; stale publication and replay-after-new-publication negative paths pass; fixture seeding uses ordinary operations. Run `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, `make test.security`, and `make test.docker` with the new suites included. Record failures/skips, command outputs and per-rule evidence. Existing Task/voice tests remain passing. Clearly retain UI and T11 acceptance limitations; no unqualified platform MVP-complete claim.

- [ ] **T10. Establish historical template deployment compatibility** — Platform/domain/infrastructure — M
  - Ticket ID: `CORE-ACP-010`
  - depends: T1, T3, T4, T5 — ref: design §3 restoration, §6, §9 Q4.
  - blocked by: platform/template owner choice of retained behavior support versus deployment restriction.
  - ready when: version compatibility and rollback policy are recorded with representative old/new fixtures.
  - DoD: supported historical revisions restore without current-creation-policy reinterpretation; unsupported deployments are prevented or rejected explicitly under approved policy; rollback rehearsal preserves readable data. Must pass before multi-version rollout; not a blocker for isolated first-version development.

- [ ] **T11. Prove binding preservation with actual assignment/execution consumers** — Cross-capability integration — M
  - Ticket ID: `CORE-ACP-011`
  - depends: T9 — ref: design §3 BR-CORE-013, §5, §9 Q5.
  - blocked by: absent assignment/execution consumers owned by their separate capability work.
  - ready when: those consumers exist with immutable published revision bindings; if multiple behavior versions are involved, T10 must also pass.
  - DoD: assignment pinned before new publication keeps its original revision at start; execution started before publication retains its original requirements; intentional new binding selects the appropriate active revision. Record real integration evidence. Do not implement the entire execution capability inside this task or count a fake consumer as full acceptance.

## Dependency Graph

```text
Q1 + Q3 decisions → T1
T2 (independent application boundaries)
T3 (independent domain version mechanics)
T1 + T3 → T4 (schema)
T1 + T3 → T5 (templates/validators)
T1 + T2 + T3 + T5 → T6 (application)
T1 + T2 + T3 + T4 + T5 + T6 → T7 (PostgreSQL adapter)
T1 + T2 + T5 + T6 + T7 + Q2 access implementation → T8 (HTTP)
T4 + T5 + T6 + T7 + T8 → T9 (backend acceptance)
T1 + T3 + T4 + T5 + Q4 → T10 (multi-version deployment gate)
T9 + Q5 consumers [and T10 for multiple behavior versions] → T11
```

## Revision History

| Version | Date | Change |
|---|---|---|
| v1 | 2026-11-01 | Initial remaining-work plan; all tasks Todo and decision dependencies explicit |
| v1.1 | 2026-11-01 | Add capability-group ticket prefix CORE-ACP and stable ticket IDs for T1–T11; task statuses and dependencies unchanged |
