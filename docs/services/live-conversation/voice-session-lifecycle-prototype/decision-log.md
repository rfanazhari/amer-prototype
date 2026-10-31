# Decision Log: Voice session lifecycle prototype

Technical decisions in this feature that exist because of business decisions or constraints. Each entry records why, what it means for the product, and when it should be revisited.

## D1: The provider account key stays on the server
- **Business driver:** The long-lived provider key must not reach the browser (`BR-LIVE-POC-020`). The candidate is running a local feasibility trial, and a key in the page could spend the provider account.
- **Technical decision:** The Go service sends the key to the provider while creating the session. The browser receives only the WebRTC answer it needs to connect audio.
- **What this means for the product:** Opening the prototype page does not reveal a reusable account credential. A person still needs the local service, with the key in its environment, before a billed session can start.
- **Alternatives considered:** Sending the long-lived key to the browser was rejected because it breaks the trial's closed local boundary. A short-lived browser credential was rejected for this version because the accepted transport decision keeps session setup on the server.
- **Trade-offs accepted:** The Go service must be running for every session, and a setup failure on the server becomes a visible error even when the microphone works.
- **Revisit when:** A later capability lets candidates connect without a developer-operated local service.
- **Status:** active

## D2: The server opens the session and does not conduct each turn
- **Business driver:** This capability proves one continuing spoken conversation (`BR-LIVE-POC-003`, `BR-LIVE-POC-012`). Scoring, a fixed HR question list, and hiring policy are outside it (`BR-LIVE-POC-013`, `BR-LIVE-POC-014`).
- **Technical decision:** Go creates the provider session and sets the interviewer instructions. The browser then keeps one live audio connection. The provider hears the end of a turn and speaks the next reply. Go does not receive each answer and does not choose the next question.
- **What this means for the product:** The candidate can continue the same conversation across several spoken turns. The team does not get a score, a required question list, or a server-side record of each turn.
- **Alternatives considered:** Routing every utterance through Go would let a later interview policy choose questions, but it would turn this trial into the interview product and add an audio relay the trial does not need.
- **Trade-offs accepted:** The team cannot inspect each turn inside Go. Whether a later reply really refers to an earlier answer has to be heard in a live trial, not proven by a database row.
- **Revisit when:** A later capability adds HR questions, scoring, or server-side interview control.
- **Status:** active

## D3: The prototype keeps no transcript and no score
- **Business driver:** A stored grade would make a connectivity trial look like an assessment (`BR-LIVE-POC-013`). A transcript kept after the session would retain speech this prototype has not accepted (`BR-LIVE-POC-022`).
- **Technical decision:** The service does not add a voice database, a score field, or a transcript file. Any on-page diagnostic text lives in memory and is cleared when the session ends.
- **What this means for the product:** After Stop, or after a failure, the prototype has no answer record to review. The evidence of success is the live conversation and the tester's notes.
- **Alternatives considered:** Saving a transcript for debugging was rejected because the capability has not accepted retained speech.
- **Trade-offs accepted:** Failures that were only audible cannot be replayed from stored text. The tester needs notes or a separate recording made outside the product.
- **Revisit when:** A later capability explicitly accepts retained interview evidence.
- **Status:** active

## D4: A failed attempt waits for Start
- **Business driver:** An automatic retry would look like a session the candidate recovered on purpose, and the team would lose track of which attempt failed (`BR-LIVE-POC-004`). A session also starts only when the candidate asks for it (`BR-LIVE-POC-001`).
- **Technical decision:** Denied microphone permission, setup failure, and a dropped connection show an error and leave Start enabled. Neither the page nor the server opens the next provider session by itself. Permission is requested before a billed setup call.
- **What this means for the product:** Every session the team reviews corresponds to a Start the candidate chose. A failure stays on screen until the candidate decides to try again.
- **Alternatives considered:** Reconnecting automatically would hide the failure and could open another billed session during the same attempt.
- **Trade-offs accepted:** The candidate must press Start again after every failure, including a brief network drop.
- **Revisit when:** The product has an explicit recovery rule that distinguishes an automatic reconnect from a new attempt.
- **Status:** active

## D5: The local trial has no sign-in
- **Business driver:** The functional baseline is a localhost feasibility trial for one test candidate, using synthetic personal details. It is not a shared interview service.
- **Technical decision:** The voice page and setup route do not require a user account. The provider key stays in the server environment, and ordinary Task startup does not require that key.
- **What this means for the product:** Anyone who can open the local service can start a billed voice session. The page tells the candidate to use fictional information.
- **Alternatives considered:** Adding accounts now would protect a route that is only meant to run on the developer machine, and it is outside this capability.
- **Trade-offs accepted:** The setup route is safe only while the service stays local. Putting it on a shared network would let others spend the provider account.
- **Revisit when:** The page or setup route becomes reachable beyond the developer machine.
- **Status:** active
