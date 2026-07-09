// Package placement implements the adaptive placement-test rules
// (CAMST band walk — research R10, FR-003..FR-005). Pure functions only.
package placement

import "github.com/pradella/fluentdev/backend/internal/domain"

// Band is a CEFR difficulty band used by the question bank.
type Band string

const (
	BandA1 Band = "A1"
	BandA2 Band = "A2"
	BandB1 Band = "B1"
	BandB2 Band = "B2"
	BandC1 Band = "C1"
)

const (
	// StartBand is where every session begins (research R10).
	StartBand = BandB1
	// TestletSize is the number of questions per testlet.
	TestletSize = 3
	// MaxQuestions is the hard stop (FR-005).
	MaxQuestions = 12
	// MaxTestlets derives from the two constants above.
	MaxTestlets = MaxQuestions / TestletSize
)

var bandOrder = []Band{BandA1, BandA2, BandB1, BandB2, BandC1}

func bandIndex(b Band) int {
	for i, x := range bandOrder {
		if x == b {
			return i
		}
	}
	return -1
}

// NextBand applies the band walk after a completed testlet:
// score > 70% moves up one band, < 40% moves down one band, otherwise stay.
func NextBand(current Band, correct int) Band {
	i := bandIndex(current)
	if i < 0 {
		return current
	}
	ratio := float64(correct) / float64(TestletSize)
	switch {
	case ratio > 0.7 && i < len(bandOrder)-1:
		return bandOrder[i+1]
	case ratio < 0.4 && i > 0:
		return bandOrder[i-1]
	default:
		return current
	}
}

// TestletIndex returns the 0-based testlet an answer belongs to given how
// many questions were already served (0..MaxQuestions-1).
func TestletIndex(questionsServed int) int {
	return questionsServed / TestletSize
}

// TestletComplete reports whether serving one more answer finishes a testlet.
func TestletComplete(questionsServed int) bool {
	return questionsServed%TestletSize == 0 && questionsServed > 0
}

// Done reports whether the session reached the hard stop.
func Done(questionsServed int) bool {
	return questionsServed >= MaxQuestions
}

// LevelForBand maps a CEFR band to a proficiency level
// (Basic = A1/A2, Intermediate = B1/B2, Advanced = C1).
func LevelForBand(b Band) domain.Level {
	switch b {
	case BandA1, BandA2:
		return domain.LevelBasic
	case BandB1, BandB2:
		return domain.LevelIntermediate
	case BandC1:
		return domain.LevelAdvanced
	default:
		return domain.LevelBasic
	}
}

// FinalLevel derives the assigned level from the bands at which the last two
// testlets were taken (research R10): the average band index, rounded down.
func FinalLevel(thirdTestletBand, fourthTestletBand Band) domain.Level {
	i := (bandIndex(thirdTestletBand) + bandIndex(fourthTestletBand)) / 2
	if i < 0 {
		i = 0
	}
	return LevelForBand(bandOrder[i])
}
