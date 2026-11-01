package candidateinterview

// maxIdentifierLength is the longest accepted notice version or controller id.
const maxIdentifierLength = 64

// validIdentifier applies ^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$ without trimming or case folding.
func validIdentifier(raw string) bool {
	if raw == "" || len(raw) > maxIdentifierLength {
		return false
	}
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		alnum := ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
		if alnum {
			continue
		}
		if i == 0 {
			return false
		}
		switch ch {
		case '.', '_', ':', '-':
		default:
			return false
		}
	}
	return true
}
