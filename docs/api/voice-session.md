# Voice session setup

`POST /voice-sessions` accepts one JSON object `{"sdp":"<WebRTC offer>"}` and returns `201` with the provider SDP answer. Success data contains `sdp` only. The response does not include the provider key, an authorization header, session instructions, a score, or a transcript.

```json
{"status":"success","data":{"sdp":"<SDP answer>"}}
```

Errors use `{"status":"error","data":{},"error":{"code":"...","message":"..."}}`. `Accept-Language: id` selects Indonesian. The default is English. Request bodies are limited to 1 MiB. Unknown fields, malformed JSON, a non-object body, a missing `sdp`, and a blank `sdp` are `400` `INVALID_INPUT`. Any method other than POST, including `GET /voice-sessions`, is `405` `METHOD_NOT_ALLOWED`. Provider setup failure, a provider call that cannot be completed, or an unset provider key is `503` `SESSION_UNAVAILABLE`. That message is `voice session unavailable` in English and `sesi suara tidak tersedia` in Indonesian. The message and body do not include the provider key, the provider response body, or other technical causes. Unexpected failures are `500` `INTERNAL_ERROR`.

The browser calls this route on the same origin. No CORS surface is added. Task request and response schemas are unchanged. The machine-readable specification is [openapi.yaml](openapi.yaml); contract changes are recorded in [CHANGELOG.md](CHANGELOG.md).
