package speech

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	assert.Equal(t,
		[]string{"i", "am", "a", "developer"},
		Normalize("I'm a Developer!"))
	assert.Equal(t,
		[]string{"do", "not", "panic"},
		Normalize("  Don't   panic... "))
	assert.Empty(t, Normalize("   "))
}

func TestScoreTranscriptPerfectMatch(t *testing.T) {
	s := ScoreTranscript("I would like a window seat please", "I would like a window seat, please.")
	assert.Equal(t, 1.0, s.Similarity)
	assert.True(t, s.Passed)
	assert.Empty(t, s.MissedWords)
}

func TestScoreTranscriptOmittedWordIsMissed(t *testing.T) {
	// 1 deletion over 7 words: WER = 1/7, similarity ≈ 0.857 — passes,
	// and the omitted word is highlighted (FR-015).
	s := ScoreTranscript("I would like a window seat please", "I would like a seat please")
	assert.InDelta(t, 1.0-1.0/7.0, s.Similarity, 1e-9)
	assert.True(t, s.Passed)
	assert.Equal(t, []string{"window"}, s.MissedWords)
}

func TestScoreTranscriptThreshold(t *testing.T) {
	// 1 error over 5 words → 0.80: passes exactly at threshold (FR-014).
	s := ScoreTranscript("one two three four five", "one two three four six")
	assert.InDelta(t, 0.80, s.Similarity, 1e-9)
	assert.True(t, s.Passed)
	assert.Equal(t, []string{"five"}, s.MissedWords)

	// 2 errors over 5 words → 0.60: fails.
	s = ScoreTranscript("one two three four five", "one two seven four six")
	assert.InDelta(t, 0.60, s.Similarity, 1e-9)
	assert.False(t, s.Passed)
	assert.Equal(t, []string{"three", "five"}, s.MissedWords)
}

func TestScoreTranscriptInsertionsDoNotFlagTargetWords(t *testing.T) {
	s := ScoreTranscript("thank you", "well thank you very much")
	// 3 insertions over 2 ref words → WER 1.5, similarity clamps to 0.
	assert.Equal(t, 0.0, s.Similarity)
	assert.False(t, s.Passed)
	assert.Empty(t, s.MissedWords) // nothing in the target was missed
}

func TestScoreTranscriptContractions(t *testing.T) {
	s := ScoreTranscript("I am staying at the Hilton hotel", "I'm staying at the Hilton hotel")
	assert.Equal(t, 1.0, s.Similarity)
	assert.True(t, s.Passed)
}

func TestScoreTranscriptEmptyTranscriptMissesEverything(t *testing.T) {
	s := ScoreTranscript("hello world", "")
	assert.Equal(t, 0.0, s.Similarity)
	assert.False(t, s.Passed)
	assert.Equal(t, []string{"hello", "world"}, s.MissedWords)
}
