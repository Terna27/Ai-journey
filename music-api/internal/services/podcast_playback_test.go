package services

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"music-api/internal/models"
	"music-api/internal/repository"
)

// fakePodcastPlaybackRepo is an in-memory implementation of
// PodcastPlaybackRepo used by the podcast playback service
// tests. It mirrors the repository's server-side rules
// closely enough to test the service layer's validation
// and delegation decisions.
type fakePodcastPlaybackRepo struct {
	createErr error

	createdSession models.PodcastPlaybackSession

	getSession models.PodcastPlaybackSession

	getErr error

	updateCalled bool

	updatedSession models.PodcastPlaybackSession

	updateErr error

	completeCalled bool

	completedSession models.PodcastPlaybackSession

	completeErr error

	historyItems []models.PodcastListeningHistoryItem

	continueItems []models.PodcastContinueListeningItem

	historyLimit  int
	historyOffset int

	continueLimit  int
	continueOffset int
}

func (f *fakePodcastPlaybackRepo) CreateSession(
	ctx context.Context,
	sessionID string,
	userID int,
	episodeID int64,
	durationMS int64,
	positionMS int64,
) (models.PodcastPlaybackSession, error) {
	if f.createErr != nil {
		return models.PodcastPlaybackSession{},
			f.createErr
	}

	f.createdSession = models.PodcastPlaybackSession{
		ID:         sessionID,
		UserID:     userID,
		EpisodeID:  episodeID,
		DurationMS: durationMS,
		PositionMS: positionMS,
	}

	return f.createdSession, nil
}

func (f *fakePodcastPlaybackRepo) GetSession(
	ctx context.Context,
	sessionID string,
	userID int,
) (models.PodcastPlaybackSession, error) {
	if f.getErr != nil {
		return models.PodcastPlaybackSession{},
			f.getErr
	}

	return f.getSession, nil
}

func (f *fakePodcastPlaybackRepo) UpdateProgress(
	ctx context.Context,
	sessionID string,
	userID int,
	positionMS int64,
	durationMS int64,
	listenedMS int64,
) (models.PodcastPlaybackSession, error) {
	if f.updateErr != nil {
		return models.PodcastPlaybackSession{},
			f.updateErr
	}

	f.updateCalled = true

	return f.updatedSession, nil
}

func (f *fakePodcastPlaybackRepo) CompleteSession(
	ctx context.Context,
	sessionID string,
	userID int,
	positionMS int64,
	durationMS int64,
	listenedMS int64,
) (models.PodcastPlaybackSession, error) {
	if f.completeErr != nil {
		return models.PodcastPlaybackSession{},
			f.completeErr
	}

	f.completeCalled = true

	return f.completedSession, nil
}

func (f *fakePodcastPlaybackRepo) GetPodcastListeningHistory(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.PodcastListeningHistoryItem, error) {
	f.historyLimit = limit
	f.historyOffset = offset

	return f.historyItems, nil
}

func (f *fakePodcastPlaybackRepo) GetContinueListening(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.PodcastContinueListeningItem, error) {
	f.continueLimit = limit
	f.continueOffset = offset

	return f.continueItems, nil
}

func TestCreatePodcastSessionRequiresUser(
	t *testing.T,
) {
	service := NewPodcastPlaybackService(
		&fakePodcastPlaybackRepo{},
	)

	_, err := service.CreateSession(
		context.Background(),
		0,
		CreatePodcastPlaybackSessionInput{
			EpisodeID: 5,
		},
	)

	if !errors.Is(
		err,
		ErrPodcastPlaybackUserRequired,
	) {
		t.Fatalf(
			"expected ErrPodcastPlaybackUserRequired, got %v",
			err,
		)
	}
}

func TestCreatePodcastSessionRequiresEpisode(
	t *testing.T,
) {
	service := NewPodcastPlaybackService(
		&fakePodcastPlaybackRepo{},
	)

	_, err := service.CreateSession(
		context.Background(),
		3,
		CreatePodcastPlaybackSessionInput{},
	)

	if !errors.Is(
		err,
		ErrPodcastEpisodeRequired,
	) {
		t.Fatalf(
			"expected ErrPodcastEpisodeRequired, got %v",
			err,
		)
	}
}

func TestCreatePodcastSessionRejectsPositionBeyondDuration(
	t *testing.T,
) {
	service := NewPodcastPlaybackService(
		&fakePodcastPlaybackRepo{},
	)

	_, err := service.CreateSession(
		context.Background(),
		3,
		CreatePodcastPlaybackSessionInput{
			EpisodeID:  5,
			DurationMS: 60_000,
			PositionMS: 120_000,
		},
	)

	if !errors.Is(
		err,
		ErrPodcastPlaybackPositionExceedsDuration,
	) {
		t.Fatalf(
			"expected ErrPodcastPlaybackPositionExceedsDuration, got %v",
			err,
		)
	}
}

func TestCreatePodcastSessionAcceptsResumePosition(
	t *testing.T,
) {
	repo := &fakePodcastPlaybackRepo{}

	service := NewPodcastPlaybackService(
		repo,
	)

	session, err := service.CreateSession(
		context.Background(),
		3,
		CreatePodcastPlaybackSessionInput{
			EpisodeID:  5,
			DurationMS: 600_000,
			PositionMS: 250_000,
		},
	)

	if err != nil {
		t.Fatalf(
			"expected resumed session creation to succeed, got %v",
			err,
		)
	}

	if session.EpisodeID != 5 ||
		session.PositionMS != 250_000 {
		t.Fatalf(
			"expected episode 5 at position 250000, got episode %d at %d",
			session.EpisodeID,
			session.PositionMS,
		)
	}
}

func TestCreatePodcastSessionMapsUnplayableEpisode(
	t *testing.T,
) {
	repo := &fakePodcastPlaybackRepo{
		createErr: pgx.ErrNoRows,
	}

	service := NewPodcastPlaybackService(
		repo,
	)

	_, err := service.CreateSession(
		context.Background(),
		3,
		CreatePodcastPlaybackSessionInput{
			EpisodeID: 5,
		},
	)

	if !errors.Is(
		err,
		repository.ErrPodcastPlaybackSessionNotFound,
	) {
		t.Fatalf(
			"expected ErrPodcastPlaybackSessionNotFound, got %v",
			err,
		)
	}
}

func TestPodcastUpdateProgressRejectsInvalidSessionID(
	t *testing.T,
) {
	service := NewPodcastPlaybackService(
		&fakePodcastPlaybackRepo{},
	)

	_, err := service.UpdateProgress(
		context.Background(),
		"not-a-uuid",
		3,
		UpdatePodcastPlaybackProgressInput{},
	)

	if !errors.Is(
		err,
		ErrPodcastPlaybackSessionRequired,
	) {
		t.Fatalf(
			"expected ErrPodcastPlaybackSessionRequired, got %v",
			err,
		)
	}
}

func TestPodcastUpdateProgressRejectsNegativeListened(
	t *testing.T,
) {
	service := NewPodcastPlaybackService(
		&fakePodcastPlaybackRepo{},
	)

	_, err := service.UpdateProgress(
		context.Background(),
		"11111111-1111-1111-1111-111111111111",
		3,
		UpdatePodcastPlaybackProgressInput{
			ListenedMS: -1,
		},
	)

	if !errors.Is(
		err,
		ErrInvalidPodcastListenedDuration,
	) {
		t.Fatalf(
			"expected ErrInvalidPodcastListenedDuration, got %v",
			err,
		)
	}
}

func TestPodcastUpdateProgressRejectsCompletedSession(
	t *testing.T,
) {
	repo := &fakePodcastPlaybackRepo{
		getSession: models.PodcastPlaybackSession{
			ID:        "11111111-1111-1111-1111-111111111111",
			UserID:    3,
			EpisodeID: 5,
			Completed: true,
		},
	}

	service := NewPodcastPlaybackService(
		repo,
	)

	_, err := service.UpdateProgress(
		context.Background(),
		"11111111-1111-1111-1111-111111111111",
		3,
		UpdatePodcastPlaybackProgressInput{
			DurationMS: 60_000,
			PositionMS: 1_000,
			ListenedMS: 1_000,
		},
	)

	if !errors.Is(
		err,
		repository.ErrPodcastPlaybackSessionCompleted,
	) {
		t.Fatalf(
			"expected ErrPodcastPlaybackSessionCompleted, got %v",
			err,
		)
	}

	if repo.updateCalled {
		t.Fatal(
			"repository UpdateProgress must not run for a completed session",
		)
	}
}

func TestPodcastCompleteSessionIsIdempotent(
	t *testing.T,
) {
	repo := &fakePodcastPlaybackRepo{
		getSession: models.PodcastPlaybackSession{
			ID:        "11111111-1111-1111-1111-111111111111",
			UserID:    3,
			EpisodeID: 5,
			Completed: true,
		},
	}

	service := NewPodcastPlaybackService(
		repo,
	)

	session, err := service.CompleteSession(
		context.Background(),
		"11111111-1111-1111-1111-111111111111",
		3,
		UpdatePodcastPlaybackProgressInput{},
	)

	if err != nil {
		t.Fatalf(
			"expected completing a completed session to succeed, got %v",
			err,
		)
	}

	if !session.Completed {
		t.Fatal(
			"expected the already completed session to be returned",
		)
	}

	if repo.completeCalled {
		t.Fatal(
			"repository CompleteSession must not run twice",
		)
	}
}

func TestPodcastHistoryPaginationDefaultsAndCaps(
	t *testing.T,
) {
	repo := &fakePodcastPlaybackRepo{}

	service := NewPodcastPlaybackService(
		repo,
	)

	_, err := service.GetPodcastListeningHistory(
		context.Background(),
		3,
		PodcastHistoryQuery{},
	)

	if err != nil {
		t.Fatalf(
			"expected history query to succeed, got %v",
			err,
		)
	}

	if repo.historyLimit != 20 ||
		repo.historyOffset != 0 {
		t.Fatalf(
			"expected default limit 20 offset 0, got %d/%d",
			repo.historyLimit,
			repo.historyOffset,
		)
	}

	_, err = service.GetContinueListening(
		context.Background(),
		3,
		PodcastHistoryQuery{
			Limit:  500,
			Offset: -5,
		},
	)

	if err != nil {
		t.Fatalf(
			"expected continue listening query to succeed, got %v",
			err,
		)
	}

	if repo.continueLimit != 100 ||
		repo.continueOffset != 0 {
		t.Fatalf(
			"expected capped limit 100 offset 0, got %d/%d",
			repo.continueLimit,
			repo.continueOffset,
		)
	}
}

func TestPodcastHistoryRequiresUser(
	t *testing.T,
) {
	service := NewPodcastPlaybackService(
		&fakePodcastPlaybackRepo{},
	)

	_, err := service.GetPodcastListeningHistory(
		context.Background(),
		0,
		PodcastHistoryQuery{},
	)

	if !errors.Is(
		err,
		ErrPodcastPlaybackUserRequired,
	) {
		t.Fatalf(
			"expected ErrPodcastPlaybackUserRequired, got %v",
			err,
		)
	}
}
