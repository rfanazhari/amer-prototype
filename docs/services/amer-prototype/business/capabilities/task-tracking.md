---
id: task-tracking
service: amer-prototype
status: partial
rules: [BR-AMER-001, BR-AMER-002, BR-AMER-003, BR-AMER-004, BR-AMER-005, BR-AMER-006, BR-AMER-007, BR-AMER-008, BR-AMER-009]
updated: 2026-10-31
implementation: [domain/aggregate, domain/valueobject, application/usecase/taskusecase, application/query/taskquery]
---

# Task tracking

## Purpose

Record work, follow its progress, and identify overdue tasks.

## Ubiquitous language

A Task represents work. Todo means not started; In Progress means started; Done means completed and final. Overdue is an observation, not a separate state.

## Actors and triggers

A general user creates, starts, completes, reschedules and views tasks. Authentication, ownership and assignment are outside this reference feature.

## Rules

### BR-AMER-001 — A task has a valid title

**Given** a person records work. **When** they submit a title. **Then** leading and trailing Unicode whitespace is removed and the remaining title has 1–100 Unicode code points. **Why** tasks need a useful and bounded name.

### BR-AMER-002 — New tasks begin as Todo

**Given** a task is accepted. **When** it is first recorded. **Then** its state is Todo. **Why** work starts from a consistent lifecycle state.

### BR-AMER-003 — Due dates are optional and cannot be in the past when changed

**Given** a task is created or a non-Done task is rescheduled. **When** a due date is supplied. **Then** it is accepted only when it is equal to or later than current time; omitting a date on creation means no date, and clearing an existing date is allowed. **Why** dates should represent actionable plans.

### BR-AMER-004 — Only Todo tasks can start

**Given** a recorded task exists. **When** a person starts it. **Then** only Todo changes to In Progress. **Why** repeated starts must not rewrite history.

### BR-AMER-005 — Only In Progress tasks can complete

**Given** a recorded task exists. **When** a person completes it. **Then** only In Progress changes to Done. **Why** unstarted work cannot be recorded as completed.

### BR-AMER-006 — Done is final

**Given** a task is Done. **When** a person attempts to start, complete or reschedule it. **Then** the operation is rejected and recorded facts remain unchanged. **Why** completed history should remain trustworthy.

### BR-AMER-007 — Overdue is derived from date and state

**Given** a task is viewed. **When** current time is strictly after its due date. **Then** it is overdue unless it is Done; equality and the absence of a date are never overdue. **Why** users need an accurate view without changing task history.

### BR-AMER-008 — The system owns creation and completion times

**Given** a task is created or successfully completed. **When** the operation succeeds. **Then** the system records its creation or completion time and rejected operations preserve existing facts. **Why** users should not choose historical system facts.

### BR-AMER-009 — Completion produces an internal event

**Given** an In Progress task is successfully completed. **When** the new state is accepted. **Then** its completion event is accepted together with persistence; refusal preserves the old task and repeated completion produces no additional event. **Why** other modules can react without coupling to task commands.

## Edge cases

Whitespace-only titles and titles beyond the limit are rejected. Due dates equal to current time are valid. Done tasks never become overdue. Failed operations preserve state and recorded times. Events are process-local and can be lost on unexpected process termination; subscriber failure is logged without retry.

## Examples

Prepare report starts as Todo, can be started and then completed. Attempting to complete it before starting is rejected. Rescheduling a non-Done task can remove its due date.

## Known gaps

Historical RED/GREEN evidence is incomplete. Full runtime acceptance and database verification results are maintained in the owning feature design.

## Open questions

The owner has not yet selected a change from AutoMigrate to SQL migrations. Local Docker deployment artifacts are now requested; Redis and RabbitMQ are optional local services and are not Task dependencies.

## Related

[Business index](../README.md), [feature design](../../../../design/task-tracking.md), [HTTP contract](../../../../api/openapi.yaml), and [audit](../../../../gonesis-audit.md).
