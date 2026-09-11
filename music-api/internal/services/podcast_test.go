package services

import (
	"errors"
	"testing"
	"time"

	"music-api/internal/models"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple title",
			in:   "My First Podcast",
			want: "my-first-podcast",
		},
		{
			name: "trims whitespace",
			in:   "   Hello World   ",
			want: "hello-world",
		},
		{
			name: "collapses punctuation",
			in:   "Music!!! Talk --- Daily",
			want: "music-talk-daily",
		},
		{
			name: "keeps unicode letters",
			in:   "Tiv Ñyian Podcast",
			want: "tiv-ñyian-podcast",
		},
		{
			name: "fallback when no letters or numbers",
			in:   "!!!",
			want: "podcast",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				got := slugify(tt.in)

				if got != tt.want {
					t.Fatalf(
						"slugify(%q) = %q; want %q",
						tt.in,
						got,
						tt.want,
					)
				}
			},
		)
	}
}

func TestSlugForAttempt(t *testing.T) {
	tests := []struct {
		attempt int
		want    string
	}{
		{
			attempt: 0,
			want:    "daily-show",
		},
		{
			attempt: 1,
			want:    "daily-show-2",
		},
		{
			attempt: 2,
			want:    "daily-show-3",
		},
	}

	for _, tt := range tests {
		got := slugForAttempt(
			"daily-show",
			tt.attempt,
		)

		if got != tt.want {
			t.Fatalf(
				"slugForAttempt attempt %d = %q; want %q",
				tt.attempt,
				got,
				tt.want,
			)
		}
	}
}

func TestBuildPodcastEpisodeAllowsDraftWithoutAudio(
	t *testing.T,
) {
	episode, err := buildPodcastEpisode(
		1,
		"Draft Episode",
		"",
		1,
		nil,
		models.PodcastEpisodeTypeFull,
		nil,
		nil,
		0,
		models.PodcastEpisodeStatusDraft,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"expected draft without audio to succeed: %v",
			err,
		)
	}

	if episode.Status !=
		models.PodcastEpisodeStatusDraft {
		t.Fatalf(
			"unexpected status: %s",
			episode.Status,
		)
	}

	if episode.AudioURL != "" {
		t.Fatalf(
			"expected draft audio_url to be empty",
		)
	}

	if episode.PublishedAt != nil {
		t.Fatalf(
			"draft must not have published_at",
		)
	}
}

func TestBuildPodcastEpisodeRejectsPublishedWithoutAudio(
	t *testing.T,
) {
	_, err := buildPodcastEpisode(
		1,
		"Published Episode",
		"",
		1,
		nil,
		models.PodcastEpisodeTypeFull,
		nil,
		nil,
		0,
		models.PodcastEpisodeStatusPublished,
		false,
		nil,
	)

	if !errors.Is(
		err,
		ErrPodcastEpisodeAudioRequired,
	) {
		t.Fatalf(
			"expected ErrPodcastEpisodeAudioRequired; got %v",
			err,
		)
	}
}

func TestBuildPodcastEpisodePublishesWithAudio(
	t *testing.T,
) {
	episode, err := buildPodcastEpisode(
		1,
		"Published Episode",
		"",
		1,
		nil,
		models.PodcastEpisodeTypeFull,
		&PodcastMediaAsset{
			URL:      "https://example.com/audio.mp3",
			PublicID: "podcasts/audio/example",
		},
		nil,
		120_000,
		models.PodcastEpisodeStatusPublished,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"expected published episode with audio to succeed: %v",
			err,
		)
	}

	if episode.PublishedAt == nil {
		t.Fatalf(
			"published episode must have published_at",
		)
	}

	if episode.ScheduledAt != nil {
		t.Fatalf(
			"published episode must not have scheduled_at",
		)
	}
}

func TestBuildPodcastEpisodeRejectsScheduledWithoutAudio(
	t *testing.T,
) {
	future := time.Now().
		UTC().
		Add(2 * time.Hour)

	_, err := buildPodcastEpisode(
		1,
		"Scheduled Episode",
		"",
		1,
		nil,
		models.PodcastEpisodeTypeFull,
		nil,
		nil,
		0,
		models.PodcastEpisodeStatusScheduled,
		false,
		&future,
	)

	if !errors.Is(
		err,
		ErrPodcastEpisodeAudioRequired,
	) {
		t.Fatalf(
			"expected ErrPodcastEpisodeAudioRequired; got %v",
			err,
		)
	}
}

func TestBuildPodcastEpisodeRejectsPastSchedule(
	t *testing.T,
) {
	past := time.Now().
		UTC().
		Add(-1 * time.Hour)

	_, err := buildPodcastEpisode(
		1,
		"Scheduled Episode",
		"",
		1,
		nil,
		models.PodcastEpisodeTypeFull,
		&PodcastMediaAsset{
			URL:      "https://example.com/audio.mp3",
			PublicID: "podcasts/audio/example",
		},
		nil,
		120_000,
		models.PodcastEpisodeStatusScheduled,
		false,
		&past,
	)

	if !errors.Is(
		err,
		ErrPodcastScheduleMustBeFuture,
	) {
		t.Fatalf(
			"expected ErrPodcastScheduleMustBeFuture; got %v",
			err,
		)
	}
}

func TestBuildPodcastEpisodeAcceptsFutureSchedule(
	t *testing.T,
) {
	future := time.Now().
		UTC().
		Add(2 * time.Hour)

	episode, err := buildPodcastEpisode(
		1,
		"Scheduled Episode",
		"",
		1,
		nil,
		models.PodcastEpisodeTypeFull,
		&PodcastMediaAsset{
			URL:      "https://example.com/audio.mp3",
			PublicID: "podcasts/audio/example",
		},
		nil,
		120_000,
		models.PodcastEpisodeStatusScheduled,
		false,
		&future,
	)
	if err != nil {
		t.Fatalf(
			"expected valid schedule: %v",
			err,
		)
	}

	if episode.ScheduledAt == nil {
		t.Fatalf(
			"scheduled episode must have scheduled_at",
		)
	}

	if episode.PublishedAt != nil {
		t.Fatalf(
			"scheduled episode must not have published_at",
		)
	}
}

func TestBuildPodcastEpisodeRejectsInvalidSeason(
	t *testing.T,
) {
	_, err := buildPodcastEpisode(
		1,
		"Episode",
		"",
		0,
		nil,
		models.PodcastEpisodeTypeFull,
		nil,
		nil,
		0,
		models.PodcastEpisodeStatusDraft,
		false,
		nil,
	)

	if !errors.Is(
		err,
		ErrInvalidPodcastSeasonNumber,
	) {
		t.Fatalf(
			"expected ErrInvalidPodcastSeasonNumber; got %v",
			err,
		)
	}
}

func TestBuildPodcastEpisodeRejectsInvalidEpisodeNumber(
	t *testing.T,
) {
	number := 0

	_, err := buildPodcastEpisode(
		1,
		"Episode",
		"",
		1,
		&number,
		models.PodcastEpisodeTypeFull,
		nil,
		nil,
		0,
		models.PodcastEpisodeStatusDraft,
		false,
		nil,
	)

	if !errors.Is(
		err,
		ErrInvalidPodcastEpisodeNumber,
	) {
		t.Fatalf(
			"expected ErrInvalidPodcastEpisodeNumber; got %v",
			err,
		)
	}
}

func TestApplyEpisodeLifecyclePreservesPublishedAt(
	t *testing.T,
) {
	publishedAt := time.Now().
		UTC().
		Add(-24 * time.Hour)

	existing := models.PodcastEpisode{
		PublishedAt: &publishedAt,
	}

	episode := models.PodcastEpisode{
		AudioURL:      "https://example.com/audio.mp3",
		AudioPublicID: "podcasts/audio/example",
	}

	err := applyEpisodeLifecycle(
		&episode,
		existing,
		models.PodcastEpisodeStatusPublished,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"unexpected lifecycle error: %v",
			err,
		)
	}

	if episode.PublishedAt == nil {
		t.Fatalf(
			"published_at should be preserved",
		)
	}

	if !episode.PublishedAt.Equal(
		publishedAt,
	) {
		t.Fatalf(
			"published_at changed: got %v; want %v",
			episode.PublishedAt,
			publishedAt,
		)
	}
}

func TestValidatePodcastText(t *testing.T) {
	title, description, category, err :=
		validatePodcastText(
			"  My Podcast  ",
			"  A description  ",
			"  Technology  ",
		)
	if err != nil {
		t.Fatalf(
			"unexpected validation error: %v",
			err,
		)
	}

	if title != "My Podcast" {
		t.Fatalf(
			"unexpected title %q",
			title,
		)
	}

	if description != "A description" {
		t.Fatalf(
			"unexpected description %q",
			description,
		)
	}

	if category != "Technology" {
		t.Fatalf(
			"unexpected category %q",
			category,
		)
	}
}

func TestValidatePodcastTextRequiresTitle(
	t *testing.T,
) {
	_, _, _, err :=
		validatePodcastText(
			"   ",
			"",
			"",
		)

	if !errors.Is(
		err,
		ErrPodcastTitleRequired,
	) {
		t.Fatalf(
			"expected ErrPodcastTitleRequired; got %v",
			err,
		)
	}
}

func TestNormalizePodcastPagination(t *testing.T) {
	limit, offset :=
		normalizePodcastPagination(
			0,
			-10,
		)

	if limit != defaultPodcastPageLimit {
		t.Fatalf(
			"expected default limit %d; got %d",
			defaultPodcastPageLimit,
			limit,
		)
	}

	if offset != 0 {
		t.Fatalf(
			"expected offset 0; got %d",
			offset,
		)
	}

	limit, offset =
		normalizePodcastPagination(
			1000,
			5,
		)

	if limit != maxPodcastPageLimit {
		t.Fatalf(
			"expected max limit %d; got %d",
			maxPodcastPageLimit,
			limit,
		)
	}

	if offset != 5 {
		t.Fatalf(
			"expected offset 5; got %d",
			offset,
		)
	}
}
