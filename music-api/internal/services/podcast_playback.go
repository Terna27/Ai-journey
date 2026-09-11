package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"music-api/internal/models"
	"music-api/internal/repository"
)

const (
	defaultPodcastHistoryLimit = 20
	maxPodcastHistoryLimit     = 100
)

var (
	ErrPodcastPlaybackUserRequired = errors.New(
		"user is required",
	)

	ErrPodcastEpisodeRequired = errors.New(
		"podcast episode is required",
	)

	ErrInvalidPodcastPlaybackDuration = errors.New(
		"playback duration must be zero or greater",
	)

	ErrInvalidPodcastPlaybackPosition = errors.New(
		"playback position must be zero or greater",
	)

	ErrInvalidPodcastListenedDuration = errors.New(
		"listened duration must be zero or greater",
	)

	ErrPodcastPlaybackPositionExceedsDuration = errors.New(
		"playback position exceeds episode duration",
	)

	ErrPodcastPlaybackSessionRequired = errors.New(
		"playback session id is required",
	)
)

type PodcastPlaybackRepo interface {
	CreateSession(
		ctx context.Context,
		sessionID string,
		userID int,
		episodeID int64,
		durationMS int64,
		positionMS int64,
	) (models.PodcastPlaybackSession, error)

	GetSession(
		ctx context.Context,
		sessionID string,
		userID int,
	) (models.PodcastPlaybackSession, error)

	UpdateProgress(
		ctx context.Context,
		sessionID string,
		userID int,
		positionMS int64,
		durationMS int64,
		listenedMS int64,
	) (models.PodcastPlaybackSession, error)

	CompleteSession(
		ctx context.Context,
		sessionID string,
		userID int,
		positionMS int64,
		durationMS int64,
		listenedMS int64,
	) (models.PodcastPlaybackSession, error)

	GetPodcastListeningHistory(
		ctx context.Context,
		userID int,
		limit int,
		offset int,
	) ([]models.PodcastListeningHistoryItem, error)

	GetContinueListening(
		ctx context.Context,
		userID int,
		limit int,
		offset int,
	) ([]models.PodcastContinueListeningItem, error)
}

type PodcastPlaybackService struct {
	repo PodcastPlaybackRepo
}

func NewPodcastPlaybackService(
	repo PodcastPlaybackRepo,
) *PodcastPlaybackService {
	return &PodcastPlaybackService{
		repo: repo,
	}
}

type CreatePodcastPlaybackSessionInput struct {
	EpisodeID  int64 `json:"episode_id"`
	DurationMS int64 `json:"duration_ms"`
	PositionMS int64 `json:"position_ms"`
}

type UpdatePodcastPlaybackProgressInput struct {
	DurationMS int64 `json:"duration_ms"`
	PositionMS int64 `json:"position_ms"`
	ListenedMS int64 `json:"listened_ms"`
}

type PodcastHistoryQuery struct {
	Limit  int
	Offset int
}

func (s *PodcastPlaybackService) CreateSession(
	ctx context.Context,
	userID int,
	input CreatePodcastPlaybackSessionInput,
) (models.PodcastPlaybackSession, error) {
	if userID <= 0 {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackUserRequired
	}

	if input.EpisodeID <= 0 {
		return models.PodcastPlaybackSession{},
			ErrPodcastEpisodeRequired
	}

	if err := validatePodcastPlaybackPosition(
		input.DurationMS,
		input.PositionMS,
	); err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	sessionID := uuid.NewString()

	session, err := s.repo.CreateSession(
		ctx,
		sessionID,
		userID,
		input.EpisodeID,
		input.DurationMS,
		input.PositionMS,
	)
	// The INSERT ... SELECT produced no row: the episode
	// does not exist or is not publicly playable.
	if errors.Is(err, pgx.ErrNoRows) {
		return models.PodcastPlaybackSession{},
			repository.ErrPodcastPlaybackSessionNotFound
	}

	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	return session, nil
}

func (s *PodcastPlaybackService) GetSession(
	ctx context.Context,
	sessionID string,
	userID int,
) (models.PodcastPlaybackSession, error) {
	if userID <= 0 {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackUserRequired
	}

	if err := validatePodcastSessionID(
		sessionID,
	); err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	return s.repo.GetSession(
		ctx,
		sessionID,
		userID,
	)
}

func (s *PodcastPlaybackService) UpdateProgress(
	ctx context.Context,
	sessionID string,
	userID int,
	input UpdatePodcastPlaybackProgressInput,
) (models.PodcastPlaybackSession, error) {
	if userID <= 0 {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackUserRequired
	}

	if err := validatePodcastSessionID(
		sessionID,
	); err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	if err := validatePodcastProgressInput(
		input,
	); err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	current, err := s.repo.GetSession(
		ctx,
		sessionID,
		userID,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	// A completed session no longer accepts progress; the
	// completion endpoint stays retryable.
	if current.Completed {
		return models.PodcastPlaybackSession{},
			repository.
				ErrPodcastPlaybackSessionCompleted
	}

	effectiveDurationMS := input.DurationMS

	if effectiveDurationMS == 0 {
		effectiveDurationMS =
			current.DurationMS
	}

	if effectiveDurationMS > 0 &&
		input.PositionMS >
			effectiveDurationMS+
				playbackPositionToleranceMS {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackPositionExceedsDuration
	}

	// Qualification and completion are intentionally NOT
	// decided here: the repository performs the
	// authoritative decision inside the row-locked
	// transaction, after clamping client-reported
	// listened_ms against server wall-clock time.
	return s.repo.UpdateProgress(
		ctx,
		sessionID,
		userID,
		input.PositionMS,
		effectiveDurationMS,
		input.ListenedMS,
	)
}

func (s *PodcastPlaybackService) CompleteSession(
	ctx context.Context,
	sessionID string,
	userID int,
	input UpdatePodcastPlaybackProgressInput,
) (models.PodcastPlaybackSession, error) {
	if userID <= 0 {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackUserRequired
	}

	if err := validatePodcastSessionID(
		sessionID,
	); err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	if err := validatePodcastProgressInput(
		input,
	); err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	current, err := s.repo.GetSession(
		ctx,
		sessionID,
		userID,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	// Completion is intentionally safe to retry.
	if current.Completed {
		return current, nil
	}

	effectiveDurationMS := input.DurationMS

	if effectiveDurationMS == 0 {
		effectiveDurationMS =
			current.DurationMS
	}

	if effectiveDurationMS > 0 &&
		input.PositionMS >
			effectiveDurationMS+
				playbackPositionToleranceMS {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackPositionExceedsDuration
	}

	return s.repo.CompleteSession(
		ctx,
		sessionID,
		userID,
		input.PositionMS,
		effectiveDurationMS,
		input.ListenedMS,
	)
}

func (s *PodcastPlaybackService) GetPodcastListeningHistory(
	ctx context.Context,
	userID int,
	query PodcastHistoryQuery,
) ([]models.PodcastListeningHistoryItem, error) {
	if userID <= 0 {
		return nil, ErrPodcastPlaybackUserRequired
	}

	limit, offset := normalizePodcastHistoryQuery(
		query,
	)

	return s.repo.GetPodcastListeningHistory(
		ctx,
		userID,
		limit,
		offset,
	)
}

func (s *PodcastPlaybackService) GetContinueListening(
	ctx context.Context,
	userID int,
	query PodcastHistoryQuery,
) ([]models.PodcastContinueListeningItem, error) {
	if userID <= 0 {
		return nil, ErrPodcastPlaybackUserRequired
	}

	limit, offset := normalizePodcastHistoryQuery(
		query,
	)

	return s.repo.GetContinueListening(
		ctx,
		userID,
		limit,
		offset,
	)
}

func normalizePodcastHistoryQuery(
	query PodcastHistoryQuery,
) (int, int) {
	limit := query.Limit

	if limit <= 0 {
		limit = defaultPodcastHistoryLimit
	}

	if limit > maxPodcastHistoryLimit {
		limit = maxPodcastHistoryLimit
	}

	offset := query.Offset

	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

func validatePodcastSessionID(
	sessionID string,
) error {
	if sessionID == "" {
		return ErrPodcastPlaybackSessionRequired
	}

	if _, err := uuid.Parse(sessionID); err != nil {
		return ErrPodcastPlaybackSessionRequired
	}

	return nil
}

func validatePodcastProgressInput(
	input UpdatePodcastPlaybackProgressInput,
) error {
	if input.DurationMS < 0 {
		return ErrInvalidPodcastPlaybackDuration
	}

	if input.PositionMS < 0 {
		return ErrInvalidPodcastPlaybackPosition
	}

	if input.ListenedMS < 0 {
		return ErrInvalidPodcastListenedDuration
	}

	return nil
}

func validatePodcastPlaybackPosition(
	durationMS int64,
	positionMS int64,
) error {
	if durationMS < 0 {
		return ErrInvalidPodcastPlaybackDuration
	}

	if positionMS < 0 {
		return ErrInvalidPodcastPlaybackPosition
	}

	if durationMS > 0 &&
		positionMS >
			durationMS+
				playbackPositionToleranceMS {
		return ErrPodcastPlaybackPositionExceedsDuration
	}

	return nil
}

func IsPodcastPlaybackValidationError(
	err error,
) bool {
	return errors.Is(
		err,
		ErrPodcastPlaybackUserRequired,
	) ||
		errors.Is(
			err,
			ErrPodcastEpisodeRequired,
		) ||
		errors.Is(
			err,
			ErrInvalidPodcastPlaybackDuration,
		) ||
		errors.Is(
			err,
			ErrInvalidPodcastPlaybackPosition,
		) ||
		errors.Is(
			err,
			ErrInvalidPodcastListenedDuration,
		) ||
		errors.Is(
			err,
			ErrPodcastPlaybackPositionExceedsDuration,
		) ||
		errors.Is(
			err,
			ErrPodcastPlaybackSessionRequired,
		)
}
