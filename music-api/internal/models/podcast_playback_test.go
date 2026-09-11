package models

import "testing"

// TestIsPodcastPlaybackNearlyComplete verifies the
// server-consistent podcast completion rule: remaining
// time <= 15 seconds OR position >= 95% of a known
// duration. Unknown duration is never nearly complete.
func TestIsPodcastPlaybackNearlyComplete(
	t *testing.T,
) {
	cases := []struct {
		name       string
		positionMS int64
		durationMS int64
		want       bool
	}{
		{
			name:       "unknown duration is never nearly complete",
			positionMS: 500_000,
			durationMS: 0,
			want:       false,
		},
		{
			name:       "zero position is never nearly complete",
			positionMS: 0,
			durationMS: 600_000,
			want:       false,
		},
		{
			name:       "early position is not nearly complete",
			positionMS: 100_000,
			durationMS: 600_000,
			want:       false,
		},
		{
			name:       "halfway position is not nearly complete",
			positionMS: 300_000,
			durationMS: 600_000,
			want:       false,
		},
		{
			name:       "within final 15 seconds completes",
			positionMS: 590_000,
			durationMS: 600_000,
			want:       true,
		},
		{
			name:       "exactly 15 seconds remaining completes",
			positionMS: 585_000,
			durationMS: 600_000,
			want:       true,
		},
		{
			name:       "16 seconds remaining completes by percentage",
			positionMS: 584_000,
			durationMS: 600_000,
			want:       true,
		},
		{
			name:       "95 percent of a long episode completes",
			positionMS: 570_000,
			durationMS: 600_000,
			want:       true,
		},
		{
			name:       "position beyond duration completes",
			positionMS: 700_000,
			durationMS: 600_000,
			want:       true,
		},
		{
			name:       "short episode near end completes",
			positionMS: 20_000,
			durationMS: 21_000,
			want:       true,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got := IsPodcastPlaybackNearlyComplete(
					testCase.positionMS,
					testCase.durationMS,
				)

				if got != testCase.want {
					t.Fatalf(
						"IsPodcastPlaybackNearlyComplete(%d, %d) = %v, want %v",
						testCase.positionMS,
						testCase.durationMS,
						got,
						testCase.want,
					)
				}
			},
		)
	}
}
