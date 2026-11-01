# Decision Log: Interview Runtime

Technical choices in this feature that exist because of business decisions. Each entry says why, what candidates and HR will experience, and when to reconsider it.

The rules `BR-CORE-060` through `BR-CORE-071` are planned. These decisions do not mean the interview already behaves this way.

## D1: The interview workflow allows a stage change; the voice model only suggests it

- **Business driver:** `BR-CORE-064`. HR's published rules decide when the conversation may move. The model may suggest that it is time.
- **Technical decision:** A stage change is stored only after the interview progress rules accept the suggestion. A rejected suggestion leaves the stage as it was.
- **What this means for the product:** The candidate stays in the current part of the interview when required questions are still open or a recording failed. The model cannot skip ahead by itself.
- **Alternatives considered:** Letting the provider session advance on its own is simpler to wire, and it is what the voice prototype does. That would make HR's question list optional. Putting the same rules in the generic execution record would force every future agent to look like an interview.
- **Trade-offs accepted:** The model can sound ready before the system agrees, so the candidate may hear a transition that the system then refuses to commit. The product needs that refusal to be uneventful rather than a second interview.
- **Revisit when:** Product allows the model to own progression for a different agent, or HR wants a stage the model may skip without a published rule.
- **Status:** active

## D2: The next part of the interview waits until the current part is saved

- **Business driver:** `BR-CORE-065` and acceptance criterion `H-05`. The next stage has to rest on a record HR can inspect.
- **Technical decision:** The system saves the source handoff first. Only then may it open the next voice session. If a short written summary fails, the saved structured record is enough. If that record is missing, the next session is not opened.
- **What this means for the product:** HR can see what was covered before the interview moves on. A missing save stops the move. A failed summary does not erase the save.
- **Alternatives considered:** Opening the next provider session and saving afterward is faster when the provider is healthy, and it loses the record if the save then fails. Saving only a free-text summary is shorter, and it cannot support coverage counts.
- **Trade-offs accepted:** A pause between stages while the save completes. The candidate may need a new connection for the next stage even though the interview is the same one.
- **Revisit when:** Product accepts a defined repair that can rebuild a missing handoff from raw provider audio. This slice does not keep raw audio.
- **Status:** active

## D3: Each active stage has its own voice session

- **Business driver:** `BR-CORE-066` and `H-04`. A stage boundary is a real break in the provider conversation. Replacing a session after a fault is still the same interview.
- **Technical decision:** Opening a stage opens one voice session. A fault opens a replacement session for that same stage and the same execution. The system does not keep two sessions open for one execution.
- **What this means for the product:** The candidate experiences a new connection at a stage boundary or after a fault, and HR still sees one interview.
- **Alternatives considered:** One provider session for the whole interview avoids reconnects and lets hidden provider memory cross the stage boundary, which the approved recovery rules reject. A new execution per stage would split one invitation into several interviews.
- **Trade-offs accepted:** More provider sessions per interview, and a short setup at each boundary. Context for the next session is limited to what was saved.
- **Revisit when:** A reviewed provider capability can reset context inside one session without keeping hidden memory, and product accepts that as the stage break.
- **Status:** active

## D4: A lost connection continues the same interview

- **Business driver:** `BR-CORE-069` and `H-08`. One deliberate disconnect and reconnect is the acceptance case. Further kinds of realtime failure stay outside this capability until product extends them.
- **Technical decision:** Reconnect, within the approved window, opens a new voice session on the same execution and continues from the last saved checkpoint. Speech that was never saved may need to be repeated. The operational ceiling from the later technical design is 120 seconds from the moment the backend detects the drop, two replacement attempts in an episode, and two episodes. The acceptance test exercises one episode.
- **What this means for the product:** A dropped call is a reason to continue, not a reason to start a second interview. The candidate may have to repeat an unsaved answer. A third episode, or a return after the window, ends in a failed interview with the saved progress kept.
- **Alternatives considered:** Cancelling on every drop is simpler and throws away a usable interview. Unlimited reconnects hide provider trouble and were not accepted. Treating the browser's timer as the start of the window would disagree with the backend record.
- **Trade-offs accepted:** The implemented ceiling is wider than the single reconnect the demo must show. Passing the one-reconnect demo does not mean every glitch recovers.
- **Revisit when:** Product expands recovery past one deliberate disconnect, or measured provider behavior requires a different window.
- **Status:** active

## D5: The salary ceiling stays on the server and out of the model's context

- **Business driver:** `BR-CORE-071` and `H-12`. The ceiling is an internal limit. The candidate-facing conversation may discuss expectations. The bound revision's mode is `expectations_only`.
- **Technical decision:** Instructions, candidate responses, and logs do not receive the ceiling reference or any amount. The published revision stores a reference id, and the local resolver does not store an amount. This slice does not invent an amount and does not compare a spoken expectation with a number.
- **What this means for the product:** The interview can ask what the candidate expects. It does not speak or show the company's ceiling, including when the candidate asks for it.
- **Alternatives considered:** Putting the ceiling in the prompt would let the model police numbers and would also let the model reveal them. Hiding the ceiling only in the admin UI would leave the voice model free to repeat it.
- **Trade-offs accepted:** Until product supplies a confidential amount and a comparison rule, the server enforces secrecy and the expectations-only mode. It does not enforce a numeric cap.
- **Revisit when:** Product stores a ceiling amount outside the model context and decides whether an expectation above that amount must change the conversation.
- **Status:** active

## D6: The interview does not hire or reject

- **Business driver:** `BR-CORE-062` and `H-11`. Hiring judgement belongs to HR after the interview. An answered question means a response was recorded.
- **Technical decision:** Coverage has no competence, hire, or reject field. Model instructions do not ask for a verdict. Completion does not produce one.
- **What this means for the product:** HR receives what was said and what was missed. The product does not present a pass or fail for the candidate.
- **Alternatives considered:** A model score would be easy to display and would be an autonomous hiring decision, which the scope freeze excludes.
- **Trade-offs accepted:** HR has more reading to do after the interview. The live demo cannot "show the outcome" as a yes or no.
- **Revisit when:** Product explicitly asks for a recommendation and separates it from the interview record.
- **Status:** active

## D7: Lost audio is not treated as a skipped question

- **Business driver:** `BR-CORE-063`. A missing recording must not be rewritten as the candidate having skipped.
- **Technical decision:** Recording or connection loss is stored as a technical gap. The question stays pending or asked. That gap cannot authorize the next stage and cannot be saved as an omission.
- **What this means for the product:** HR can tell a technical failure from a candidate who skipped or stayed silent. The interview does not move on because audio was lost.
- **Alternatives considered:** Marking the question omitted would let the stage finish and would misstate the candidate's behavior. Leaving the gap only in provider logs would give HR no durable explanation.
- **Trade-offs accepted:** A fault can block progression until the question is actually captured or the hard deadline fails the interview. The saved earlier stages remain.
- **Revisit when:** Product defines a reviewed way to recover the lost utterance itself. Guessing the answer is not that way.
- **Status:** active

## D8: Finishing the interview does not write the HR report

- **Business driver:** The reporting capability owns the write-up. Completion makes a finished execution available to that sibling. Report failure must not turn a finished interview into a failed one (`H-10`). The user scope for this slice excludes the report body.
- **Technical decision:** After the required interview record is stored, this slice asks the lifecycle to mark the execution complete. It does not create a report job, a report row, or a broker message.
- **What this means for the product:** The candidate can reach a finished interview without waiting for the write-up. HR's report arrives through the reporting work, and a report problem leaves the interview finished.
- **Alternatives considered:** Creating the report in the same save as completion matches one row in the platform technical design and couples the two outcomes. This slice follows the capability split instead.
- **Trade-offs accepted:** Completion does not, by itself, start the report. The reporting slice has to notice the completed execution. That hand-off is outside this plan.
- **Revisit when:** Product wants the report to start automatically and assigns that trigger to a named owner. The interview outcome still stays independent of report success.
- **Status:** active

## D9: Stage time comes from the published revision; each stage allows two follow-ups

- **Business driver:** `BR-CORE-060` and `BR-CORE-061`. The candidate and HR must be able to say which questions and time budgets governed the conversation. The representative interview includes a follow-up that refers to what the candidate said, inside the follow-up limit.
- **Technical decision:** Soft stage time is the published `stageSeconds` for `background`, `screening`, `compensation`, and `closing`. The roughly 2 / 4 / 2 minute showcase is guidance for a demo, not the timer. The published form does not contain a follow-up number. This slice stores a limit of two follow-ups on the interview when it activates, from the later platform rule, and keeps that stored number for the life of the run.
- **What this means for the product:** An interview published with its own stage times runs to those times. Each stage can add at most two follow-ups that point at something the candidate already said. A third follow-up is not part of the record.
- **Alternatives considered:** Using the showcase minutes would ignore the revision HR published. Reading "two" from a global setting on every turn would let a later setting change an interview that had already started. Adding a new publication field is configuration work this slice was told not to reopen.
- **Trade-offs accepted:** HR cannot set a different follow-up count per revision until configuration gains that field. The screening spare-time check used at publication remains a publishing rule, not a second live counter.
- **Revisit when:** Product adds an explicit follow-up limit to the published revision. Runs already activated keep the limit they stored.
- **Status:** active
