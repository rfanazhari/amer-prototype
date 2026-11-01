# Decision Log: Candidate Interview

Technical decisions in this feature that exist because of business decisions or constraints. Each entry records why, what it means for the product, and when it should be revisited. None of these entries means the behavior is already built or enforced. BR-CORE-040..047 remain planned.

## D1: This phase has a candidate screen and no HR or admin screen

- **Business driver:** The Candidate Interview capability says the screen is the candidate's and that this phase does not add an HR or admin screen. The same boundary is in the request for this plan.
- **Technical decision:** The interview page is part of this work. It is a single candidate page served by this service. HR and admin do not get a new page for notices, live monitoring, or results in this phase.
- **What this means for the product:** A candidate who already has access can read the notice, accept it, follow the interview, switch tabs, leave early, and see the outcome. HR continues to use the access and configuration tools that already exist. There is no new HR screen to watch the interview or edit the notice.
- **Alternatives considered:** An API without a page would leave the candidate outside the product experience this phase describes. Building an HR console in the same phase would serve a second actor and delay the candidate screen.
- **Trade-offs accepted:** The notice text cannot be edited by HR in a screen. Someone must supply it through the approved catalog (see the open notice question in the technical design). Operators cannot supervise the live interview from a new HR page.
- **Revisit when:** The product asks for an HR monitoring screen or for HR to change the notice without a software change.
- **Status:** active

## D2: The candidate accepts the notice before the interview can start

- **Business driver:** BR-CORE-040. An invitation shows that the candidate may open this assignment. It does not show that they agreed to the notice the company must be able to point back to. Platform section 10 requires the notice version and the time of acceptance to be recorded before the interview starts.
- **Technical decision:** The candidate stays on the notice until they explicitly accept the current version. That acceptance is stored. Starting, or starting twice after a lost response, uses the stored acceptance. Opening the link, refreshing, or a failed start does not count as acceptance.
- **What this means for the product:** The candidate sees the notice first and takes an explicit accept action. Only then can they start. If the accept response is lost, repeating accept returns the same acceptance. If the company changes the notice, the candidate must accept the new version before starting.
- **Alternatives considered:** Treating the invitation as consent would leave no acceptance record. Combining accept and start into one click would skip the notice the rule says the candidate remains on. Letting each browser tab store its own acceptance would make the record depend on the device rather than the assignment.
- **Trade-offs accepted:** The candidate has an extra step. The company keeps one acceptance per notice version for that assignment, including versions that are no longer current. What should happen if the candidate withdraws consent after the interview has started is still unresolved, and this decision does not invent that behavior.
- **Revisit when:** The company changes what a candidate must agree to, or decides how a withdrawal after start should affect the interview record.
- **Status:** active

## D3: The candidate's final screen does not wait for or show the HR report

- **Business driver:** BR-CORE-047. The candidate's part ends with the interview. Interview Reporting owns the later HR write-up, and that write-up can fail or still be in progress without changing the interview. Journey A keeps the candidate's closing moment separate from the HR report. The request for this plan states the same rule.
- **Technical decision:** When the interview is completed, failed, or cancelled, the candidate screen shows that outcome immediately. It does not ask whether an HR report exists, and it does not display the report.
- **What this means for the product:** The candidate is not left on a waiting screen while HR's write-up is prepared. A delay or failure in the write-up does not change what the candidate sees. HR still receives the report through the reporting capability, outside this screen.
- **Alternatives considered:** A "preparing your results" screen would hold the candidate for work they are not meant to see. Showing a shortened report would turn the candidate screen into an HR document.
- **Trade-offs accepted:** The candidate does not receive a written summary of the interview. Cancelled, failed, and completed must stay visibly different so an early exit is not presented as a finished interview.
- **Revisit when:** The product explicitly wants the candidate to see a closing summary. That summary still has to stay separate from the HR report.
- **Status:** active
