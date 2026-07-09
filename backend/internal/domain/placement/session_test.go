package placement_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/placement"
)

func TestNextBand(t *testing.T) {
	tests := []struct {
		name    string
		current placement.Band
		correct int
		want    placement.Band
	}{
		{"3/3 (>70%) moves up", placement.BandB1, 3, placement.BandB2},
		{"2/3 (66%) stays", placement.BandB1, 2, placement.BandB1},
		{"1/3 (33%, <40%) moves down", placement.BandB1, 1, placement.BandA2},
		{"0/3 moves down", placement.BandB1, 0, placement.BandA2},
		{"cannot move above C1", placement.BandC1, 3, placement.BandC1},
		{"cannot move below A1", placement.BandA1, 0, placement.BandA1},
		{"A2 up to B1", placement.BandA2, 3, placement.BandB1},
		{"B2 down to B1", placement.BandB2, 1, placement.BandB1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, placement.NextBand(tt.current, tt.correct))
		})
	}
}

func TestStartBandIsB1(t *testing.T) {
	assert.Equal(t, placement.BandB1, placement.StartBand)
}

func TestHardStopAt12Questions(t *testing.T) {
	assert.False(t, placement.Done(11))
	assert.True(t, placement.Done(12))
	assert.Equal(t, 12, placement.MaxQuestions)
	assert.Equal(t, 4, placement.MaxTestlets)
}

func TestTestletBoundaries(t *testing.T) {
	assert.Equal(t, 0, placement.TestletIndex(0))
	assert.Equal(t, 0, placement.TestletIndex(2))
	assert.Equal(t, 1, placement.TestletIndex(3))
	assert.Equal(t, 3, placement.TestletIndex(11))

	assert.False(t, placement.TestletComplete(0))
	assert.False(t, placement.TestletComplete(2))
	assert.True(t, placement.TestletComplete(3))
	assert.True(t, placement.TestletComplete(6))
	assert.True(t, placement.TestletComplete(12))
}

func TestLevelForBand(t *testing.T) {
	assert.Equal(t, domain.LevelBasic, placement.LevelForBand(placement.BandA1))
	assert.Equal(t, domain.LevelBasic, placement.LevelForBand(placement.BandA2))
	assert.Equal(t, domain.LevelIntermediate, placement.LevelForBand(placement.BandB1))
	assert.Equal(t, domain.LevelIntermediate, placement.LevelForBand(placement.BandB2))
	assert.Equal(t, domain.LevelAdvanced, placement.LevelForBand(placement.BandC1))
}

func TestFinalLevelFromLastTwoTestlets(t *testing.T) {
	// Average band index rounded down (research R10).
	assert.Equal(t, domain.LevelAdvanced, placement.FinalLevel(placement.BandC1, placement.BandC1))
	assert.Equal(t, domain.LevelIntermediate, placement.FinalLevel(placement.BandB2, placement.BandC1))
	assert.Equal(t, domain.LevelIntermediate, placement.FinalLevel(placement.BandB1, placement.BandB2))
	assert.Equal(t, domain.LevelBasic, placement.FinalLevel(placement.BandA1, placement.BandA2))
	assert.Equal(t, domain.LevelBasic, placement.FinalLevel(placement.BandA2, placement.BandB1))
	assert.Equal(t, domain.LevelIntermediate, placement.FinalLevel(placement.BandB1, placement.BandB1))
}
