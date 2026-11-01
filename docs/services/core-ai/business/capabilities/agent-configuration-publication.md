---
id: agent-configuration-publication
service: core-ai
status: planned
rules: BR-CORE-001..BR-CORE-014
updated: 2026-11-02
implementation:
---

# Agent Configuration and Publication

## Purpose

Enable authorized HR users to prepare reusable HR Interviewer and Document Analyzer agents through predefined configuration fields without changing source code. Publish only the configuration that was validated, so users can rely on stable agent behavior while preparing later changes independently.

This capability covers template selection, instance drafts, validation, and publication. Execution admission, interview consent and access, domain execution workflows, and reporting are separate capabilities; the rules here describe their configuration boundary only.

## Ubiquitous language

The shared definitions of **Agent Template**, **Agent Instance**, **Published Revision**, and **Agent Execution** are maintained in the [architecture design, section 4](../../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md#4-domain-and-platform-concepts). **Draft**, **Validate**, and **Active revision** follow the [scope freeze, section 4.1](../../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md#41-template--instance--published-revision--execution).

| Term | Meaning |
|---|---|
| **Publication-blocking error** | A validation finding that makes the proposed agent configuration unsuitable for publication under its template's requirements. |
| **Validation warning** | A finding surfaced for user attention that does not necessarily prevent publication; the precise warning policy remains an open product decision. |

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| Select a predefined template and create an instance | Authorized HR user | Establish a reusable agent whose configuration is constrained by that template version. |
| Replace a working draft | Authorized HR user | Record a new draft version without changing the published configuration. |
| Request validation | Authorized HR user | Assess one exact draft and template version and expose validation findings. |
| Request publication | Authorized HR user | Publish the eligible validated draft as a new immutable active revision. |
| Repeat a publication request after an uncertain response | Authorized HR user | Resolve the original logical publication without creating a duplicate revision. |
| Assign one existing candidate to the current published revision | Authorized HR user | Create one interview assignment that pins the current HR Interviewer active revision. A later publication does not move the pin. Issuing an invitation is a separate access action and needs the new assignment id. |

The current delivery slice may expose these actions through admin APIs. The final MVP still requires the non-developer configuration journey; this document does not claim that an admin UI exists.

## Rules

### BR-CORE-001 — Agent instances use predefined, versioned templates

**Given** an authorized user wants to configure an agent
**When** the user selects a template to create an instance
**Then** the platform supports the predefined HR Interviewer and Document Analyzer templates and associates the instance with its selected template version.

**Why:** Users need a supported starting point whose capabilities and interpretation are known; an unversioned blueprint would make later configuration changes difficult to explain.

### BR-CORE-002 — Configuration stays within the selected template's supported choices

**Given** an instance is based on a predefined template
**When** its configuration is assessed for publication
**Then** unsupported fields, values, or capabilities cannot qualify as a valid configuration for that template.

**Why:** Constrained configuration lets non-developers configure useful agents without accidentally introducing behavior that the platform cannot control or support.

### BR-CORE-003 — Configuration management requires HR authority

**Given** an actor requests instance creation, draft editing, validation, or publication
**When** the platform authorizes the action
**Then** the action is permitted only with current HR authority in the single company's shared configuration scope; the ADMIN role manages HR accounts and grants no configuration-management authority, and candidate interview access grants none either.

**Why:** Interview policy belongs to HR. Account administration and interview participation must not confer the ability to change the questions or rules under which interviews are conducted.

### BR-CORE-004 — Draft replacement protects against stale edits

**Given** a user submits a full draft replacement against an expected draft version
**When** the current version is checked
**Then** the replacement creates the next draft version only if the expected version is current; otherwise it is rejected without overwriting the current draft.

**Why:** A user working from an older view must not silently erase another accepted change and later publish a configuration neither user intended.

### BR-CORE-005 — Editing a draft leaves published configuration intact

**Given** an instance has an active published revision
**When** a user changes its working draft
**Then** the active published revision remains unchanged until a new publication succeeds.

**Why:** Preparing future changes must not disrupt the stable configuration that other users already rely on.

### BR-CORE-006 — Validation evaluates one exact configuration deterministically

**Given** a draft and its associated template version are available
**When** validation is requested
**Then** the result assesses that exact draft/template version against structural, semantic, stable-reference, and locally checkable readiness requirements without a model call.

**Why:** Publication eligibility must reflect repeatable checks of the configuration the user actually submitted, rather than a model's variable judgment or a different draft.

### BR-CORE-007 — Validation errors block publication with an intelligible reason

**Given** validation identifies a publication-blocking error
**When** the user reviews the validation result or attempts to publish that draft
**Then** publication is blocked with an intelligible explanation of the unmet requirement.

**Why:** Users need to understand what must be corrected before they can prepare a usable agent without developer intervention.

### BR-CORE-008 — Publication requires validation applicable to the exact current draft

**Given** a user requests publication with an expected draft version and validation identity
**When** the publication is evaluated
**Then** it is eligible only if the expected draft is current and the validation applies to that exact draft and template version with no blocking findings.

**Why:** Approval of an earlier configuration cannot provide assurance about changes introduced afterward, even when those changes appear minor.

### BR-CORE-009 — A published revision is immutable

**Given** a configuration has been published
**When** a user wants to change its behavior
**Then** the change must go through a draft and a new validated publication; the existing published revision cannot be edited.

**Why:** Users must be able to identify which settings governed a result without the historical configuration changing underneath it.

### BR-CORE-010 — Successful publication establishes the active revision consistently

**Given** a new logical publication is authorized and eligible
**When** publication succeeds
**Then** the new immutable revision becomes the instance's active revision as one committed publication outcome; an unsuccessful publication does not partially replace the active revision.

**Why:** Users need one unambiguous answer to which configuration is currently available for new bindings, including when publication fails partway through.

### BR-CORE-011 — Repeating the same logical publication returns its original outcome

**Given** a publication request has already committed successfully
**When** an authorized user repeats the same request identity with the same intent
**Then** the original publication result is returned without creating another revision or changing the active selection again.

**Why:** A lost response or repeated click must not produce extra versions or restore an older active revision after a newer publication has succeeded.

### BR-CORE-012 — Reusing a publication identity for different intent is a conflict

**Given** a request identity already identifies a publication intent
**When** it is reused within the same scope for a different intent
**Then** the request is rejected as a conflict rather than treated as the original publication or a new one.

**Why:** Users must not receive apparent confirmation for a configuration different from the one they intended to publish.

### BR-CORE-013 — Later publication preserves existing configuration bindings

**Given** an HR assignment or execution already references a published configuration
**When** a newer revision becomes active
**Then** the existing binding retains its original behavioral configuration; the new active revision applies only where a new binding selects it.

**Why:** Changing questions or extraction requirements after an assignment or execution is prepared would make its outcome inconsistent with the conditions originally selected.

### BR-CORE-014 — Publication does not promise immediate execution readiness

**Given** a draft satisfies its publication requirements
**When** it is published
**Then** publication establishes configuration eligibility only; current provider availability, execution authorization, and live safety restrictions remain execution-time concerns.

**Why:** A reusable configuration should not become invalid merely because a provider is temporarily unavailable, and prior publication must never be mistaken for permanent permission to execute.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| A candidate with valid interview access attempts to edit its agent | Deny configuration management. | `BR-CORE-003` |
| Two editors replace draft version 4 | Once one replacement becomes version 5, the other version-4 replacement is rejected. | `BR-CORE-004` |
| A draft changes while revision 2 is active | Revision 2 remains active until another publication succeeds. | `BR-CORE-005` |
| The draft changes after validation but before publication | Reject publication using the old draft/validation combination. | `BR-CORE-008` |
| Validation identifies unsupported invoice configuration | Block publication and explain the unsupported choice. | `BR-CORE-002`, `BR-CORE-007` |
| Publication cannot commit a complete outcome | Do not expose a partial change to the active revision. | `BR-CORE-010` |
| Publication succeeded but its response was lost | A matching retry resolves the original published revision. | `BR-CORE-011` |
| A retry for revision 2 arrives after revision 3 became active | Return the revision-2 publication result; revision 3 stays active. | `BR-CORE-011` |
| A publication request identity is reused for another draft | Reject as conflicting intent. | `BR-CORE-012` |
| An assignment pins revision 1, then revision 2 is published before interview start | The assignment remains bound to revision 1. | `BR-CORE-013` |
| HR issues an invitation with a candidate id and no assignment exists | Do not create an assignment. The invitation can target only an existing assignment. | sibling `BR-CORE-030`, `BR-CORE-031` |
| A provider is unavailable while a locally valid draft is being published | Provider availability alone does not invalidate the configuration; activation checks availability later. | `BR-CORE-014` |
| Access is revoked after publication | Publication does not override the current restriction. | `BR-CORE-014` |

## Examples

| Input | Expected outcome |
|---|---|
| Document Analyzer draft 3 selects `invoice_number`, `total`, and `currency` as required fields, with all other template checks passing | Validation can qualify draft 3 for publication using this supported subset. |
| Document Analyzer draft 4 requests an arbitrary new field named `customer_credit_score` | The unsupported configuration cannot be published as valid. |
| HR Interviewer draft 7 was validated; the user replaces it with draft 8 and submits the validation for draft 7 | Publication is rejected; draft 8 needs applicable validation. |
| Request `publish-invoice-007` publishes draft 3 as revision 1; the identical request is retried after a response timeout | Return revision 1 without creating revision 2. |
| Request `publish-invoice-007` is reused to publish draft 4 | Reject the conflicting intent. |
| Assignment `interview-demo-042` pins HR revision 2; HR revision 3 becomes active before the candidate starts | The assignment retains revision 2. |
| Document execution `invoice-demo-018` uses revision 1 requiring `total`; revision 2 later adds `currency` to the required set | The existing execution retains revision 1's requirements. |

## Known gaps

The assignment-create contract is approved in the Agent Configuration technical design, section 11, and T13 mounts `POST /admin/interview-assignments`. Calling the invitation path with a candidate id still returns not found.

Consent, execution start, a provider session, voice binding, Document Analyzer jobs, brokers, GET/revoke/report assignment routes, and a job-description CMS stay outside this capability.

## Open questions

- **Product owner and template owners:** Which concrete findings are warnings for each template, and do any warnings require acknowledgment before publication? The sources establish that errors block publication and warnings do not necessarily block it, but do not define a complete warning policy.
- **Platform owner and template owners:** When a new template/behavior version is deployed, what compatibility or deployment restriction preserves already referenced behavior versions? The technical design leaves the exact policy open; existing bindings must remain stable.
- **Product owner:** Confirm that the business consequences expressed in each **Why** line capture the intended rationale. These are interpretations of the approved design, not independently recorded stakeholder statements.

## Related

- Sibling capability: [Identity and Access](./identity-access.md), BR-CORE-020..039 — owner-approved ADMIN/HR separation, current session authority, shared company scope, candidate exclusion, and invitation targeting an existing assignment (`BR-CORE-030`, `BR-CORE-031`).
- Source document: [Agent Configuration technical design, section 11](../../agent-configuration-publication/technical-design.md#11-interview-assignment-create-contract-approved-t12) — approved interview-assignment create contract. T13 implements the route without new SQL.

- Source document: [Core AI Agent Runtime — Design Direction v0.1](../../../../design/core-ai/core_ai_agent_runtime_v0.1.md) — reusable runtime direction and separation from agent-specific behavior.
- Source document: [Shared AI Agent Execution Platform — Architecture Design v0.1](../../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md) — shared terminology and configuration lifecycle; later explicit decisions take precedence where amended.
- Source document: [Consolidated Architecture & MVP Scope Freeze v0.1](../../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) — sections 4.1 and 7.1 establish publication semantics and P-01–P-05 acceptance targets.
- Source document: [Technical Design v0.1](../../../../design/core-ai/technical-design.md) — sections 3–5 refine draft versions, validation, publication identity, configuration binding, and transactional consistency; sections 9–10 define administrative access direction.
- Sibling capabilities: None documented yet for `core-ai`. Execution admission/lifecycle, HR access/interview/reporting, and invoice analysis remain separate documentation candidates.

The approved numbering block is `BR-CORE-001..BR-CORE-014`, covering configuration and publication. These rules describe intended behavior; the source designs and acceptance targets do not establish implementation evidence.
