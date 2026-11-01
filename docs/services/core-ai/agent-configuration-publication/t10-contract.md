# T10 contract record — CORE-ACP-010

Status: **approved; T10 Done**. Prepared and recorded on 2026-11-01.

## Approval

- Confirming actor: the user in this conversation; no separate external sign-off is claimed.
- Date: 2026-11-01.
- Exact confirmation: exclusive T10 execution with a Done checklist for `CORE-ACP-010`.
- Scope: historical template/behavior deployment compatibility (Q4). First-version publication and T9 acceptance remain valid; T11 still waits on assignment/execution consumers.
- Canonical accepted text is now in [technical-design.md, section 6.6](technical-design.md#66-historical-deployment-compatibility-approved-t10).

| Approved section | Canonical location |
|---|---|
| Retain-and-restrict deployment | [§6.6 policy A](technical-design.md#66-historical-deployment-compatibility-approved-t10) |
| Restoration without reinterpretation | [§6.6 policy B](technical-design.md#66-historical-deployment-compatibility-approved-t10) |
| Representative fixtures | [§6.6 fixtures](technical-design.md#66-historical-deployment-compatibility-approved-t10) and `test/testdata/agentconfig/historical_hr_revision_v1_sparse.json` |

## Policy summary

1. **Retain-and-restrict deployment:** Each release embeds a restoration catalog. Startup runs `RequireAgentConfigurationDeploymentCompatible` and refuses boot when any stored `(template_id, template_version, behavior_version)` tuple is outside that catalog. Operators add restoration support before persisting a new behavior version, or roll back the binary only while stored tuples remain readable.
2. **Restoration without reinterpretation:** Adapter restoration decodes stored JSON and applies domain `Restore*` only. It does not rerun `AssessHR`/`AssessDocument` or publication eligibility against today's rules.
3. **Immutable version identity:** Replacing validator behavior under an unchanged version string remains forbidden; new behavior ships under a new behavior version with explicit restoration registration.

## Representative fixtures

- **Supported historical revision:** HR Interviewer template `@1` behavior `1` with a single stored mandatory question (see sparse JSON fixture). Restoration succeeds; current assessors still report ineligible findings without mutating stored content.
- **Unsupported deployment:** Any row carrying behavior `2` while the binary restoration catalog retains only behavior `1` is rejected at startup and on adapter restoration.

Verification uses sqlmock for deployment SQL and domain/repository tests for restoration; real PostgreSQL behavior remains covered by existing T7/T9 suites for v1 data.
