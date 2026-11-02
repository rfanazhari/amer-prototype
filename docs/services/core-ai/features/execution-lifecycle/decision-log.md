# Decision Log: Execution Lifecycle

Technical choices in this feature that exist because of business rules. Each entry says why the rule forces the choice, what people will observe, and when to look again. None of these entries means the behavior is already built or already tested.

The rules are BR-CORE-050 through BR-CORE-059 in the Execution Lifecycle capability. That capability is still planned. The short "why" lines in the capability document are the approved reading of the platform design. They are not a second sign-off from a separate stakeholder workshop.

## D1: One assignment has one execution

- **Business driver:** BR-CORE-050. HR and the candidate must be able to point at one interview for one invitation. Two runs would make the outcome ambiguous. The current database already allows only one execution row per assignment, but that row does not yet have a lifecycle.
- **Technical decision:** Starting an interview locks the assignment and then writes at most one execution. A repeated start returns that same execution. A future document task is not forced to use an assignment. One-per-assignment is the interview rule, not a rule for every kind of agent.
- **What this means for the product:** A double click or a lost response still shows one interview. HR cannot accidentally hold two interviews for the same invitation. A later invoice task can still be run more than once when that product says so.
- **Alternatives considered:** Letting every click create a run is simple and makes the outcome ambiguous. Treating "one input, one run" as a universal law would stop a legitimate second pass over another invoice.
- **Trade-offs accepted:** The interview start path has to know about assignments. The shared record of a run does not. Callers must use the interview start for interviews and the shared admit path for anything else.
- **Revisit when:** Product allows more than one interview on the same assignment, or asks for a universal one-run-per-input rule.
- **Status:** active

## D2: An accepted start is not the interview actually beginning

- **Business driver:** BR-CORE-051. The candidate and HR need a durable answer that the interview exists even when the voice provider has not connected. The later platform design names that accepted-but-not-started condition `CREATED`. An earlier architecture sketch that added a waiting state is not the rule we are implementing.
- **Technical decision:** Saving the start stores the identity and the configuration binding and marks the run accepted. A separate activation step is what marks it running. Failure to connect does not create a second interview and does not, by itself, mean the run has started.
- **What this means for the product:** After a successful start, the interview exists even if no voice session has begun. The screen must not describe that moment as a live interview. Connecting later continues the same interview.
- **Alternatives considered:** Marking the run live at the moment of the click would be simpler to store and would tell HR the conversation had started when it had not. Adding extra public states for "connecting" and "reconnecting" would make every reader learn provider mechanics.
- **Trade-offs accepted:** Start and activation are two steps even when the product later calls them one after another. How many connection retries are allowed is left to the interview runtime, not fixed here.
- **Revisit when:** Product wants the candidate to see a distinct "connecting" state of its own, with copy and a timeout that have been explicitly chosen.
- **Status:** active

## D3: A finished execution never reopens

- **Business driver:** BR-CORE-054. Everyone reading the interview must see one current answer, and a finished interview must not quietly become unfinished. The platform also says the first saved ending wins when completion and cancellation happen together.
- **Technical decision:** The only endings are completed, failed, and cancelled. After one of those is saved, later requests cannot move the run backward. If two endings race, the one saved first remains. Repeating that same ending returns it. Asking for a different ending does not change the record.
- **What this means for the product:** A completed interview stays completed. Cancelling it afterward does not undo it. Failing it afterward does not undo it. A candidate who refreshes still sees that same ending.
- **Alternatives considered:** Reopening a finished run would let a late provider message rewrite the result HR already saw. Keeping only the latest request, whatever it is, would let a slow cancellation replace a completion that already succeeded, or the reverse.
- **Trade-offs accepted:** A mistaken ending cannot be repaired by running the interview again on the same record. A new interview needs a new assignment. Operators need a separate, explicit repair process if bad data is ever saved.
- **Revisit when:** Product defines a supervised correction process for a wrong ending. That process has to say what happens to the record people have already seen.
- **Status:** active

## D4: Completion requires the saved interview record, not a successful provider connection

- **Business driver:** BR-CORE-056. A model finishing its connection is not evidence that the interview, or a later document task, produced the record the business agreed to keep. The interview runtime is what decides which record is required. This lifecycle only refuses to complete without a committed reference to that record.
- **Technical decision:** The complete action has no "the provider succeeded" switch. The run becomes completed only from the running state, and only after the owning workflow confirms the required progress is already saved. The shared lifecycle stores a reference to that progress. It does not store the transcript or the invoice.
- **What this means for the product:** The interview is not complete just because the voice session ended. If the required record is missing, the interview stays unfinished or is failed with whatever progress was truly saved. HR does not see a successful interview that has no record.
- **Alternatives considered:** Completing when the provider disconnects cleanly would mark interviews successful before the transcript exists. Letting this shared layer understand questions or invoice fields would force every future agent to look like an interview.
- **Trade-offs accepted:** The interview runtime must actually save progress before it asks for completion. Until that runtime exists, tests can only simulate the confirmation. A completed run can still lack an HR report; the report is a later document and must not change this ending.
- **Revisit when:** The definition of "required progress" changes for interviews or for document tasks. The confirmation step stays; only the workflow behind it should change.
- **Status:** active

## D5: Later publication does not rewrite the run that already started

- **Business driver:** BR-CORE-058, together with the configuration rule that an existing binding keeps its revision. Questions or extraction rules changed after a run begins would make the result describe a different piece of work from the one that was started.
- **Technical decision:** When an interview start is saved, it copies the revision already pinned on the assignment. It does not read the newest published revision. Nothing in the lifecycle is allowed to change that copy afterward. Publishing again changes only what a new assignment or a new document run will pick up.
- **What this means for the product:** HR can publish better questions while a candidate is in an interview. That candidate's interview keeps the questions it started with. The new questions apply to interviews that start afterward.
- **Alternatives considered:** Always using the newest published settings would make a live interview change meaning in the middle. Asking the candidate's browser to send the revision would let a client pick a configuration the assignment did not pin.
- **Trade-offs accepted:** Two candidates invited at different times can be on different question sets. Support has to read the revision stored on the run, not the agent's current published version.
- **Revisit when:** Product wants an explicit, visible migration of an interview that has not yet started, with the candidate told which question set will be used.
- **Status:** active

## D6: The shared lifecycle stays free of interview stages and invoice fields

- **Business driver:** BR-CORE-059. A shared ending is useful only when a non-interview agent can reach it without pretending to be an interview. The platform acceptance criterion P-12 asks for the same separation. The runnable invoice product is not part of this work.
- **Technical decision:** The shared record stores identity, state, whose attempt is current, and a reference to progress that was already saved. It does not store stage names, question lists, transcript text, or invoice fields. Interview-only uniqueness stays on the interview assignment link. A document run can be saved without that link.
- **What this means for the product:** The interview and a future document task can share one notion of "accepted, running, and finished." Building the document task does not require fake interview stages. This plan does not itself deliver document extraction.
- **Alternatives considered:** Adding the lifecycle columns only onto the interview table would ship faster and would make the document task carry an assignment it does not have. Copying the whole state machine later would risk two different meanings of "finished."
- **Trade-offs accepted:** Interview start is a separate step from the shared save. The full proof that a real document task uses this lifecycle waits for that task to be designed. Until then, the proof is that the shared model can finish a run that has no interview row.
- **Revisit when:** A second agent cannot meet its completion rule without a field this shared record refuses to store. Add that fact to the agent's own record, or revisit this decision openly. Do not quietly add interview or invoice columns to the shared run.
- **Status:** active

## D7: Revocation or expiry during a live run cancels it

- **Business driver:** BR-CORE-055. Closing the candidate's right to be in the interview must also close the interview. Leaving it running after access was withdrawn would record an interview the company no longer authorizes. Replacing an invitation link is not the same event: that action must not cancel an interview already underway.
- **Technical decision:** When assignment access is revoked, or the candidate's authority to continue expires during the run, the lifecycle saves cancellation if the run is not already finished. The save locks the assignment first and the run second, so it cannot race an unprotected start. A run that is already finished stays finished. The lifecycle does not decide the two-hour session clock and does not treat a replacement invitation as expiry.
- **What this means for the product:** If HR withdraws the assignment while the interview is still going, that interview ends as cancelled. If the candidate's authority lapses during the interview, the interview ends as cancelled. If HR only replaces the entry link, the interview continues. A browser disconnect, by itself, does not cancel the interview.
- **Alternatives considered:** Only rejecting the next click would leave a running interview in place after access was removed. Cancelling on every disconnect would throw away an interview the candidate can still rejoin. Cancelling when a link is replaced would interrupt an interview that the access rules say should continue.
- **Trade-offs accepted:** Something outside this lifecycle has to notice revocation and expiry and ask for the cancellation. Today no product command does that; tests can only simulate the request. Until that caller exists, a revoked row in a fixture is not the same as an HR button. D8 names the approved revocation caller. The expiry half of this decision stays open.
- **Revisit when:** Product changes what "access ended during the interview" includes, especially if replacing an invitation should also end the live interview.
- **Status:** active

## D8: The HR revocation command lives in Identity and Access

- **Business driver:** BR-CORE-055 needs a real withdrawal before a live interview can be shown as cancelled. On 2026-11-02 the owner approved putting that command in Identity and Access, leaving this lifecycle's T10 as the later proof, and leaving session expiry during a run undecided.
- **Technical decision:** Identity and Access T22 (`CORE-IAM-022`) stores the withdrawal time and then asks this lifecycle to cancel a run that is not finished, using the existing cancellation operation and the reason `access_revoked`. This capability does not gain an HR route. Replacing an invitation still does not cancel the interview. A candidate who is merely refused access is not treated as having cancelled it. A session that runs out during the interview waits for a separate decision.
- **What this means for the product:** Once that access command exists, HR can withdraw an assignment and the unfinished interview ends as cancelled. Until then, a test that stamps the withdrawal by hand is not the product behavior. An interview is not yet cancelled only because the candidate's time ran out.
- **Alternatives considered:** Building the HR command inside this lifecycle would mix access administration into the run record. Waiting for the candidate screen to cancel it would make the candidate the actor who withdraws the assignment.
- **Trade-offs accepted:** The withdrawal and the cancellation are two saves. The access command reports success only when both have completed. This lifecycle does not change its cancellation operation for that command.
- **Revisit when:** The session-expiry trigger is decided, or the withdrawal command needs a different actor.
- **Status:** active
