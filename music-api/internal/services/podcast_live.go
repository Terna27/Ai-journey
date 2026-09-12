package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"music-api/internal/models"
	"music-api/internal/repository"
)

const (
	defaultPodcastLivePageLimit = 20
	maxPodcastLivePageLimit     = 100

	// A broadcast may not be scheduled more than one year
	// ahead.
	maxPodcastLiveScheduleAhead = 365 * 24 * time.Hour

	// Hosts may start the broadcast a few minutes before the
	// scheduled time (room preparation). Starting after the
	// scheduled time is always allowed; the exact start
	// moment is stamped server-side.
	podcastLiveStartEarlyTolerance = 10 * time.Minute
)

var (
	ErrPodcastLiveScheduleRequired = errors.New(
		"scheduled_start_at is required",
	)

	ErrPodcastLiveScheduleMustBeFuture = errors.New(
		"scheduled_start_at must be in the future",
	)

	ErrPodcastLiveScheduleTooFarAhead = errors.New(
		"scheduled_start_at is too far in the future",
	)

	// The episode is not in a state a live broadcast can be
	// scheduled from (only DRAFT episodes are schedulable).
	ErrPodcastLiveEpisodeNotSchedulable = errors.New(
		"episode cannot be scheduled for a live broadcast",
	)

	// The parent podcast is archived (or otherwise not a
	// valid live host).
	ErrPodcastLivePodcastNotAllowed = errors.New(
		"podcast cannot host a live broadcast",
	)

	ErrPodcastLiveAlreadyScheduled = errors.New(
		"episode already has a live session",
	)

	ErrPodcastLiveStartTooEarly = errors.New(
		"live broadcast cannot start this early",
	)

	// The session is not currently joinable by listeners
	// (not scheduled/live, or not publicly visible).
	ErrPodcastLiveNotJoinable = errors.New(
		"live session is not joinable",
	)

	ErrPodcastLiveTokenHostOnly = errors.New(
		"host tokens are only available to the session host",
	)
)

// PodcastLiveRepo is the persistence surface the live
// scheduling service depends on.
type PodcastLiveRepo interface {
	ScheduleLiveSession(
		ctx context.Context,
		session models.PodcastLiveSession,
	) (models.PodcastLiveSession, error)

	GetLiveSessionByID(
		ctx context.Context,
		sessionID string,
	) (models.PodcastLiveSession, error)

	GetLiveSessionByEpisodeID(
		ctx context.Context,
		episodeID int64,
	) (models.PodcastLiveSession, error)

	GetOwnedLiveSession(
		ctx context.Context,
		sessionID string,
		hostUserID int,
	) (models.PodcastLiveSession, error)

	GetOwnedLiveDetails(
		ctx context.Context,
		sessionID string,
		hostUserID int,
	) (models.PodcastLiveDetails, error)

	ListUpcomingLiveSessions(
		ctx context.Context,
		limit int,
		offset int,
	) ([]models.PodcastLiveBroadcast, error)

	ListCurrentlyLiveSessions(
		ctx context.Context,
		limit int,
		offset int,
	) ([]models.PodcastLiveBroadcast, error)

	GetPublicLiveSessionByID(
		ctx context.Context,
		sessionID string,
	) (models.PodcastLiveBroadcast, error)

	StartLiveSession(
		ctx context.Context,
		sessionID string,
		hostUserID int,
	) (models.PodcastLiveSession, error)

	EndLiveSession(
		ctx context.Context,
		sessionID string,
		hostUserID int,
	) (models.PodcastLiveSession, error)

	CancelLiveSession(
		ctx context.Context,
		sessionID string,
		hostUserID int,
	) (models.PodcastLiveSession, error)

	PublishLiveRecording(
		ctx context.Context,
		sessionID string,
		hostUserID int,
	) (models.PodcastLiveSession, error)
}

// PodcastLiveEpisodeRepo is the ownership surface over
// podcasts/episodes the live service needs.
type PodcastLiveEpisodeRepo interface {
	GetOwnedPodcastByID(
		ctx context.Context,
		id int64,
		ownerUserID int,
	) (models.Podcast, error)

	GetOwnedEpisodeByID(
		ctx context.Context,
		id int64,
		ownerUserID int,
	) (models.PodcastEpisode, error)
}

type PodcastLiveService struct {
	repo        PodcastLiveRepo
	episodeRepo PodcastLiveEpisodeRepo

	provider LiveAudioProvider
}

func NewPodcastLiveService(
	repo PodcastLiveRepo,
	episodeRepo PodcastLiveEpisodeRepo,
	provider LiveAudioProvider,
) *PodcastLiveService {
	return &PodcastLiveService{
		repo:        repo,
		episodeRepo: episodeRepo,
		provider:    provider,
	}
}

// ScheduleLiveEpisodeInput is the client request for
// scheduling a live broadcast.
//
// Deliberately minimal: the host, provider room, and every
// runtime timestamp are decided server-side.
type ScheduleLiveEpisodeInput struct {
	ScheduledStartAt time.Time
}

// -----------------------------------------------------------------
// Scheduling
// -----------------------------------------------------------------

// ScheduleLiveEpisode schedules a live broadcast for a DRAFT
// episode owned by hostUserID.
//
// State machine (episode): DRAFT -> SCHEDULED.
// State machine (session): (none) -> SCHEDULED.
//
// The room name is generated server-side and is
// unpredictable, so room names cannot be guessed and cannot
// collide.
func (s *PodcastLiveService) ScheduleLiveEpisode(
	ctx context.Context,
	hostUserID int,
	episodeID int64,
	in ScheduleLiveEpisodeInput,
) (models.PodcastLiveDetails, error) {
	if hostUserID <= 0 {
		return models.PodcastLiveDetails{},
			ErrUnauthorized
	}

	if episodeID <= 0 {
		return models.PodcastLiveDetails{},
			repository.ErrPodcastLiveSessionNotFound
	}

	scheduled := in.ScheduledStartAt.UTC()

	if scheduled.IsZero() {
		return models.PodcastLiveDetails{},
			ErrPodcastLiveScheduleRequired
	}

	now := time.Now().UTC()

	if !scheduled.After(now) {
		return models.PodcastLiveDetails{},
			ErrPodcastLiveScheduleMustBeFuture
	}

	if scheduled.After(now.Add(maxPodcastLiveScheduleAhead)) {
		return models.PodcastLiveDetails{},
			ErrPodcastLiveScheduleTooFarAhead
	}

	// Ownership: another user's episode must be
	// indistinguishable from a missing one.
	episode, err :=
		s.episodeRepo.GetOwnedEpisodeByID(
			ctx,
			episodeID,
			hostUserID,
		)
	if err != nil {
		return models.PodcastLiveDetails{},
			mapPodcastLiveNotFound(err)
	}

	// Check whether this episode already has a live-session
	// history before applying the generic episode-state rule.
	//
	// A non-cancelled session owns the episode's live slot and
	// must produce the specific duplicate-scheduling conflict.
	// A CANCELLED session is retained for audit but deliberately
	// releases that slot, so the episode may be scheduled again.
	existingSession, existingErr :=
		s.repo.GetLiveSessionByEpisodeID(
			ctx,
			episodeID,
		)

	switch {
	case existingErr == nil:
		if existingSession.Status !=
			models.PodcastLiveStatusCancelled {
			return models.PodcastLiveDetails{},
				ErrPodcastLiveAlreadyScheduled
		}

	case errors.Is(
		existingErr,
		repository.ErrPodcastLiveSessionNotFound,
	):
		// No live-session history exists yet. Continue with the
		// normal scheduling checks below.

	default:
		return models.PodcastLiveDetails{}, existingErr
	}

	// Only a DRAFT episode can be scheduled for a live
	// broadcast. Published/archived/live/ended episodes are
	// rejected. A cancelled live session returns its episode to
	// DRAFT, so legitimate rescheduling still passes here.
	if episode.Status != models.PodcastEpisodeStatusDraft {
		return models.PodcastLiveDetails{},
			ErrPodcastLiveEpisodeNotSchedulable
	}

	podcast, err :=
		s.episodeRepo.GetOwnedPodcastByID(
			ctx,
			episode.PodcastID,
			hostUserID,
		)
	if err != nil {
		return models.PodcastLiveDetails{},
			mapPodcastLiveNotFound(err)
	}

	// An archived podcast cannot host live broadcasts.
	if podcast.Status == models.PodcastStatusArchived {
		return models.PodcastLiveDetails{},
			ErrPodcastLivePodcastNotAllowed
	}

	session, err :=
		s.repo.ScheduleLiveSession(
			ctx,
			models.PodcastLiveSession{
				ID: uuid.NewString(),

				EpisodeID:  episodeID,
				HostUserID: hostUserID,

				Provider:         "LIVEKIT",
				ProviderRoomName: newPodcastLiveRoomName(),

				Status:           models.PodcastLiveStatusScheduled,
				ScheduledStartAt: scheduled,
			},
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrPodcastLiveEpisodeTaken,
		) {
			return models.PodcastLiveDetails{},
				ErrPodcastLiveAlreadyScheduled
		}

		if errors.Is(
			err,
			repository.ErrPodcastLiveEpisodeInconsistent,
		) {
			// The episode stopped being a draft between the
			// check above and the transaction.
			return models.PodcastLiveDetails{},
				ErrPodcastLiveEpisodeNotSchedulable
		}

		return models.PodcastLiveDetails{}, err
	}

	return s.getOwnedDetails(
		ctx,
		session.ID,
		hostUserID,
	)
}

// GetOwnedLiveByEpisode returns the owner-facing live session
// of an episode (latest session row, including cancelled
// ones).
func (s *PodcastLiveService) GetOwnedLiveByEpisode(
	ctx context.Context,
	hostUserID int,
	episodeID int64,
) (models.PodcastLiveDetails, error) {
	if hostUserID <= 0 {
		return models.PodcastLiveDetails{},
			ErrUnauthorized
	}

	if episodeID <= 0 {
		return models.PodcastLiveDetails{},
			repository.ErrPodcastLiveSessionNotFound
	}

	// Ownership check first: another user's episode must be
	// indistinguishable from a missing one.
	_, err :=
		s.episodeRepo.GetOwnedEpisodeByID(
			ctx,
			episodeID,
			hostUserID,
		)
	if err != nil {
		return models.PodcastLiveDetails{},
			mapPodcastLiveNotFound(err)
	}

	session, err :=
		s.repo.GetLiveSessionByEpisodeID(
			ctx,
			episodeID,
		)
	if err != nil {
		return models.PodcastLiveDetails{},
			mapPodcastLiveNotFound(err)
	}

	return s.getOwnedDetails(
		ctx,
		session.ID,
		hostUserID,
	)
}

// -----------------------------------------------------------------
// State transitions
// -----------------------------------------------------------------

// StartLiveEpisode starts a scheduled broadcast.
//
// State machine (episode): SCHEDULED -> LIVE.
// State machine (session): SCHEDULED -> LIVE.
//
// started_at is stamped server-side. The host may start a few
// minutes before the scheduled time; client timestamps are
// never trusted.
func (s *PodcastLiveService) StartLiveEpisode(
	ctx context.Context,
	hostUserID int,
	episodeID int64,
) (models.PodcastLiveDetails, error) {
	session, err :=
		s.resolveOwnedLiveByEpisode(
			ctx,
			hostUserID,
			episodeID,
		)
	if err != nil {
		return models.PodcastLiveDetails{}, err
	}

	if session.Status != models.PodcastLiveStatusScheduled {
		return models.PodcastLiveDetails{},
			repository.ErrPodcastLiveSessionNotScheduled
	}

	// Small practical tolerance so hosts can open the room
	// before the announced start.
	now := time.Now().UTC()

	if session.ScheduledStartAt.Sub(now) >
		podcastLiveStartEarlyTolerance {
		return models.PodcastLiveDetails{},
			ErrPodcastLiveStartTooEarly
	}

	if _, err := s.repo.StartLiveSession(
		ctx,
		session.ID,
		hostUserID,
	); err != nil {
		return models.PodcastLiveDetails{}, err
	}

	return s.getOwnedDetails(
		ctx,
		session.ID,
		hostUserID,
	)
}

// EndLiveEpisode ends a running broadcast.
//
// State machine (episode): LIVE -> ENDED.
// State machine (session): LIVE -> ENDED.
func (s *PodcastLiveService) EndLiveEpisode(
	ctx context.Context,
	hostUserID int,
	episodeID int64,
) (models.PodcastLiveDetails, error) {
	session, err :=
		s.resolveOwnedLiveByEpisode(
			ctx,
			hostUserID,
			episodeID,
		)
	if err != nil {
		return models.PodcastLiveDetails{}, err
	}

	if session.Status != models.PodcastLiveStatusLive {
		return models.PodcastLiveDetails{},
			repository.ErrPodcastLiveSessionNotLive
	}

	if _, err := s.repo.EndLiveSession(
		ctx,
		session.ID,
		hostUserID,
	); err != nil {
		return models.PodcastLiveDetails{}, err
	}

	return s.getOwnedDetails(
		ctx,
		session.ID,
		hostUserID,
	)
}

// CancelLiveEpisode cancels a scheduled broadcast.
//
// State machine (session): SCHEDULED -> CANCELLED.
// The episode returns to DRAFT (editable, scheduled_at
// cleared), so the owner can reschedule later.
func (s *PodcastLiveService) CancelLiveEpisode(
	ctx context.Context,
	hostUserID int,
	episodeID int64,
) (models.PodcastLiveDetails, error) {
	session, err :=
		s.resolveOwnedLiveByEpisode(
			ctx,
			hostUserID,
			episodeID,
		)
	if err != nil {
		return models.PodcastLiveDetails{}, err
	}

	if session.Status != models.PodcastLiveStatusScheduled {
		return models.PodcastLiveDetails{},
			repository.ErrPodcastLiveSessionNotCancellable
	}

	if _, err := s.repo.CancelLiveSession(
		ctx,
		session.ID,
		hostUserID,
	); err != nil {
		return models.PodcastLiveDetails{}, err
	}

	return s.getOwnedDetails(
		ctx,
		session.ID,
		hostUserID,
	)
}

// PublishLiveRecording publishes the finished recording of an
// ENDED live session as the episode's regular audio.
//
// State machine (episode): ENDED -> PUBLISHED (published_at
// stamped server-side). Owner confirmation is explicit: this
// only runs when the host asks for it.
//
// After publishing, the episode is an ordinary on-demand
// episode served through the normal Phase 4.1 playback
// pipeline.
func (s *PodcastLiveService) PublishLiveRecording(
	ctx context.Context,
	hostUserID int,
	sessionID string,
) (models.PodcastLiveDetails, error) {
	if hostUserID <= 0 {
		return models.PodcastLiveDetails{},
			ErrUnauthorized
	}

	if !isPodcastLiveSessionID(sessionID) {
		return models.PodcastLiveDetails{},
			repository.ErrPodcastLiveSessionNotFound
	}

	session, err :=
		s.repo.PublishLiveRecording(
			ctx,
			sessionID,
			hostUserID,
		)
	if err != nil {
		return models.PodcastLiveDetails{}, err
	}

	return s.getOwnedDetails(
		ctx,
		session.ID,
		hostUserID,
	)
}

// -----------------------------------------------------------------
// Participant tokens
// -----------------------------------------------------------------

// GetHostToken mints a short-lived host token for the
// session's room. Only the session host can obtain one; the
// identity is derived server-side and never accepted from the
// client.
func (s *PodcastLiveService) GetHostToken(
	ctx context.Context,
	hostUserID int,
	sessionID string,
) (LiveAccessToken, error) {
	if hostUserID <= 0 {
		return LiveAccessToken{}, ErrUnauthorized
	}

	if !isPodcastLiveSessionID(sessionID) {
		return LiveAccessToken{},
			repository.ErrPodcastLiveSessionNotFound
	}

	session, err :=
		s.repo.GetOwnedLiveSession(
			ctx,
			sessionID,
			hostUserID,
		)
	if err != nil {
		return LiveAccessToken{},
			mapPodcastLiveNotFound(err)
	}

	// Tokens exist for scheduled and live sessions only.
	if session.Status != models.PodcastLiveStatusScheduled &&
		session.Status != models.PodcastLiveStatusLive {
		return LiveAccessToken{},
			repository.ErrPodcastLiveSessionNotScheduled
	}

	return s.provider.HostToken(
		session.ProviderRoomName,
		fmt.Sprintf(
			"host-%d",
			hostUserID,
		),
	)
}

// GetListenerToken mints a short-lived subscribe-only token
// for a publicly joinable live session.
//
// Anonymous listening is allowed: callers without an account
// receive a token with a server-generated ephemeral identity.
// Authenticated users get a stable derived identity. The
// caller never supplies an identity or permissions.
func (s *PodcastLiveService) GetListenerToken(
	ctx context.Context,
	sessionID string,
	userID int,
) (LiveAccessToken, error) {
	if !isPodcastLiveSessionID(sessionID) {
		return LiveAccessToken{},
			repository.ErrPodcastLiveSessionNotFound
	}

	// Public visibility check: the broadcast must belong to a
	// published podcast and be scheduled or live. Cancelled
	// and invisible sessions are indistinguishable from
	// missing ones.
	broadcast, err :=
		s.repo.GetPublicLiveSessionByID(
			ctx,
			sessionID,
		)
	if err != nil {
		return LiveAccessToken{},
			mapPodcastLiveNotFound(err)
	}

	if broadcast.LiveSession.Status != models.PodcastLiveStatusScheduled &&
		broadcast.LiveSession.Status != models.PodcastLiveStatusLive {
		return LiveAccessToken{},
			ErrPodcastLiveNotJoinable
	}

	// The internal session row carries the room name.
	session, err :=
		s.repo.GetLiveSessionByID(
			ctx,
			sessionID,
		)
	if err != nil {
		return LiveAccessToken{},
			mapPodcastLiveNotFound(err)
	}

	identity := fmt.Sprintf(
		"listener-%s",
		uuid.NewString()[:8],
	)

	if userID > 0 {
		identity = fmt.Sprintf(
			"listener-%d",
			userID,
		)
	}

	return s.provider.ListenerToken(
		session.ProviderRoomName,
		identity,
	)
}

// -----------------------------------------------------------------
// Public discovery
// -----------------------------------------------------------------

func (s *PodcastLiveService) ListUpcomingLive(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.PodcastLiveBroadcast, error) {
	limit, offset = normalizePodcastLivePagination(
		limit,
		offset,
	)

	return s.repo.ListUpcomingLiveSessions(
		ctx,
		limit,
		offset,
	)
}

func (s *PodcastLiveService) ListCurrentlyLive(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.PodcastLiveBroadcast, error) {
	limit, offset = normalizePodcastLivePagination(
		limit,
		offset,
	)

	return s.repo.ListCurrentlyLiveSessions(
		ctx,
		limit,
		offset,
	)
}

func (s *PodcastLiveService) GetPublicLive(
	ctx context.Context,
	sessionID string,
) (models.PodcastLiveBroadcast, error) {
	if !isPodcastLiveSessionID(sessionID) {
		return models.PodcastLiveBroadcast{},
			repository.ErrPodcastLiveSessionNotFound
	}

	broadcast, err :=
		s.repo.GetPublicLiveSessionByID(
			ctx,
			sessionID,
		)
	if err != nil {
		return models.PodcastLiveBroadcast{},
			mapPodcastLiveNotFound(err)
	}

	return broadcast, nil
}

// -----------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------

// resolveOwnedLiveByEpisode loads the episode's live session
// after verifying the caller owns the episode.
func (s *PodcastLiveService) resolveOwnedLiveByEpisode(
	ctx context.Context,
	hostUserID int,
	episodeID int64,
) (models.PodcastLiveSession, error) {
	if hostUserID <= 0 {
		return models.PodcastLiveSession{},
			ErrUnauthorized
	}

	if episodeID <= 0 {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotFound
	}

	_, err :=
		s.episodeRepo.GetOwnedEpisodeByID(
			ctx,
			episodeID,
			hostUserID,
		)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	session, err :=
		s.repo.GetLiveSessionByEpisodeID(
			ctx,
			episodeID,
		)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	return session, nil
}

func (s *PodcastLiveService) getOwnedDetails(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveDetails, error) {
	details, err :=
		s.repo.GetOwnedLiveDetails(
			ctx,
			sessionID,
			hostUserID,
		)
	if err != nil {
		return models.PodcastLiveDetails{},
			mapPodcastLiveNotFound(err)
	}

	return details, nil
}

func newPodcastLiveRoomName() string {
	// Random UUID suffix: unpredictable (room names cannot be
	// guessed) and collision-free in practice.
	return fmt.Sprintf(
		"podcast-live-%s",
		uuid.NewString(),
	)
}

// isPodcastLiveSessionID reports whether value is a well-formed
// live session identifier (UUID). Rejecting malformed IDs here
// keeps them from reaching the database as uncontrolled errors.
func isPodcastLiveSessionID(
	value string,
) bool {
	value = strings.TrimSpace(value)

	if value == "" {
		return false
	}

	_, err := uuid.Parse(value)

	return err == nil
}

func normalizePodcastLivePagination(
	limit int,
	offset int,
) (int, int) {
	if limit <= 0 {
		limit = defaultPodcastLivePageLimit
	}

	if limit > maxPodcastLivePageLimit {
		limit = maxPodcastLivePageLimit
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

func mapPodcastLiveNotFound(
	err error,
) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.ErrPodcastLiveSessionNotFound
	}

	return err
}

// IsPodcastLiveValidationError reports whether err is a
// client-fixable live scheduling error.
func IsPodcastLiveValidationError(
	err error,
) bool {
	validationErrors := []error{
		ErrPodcastLiveScheduleRequired,
		ErrPodcastLiveScheduleMustBeFuture,
		ErrPodcastLiveScheduleTooFarAhead,
		ErrPodcastLiveEpisodeNotSchedulable,
		ErrPodcastLivePodcastNotAllowed,
		ErrPodcastLiveAlreadyScheduled,
		ErrPodcastLiveStartTooEarly,
		ErrPodcastLiveNotJoinable,
	}

	for _, candidate := range validationErrors {
		if errors.Is(err, candidate) {
			return true
		}
	}

	return false
}
