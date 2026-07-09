package speech

// WritingResult is the outcome of validating a written answer (FR-009).
type WritingResult struct {
	Correct        bool
	ToleratedTypos []string // words accepted despite a small misspelling
}

// typoTolerance returns the allowed character edit distance for a target
// word: ≤1 for words of ≤5 characters, ≤2 for longer words (research R8).
func typoTolerance(word string) int {
	if len([]rune(word)) <= 5 {
		return 1
	}
	return 2
}

// ValidateWriting compares a written answer with the target sentence.
// Words must align one-to-one after normalization; a substituted word within
// typo tolerance is accepted (and reported), any other difference — including
// missing or extra words — fails the answer (semantic misses are rejected).
func ValidateWriting(target, answer string) WritingResult {
	want := Normalize(target)
	got := Normalize(answer)

	if len(want) != len(got) {
		return WritingResult{Correct: false}
	}

	var typos []string
	for i := range want {
		if want[i] == got[i] {
			continue
		}
		if levenshtein(want[i], got[i]) <= typoTolerance(want[i]) {
			typos = append(typos, got[i])
			continue
		}
		return WritingResult{Correct: false}
	}
	return WritingResult{Correct: true, ToleratedTypos: typos}
}
