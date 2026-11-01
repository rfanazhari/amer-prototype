package candidateinterview

// NoticeVersion identifies one version of the candidate notice.
type NoticeVersion struct{ value string }

// ParseNoticeVersion validates the identifier character rule without trimming or case folding.
func ParseNoticeVersion(raw string) (NoticeVersion, error) {
	return NoticeVersion{}, ErrInvalidNoticeVersion
}

// String returns the version text.
func (v NoticeVersion) String() string { return v.value }
