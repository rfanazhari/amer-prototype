# Decision Log: Interview Reporting

Technical choices in this feature that exist because of a business decision. None of these entries means the behavior is already built or that BR-CORE-080..086 are enforced. Those rules are planned.

## D1: The report is a separate lifecycle from the interview

- **Business driver:** BR-CORE-080. HR needs a write-up after the interview, and that write-up is not the interview result. Approved in the Interview Reporting capability and in architecture decision ADR-09.
- **Technical decision:** Store the report, its generations, and its background job in their own records. The interview execution remains the interview. Starting or replacing a report does not reopen the interview.
- **What this means for the product:** HR can have a pending, failed, or replaced write-up while the interview stays completed. The two outcomes are visible as two facts.
- **Alternatives considered:** Storing the write-up as another interview status would make a report retry look like the interview had started again. Keeping only the latest text would hide whether an earlier write-up failed.
- **Trade-offs accepted:** HR can see a previous write-up while a new one is still running. The product keeps that history instead of erasing it.
- **Revisit when:** The business wants one interview status to include the write-up, or wants old write-ups deleted.
- **Status:** active

## D2: A failed report leaves the completed interview alone

- **Business driver:** BR-CORE-081 and scope freeze H-10. A problem in the later write-up must not make the candidate repeat the interview.
- **Technical decision:** The interview is marked completed by the interview flow. In that same save, the service also stores the queued report work. If the write-up later fails, only the report records change. The candidate's finished screen does not wait on the write-up (sibling BR-CORE-047).
- **What this means for the product:** The candidate sees the interview outcome even when HR has no report yet, or when the report failed. HR sees the failure on the report, not as a failed interview.
- **Alternatives considered:** Waiting to complete the interview until the report exists would hold the candidate on a loading screen. Marking the interview failed when the report fails would discard a finished conversation.
- **Trade-offs accepted:** A completed interview can exist for a while with no readable account. Operators have to look at the report state, not the interview state, to see that delay.
- **Revisit when:** The business wants the candidate screen to wait for HR's account, or wants report failure to invalidate the interview.
- **Status:** active

## D3: The report states observations, gaps, and uncertainty, and it does not hire, reject, or rank

- **Business driver:** BR-CORE-082 and BR-CORE-084, scope freeze H-11. Hiring stays a human decision.
- **Technical decision:** The stored account has only observations, gaps, and uncertainties. There is no recommendation, rank, or decision field. Wording that is exactly a hire, reject, or rank verdict is refused. The service does not add a score.
- **What this means for the product:** HR sees what was covered, what was missing, and what is unclear. The product does not present a hiring verdict.
- **Alternatives considered:** A recommended outcome would be shorter for HR and would contradict the rule. A free-form essay with no kinds would make gaps optional and easy to omit.
- **Trade-offs accepted:** The wording check catches direct verdicts such as "hire" and will miss softer paraphrases until a later writing rule is approved. The first version prefers a visible gap over a confident summary.
- **Revisit when:** HR asks for a formal recommendation field, or approves a stronger review of paraphrased verdicts.
- **Status:** active

## D4: Statements cite the finalized interview record

- **Business driver:** BR-CORE-083. HR must be able to trace a finding to the interview, and a missing record must not look like an answer.
- **Technical decision:** A statement presented as the candidate's answer stores the finalized evidence entry it relies on. A gap does not count as an answer. The report writer is not allowed to invent a citation. Partial or unfinished speech is not evidence.
- **What this means for the product:** An HR reader can see which interview record supports an observation. A topic with no record appears as a gap or an uncertainty, not as something the candidate demonstrated.
- **Alternatives considered:** Letting the write-up stand without citations would be faster to generate and would allow claims the interview did not record.
- **Trade-offs accepted:** The report cannot be more specific than the finalized record. Until that record exists in the interview flow, a live report cannot be produced. The plan keeps that adapter blocked rather than inventing citations.
- **Revisit when:** The business allows summaries that are not tied to a specific finalized entry.
- **Status:** active

## D5: The candidate does not receive the report

- **Business driver:** BR-CORE-086. The account is HR's internal write-up, including gaps and compensation context.
- **Technical decision:** The report is available only on the HR service routes. The candidate's finished-interview response is unchanged and does not include the report.
- **What this means for the product:** A candidate who finishes, refreshes, or opens the interview again sees the interview outcome only. The report is not a result handed back at the end of the call.
- **Alternatives considered:** Showing the report to the candidate would make the write-up a candidate-facing score, which this phase rejects.
- **Trade-offs accepted:** Candidates are not told what HR's account says, including gaps that concern them. A later candidate-facing summary would be a new decision.
- **Revisit when:** The business approves a separate candidate-facing summary with its own content rules.
- **Status:** active

## D6: This phase serves the report to current HR through the service, with no new screen

- **Business driver:** The capability's product-owner note for this phase. The reader is the HR colleague already allowed to operate interviews. ADMIN account administration does not include report access (BR-CORE-022). An admin or HR screen is a later decision.
- **Technical decision:** Add two service routes for current HR, and reuse the existing HR sign-in check. Do not build a second sign-in component and do not add a user interface.
- **What this means for the product:** An HR user can retrieve and request a report through the service. An administrator who only manages HR accounts cannot. Nobody gets a new report screen in this phase.
- **Alternatives considered:** A screen now would ship a workflow the product owner postponed. Giving administrators automatic report access would expose HR outcomes to the account-management role.
- **Trade-offs accepted:** HR needs an API client, not a page, to read the report. A screen later has to be designed against this same access rule.
- **Revisit when:** The business approves an HR or admin screen, or wants another role to read reports.
- **Status:** active

## D7: Queued background work uses the accepted message broker, and the live call does not

- **Business driver:** The candidate's finished screen must not wait on the write-up (BR-CORE-047, D2). The platform owner accepted RabbitMQ plus a durable outbox for document and report work (technical design TD-09), instead of polling only. The same decision says this queue is not the live voice path.
- **Technical decision:** When the interview is saved as completed, the queued report work is saved with it. A worker publishes that work to RabbitMQ and writes the account afterward. The voice conversation stays on its current path. The exact dead-letter setup is still undecided, so malformed-message handling waits. The durable save does not wait on that decision.
- **What this means for the product:** A broker outage can delay the write-up. It does not undo the interview, and it does not interrupt the live call. A garbled queue message is not given an invented discard bin.
- **Alternatives considered:** Writing the report inside the candidate's final request would make the candidate wait. Keeping the job only in process memory would lose it on restart. Inventing a dead-letter layout would pretend an open platform choice was settled.
- **Trade-offs accepted:** Report text waits on the worker. Operations need the broker for timely reports. Malformed messages stay an open operational question (task CORE-IRP-012).
- **Revisit when:** The platform owner closes the dead-letter layout, or replaces RabbitMQ as the report mechanism.
- **Status:** active

## D8: The salary ceiling stays out of the report

- **Business driver:** Sibling BR-CORE-071 and scope freeze H-12, applied to the report by the capability's compensation edge case. The ceiling is an internal limit, not something HR's account should disclose. The report may still describe the candidate's compensation expectation and its uncertainty.
- **Technical decision:** The report store, the HR response, the logs, and the writing request do not include the ceiling value or its internal reference. The report has no ceiling field to fill in.
- **What this means for the product:** HR does not receive the company's salary ceiling inside the interview report. A compensation gap or an unclear expectation can still appear.
- **Alternatives considered:** Copying the ceiling into the report would make the limit easy to quote and would break the confidentiality rule. Blocking all compensation language would hide a gap HR asked to see.
- **Trade-offs accepted:** The product currently stores a reference to the ceiling, not the amount. This phase can guarantee the report has no ceiling field. It cannot scan prose for an amount the service does not store. That scan waits for a real amount store.
- **Revisit when:** A salary-ceiling amount is stored and the business wants prose scanned for it, or compensation expectations become too sensitive to include.
- **Status:** active
