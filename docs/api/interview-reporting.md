# Interview reporting API

Ticket: `CORE-IRP-008` (T8). These handlers are implemented and tested but **not mounted in runtime**. Registration belongs to T15 after the writer and evidence adapters exist. See [OpenAPI](openapi.yaml) for schemas and [changelog](CHANGELOG.md) for version 0.16.0.

Both routes require a current HR administrative session and an exact configured `Origin`. HTTP development uses `ia-admin`; HTTPS uses `__Host-ia-admin`. Candidate cookies cannot substitute for the administrative cookie. ADMIN receives `403 IRP_FORBIDDEN` before any operation or assignment lookup. Responses use `Cache-Control: no-store` and the standard JSON envelope. `Accept-Language: id` selects Indonesian messages; English is the default.

## Read

`GET /admin/interview-assignments/{id}/report` accepts no body or query parameters. The id is a nonzero hyphenated UUID; uppercase hex is normalized to lowercase.

`200` returns the newest generation alongside the latest succeeded account. The account remains visible while a later generation is pending, running or failed; it is null when none succeeded.

```json
{
  "status": "success",
  "data": {
    "assignmentId": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "executionId": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
    "generation": {
      "id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
      "number": 1,
      "state": "pending",
      "failureReason": null,
      "createdAt": "2026-11-03T03:00:00.123Z"
    },
    "report": null
  }
}
```

A succeeded account has exactly `generationId`, `observations`, `gaps` and `uncertainties`. Each collection is an array, including when empty. Observations have `body`, `evidenceEntryId` and positive `evidenceRevision`. Gaps have `body` and nullable `questionId`, with no evidence fields. Uncertainties have `body` and optional evidence fields, omitted when uncited. Generations have exactly the five fields above; `failureReason` is nullable. All timestamps use UTC milliseconds.

There are no recommendation, decision, rank, score, `salaryCeiling` or `salaryCeilingRef` fields. Domain rules govern statement wording and valid citations; the handler maps detached application DTOs.

## Regenerate

`POST /admin/interview-assignments/{id}/report-generations` requires the same cookie and Origin, plus one session-bound `X-CSRF-Token`, JSON content type and the body `{}`. Missing/non-object bodies, unknown or duplicate members, trailing JSON, queries and bodies larger than 1 MiB are rejected.

Supply exactly one `Idempotency-Key`, matching `[A-Za-z0-9][A-Za-z0-9._:-]{0,127}` without trimming. Duplicate headers are rejected. The key is scoped to the current HR subject. Its intent fingerprint is lowercase hex SHA-256 of the compact UTF-8 JSON array `["regenerate-interview-report/v1","<assignment-uuid-lowercase>"]`.

- `201`: a new pending generation has committed.
- `200`: the same key and intent returns its original generation, even if later generations exist.
- The response `data` is the generation object, without a report wrapper.

A new key requires a completed interview, finalized evidence and no pending/running generation. Neither route changes interview completion or execution state. A broker outage after the durable outbox commit does not change success into `503`.

## Errors and precedence

After route/method dispatch, requests are checked in this order:

1. Missing, candidate, expired, revoked, disabled or unbound administrative session: `401 IRP_UNAUTHENTICATED`.
2. Non-HR, missing/mismatched Origin, or missing/invalid POST CSRF: `403 IRP_FORBIDDEN`.
3. Malformed id, body, query, content type or key: `400 INVALID_INPUT`.
4. Unknown assignment: `404 NOT_FOUND`.
5. Same key with a different intent: `409 IRP_IDEMPOTENCY_CONFLICT`.
6. Same key and intent: `200` replay, before eligibility checks.
7. Interview incomplete: `409 IRP_NOT_COMPLETED`.
8. No finalized evidence: `409 IRP_EVIDENCE_UNAVAILABLE`.
9. Another active generation: `409 IRP_GENERATION_ACTIVE`.
10. Store unavailable before commit: `503 IRP_DEPENDENCY_UNAVAILABLE`.
11. Corrupt stored rows or unclassified technical failure: `500 INTERNAL_ERROR`.

Unsupported methods return `405 METHOD_NOT_ALLOWED`; unknown routes return `404 NOT_FOUND`. Failures have `status: "error"`, `data: {}`, and `error` containing only the safe localized `code` and `message`. Cookies, CSRF values, provider payloads and internal failure causes are never serialized. Candidate and voice routes retain their existing contracts.
