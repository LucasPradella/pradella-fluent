package speech

// PassThreshold is the minimum similarity for a speaking pass (FR-014).
const PassThreshold = 0.80

// SpeechScore is the outcome of scoring a transcript against a target sentence.
type SpeechScore struct {
	Similarity  float64  // 1 − WER, clamped to [0, 1]
	Passed      bool     // Similarity >= PassThreshold
	MissedWords []string // target words deleted or substituted (FR-015)
}

// ScoreTranscript computes similarity = 1 − WER via word-level Levenshtein
// alignment and collects the target words the speaker missed (research R8).
func ScoreTranscript(target, transcript string) SpeechScore {
	ref := Normalize(target)
	hyp := Normalize(transcript)

	if len(ref) == 0 {
		return SpeechScore{Similarity: 1, Passed: true}
	}

	// DP edit-distance matrix over words, then backtrace for the alignment.
	n, m := len(ref), len(hyp)
	d := make([][]int, n+1)
	for i := range d {
		d[i] = make([]int, m+1)
		d[i][0] = i
	}
	for j := 0; j <= m; j++ {
		d[0][j] = j
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 1
			if ref[i-1] == hyp[j-1] {
				cost = 0
			}
			d[i][j] = min3(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
		}
	}

	var missed []string
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && d[i][j] == d[i-1][j-1] && ref[i-1] == hyp[j-1]:
			i, j = i-1, j-1 // match
		case i > 0 && j > 0 && d[i][j] == d[i-1][j-1]+1:
			missed = append(missed, ref[i-1]) // substitution
			i, j = i-1, j-1
		case i > 0 && d[i][j] == d[i-1][j]+1:
			missed = append(missed, ref[i-1]) // deletion (word omitted)
			i--
		default:
			j-- // insertion (extra spoken word) — not a missed target word
		}
	}
	// backtrace collected in reverse order
	for a, b := 0, len(missed)-1; a < b; a, b = a+1, b-1 {
		missed[a], missed[b] = missed[b], missed[a]
	}

	wer := float64(d[n][m]) / float64(n)
	sim := 1 - wer
	if sim < 0 {
		sim = 0
	}
	return SpeechScore{
		Similarity:  sim,
		Passed:      sim >= PassThreshold,
		MissedWords: missed,
	}
}
