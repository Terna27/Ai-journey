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
	defaultHistoryLimit = 20
	maxHistoryLimit     = 100

	// A client may report playback slightly ahead because
	// browser media timing is not exact. We allow a small
	// tolerance but reject obviously impossible values.
	playbackPositionToleranceMS int64 = 2_000
)

var (
	ErrPlaybackUserRequired = errors.New(
		"user is required",
	)

	ErrPlaybackMusicRequired = errors.New(
		"music is required",
	)

	ErrInvalidPlaybackDuration = errors.New(
		"playback duration must be zero or greater",
	)

	ErrInvalidPlaybackPosition = errors.New(
		"playback position must be zero or greater",
	)

	ErrInvalidListenedDuration = errors.New(
		"listened duration must be zero or greater",
	)

	ErrPlaybackPositionExceedsDuration = errors.New(
		"playback position exceeds track duration",
	)

	ErrListenedDurationExceedsSession = errors.New(
		"listened duration exceeds possible session duration",
	)

	ErrPlaybackSessionRequired = errors.New(
		"playback session id is required",
	)
)

type PlaybackRepo interface {
	CreateSession(
		ctx context.Context,
		sessionID string,
		userID int,
		musicID int,
		durationMS int64,
		positionMS int64,
	) (models.PlaybackSession, error)

	GetSession(
		ctx context.Context,
		sessionID string,
		userID int,
	) (models.PlaybackSession, error)

	UpdateProgress(
		ctx context.Context,
		sessionID string,
		userID int,
		positionMS int64,
		durationMS int64,
		listenedMS int64,
	) (models.PlaybackSession, error)

	CompleteSession(
		ctx context.Context,
		sessionID string,
		userID int,
		positionMS int64,
		durationMS int64,
		listenedMS int64,
	) (models.PlaybackSession, error)

	GetListeningHistory(
		ctx context.Context,
		userID int,
		limit int,
		offset int,
	) ([]models.ListeningHistoryItem, error)
}

type PlaybackService struct {
	repo PlaybackRepo
}

func NewPlaybackService(
	repo PlaybackRepo,
) *PlaybackService {
	return &PlaybackService{
		repo: repo,
	}
}

type CreatePlaybackSessionInput struct {
	MusicID    int   `json:"music_id"`
	DurationMS int64 `json:"duration_ms"`
	PositionMS int64 `json:"position_ms"`
}

type UpdatePlaybackProgressInput struct {
	DurationMS int64 `json:"duration_ms"`
	PositionMS int64 `json:"position_ms"`
	ListenedMS int64 `json:"listened_ms"`
}

type PlaybackHistoryQuery struct {
	Limit  int
	Offset int
}

func (s *PlaybackService) CreateSession(
	ctx context.Context,
	userID int,
	input CreatePlaybackSessionInput,
) (models.PlaybackSession, error) {
	if userID <= 0 {
		return models.PlaybackSession{},
			ErrPlaybackUserRequired
	}

	if input.MusicID <= 0 {
		return models.PlaybackSession{},
			ErrPlaybackMusicRequired
	}

	if err := validatePlaybackPosition(
		input.DurationMS,
		input.PositionMS,
	); err != nil {
		return models.PlaybackSession{}, err
	}

	sessionID := uuid.NewString()

	session, err := s.repo.CreateSession(
		ctx,
		sessionID,
		userID,
		input.MusicID,
		input.DurationMS,
		input.PositionMS,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.PlaybackSession{},
			repository.ErrPlaybackSessionNotFound
	}

	if err != nil {
		return models.PlaybackSession{}, err
	}

	return session, nil
}

func (s *PlaybackService) GetSession(
	ctx context.Context,
	sessionID string,
	userID int,
) (models.PlaybackSession, error) {
	if userID <= 0 {
		return models.PlaybackSession{},
			ErrPlaybackUserRequired
	}

	if sessionID == "" {
		return models.PlaybackSession{},
			ErrPlaybackSessionRequired
	}

	if _, err := uuid.Parse(sessionID); err != nil {
		return models.PlaybackSession{},
			ErrPlaybackSessionRequired
	}

	return s.repo.GetSession(
		ctx,
		sessionID,
		userID,
	)
}

func (s *PlaybackService) UpdateProgress(
	ctx context.Context,
	sessionID string,
	userID int,
	input UpdatePlaybackProgressInput,
) (models.PlaybackSession, error) {
	if userID <= 0 {
		return models.PlaybackSession{},
			ErrPlaybackUserRequired
	}

	if err := validateSessionID(
		sessionID,
	); err != nil {
		return models.PlaybackSession{}, err
	}

	if err := validateProgressInput(
		input,
	); err != nil {
		return models.PlaybackSession{}, err
	}

	current, err := s.repo.GetSession(
		ctx,
		sessionID,
		userID,
	)
	if err != nil {
		return models.PlaybackSession{}, err
	}

	// A completed session no longer accepts progress.
	// Retrying completion (not progress) is the supported
	// way to finish a session.
	if current.Completed {
		return models.PlaybackSession{},
			repository.
				ErrPlaybackSessionCompleted
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
		return models.PlaybackSession{},
			ErrPlaybackPositionExceedsDuration
	}

	// Qualification is intentionally NOT decided here.
	//
	// The repository performs the authoritative decision
	// inside the row-locked transaction, after clamping
	// the client-reported listened_ms against server
	// wall-clock time. Deciding here from raw input would
	// let a malicious client qualify instantly by sending
	// a huge listened_ms value.
	return s.repo.UpdateProgress(
		ctx,
		sessionID,
		userID,
		input.PositionMS,
		effectiveDurationMS,
		input.ListenedMS,
	)
}

func (s *PlaybackService) CompleteSession(
	ctx context.Context,
	sessionID string,
	userID int,
	input UpdatePlaybackProgressInput,
) (models.PlaybackSession, error) {
	if userID <= 0 {
		return models.PlaybackSession{},
			ErrPlaybackUserRequired
	}

	if err := validateSessionID(
		sessionID,
	); err != nil {
		return models.PlaybackSession{}, err
	}

	if err := validateProgressInput(
		input,
	); err != nil {
		return models.PlaybackSession{}, err
	}

	current, err := s.repo.GetSession(
		ctx,
		sessionID,
		userID,
	)
	if err != nil {
		return models.PlaybackSession{}, err
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
		return models.PlaybackSession{},
			ErrPlaybackPositionExceedsDuration
	}

	// Qualification is intentionally NOT decided here —
	// same rationale as UpdateProgress: the repository
	// evaluates it inside the locked transaction from the
	// accepted, wall-clock-clamped listened time.
	return s.repo.CompleteSession(
		ctx,
		sessionID,
		userID,
		input.PositionMS,
		effectiveDurationMS,
		input.ListenedMS,
	)
}

func (s *PlaybackService) GetListeningHistory(
	ctx context.Context,
	userID int,
	query PlaybackHistoryQuery,
) ([]models.ListeningHistoryItem, error) {
	if userID <= 0 {
		return nil, ErrPlaybackUserRequired
	}

	limit := query.Limit

	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}

	offset := query.Offset

	if offset < 0 {
		offset = 0
	}

	return s.repo.GetListeningHistory(
		ctx,
		userID,
		limit,
		offset,
	)
}

func validateSessionID(
	sessionID string,
) error {
	if sessionID == "" {
		return ErrPlaybackSessionRequired
	}

	if _, err := uuid.Parse(sessionID); err != nil {
		return ErrPlaybackSessionRequired
	}

	return nil
}

func validateProgressInput(
	input UpdatePlaybackProgressInput,
) error {
	if input.DurationMS < 0 {
		return ErrInvalidPlaybackDuration
	}

	if input.PositionMS < 0 {
		return ErrInvalidPlaybackPosition
	}

	if input.ListenedMS < 0 {
		return ErrInvalidListenedDuration
	}

	return nil
}

func validatePlaybackPosition(
	durationMS int64,
	positionMS int64,
) error {
	if durationMS < 0 {
		return ErrInvalidPlaybackDuration
	}

	if positionMS < 0 {
		return ErrInvalidPlaybackPosition
	}

	if durationMS > 0 &&
		positionMS >
			durationMS+
				playbackPositionToleranceMS {
		return ErrPlaybackPositionExceedsDuration
	}

	return nil
}

func IsPlaybackValidationError(
	err error,
) bool {
	return errors.Is(
		err,
		ErrPlaybackUserRequired,
	) ||
		errors.Is(
			err,
			ErrPlaybackMusicRequired,
		) ||
		errors.Is(
			err,
			ErrInvalidPlaybackDuration,
		) ||
		errors.Is(
			err,
			ErrInvalidPlaybackPosition,
		) ||
		errors.Is(
			err,
			ErrInvalidListenedDuration,
		) ||
		errors.Is(
			err,
			ErrPlaybackPositionExceedsDuration,
		) ||
		errors.Is(
			err,
			ErrListenedDurationExceedsSession,
		) ||
		errors.Is(
			err,
			ErrPlaybackSessionRequired,
		)
}

// QualifiedPlayThresholdMS returns how much genuine
// listening time qualifies a play for the given track
// duration. See models.QualifiedPlayThresholdMS for the
// rule and its rationale.
func QualifiedPlayThresholdMS(
	durationMS int64,
) int64 {
	return models.
		QualifiedPlayThresholdMS(
			durationMS,
		)
}
