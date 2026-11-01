package runtime

// CandidateNotice is the notice version and statement a candidate is asked to accept before an interview starts.
type CandidateNotice struct {
	// Version identifies the statement and is stored with each acceptance.
	Version string
	// Statement is the full approved text shown to the candidate.
	Statement string
}

// CurrentCandidateNotice returns the notice catalog value composed into the candidate interview.
func CurrentCandidateNotice() CandidateNotice {
	return CandidateNotice{}
}
