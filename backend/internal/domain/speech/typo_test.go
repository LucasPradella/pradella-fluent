package speech

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateWritingExactMatch(t *testing.T) {
	r := ValidateWriting("I have one bag to check", "I have one bag to check")
	assert.True(t, r.Correct)
	assert.Empty(t, r.ToleratedTypos)
}

func TestValidateWritingCaseAndPunctuationInsensitive(t *testing.T) {
	r := ValidateWriting("I have one bag to check", "i have one bag to check!")
	assert.True(t, r.Correct)
	assert.Empty(t, r.ToleratedTypos)
}

func TestValidateWritingToleratesSmallTypoShortWord(t *testing.T) {
	// "bag" -> "bga": distance 2 on a <=5-char word — NOT tolerated.
	r := ValidateWriting("I have one bag to check", "I have one bga to check")
	assert.False(t, r.Correct)

	// "check" -> "chek": distance 1 on a 5-char word — tolerated.
	r = ValidateWriting("I have one bag to check", "I have one bag to chek")
	assert.True(t, r.Correct)
	assert.Equal(t, []string{"chek"}, r.ToleratedTypos)
}

func TestValidateWritingToleranceScalesWithLength(t *testing.T) {
	// "reservation" (11 chars) allows distance <=2.
	r := ValidateWriting("I have a reservation", "I have a reservatoin")
	assert.True(t, r.Correct)
	assert.Equal(t, []string{"reservatoin"}, r.ToleratedTypos)

	// Distance 3 fails.
	r = ValidateWriting("I have a reservation", "I have a reservtaoin")
	assert.False(t, r.Correct)
}

func TestValidateWritingRejectsSemanticMiss(t *testing.T) {
	// "dog" vs "bag" — distance 2 on a 3-char word: rejected.
	r := ValidateWriting("I have one bag to check", "I have one dog to check")
	assert.False(t, r.Correct)
}

func TestValidateWritingRejectsMissingOrExtraWords(t *testing.T) {
	assert.False(t, ValidateWriting("I have one bag to check", "I have one bag").Correct)
	assert.False(t, ValidateWriting("I have one bag", "I have one big red bag").Correct)
}

func TestValidateWritingContractionEquivalence(t *testing.T) {
	r := ValidateWriting("I am a junior developer", "I'm a junior developer")
	assert.True(t, r.Correct)
	assert.Empty(t, r.ToleratedTypos)
}

func TestTypoToleranceThresholds(t *testing.T) {
	assert.Equal(t, 1, typoTolerance("bag"))   // <=5 chars
	assert.Equal(t, 1, typoTolerance("check")) // exactly 5
	assert.Equal(t, 2, typoTolerance("boarding"))
}
