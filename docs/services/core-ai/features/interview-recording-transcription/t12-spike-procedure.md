# T12 Spike Procedure: Live Transcript Events

Ticket `CORE-IRTREC-012`. Prepared 2026-11-04. **Status: prepared, not run.** No provider call, database query or recording download has been made for this task. Running it needs the explicit owner authorizations in section 2. The decision it feeds is the proposed D5 ([proposal](proposal-per-stage-live-transcript.md)); accepted decisions D10 and D11 do not depend on it.

## 1. What the spike answers

| # | Question | Task DoD item |
| --- | --- | --- |
| 1 | Do both transcript event types arrive on our account, with `start_ms`, `end_ms` and an id? Which events are missing, late, duplicated or out of order? | 1 |
| 2 | How does an event interval relate to the same speech in the downloaded stereo WAV (constant shift, or only approximate)? | 2 |
| 3 | How does the live text compare with `gpt-transcribe` per channel, especially roles, technical terms and numbers? | 3 |
| 4 | How long does a provider session take to become ready, and what pause does a candidate perceive between two sessions? | 4 |
| 5 | What is lost when the observing sideband is cut mid-session, does it recover, and does the close confirmation still arrive? | 5 |
| 6 | Which event carries a model function call (the stage-end trigger candidate)? | 6, **not covered here** (see section 9) |

The spike measures. It sets no threshold. The owner writes acceptance thresholds in section 7 **before** the run, so the result cannot choose its own bar.

## 2. Authorizations needed before running

The owner confirms each one; none is assumed.

- [ ] Provider calls on the owner's account, including cost. State a ceiling.
- [ ] One or two interview runs with a **synthetic** candidate speaking the script in section 5, on a test assignment. Recording is already requested for new sessions (`session.store: true`).
- [ ] A read-only lookup of `interview_voice_sessions` on the development database (provider session id and timestamps only). The database must be one authorized for use.
- [ ] Installing `websocket-client==1.9.0` into a temporary directory outside the repository (the T1 precedent).
- [ ] Storing the run's audio and text outside the repository in a private directory, and deleting them afterwards (section 8).

## 3. Safety and data handling

- Use only the synthetic script. Do not use a real candidate. The earlier sample interview was not asserted to be synthetic.
- The key stays in `OPENAI_API_KEY`. Never paste it into a command line, a note or a commit. Use the header file described in step 6 for `curl`, so the key does not appear in process arguments.
- Raw audio, text deltas, transcripts and header files stay in a private directory (`0700`) **outside any git working tree**. The harness refuses to write inside one.
- What may be recorded in the repository: counts, timings, event type names, key names, offset statistics, error rates and HTTP status codes. Never text, audio, ids of real people, signed URLs or credentials.

## 4. Harness (kept outside the repository)

Prepared next to this procedure but **not** committed, following the T1 precedent that investigation helpers are temporary and are not production tooling:

- `t12_harness.py`: `observe`, `summarize`, `align`, `wer`, `split`, `join`, `livetext`. Standard library only, except `observe`, which imports `websocket-client` from `T12_DEPS`.
- `test_harness_offline.py`: nine offline checks on synthetic data (event counting and duplicates, recovery of an injected 150 ms shift against a synthetic WAV, WER and watch-list, split/join/livetext, private modes, the git-tree refusal, argument guards, nested event extraction). They passed on 2026-11-04. This shows the harness logic works on synthetic data; it says nothing about the real event shape.

Assumptions the first real run may break: the transcript events are found by their `type` anywhere inside the frame (top level or nested), and the stereo WAV puts the candidate on the left and the interviewer on the right (documented, and used in T1). Any other event types are logged by name and key names only.

## 5. Synthetic script and watch list

Read naturally in Bahasa Indonesia, with one deliberate pause of about four seconds inside an answer and one interruption of the interviewer. All details are invented.

> Halo, nama saya Budi Contoh. Saya pernah memimpin proyek aplikasi survei online dengan tim satu back-end, satu front-end, dan satu QA selama dua belas bulan. Saya memastikan setiap isu produksi bisa direproduksi dan diperbaiki. Untuk kolaborasi, saya meminta log dari tim back-end lalu meminta QA memeriksa apakah isu itu berasal dari front-end atau dari gap bisnis. Kalau stakeholder meminta perubahan di tengah sprint, saya menilai dulu apakah perubahan itu urgent. Ekspektasi gaji saya dua puluh lima juta rupiah per bulan karena pengalaman saya lebih dari lima tahun. Saya juga meminta asuransi swasta. Pertanyaan saya, apa target sembilan puluh hari pertama? Terima kasih.

Watch list (`terms.txt`; alternatives separated by `|`):

```text
qa
back-end|backend
front-end|frontend
dua puluh lima juta|25 juta
lima tahun|5 tahun
dua belas bulan|12 bulan
sembilan puluh hari|90 hari
```

Save the script as `reference_candidate.txt`. The interviewer has no fixed script, so interviewer text is compared only between the live events and `gpt-transcribe`.

## 6. Procedure

Variables used below (set them yourself; do not echo secrets): `OUT` a private directory outside the repo, `T12_DEPS` the temporary dependency directory, `H` the path to `t12_harness.py`, `PID` the provider session id, `EXEC` the execution id.

**Step 0, setup.**
1. `umask 077; mkdir -p "$OUT" "$T12_DEPS"`.
2. `python3 -m pip install --target "$T12_DEPS" websocket-client==1.9.0`.
3. Run `python3 -I test_harness_offline.py` once more; all checks must pass.

**Step 1, start a run.** Start the application as in the root `README.md`, open the candidate page for the test assignment and accept the notice. Keep the microphone muted until the observer prints `attached`. The interviewer's opening line is spoken regardless of the microphone, so its first fragments may be missed if attach is late. Record that as a finding; it is the late-attach risk in the concept.

**Step 2, find the provider session id (read-only).** After the page requests the voice session:

```sql
SELECT id, stage, provider_session_id, opened_at, readiness_at, closed_at
FROM interview_voice_sessions
WHERE execution_id = '<EXEC>'
ORDER BY opened_at;
```

Use the connection string you were authorized to use; do not print it.

**Step 3, attach the observer (covers items 1 and 5).**

```bash
T12_DEPS="$T12_DEPS" python3 -I "$H" observe --session-id "$PID" --out "$OUT/run1" --cut-after 60 --cut-for 10
```

`--cut-after` closes only the observer's own connection at 60 seconds, waits 10 seconds and reattaches. It does not touch the application's connection or the WebRTC media. If the first attach fails (status is printed, nothing else), stop and record the status: the provider may allow only one sideband per session, which would itself be the finding. A temporary local logging change in `Driver.Observe` is a production-code change and needs a separate approval; it is not part of this procedure.

**Step 4, speak the script** with the pause and the interruption. Then press the candidate "Selesai" control. The application closes the provider session; the observer should log whether `session.closed` reaches it (item 5, and the close observation still missing from T1).

**Step 5, repeat once** with a second execution (or a replacement session) to get a second session. This gives two setup times and a perceived gap. It is **not** a true stage handoff, which does not exist until `CORE-IRT-023`.

**Step 6, download each recording.** Create `auth.hdr` with mode `0600` holding the `Authorization: Bearer ...` line, then:

```bash
curl -sS -H @"$OUT/auth.hdr" -o "$OUT/run1/session.wav" -w '%{http_code} %{size_download} %{time_total}\n' "https://api.openai.com/v1/live/sessions/$PID/content"
```

If the status is not 200, retry every 5 seconds for at most 10 minutes (spike parameters, not production budgets) and record each status code with its elapsed time. This is the first real data for pending versus unavailable; do not infer a classification from one 404 (T1). Record the byte count and SHA-256 of a successful download. Delete `auth.hdr` afterwards.

**Step 7, event summary (item 1).**

```bash
python3 -I "$H" summarize --events "$OUT/run1/events.jsonl"
```

**Step 8, alignment against the WAV (item 2).**

```bash
python3 -I "$H" align --events "$OUT/run1/events.jsonl" --wav "$OUT/run1/session.wav" --out "$OUT/run1/align.json"
```

The merge gap, search window and speech threshold are tool parameters, not decisions. Report the values used.

**Step 9, text comparison (item 3).**
1. `python3 -I "$H" split --wav "$OUT/run1/session.wav" --out "$OUT/run1/chunks"` (120-second fixed cuts, as in T1).
2. For each chunk: `curl -sS -H @"$OUT/auth.hdr" -F model=gpt-transcribe -F file=@"$OUT/run1/chunks/left_01.wav" https://api.openai.com/v1/audio/transcriptions -o "$OUT/run1/chunks/left_01.json"` (same for every left and right chunk).
3. `python3 -I "$H" join --inputs <left chunk JSONs in order> --out "$OUT/run1/gt_candidate.txt"`, and likewise for the interviewer from the right chunks.
4. `python3 -I "$H" livetext --text "$OUT/run1/text.jsonl" --speaker candidate --out "$OUT/run1/live_candidate.txt"`, and likewise for the interviewer.
5. Compare: `wer --reference reference_candidate.txt --hypothesis live_candidate.txt --terms terms.txt`, the same with `gt_candidate.txt`, and live against `gt_` for each speaker.

Split cuts speech at fixed boundaries; a word crossing a boundary counts as an error in the chunked text. Note that effect rather than hiding it.

**Step 10, timings (item 4).** From the SQL above, setup time is `readiness_at - opened_at` for each session. Add the pause you hear between the two sessions by stopwatch, labelled as a human measurement.

## 7. Thresholds the owner sets before the run

Leave blank until the owner fills them; none is proposed here.

| Measure | Acceptable for D5 if |
| --- | --- |
| Share of expected events present and not duplicated | |
| Onset offset spread against the WAV (constant shift vs approximate) | |
| Live WER against the script, and watch-list terms found | |
| Live text compared with `gpt-transcribe` | |
| Session setup time and perceived handoff pause (revisit trigger for D10) | |
| Behavior after the observer is cut, and arrival of `session.closed` | |

## 8. Record and cleanup

Write the result into [technical-design.md](technical-design.md) §6 as a new dated, sanitized subsection and update the T12 evidence line in the [task plan](task-plan.md). Include: run count, provider session count, event type names and key names, counts per type, duplicate and out-of-order counts, arrival-lag spread, alignment statistics with the tool parameters, WER and watch-list results, HTTP status codes with elapsed time for the recording download, setup times, the cut-and-reattach result, and whether `session.closed` arrived. State what the run did **not** show. Do not include text, audio or identifiers.

Afterwards delete `$OUT` and `auth.hdr`, or move them somewhere private under the owner's retention rules. No retention period is approved for them (Q2).

## 9. Limits of this spike

- Item 6 (function call event) needs a session created with delegation and a function tool. That changes the create payload and belongs to the `CORE-IRT-021` live spike (Step 0 there). It is not covered here.
- Two ordinary sessions are not a stage handoff. The real handoff pause is measured only after `CORE-IRT-023` exists, or approximated here by setup time plus a human stopwatch.
- The alignment tool uses a simple energy threshold. It finds speech activity, not words; overlapping speech on one channel can merge. It shows whether an offset is constant, not word-level accuracy.
- A single scripted speaker and two runs show feasibility and failure modes, not a distribution. Do not generalize rates from them.
- The provider's event shape, whether a second sideband attach is allowed, and whether a late attach receives earlier events are unknown until the run.
