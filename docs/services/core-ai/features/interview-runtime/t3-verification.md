# T3 verification — CORE-IRT-003

Only T3 is implemented. The workflow is unmounted and uses interview-owned ports.
No SQL, migration, provider adapter, HTTP route, classifier, report, or real lifecycle
fence is added. SQL mocking is not applicable; interface mocks use Mockery.

## Design mapping and assessment

Design §5–§6 and decisions D1–D4 govern the implementation.
`domain/interview.Progress` retains authorization, evidence, stage order, stored
follow-up policy, hard deadline and reconnect counters. `Workflow` coordinates:

- `Propose`: fence/read, authorize, optional narrative, then reauthorize inside
  `WithProgress` and `ProgressStore.Update`; commit structured evidence and close
  the source shell. No create-session call occurs here.
- `OpenStage`: a separate call reads the committed handoff, reserves the successor
  shell, then creates and attaches the provider session outside the transaction.
- `Activate`: pass section 10 budgets to the lifecycle (2 attempts, 30 seconds each,
  90-second activation window; 120 seconds, 2 attempts and 2 episodes for recovery).
  Rejected admission opens no provider session. A provider setup retry keeps the
  same progress and execution; the mocked lifecycle refuses the third admission.
- `Disconnect`/`Reconnect`: inject normalized UTC millisecond time, store backend
  detection, reject stale authority, consume stored recovery capacity, replace the
  session on the same execution, supply only the latest committed checkpoint and
  its final entries. Uncited speech is excluded. Failure retains consumed capacity.
- All writes join the lifecycle callback context and check progress version,
  distinct from the execution version. Provider identity is stored only after
  successful create/attach; partial setup cleanup is bounded and best effort.

Assessment: **retained with rationale**. Domain invariants remain in the aggregate;
application methods own ordering and I/O orchestration. No domain service, SQL or
provider-event policy is needed. The optional `InterviewSummarizer` port is the
minimal concrete refinement needed to exercise narrative failure independently.
Its mock was generated using `make mocks`, not handwritten.

A deadline regression test showed why trusted time must be read again in the
handoff callback after optional summary work: reaching the exact hard deadline
must reject the handoff even if initial authorization succeeded. This was fixed
and the focused test rerun successfully within the same T3 implementation slice.

## RED checkpoint

Cwd for all commands: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
env GOCACHE=/tmp/irt-t3-go-cache go test ./application/usecase/interviewusecase -run '^TestWorkflow' -count=1
```

Observed intended failure: successful proposals, activation and disconnect returned
`interview.ErrConflict`; activation did not pass budgets to the lifecycle. The
scenario assertions executed. RED contained tests, generated summary mock, its
port, and only compile-required workflow fields, inputs and inert method stubs.
An initial fixture-only nil-map mismatch was corrected before the recorded RED.

SHA: `5dd60b12756cfae9bedfd607fcb40704f17f46b8`.
Subject: `test: [CORE-IRT-003][rfanazhari] RED cover interview workflow ordering and budgets`.

## GREEN verification

```sh
env GOCACHE=/tmp/irt-t3-go-cache make mocks
env GOCACHE=/tmp/irt-t3-go-cache go test ./application/usecase/interviewusecase -run '^TestWorkflow' -count=1
env GOCACHE=/tmp/irt-t3-go-cache make test
env GOCACHE=/tmp/irt-t3-go-cache make test.architecture
env GOCACHE=/tmp/irt-t3-go-cache GOLANGCI_LINT_CACHE=/tmp/irt-t3-lint-cache make lint
env GOCACHE=/tmp/irt-t3-go-cache make test.security
env -u TEST_DATABASE_URL GOCACHE=/tmp/irt-t3-go-cache make test.all
```

All final checks passed. Lint: 0 issues. Security: 0 called vulnerabilities;
1 module-only vulnerability not reached by this code. Initial security execution
could not resolve `vuln.go.dev`; the authorized network rerun passed. The first
all-package run could not bind an existing `httptest` listener; the authorized
localhost-port rerun passed. Intermediate RED test runs are not final failures.
The GREEN SHA and actual subject are supplied in the handoff.

Tests cover unauthorized and stale proposals writing nothing, handoff before
create, summary failure retaining evidence, commit failure, locked fence refusal,
activation budgets/refused third attempt, reconnect identity and checkpoint
context, old-attempt rejection, recovery exhaustion and 120-second equality,
missing handoff, invalid authority, stale progress, and create/attach failure.

Database suites and live probes were **skipped**. No integration database was
supplied. These mocked tests establish neither PostgreSQL atomicity nor actual
lifecycle fencing, provider availability, live reconnect, or H-01/H-08 acceptance.

## Subsequent task obligations

T8 must add candidate authorization, request replay/SDP answer persistence, ready,
HTTP contracts and mounting. Lifecycle replay currently refuses duplicate create;
it does not implement the HTTP replay response. T9 builds stage instructions;
T3 does not supply a production stage prompt. T10 adds real transaction proof and
post-handoff target retry behavior. T16 supplies the still-deferred retry count;
T3 has no automatic retry loop or invented next-stage budget. T11 retains terminal
completion/failure ownership. Q1/Q2/Q3 and other task checklists remain unchanged.
