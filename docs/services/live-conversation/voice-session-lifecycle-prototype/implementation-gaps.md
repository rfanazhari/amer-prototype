# Implementation gaps: Voice session lifecycle prototype

Date: 2026-10-31. Ticket: `LIVE-POC-T6`.

The confirmed code gaps below are done. The provider `invalid_offer` / SDP `EOF` rejection is recorded in [rca-invalid-offer-sdp-eof.md](rca-invalid-offer-sdp-eof.md). `T6` is done in the task plan.

## Closed code gaps

| Gap | Resolution |
| --- | --- |
| Setup failure was shown as a dropped connection | Local release sets `stopping` before close and detaches the data channel before `close`. A non-success setup stays `Error` with `error.message`. |
| Stop was replaced by the drop state | Stop sets `stopping`, releases the session, and leaves status `Idle`. The later `close` event does not call `handleDrop`. |
| ICE gathering that never completes stayed `Connecting` | `waitForIce` fails on gather failure or its time limit and the page shows `Disconnected`. |
| A recoverable ICE `disconnected` ended the session | Only `iceConnectionState` `failed` releases the session. |
| Provider failure causes were not logged | `clientmanager` v0.10.1 records the provider call on the request trace. The public `503` body still omits the key and the provider body. |
| Provider returned `invalid_offer` / `failed to unmarshal SDP: EOF` | `Negotiate` no longer sends the `TrimSpace` result. The offer keeps a final CRLF. Details are in the RCA. |
