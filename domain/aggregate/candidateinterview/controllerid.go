package candidateinterview

// ControllerID identifies the browser tab allowed to control a live interview.
type ControllerID struct{ value string }

// ParseControllerID validates the identifier character rule without trimming or case folding.
func ParseControllerID(raw string) (ControllerID, error) {
	if !validIdentifier(raw) {
		return ControllerID{}, ErrInvalidControllerID
	}
	return ControllerID{value: raw}, nil
}

// String returns the controller id text.
func (c ControllerID) String() string { return c.value }
