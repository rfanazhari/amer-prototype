package agentconfig

// DraftVersion is a positive replacement sequence; the zero value is invalid.
type DraftVersion struct{ number uint64 }

// NewDraftVersion validates a sequence supplied by creation or persistence policy.
func NewDraftVersion(number uint64) (DraftVersion, error) {
	if number == 0 {
		return DraftVersion{}, ErrInvalid
	}
	return DraftVersion{number: number}, nil
}

// Number returns the exact sequence without normalization.
func (v DraftVersion) Number() uint64 { return v.number }
