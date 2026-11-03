# T19 verification — CORE-IRT-019

## Status

Accepted 2026-11-03 (Asia/Jakarta). One live session was started from the production
candidate interview page by the owner, who confirmed the spoken provider reply and
approved this sanitized note. T19 is checked Done. Q1 is not resolved; no transcript,
scoring or report behavior is claimed.

## Setup

- Production composition through the Compose stack (`make docker.up`, schema 11), page at
  `GET /candidate/interview/page`, live provider endpoint `https://api.openai.com/v1/live/sessions`.
- Data was created through the administrative API: HR account, `hr-interviewer` instance,
  validation, publication, candidate, assignment and invitation. The candidate entered through
  `POST /candidate/access/exchange`. No harness or mock was involved.
- Browser: Chrome, microphone permission granted from the page.

## Observed (from the page and server log; no values recorded)

| Step | Result |
| --- | --- |
| Notice accepted, `POST /candidate/consents` | 201 |
| Start interview, `POST /candidate/executions` | 201, state CREATED |
| Start voice, `POST /candidate/executions/{id}/session-attempts` | 201, one provider create (201), answer SDP returned |
| Provider session id | Retained: present in `interview_voice_sessions` (count 1); value not recorded |
| Answer persisted | `interview_session_commands` succeeded with answer present (count 1) |
| Sideband attach | Not logged separately; setup returned 201, and a sideband failure is mapped to a setup failure, so attach is inferred as successful |
| `.../session-attempts/{attempt}/ready` | 200, condition `ready`; execution RUNNING, version 2 |
| Page state | "Voice session ready" |
| Cancel, `POST /candidate/executions/{id}/cancel` | 200, state CANCELLED, version 3; page shows the cancelled message |
| Spoken provider reply | Confirmed by the owner |

## Not recorded

The key, SDP, instructions, transcript, provider session id and candidate contact details are not
in this note. The page did not show them.

## Observations outside T19 scope

- While RUNNING the page polls `GET /candidate/interview` about every two seconds.
- The provider create request logs `instructions` as an empty string. Whether the pinned
  revision's instructions reach the provider is a separate question for T8/T18.
