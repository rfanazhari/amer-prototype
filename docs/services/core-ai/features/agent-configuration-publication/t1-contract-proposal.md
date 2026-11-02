# T1 contract approval record — CORE-ACP-001

Status: **approved; T1 Done**. Originally prepared as a proposal on 2026-11-01.
The filename is retained so existing review links remain valid.

## Approval

- Confirming actor: the user in this conversation; no separate external sign-off is claimed.
- Date: 2026-11-01.
- Exact confirmation: “approve. - Setujui bagian A–E sebagai kontrak T1”.
- Scope: all sections A–E as presented, without requested revisions.
- Canonical accepted text is now in [technical-design.md, section 6](technical-design.md#6-request--response).

| Approved section | Canonical location |
|---|---|
| A — Template and field schemas, defaults, normalization and examples | [§6.1](technical-design.md#61-template-and-field-schemas-approved-a) |
| B — Validation findings and informational warnings without acknowledgment | [§6.2](technical-design.md#62-validation-and-warnings-approved-b-q3-resolved) |
| C — Operation and result schemas | [§6.3](technical-design.md#63-operation-and-result-schemas-approved-c) |
| D — Publication scope, fingerprint, replay and conflicting intent | [§6.4](technical-design.md#64-publication-identity-intent-and-retries-approved-d) |
| E — Public errors and precedence | [§6.5](technical-design.md#65-public-errors-and-precedence-approved-e) |

## Design assessment and downstream obligations

Retained with rationale: the instance owns draft/version/active-selection rules;
concrete configuration values own template constraints; application authorization
and fingerprinting remain separate from transactional persistence. T2 ports and
T3 generic version/ownership mechanics remain unchanged. T5 must implement concrete
configuration values, finding severity and deterministic warning/error eligibility;
T6–T8 must apply approved normalization, replay ordering and safe filtered views.
The existing T3 `Finding` contains code/path only; that is not evidence that the
approved severity policy is implemented. The T3 wider positive version mechanics
also do not establish the approved JSON-safe public version bound.

Q1/Q3 are resolved. Q2 secure access, Q4 historical deployment compatibility and
Q5 actual binding consumers retain their gates. Only T1 documentation is changed;
T4–T11 are not executed by this request. OpenAPI/endpoint/changelog implementation
synchronization remains T8 work, together with runtime codes and EN/ID messages.

Verification is document/link/example consistency only. No Go, SQL, interface or
runtime changes; behavioral RED/GREEN commands and SHAs are not applicable. No
checkpoint commit is fabricated. Go lint/tests, security and database integration
are skipped for this document-only task. sqlmock/Mockery apply to later SQL and
interface mocking; mocks do not replace real PostgreSQL proof.

## Local verification evidence

Cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

- **Passed:** `python3 /private/tmp/core-acp-001-check-docs.py` — local temporary
  document checker; verifies five documents, local links/anchors, JSON example
  syntax and HR example coherence, T1 closure, preserved T2/T3/T4–T11 statuses,
  Q1/Q3 resolution and retained Q2/Q4/Q5 gates. This is not behavioral Go evidence.
- **Passed:** `git diff --check` for tracked changes; the document checker also
  checks trailing whitespace in the untracked design/strategy/decision files.
- **Skipped:** `make lint`, `make test`, `make test.all`, `make test.contract`,
  `make test.architecture`, `make test.security`, and database/runtime checks;
  T1 modifies documentation only, with no Go/API artifact/dependency change.
- **Not applicable:** RED command/cwd/reason and SHA, GREEN command and SHA;
  no behavioral test checkpoint or commit is created for document approval.
