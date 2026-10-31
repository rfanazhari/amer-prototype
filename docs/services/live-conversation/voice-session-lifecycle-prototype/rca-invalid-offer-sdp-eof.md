# RCA: Provider rejected the WebRTC offer

Date: 2026-10-31. Ticket: `LIVE-POC-T6`.

## Summary

`POST /voice-sessions` reached OpenAI and returned HTTP `400` `invalid_offer` with `Failed to parse offer: failed to unmarshal SDP: EOF`. The offer was present and began with `v=`. The parser failed because the last SDP line had no CRLF. `Negotiate` removed that terminator with `strings.TrimSpace` before the provider call.

## Impact

Setup could not return an SDP answer, so the browser never reached a connected voice session. The public body stayed the safe `503` `SESSION_UNAVAILABLE` envelope. The provider key was not in that body. Process logs from `clientmanager` showed the provider status and the rejected request.

## Evidence

Checked on 2026-10-31 against a local process with a provider key.

| Attempt | What was sent | Provider result |
| --- | --- | --- |
| Multipart `POST /v1/realtime/calls` | Offer about 3440 bytes, prefix `v=` | `400`, SDP `EOF` |
| Same call, part `Content-Type: application/sdp` and `Content-Disposition: form-data; name=sdp` | Same non-empty offer | `400`, SDP `EOF` |
| JSON `POST /v1/live/sessions` | `transport.sdp` held the same offer. The logged text ended at `a=max-message-size:262144` with no CRLF before the closing quote | `400`, SDP `EOF` |

A working local client (`client.Live.New`) posts JSON to `POST /v1/live/sessions` and reads the answer from `transport.sdp`. That shape was adopted after the multipart calls endpoint kept returning `EOF` for a non-empty part. The `EOF` remained until the offer kept its final CRLF.

`GET /voice-session` is the page route and writes HTTP 200. Log lines whose `msg` is `internal server error` and whose latency is 0 do not record that HTTP status. The logger uses that text when the transaction has no response code. The Chrome request for `/.well-known/appspecific/com.chrome.devtools.json` is not part of setup.

## Cause

The browser offer ends with `\r\n`. `Negotiate.Execute` used `strings.TrimSpace` both to reject a blank offer and as the value passed to the provider. `TrimSpace` deletes trailing whitespace, including that final CRLF. Internal line breaks stayed, so the logged SDP still showed `\r\n` between lines and stopped on the last attribute.

The provider unmarshals the offer as SDP and requires every line, including the last, to end in CRLF. The trimmed block ends mid-line, and that unmarshal returns `EOF`.

The page sends `localDescription.sdp` through `JSON.stringify` and does not trim it. The setup handler trims the raw JSON body only to require an object that starts with `{`. That does not change the `sdp` string.

## What this was not

- An empty offer. Measured length stayed above 3400 bytes, and the text started with `v=`.
- A missing multipart part, a quoted `name="sdp"`, or a `text/plain` part. Those variants were sent and still returned `EOF` while the offer bytes were unchanged.
- `EPHEMERAL_KEY`. Setup uses the server key. The browser does not call the provider directly.
- The page status bugs from the empty-key trial. Those were separate and are closed in [implementation-gaps.md](implementation-gaps.md).

## Fix

`TrimSpace` remains the blank-offer check only. The value sent to the provider is normalized: every line ending becomes CRLF, and the last line keeps a CRLF. The adapter appends `\r\n` when the offer it receives does not already end with a newline.

The provider call is JSON `POST /v1/live/sessions` through `clientmanager` v0.10.1. The configured realtime calls URL is sent to that Live path. The default realtime model is sent as `gpt-live-1`. The answer returned to the page is `transport.sdp`.

## Remaining limit

The setup rejection is closed, and `T6` is marked done in the task plan.
