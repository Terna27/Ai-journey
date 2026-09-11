package models

import "testing"

// TestQualifiedPlayThresholdMS verifies the qualified-play
// rule: 30 seconds OR 50% of the track duration,
// whichever comes first. Unknown duration requires the
// full 30 seconds.
func TestQualifiedPlayThresholdMS(
	t *testing.T,
) {
	cases := []struct {
		name       string
		durationMS int64
		wantMS     int64
	}{
		{
			name:       "unknown duration requires 30 seconds",
			durationMS: 0,
			wantMS:     30_000,
		},
		{
			name:       "20 second track qualifies after 10 seconds",
			durationMS: 20_000,
			wantMS:     10_000,
		},
		{
			name:       "40 second track qualifies after 20 seconds",
			durationMS: 40_000,
			wantMS:     20_000,
		},
		{
			name:       "60 second track qualifies after 30 seconds",
			durationMS: 60_000,
			wantMS:     30_000,
		},
		{
			name:       "3 minute track qualifies after 30 seconds",
			durationMS: 180_000,
			wantMS:     30_000,
		},
		{
			name:       "5 minute track qualifies after 30 seconds",
			durationMS: 300_000,
			wantMS:     30_000,
		},
		{
			name:       "odd duration rounds half up",
			durationMS: 21_001,
			wantMS:     10_501,
		},
	}

	for _, tc := range cases {
		t.Run(
			tc.name,
			func(t *testing.T) {
				got := QualifiedPlayThresholdMS(
					tc.durationMS,
				)

				if got != tc.wantMS {
					t.Fatalf(
						"QualifiedPlayThresholdMS(%d) = %d, want %d",
						tc.durationMS,
						got,
						tc.wantMS,
					)
				}
			},
		)
	}
}

// TestIsQualifiedPlay verifies the qualification verdict
// around the threshold, including that a seek-only
// position never matters — only listened time does.
func TestIsQualifiedPlay(
	t *testing.T,
) {
	cases := []struct {
		name       string
		listenedMS int64
		durationMS int64
		want       bool
	}{
		{
			name:       "just below threshold",
			listenedMS: 29_999,
			durationMS: 300_000,
			want:       false,
		},
		{
			name:       "exactly at threshold",
			listenedMS: 30_000,
			durationMS: 300_000,
			want:       true,
		},
		{
			name:       "short track half duration",
			listenedMS: 10_000,
			durationMS: 20_000,
			want:       true,
		},
		{
			name:       "short track below half",
			listenedMS: 9_999,
			durationMS: 20_000,
			want:       false,
		},
		{
			name:       "unknown duration needs full 30 seconds",
			listenedMS: 29_999,
			durationMS: 0,
			want:       false,
		},
		{
			name:       "negative listened never qualifies",
			listenedMS: -1,
			durationMS: 20_000,
			want:       false,
		},
	}

	for _, tc := range cases {
		t.Run(
			tc.name,
			func(t *testing.T) {
				got := IsQualifiedPlay(
					tc.listenedMS,
					tc.durationMS,
				)

				if got != tc.want {
					t.Fatalf(
						"IsQualifiedPlay(%d, %d) = %v, want %v",
						tc.listenedMS,
						tc.durationMS,
						got,
						tc.want,
					)
				}
			},
		)
	}
}
