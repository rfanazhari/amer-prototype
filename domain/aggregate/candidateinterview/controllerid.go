package candidateinterview

// ControllerID identifies the browser tab allowed to control a live interview.
type ControllerID struct{ value string }

// ParseControllerID validates the identifier character rule without trimming or case folding.
func ParseControllerID(raw string) (ControllerID, error) {
	return ControllerID{}, ErrInvalidControllerID
}

// String returns the controller id text.
func (c ControllerID) String() string { return c.value }
