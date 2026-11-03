package interviewreport

import (
	"strings"
	"unicode"
)

// verdictWords are the hire, reject, and rank words refused in statement text.
var verdictWords = []string{"hire", "hiring", "reject", "rejection", "rank", "ranking"}

// hasVerdictWord reports whether body contains a verdict word as a whole word, ignoring case.
// It is an English guard for the words in the business rule and does not detect paraphrase.
func hasVerdictWord(body string) bool {
	words := strings.FieldsFunc(body, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	for _, w := range words {
		for _, v := range verdictWords {
			if strings.EqualFold(w, v) {
				return true
			}
		}
	}
	return false
}
