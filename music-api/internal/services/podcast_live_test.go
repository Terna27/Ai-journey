package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"music-api/internal/models"
	"music-api/internal/repository"
)

// -----------------------------------------------------------------
// Fakes
// -----------------------------------------------------------------

type fakePodcastLiveRepo struct {
	sessions map[string]models.PodcastLiveSession

	byEpisode map[int64]string

	scheduled []models.PodcastLiveSession
	started   []string
	ended     []string
	cancelled []string
	published []string

	publicVisible map[string]bool

	// episodeRepo mirrors the real transaction: publishing a
	// recording also transitions the episode. Wired by the
	// test kit.
	episodeRepo *fakePodcastLiveEpisodeRepo
}

func newFakePodcastLiveRepo() *fakePodcastLiveRepo {
	return &fakePodcastLiveRepo{
		sessions:      map[string]models.PodcastLiveSession{},
		byEpisode:     map[int64]string{},
		publicVisible: map[string]bool{},
	}
}

func (f *fakePodcastLiveRepo) put(
	session models.PodcastLiveSession,
	public bool,
) {
	f.sessions[session.ID] = session
	f.byEpisode[session.EpisodeID] = session.ID
	f.publicVisible[session.ID] = public
}

func (f *fakePodcastLiveRepo) ScheduleLiveSession(
	ctx context.Context,
	session models.PodcastLiveSession,
) (models.PodcastLiveSession, error) {
	if _, exists := f.byEpisode[session.EpisodeID]; exists {
		current := f.sessions[f.byEpisode[session.EpisodeID]]

		if current.Status != models.PodcastLiveStatusCancelled {
			return models.PodcastLiveSession{},
				repository.ErrPodcastLiveEpisodeTaken
		}
	}

	// Mirror the real transaction: DRAFT -> SCHEDULED.
	if episode, ok :=
		f.episodeRepo.episodes[session.EpisodeID]; ok {
		if episode.Status != models.PodcastEpisodeStatusDraft {
			return models.PodcastLiveSession{},
				repository.ErrPodcastLiveEpisodeInconsistent
		}

		episode.Status = models.PodcastEpisodeStatusScheduled
		episode.ScheduledAt = &session.ScheduledStartAt

		f.episodeRepo.episodes[episode.ID] = episode
	}

	f.scheduled = append(f.scheduled, session)

	// Scheduled sessions of published podcasts are publicly
	// discoverable.
	f.put(session, true)

	return session, nil
}

func (f *fakePodcastLiveRepo) GetLiveSessionByID(
	ctx context.Context,
	sessionID string,
) (models.PodcastLiveSession, error) {
	session, ok := f.sessions[sessionID]

	if !ok {
		return models.PodcastLiveSession{},
			pgx.ErrNoRows
	}

	return session, nil
}

func (f *fakePodcastLiveRepo) GetLiveSessionByEpisodeID(
	ctx context.Context,
	episodeID int64,
) (models.PodcastLiveSession, error) {
	sessionID, ok := f.byEpisode[episodeID]

	if !ok {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotFound
	}

	return f.sessions[sessionID], nil
}

func (f *fakePodcastLiveRepo) GetOwnedLiveSession(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	session, ok := f.sessions[sessionID]

	if !ok || session.HostUserID != hostUserID {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotFound
	}

	return session, nil
}

func (f *fakePodcastLiveRepo) GetOwnedLiveDetails(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveDetails, error) {
	session, err := f.GetOwnedLiveSession(
		ctx,
		sessionID,
		hostUserID,
	)
	if err != nil {
		return models.PodcastLiveDetails{}, err
	}

	details := models.PodcastLiveDetails{
		LiveSession: session,
	}

	// Mirror the real joined payload: episode + podcast.
	if f.episodeRepo != nil {
		if episode, ok :=
			f.episodeRepo.episodes[session.EpisodeID]; ok {
			details.Episode = episode

			if podcast, ok :=
				f.episodeRepo.podcasts[episode.PodcastID]; ok {
				details.Podcast = podcast
			}
		}
	}

	return details, nil
}

func (f *fakePodcastLiveRepo) ListUpcomingLiveSessions(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.PodcastLiveBroadcast, error) {
	return nil, nil
}

func (f *fakePodcastLiveRepo) ListCurrentlyLiveSessions(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.PodcastLiveBroadcast, error) {
	return nil, nil
}

func (f *fakePodcastLiveRepo) GetPublicLiveSessionByID(
	ctx context.Context,
	sessionID string,
) (models.PodcastLiveBroadcast, error) {
	session, ok := f.sessions[sessionID]

	if !ok ||
		!f.publicVisible[sessionID] ||
		session.Status == models.PodcastLiveStatusCancelled {
		return models.PodcastLiveBroadcast{},
			repository.ErrPodcastLiveSessionNotFound
	}

	return models.PodcastLiveBroadcast{
		LiveSession: session.Public(),
	}, nil
}

func (f *fakePodcastLiveRepo) StartLiveSession(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	session, ok := f.sessions[sessionID]

	if !ok || session.HostUserID != hostUserID {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotFound
	}

	if session.Status != models.PodcastLiveStatusScheduled {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotScheduled
	}

	now := time.Now().UTC()

	session.Status = models.PodcastLiveStatusLive
	session.StartedAt = &now

	// Mirror the real transaction: SCHEDULED -> LIVE.
	if episode, ok :=
		f.episodeRepo.episodes[session.EpisodeID]; ok {
		episode.Status = models.PodcastEpisodeStatusLive

		f.episodeRepo.episodes[episode.ID] = episode
	}

	f.put(session, f.publicVisible[sessionID])
	f.started = append(f.started, sessionID)

	return session, nil
}

func (f *fakePodcastLiveRepo) EndLiveSession(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	session, ok := f.sessions[sessionID]

	if !ok || session.HostUserID != hostUserID {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotFound
	}

	if session.Status != models.PodcastLiveStatusLive {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotLive
	}

	now := time.Now().UTC()

	session.Status = models.PodcastLiveStatusEnded
	session.EndedAt = &now

	// Mirror the real transaction: LIVE -> ENDED.
	if episode, ok :=
		f.episodeRepo.episodes[session.EpisodeID]; ok {
		episode.Status = models.PodcastEpisodeStatusEnded

		f.episodeRepo.episodes[episode.ID] = episode
	}

	f.put(session, f.publicVisible[sessionID])
	f.ended = append(f.ended, sessionID)

	return session, nil
}

func (f *fakePodcastLiveRepo) CancelLiveSession(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	session, ok := f.sessions[sessionID]

	if !ok || session.HostUserID != hostUserID {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotFound
	}

	if session.Status != models.PodcastLiveStatusScheduled {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotCancellable
	}

	session.Status = models.PodcastLiveStatusCancelled

	// Mirror the real transaction: back to DRAFT with the
	// schedule cleared.
	if episode, ok :=
		f.episodeRepo.episodes[session.EpisodeID]; ok {
		episode.Status = models.PodcastEpisodeStatusDraft
		episode.ScheduledAt = nil

		f.episodeRepo.episodes[episode.ID] = episode
	}

	f.put(session, f.publicVisible[sessionID])
	f.cancelled = append(f.cancelled, sessionID)

	return session, nil
}

func (f *fakePodcastLiveRepo) PublishLiveRecording(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	session, ok := f.sessions[sessionID]

	if !ok || session.HostUserID != hostUserID {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotFound
	}

	if session.Status != models.PodcastLiveStatusEnded {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveSessionNotEnded
	}

	if session.RecordingStatus != models.PodcastLiveRecordingReady ||
		session.RecordingURL == nil ||
		*session.RecordingURL == "" {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveRecordingNotReady
	}

	episode, ok := f.episodeRepo.episodes[session.EpisodeID]

	if !ok || episode.Status != models.PodcastEpisodeStatusEnded {
		return models.PodcastLiveSession{},
			repository.ErrPodcastLiveEpisodeInconsistent
	}

	episode.AudioURL = *session.RecordingURL
	episode.Status = models.PodcastEpisodeStatusPublished
	episode.ScheduledAt = nil

	now := time.Now().UTC()
	episode.PublishedAt = &now

	f.episodeRepo.episodes[episode.ID] = episode

	f.published = append(f.published, sessionID)

	return session, nil
}

type fakePodcastLiveEpisodeRepo struct {
	podcasts  map[int64]models.Podcast
	episodes  map[int64]models.PodcastEpisode
	ownerByID map[int64]int
}

func newFakePodcastLiveEpisodeRepo() *fakePodcastLiveEpisodeRepo {
	return &fakePodcastLiveEpisodeRepo{
		podcasts:  map[int64]models.Podcast{},
		episodes:  map[int64]models.PodcastEpisode{},
		ownerByID: map[int64]int{},
	}
}

func (f *fakePodcastLiveEpisodeRepo) GetOwnedPodcastByID(
	ctx context.Context,
	id int64,
	ownerUserID int,
) (models.Podcast, error) {
	podcast, ok := f.podcasts[id]

	if !ok || podcast.OwnerUserID != ownerUserID {
		return models.Podcast{}, pgx.ErrNoRows
	}

	return podcast, nil
}

func (f *fakePodcastLiveEpisodeRepo) GetOwnedEpisodeByID(
	ctx context.Context,
	id int64,
	ownerUserID int,
) (models.PodcastEpisode, error) {
	episode, ok := f.episodes[id]

	if !ok || f.ownerByID[id] != ownerUserID {
		return models.PodcastEpisode{}, pgx.ErrNoRows
	}

	return episode, nil
}

type fakeLiveProvider struct {
	configured bool

	hostCalls     []string
	listenerCalls []string

	hostIdentity     string
	listenerIdentity string
}

func (f *fakeLiveProvider) Configured() bool {
	return f.configured
}

func (f *fakeLiveProvider) HostToken(
	roomName string,
	identity string,
) (LiveAccessToken, error) {
	if !f.configured {
		return LiveAccessToken{},
			ErrLiveProviderNotConfigured
	}

	f.hostCalls = append(f.hostCalls, roomName)
	f.hostIdentity = identity

	return LiveAccessToken{
		Token:      "host-token",
		ConnectURL: "wss://live.example",
	}, nil
}

func (f *fakeLiveProvider) ListenerToken(
	roomName string,
	identity string,
) (LiveAccessToken, error) {
	if !f.configured {
		return LiveAccessToken{},
			ErrLiveProviderNotConfigured
	}

	f.listenerCalls = append(f.listenerCalls, roomName)
	f.listenerIdentity = identity

	return LiveAccessToken{
		Token:      "listener-token",
		ConnectURL: "wss://live.example",
	}, nil
}

// -----------------------------------------------------------------
// Test fixture
// -----------------------------------------------------------------

func newPodcastLiveServiceTestKit() (
	*PodcastLiveService,
	*fakePodcastLiveRepo,
	*fakePodcastLiveEpisodeRepo,
	*fakeLiveProvider,
) {
	repo := newFakePodcastLiveRepo()

	episodeRepo := newFakePodcastLiveEpisodeRepo()

	episodeRepo.podcasts[1] = models.Podcast{
		ID:          1,
		OwnerUserID: 7,
		Status:      models.PodcastStatusPublished,
	}

	episodeRepo.episodes[10] = models.PodcastEpisode{
		ID:        10,
		PodcastID: 1,
		Status:    models.PodcastEpisodeStatusDraft,
	}

	episodeRepo.ownerByID[10] = 7

	provider := &fakeLiveProvider{
		configured: true,
	}

	// Mirror the real transaction: publishing a recording
	// also transitions the episode row.
	repo.episodeRepo = episodeRepo

	service := NewPodcastLiveService(
		repo,
		episodeRepo,
		provider,
	)

	return service, repo, episodeRepo, provider
}

func futureTime() time.Time {
	return time.Now().UTC().Add(2 * time.Hour)
}

// -----------------------------------------------------------------
// Scheduling
// -----------------------------------------------------------------

func TestScheduleLiveEpisodeRequiresUser(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	_, err := service.ScheduleLiveEpisode(
		context.Background(),
		0,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: futureTime(),
		},
	)

	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf(
			"expected ErrUnauthorized, got %v",
			err,
		)
	}
}

func TestScheduleLiveEpisodeRequiresFutureTime(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	_, err := service.ScheduleLiveEpisode(
		context.Background(),
		7,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: time.Now().UTC().Add(-time.Hour),
		},
	)

	if !errors.Is(
		err,
		ErrPodcastLiveScheduleMustBeFuture,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveScheduleMustBeFuture, got %v",
			err,
		)
	}
}

func TestScheduleLiveEpisodeRejectsFarFuture(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	_, err := service.ScheduleLiveEpisode(
		context.Background(),
		7,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: time.Now().UTC().Add(
				2 * 365 * 24 * time.Hour,
			),
		},
	)

	if !errors.Is(
		err,
		ErrPodcastLiveScheduleTooFarAhead,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveScheduleTooFarAhead, got %v",
			err,
		)
	}
}

func TestScheduleLiveEpisodeRejectsForeignEpisode(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	// User 8 does not own episode 10.
	_, err := service.ScheduleLiveEpisode(
		context.Background(),
		8,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: futureTime(),
		},
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotFound,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotFound, got %v",
			err,
		)
	}
}

func TestScheduleLiveEpisodeRejectsNonDraftEpisode(t *testing.T) {
	service, _, episodeRepo, _ :=
		newPodcastLiveServiceTestKit()

	episodeRepo.episodes[10] = models.PodcastEpisode{
		ID:        10,
		PodcastID: 1,
		Status:    models.PodcastEpisodeStatusPublished,
	}

	_, err := service.ScheduleLiveEpisode(
		context.Background(),
		7,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: futureTime(),
		},
	)

	if !errors.Is(
		err,
		ErrPodcastLiveEpisodeNotSchedulable,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveEpisodeNotSchedulable, got %v",
			err,
		)
	}
}

func TestScheduleLiveEpisodeRejectsArchivedPodcast(t *testing.T) {
	service, _, episodeRepo, _ :=
		newPodcastLiveServiceTestKit()

	episodeRepo.podcasts[1] = models.Podcast{
		ID:          1,
		OwnerUserID: 7,
		Status:      models.PodcastStatusArchived,
	}

	_, err := service.ScheduleLiveEpisode(
		context.Background(),
		7,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: futureTime(),
		},
	)

	if !errors.Is(
		err,
		ErrPodcastLivePodcastNotAllowed,
	) {
		t.Fatalf(
			"expected ErrPodcastLivePodcastNotAllowed, got %v",
			err,
		)
	}
}

func TestScheduleLiveEpisodeCreatesScheduledSession(t *testing.T) {
	service, repo, _, _ :=
		newPodcastLiveServiceTestKit()

	scheduled := futureTime()

	details, err := service.ScheduleLiveEpisode(
		context.Background(),
		7,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: scheduled,
		},
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	session := details.LiveSession

	if session.Status != models.PodcastLiveStatusScheduled {
		t.Fatalf(
			"expected SCHEDULED session, got %s",
			session.Status,
		)
	}

	if !session.ScheduledStartAt.Equal(scheduled) {
		t.Fatalf(
			"scheduled time not preserved: %v",
			session.ScheduledStartAt,
		)
	}

	if session.ProviderRoomName == "" {
		t.Fatal(
			"expected a server-generated room name",
		)
	}

	if session.HostUserID != 7 {
		t.Fatalf(
			"expected host 7, got %d",
			session.HostUserID,
		)
	}

	if len(repo.scheduled) != 1 {
		t.Fatalf(
			"expected exactly one scheduling, got %d",
			len(repo.scheduled),
		)
	}
}

func TestScheduleLiveEpisodeRejectsDuplicate(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	_, err := service.ScheduleLiveEpisode(
		context.Background(),
		7,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: futureTime(),
		},
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	_, err = service.ScheduleLiveEpisode(
		context.Background(),
		7,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: futureTime(),
		},
	)

	if !errors.Is(
		err,
		ErrPodcastLiveAlreadyScheduled,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveAlreadyScheduled, got %v",
			err,
		)
	}
}

// -----------------------------------------------------------------
// Starting / ending / cancelling
// -----------------------------------------------------------------

func scheduleForTest(
	t *testing.T,
	service *PodcastLiveService,
	scheduled time.Time,
) models.PodcastLiveDetails {
	t.Helper()

	details, err := service.ScheduleLiveEpisode(
		context.Background(),
		7,
		10,
		ScheduleLiveEpisodeInput{
			ScheduledStartAt: scheduled,
		},
	)
	if err != nil {
		t.Fatalf(
			"scheduling failed: %v",
			err,
		)
	}

	return details
}

func TestStartLiveEpisodeRejectsTooEarly(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(2*time.Hour),
	)

	// Start through the episode-scoped API.
	_, err := service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)

	if !errors.Is(
		err,
		ErrPodcastLiveStartTooEarly,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveStartTooEarly, got %v",
			err,
		)
	}
}

func TestStartLiveEpisodeWithinTolerance(t *testing.T) {
	service, repo, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	started, err := service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if started.LiveSession.Status != models.PodcastLiveStatusLive {
		t.Fatalf(
			"expected LIVE session, got %s",
			started.LiveSession.Status,
		)
	}

	if started.LiveSession.StartedAt == nil {
		t.Fatal(
			"expected a server-stamped started_at",
		)
	}

	if len(repo.started) != 1 {
		t.Fatalf(
			"expected exactly one start, got %d",
			len(repo.started),
		)
	}
}

func TestStartLiveEpisodeRejectsDuplicateStart(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	_, err := service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	_, err = service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotScheduled,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotScheduled, got %v",
			err,
		)
	}
}

func TestEndLiveEpisodeRequiresLive(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	_, err := service.EndLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotLive,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotLive, got %v",
			err,
		)
	}
}

func TestEndLiveEpisodeRejectsDuplicateEnd(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	if _, err := service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	); err != nil {
		t.Fatalf(
			"start failed: %v",
			err,
		)
	}

	if _, err := service.EndLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	); err != nil {
		t.Fatalf(
			"end failed: %v",
			err,
		)
	}

	_, err := service.EndLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotLive,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotLive, got %v",
			err,
		)
	}
}

func TestCancelLiveEpisodeRequiresScheduled(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	if _, err := service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	); err != nil {
		t.Fatalf(
			"start failed: %v",
			err,
		)
	}

	_, err := service.CancelLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotCancellable,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotCancellable, got %v",
			err,
		)
	}
}

func TestCancelledSessionCannotStart(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	if _, err := service.CancelLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	); err != nil {
		t.Fatalf(
			"cancel failed: %v",
			err,
		)
	}

	_, err := service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotScheduled,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotScheduled, got %v",
			err,
		)
	}
}

// -----------------------------------------------------------------
// Tokens
// -----------------------------------------------------------------

func TestGetHostTokenRequiresOwnership(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		futureTime(),
	)

	_, err := service.GetHostToken(
		context.Background(),
		8,
		details.LiveSession.ID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotFound,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotFound, got %v",
			err,
		)
	}
}

func TestGetHostTokenDerivesIdentityServerSide(t *testing.T) {
	service, _, _, provider :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		futureTime(),
	)

	token, err := service.GetHostToken(
		context.Background(),
		7,
		details.LiveSession.ID,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if token.Token == "" {
		t.Fatal("expected a host token")
	}

	if provider.hostIdentity != "host-7" {
		t.Fatalf(
			"expected server-derived identity host-7, got %s",
			provider.hostIdentity,
		)
	}

	if len(provider.hostCalls) != 1 {
		t.Fatalf(
			"expected one host token mint, got %d",
			len(provider.hostCalls),
		)
	}
}

func TestGetHostTokenRejectedWhenProviderUnconfigured(t *testing.T) {
	service, _, _, provider :=
		newPodcastLiveServiceTestKit()

	provider.configured = false

	details := scheduleForTest(
		t,
		service,
		futureTime(),
	)

	_, err := service.GetHostToken(
		context.Background(),
		7,
		details.LiveSession.ID,
	)

	if !errors.Is(
		err,
		ErrLiveProviderNotConfigured,
	) {
		t.Fatalf(
			"expected ErrLiveProviderNotConfigured, got %v",
			err,
		)
	}
}

func TestGetListenerTokenAllowsAnonymous(t *testing.T) {
	service, _, _, provider :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		futureTime(),
	)

	token, err := service.GetListenerToken(
		context.Background(),
		details.LiveSession.ID,
		0,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if token.Token == "" {
		t.Fatal("expected a listener token")
	}

	if provider.listenerIdentity == "" ||
		provider.listenerIdentity == "host-7" {
		t.Fatalf(
			"expected an ephemeral listener identity, got %q",
			provider.listenerIdentity,
		)
	}
}

func TestGetListenerTokenUsesStableIdentityForUsers(t *testing.T) {
	service, _, _, provider :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		futureTime(),
	)

	_, err := service.GetListenerToken(
		context.Background(),
		details.LiveSession.ID,
		42,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if provider.listenerIdentity != "listener-42" {
		t.Fatalf(
			"expected listener-42, got %s",
			provider.listenerIdentity,
		)
	}
}

func TestGetListenerTokenRejectsInvalidID(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	_, err := service.GetListenerToken(
		context.Background(),
		"not-a-uuid",
		42,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotFound,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotFound, got %v",
			err,
		)
	}
}

func TestGetListenerTokenRejectsEnded(t *testing.T) {
	service, _, _, _ :=
		newPodcastLiveServiceTestKit()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	if _, err := service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	); err != nil {
		t.Fatalf(
			"start failed: %v",
			err,
		)
	}

	if _, err := service.EndLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	); err != nil {
		t.Fatalf(
			"end failed: %v",
			err,
		)
	}

	_, err := service.GetListenerToken(
		context.Background(),
		details.LiveSession.ID,
		42,
	)

	if !errors.Is(
		err,
		ErrPodcastLiveNotJoinable,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveNotJoinable, got %v",
			err,
		)
	}
}

// -----------------------------------------------------------------
// Recording publish
// -----------------------------------------------------------------

// endedSessionForTest drives one session through schedule ->
// start -> end, then marks its recording READY, returning the
// finished details.
func endedSessionForTest(
	t *testing.T,
	service *PodcastLiveService,
	repo *fakePodcastLiveRepo,
	recordingURL string,
) models.PodcastLiveDetails {
	t.Helper()

	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	if _, err := service.StartLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	); err != nil {
		t.Fatalf(
			"start failed: %v",
			err,
		)
	}

	ended, err := service.EndLiveEpisode(
		context.Background(),
		7,
		details.LiveSession.EpisodeID,
	)
	if err != nil {
		t.Fatalf(
			"end failed: %v",
			err,
		)
	}

	if recordingURL != "" {
		session := repo.sessions[ended.LiveSession.ID]
		session.RecordingStatus =
			models.PodcastLiveRecordingReady
		session.RecordingURL = &recordingURL
		repo.put(session, true)
	}

	return ended
}

func TestPublishLiveRecordingRequiresEndedSession(t *testing.T) {
	service, repo, _, _ :=
		newPodcastLiveServiceTestKit()

	// Still scheduled, not ended.
	details := scheduleForTest(
		t,
		service,
		time.Now().UTC().Add(5*time.Minute),
	)

	_, err := service.PublishLiveRecording(
		context.Background(),
		7,
		details.LiveSession.ID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotEnded,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotEnded, got %v",
			err,
		)
	}

	if len(repo.published) != 0 {
		t.Fatal(
			"expected no publish on a non-ended session",
		)
	}
}

func TestPublishLiveRecordingRequiresReadyRecording(t *testing.T) {
	service, repo, _, _ :=
		newPodcastLiveServiceTestKit()

	// Ended, but no recording was produced.
	details := endedSessionForTest(
		t,
		service,
		repo,
		"",
	)

	_, err := service.PublishLiveRecording(
		context.Background(),
		7,
		details.LiveSession.ID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveRecordingNotReady,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveRecordingNotReady, got %v",
			err,
		)
	}
}

func TestPublishLiveRecordingRequiresOwnership(t *testing.T) {
	service, repo, _, _ :=
		newPodcastLiveServiceTestKit()

	details := endedSessionForTest(
		t,
		service,
		repo,
		"https://recordings.example/live.m4a",
	)

	_, err := service.PublishLiveRecording(
		context.Background(),
		8,
		details.LiveSession.ID,
	)

	if !errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotFound,
	) {
		t.Fatalf(
			"expected ErrPodcastLiveSessionNotFound, got %v",
			err,
		)
	}
}

func TestPublishLiveRecordingPublishesEpisode(t *testing.T) {
	service, repo, episodeRepo, _ :=
		newPodcastLiveServiceTestKit()

	details := endedSessionForTest(
		t,
		service,
		repo,
		"https://recordings.example/live.m4a",
	)

	published, err := service.PublishLiveRecording(
		context.Background(),
		7,
		details.LiveSession.ID,
	)
	if err != nil {
		t.Fatalf(
			"publish failed: %v",
			err,
		)
	}

	if len(repo.published) != 1 {
		t.Fatalf(
			"expected exactly one publish, got %d",
			len(repo.published),
		)
	}

	// The episode became a normal published episode with the
	// recording as its audio.
	episode := episodeRepo.episodes[10]

	if episode.Status != models.PodcastEpisodeStatusPublished {
		t.Fatalf(
			"expected episode PUBLISHED, got %s",
			episode.Status,
		)
	}

	if episode.AudioURL != "https://recordings.example/live.m4a" {
		t.Fatalf(
			"expected recording URL as episode audio, got %q",
			episode.AudioURL,
		)
	}

	if episode.PublishedAt == nil {
		t.Fatal(
			"expected published_at to be stamped server-side",
		)
	}

	if episode.ScheduledAt != nil {
		t.Fatal(
			"expected scheduled_at to be cleared",
		)
	}

	if published.Episode.Status !=
		models.PodcastEpisodeStatusPublished {
		t.Fatalf(
			"expected returned episode PUBLISHED, got %s",
			published.Episode.Status,
		)
	}
}
