package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"music-api/internal/config"
	"music-api/internal/database"
	"music-api/internal/handler"
	"music-api/internal/middleware"
	"music-api/internal/repository"
	"music-api/internal/services"
)

func main() {
	// =========================
	// CONFIGURATION
	// =========================

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"failed to load configuration: %v",
			err,
		)
	}

	// =========================
	// DATABASE
	// =========================

	ctx := context.Background()

	db, err := database.Connect(
		ctx,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatalf(
			"failed to connect to database: %v",
			err,
		)
	}
	defer db.Close()

	// =========================
	// REPOSITORIES
	// =========================

	musicRepo :=
		repository.NewMusicRepository(
			db.Pool,
		)

	artistRepo :=
		repository.NewArtistRepository(
			db.Pool,
		)

	artistFollowRepo :=
		repository.NewArtistFollowRepository(
			db.Pool,
		)

	userRepo :=
		repository.NewUserRepository(
			db.Pool,
		)

	userMusicLikeRepo :=
		repository.NewUserMusicLikeRepository(
			db.Pool,
		)

	playlistRepo :=
		repository.NewPlaylistRepository(
			db.Pool,
		)

	releaseRepo :=
		repository.NewReleaseRepository(
			db.Pool,
		)

	searchRepo :=
		repository.NewSearchRepository(
			db.Pool,
		)

	discoveryRepo :=
		repository.NewDiscoveryRepository(
			db.Pool,
		)

	emailVerificationRepo :=
		repository.NewEmailVerificationRepository(
			db.Pool,
		)

	lyricsRepo :=
		repository.NewLyricsRepository(
			db.Pool,
		)

	playbackRepo :=
		repository.NewPlaybackRepository(
			db.Pool,
		)

	podcastRepo :=
		repository.NewPodcastRepository(
			db.Pool,
		)

	podcastPlaybackRepo :=
		repository.NewPodcastPlaybackRepository(
			db.Pool,
		)

	podcastLiveRepo :=
		repository.NewPodcastLiveRepository(
			db.Pool,
		)

	// =========================
	// SERVICES
	// =========================

	musicService :=
		services.NewMusicService(
			musicRepo,
			userMusicLikeRepo,
		)

	playlistService :=
		services.NewPlaylistService(
			playlistRepo,
		)

	releaseService :=
		services.NewReleaseService(
			releaseRepo,
		)

	searchService :=
		services.NewSearchService(
			searchRepo,
		)

	discoveryService :=
		services.NewDiscoveryService(
			discoveryRepo,
		)

	artistService :=
		services.NewArtistService(
			artistRepo,
		)

	artistFollowService :=
		services.NewArtistFollowService(
			artistFollowRepo,
			artistRepo,
		)

	userService :=
		services.NewUserService(
			userRepo,
		)

	lyricsService :=
		services.NewLyricsService(
			lyricsRepo,
		)

	playbackService :=
		services.NewPlaybackService(
			playbackRepo,
		)

	podcastService :=
		services.NewPodcastService(
			podcastRepo,
		)

	podcastPlaybackService :=
		services.NewPodcastPlaybackService(
			podcastPlaybackRepo,
		)

	// LiveKit is OPTIONAL: when its env vars are unset the
	// service simply reports itself unconfigured and the
	// token endpoints return a controlled 503. Scheduling
	// and state management work without a provider.
	liveKitService :=
		services.NewLiveKitService(
			cfg.LiveKitURL,
			cfg.LiveKitAPIKey,
			cfg.LiveKitAPISecret,
		)

	podcastLiveService :=
		services.NewPodcastLiveService(
			podcastLiveRepo,
			podcastRepo,
			liveKitService,
		)

	jwtService :=
		services.NewJWTService(
			cfg.JWTSecret,
		)

	// =========================
	// CLOUDINARY
	// =========================

	cloudinaryService, err :=
		services.NewCloudinaryService()

	if err != nil {
		log.Fatalf(
			"failed to initialize Cloudinary: %v",
			err,
		)
	}

	// =========================
	// EMAIL
	// =========================

	emailService, err :=
		services.NewEmailService(
			cfg.ResendAPIKey,
			cfg.EmailFrom,
		)

	if err != nil {
		log.Fatalf(
			"failed to initialize email service: %v",
			err,
		)
	}

	emailVerificationService :=
		services.NewEmailVerificationService(
			emailVerificationRepo,
			userService,
			emailService,
			cfg.FrontendURL,
		)

	// =========================
	// HANDLERS
	// =========================

	musicHandler :=
		handler.NewMusicHandler(
			musicService,
			artistService,
			userService,
			jwtService,
			cloudinaryService,
			emailVerificationService,
		)

	playlistHandler :=
		handler.NewPlaylistHandler(
			playlistService,
		)

	releaseHandler :=
		handler.NewReleaseHandler(
			releaseService,
		)

	searchHandler :=
		handler.NewSearchHandler(
			searchService,
		)

	discoveryHandler :=
		handler.NewDiscoveryHandler(
			discoveryService,
		)

	lyricsHandler :=
		handler.NewLyricsHandler(
			lyricsService,
		)

	artistFollowHandler :=
		handler.NewArtistFollowHandler(
			artistFollowService,
		)

	playbackHandler :=
		handler.NewPlaybackHandler(
			playbackService,
		)

	podcastHandler :=
		handler.NewPodcastHandler(
			podcastService,
			cloudinaryService,
		)

	podcastPlaybackHandler :=
		handler.NewPodcastPlaybackHandler(
			podcastPlaybackService,
		)

	podcastLiveHandler :=
		handler.NewPodcastLiveHandler(
			podcastLiveService,
			jwtService,
		)

	healthHandler :=
		handler.NewHealthHandler(
			db.Pool,
		)

	// =========================
	// ROUTER
	// =========================

	mux := http.NewServeMux()

	// =========================
	// HEALTH
	// =========================

	mux.HandleFunc(
		"GET /health",
		healthHandler.Health,
	)

	mux.HandleFunc(
		"GET /ready",
		healthHandler.Ready,
	)

	// =========================
	// AUTH
	// =========================

	mux.HandleFunc(
		"POST /api/v1/auth/register",
		musicHandler.RegisterUser,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		musicHandler.LoginUser,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/verify-email",
		musicHandler.VerifyEmail,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/resend-verification",
		musicHandler.ResendVerification,
	)

	// =========================
	// CURRENT USER
	// =========================

	mux.Handle(
		"GET /api/v1/me",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(
				musicHandler.GetMe,
			),
		),
	)

	// =========================
	// PUBLIC PODCASTS
	// =========================

	mux.HandleFunc(
		"GET /api/v1/podcasts",
		podcastHandler.GetPublishedPodcasts,
	)

	mux.HandleFunc(
		"GET /api/v1/podcasts/{id}",
		podcastHandler.GetPublicPodcast,
	)

	mux.HandleFunc(
		"GET /api/v1/podcasts/slug/{slug}",
		podcastHandler.GetPublicPodcastBySlug,
	)

	mux.HandleFunc(
		"GET /api/v1/podcast-episodes/{id}",
		podcastHandler.GetPublicEpisode,
	)

	// =========================
	// CREATE PODCAST
	// =========================

	mux.Handle(
		"POST /api/v1/podcasts",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.CreatePodcast,
				),
			),
		),
	)

	// =========================
	// CURRENT USER PODCASTS
	// =========================

	mux.Handle(
		"GET /api/v1/me/podcasts",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.GetMyPodcasts,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/me/podcasts/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.GetMyPodcast,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/me/podcasts/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.UpdatePodcast,
				),
			),
		),
	)

	mux.Handle(
		"DELETE /api/v1/me/podcasts/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.DeletePodcast,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/me/podcasts/{id}/artwork",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.UpdatePodcastArtwork,
				),
			),
		),
	)

	// =========================
	// PODCAST EPISODES
	// =========================

	mux.Handle(
		"POST /api/v1/me/podcasts/{id}/episodes",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.CreateEpisode,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/me/podcast-episodes/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.UpdateEpisode,
				),
			),
		),
	)

	mux.Handle(
		"DELETE /api/v1/me/podcast-episodes/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.DeleteEpisode,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/me/podcast-episodes/{id}/media",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastHandler.UpdateEpisodeMedia,
				),
			),
		),
	)

	// =========================
	// CURRENT ARTIST RELEASES
	// =========================

	mux.Handle(
		"GET /api/v1/me/releases",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						releaseHandler.GetMyReleases,
					),
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/me/releases/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						releaseHandler.GetMyRelease,
					),
				),
			),
		),
	)

	// =========================
	// LIKED MUSIC LIBRARY
	// =========================

	mux.Handle(
		"GET /api/v1/me/liked-music",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					musicHandler.GetLikedMusic,
				),
			),
		),
	)

	// =========================
	// ARTIST PROFILE
	// =========================

	mux.HandleFunc(
		"GET /api/v1/artists/{id}",
		musicHandler.GetArtistProfile,
	)

	mux.Handle(
		"POST /api/v1/artists/profile",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					musicHandler.CreateArtistProfile,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/me/artist-profile",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						musicHandler.UpdateArtistProfile,
					),
				),
			),
		),
	)

	// =========================
	// ARTIST RELEASES
	// =========================

	mux.HandleFunc(
		"GET /api/v1/artists/{id}/releases",
		releaseHandler.GetArtistReleases,
	)

	// =========================
	// PUBLIC RELEASES
	// =========================

	mux.HandleFunc(
		"GET /api/v1/releases/{id}",
		releaseHandler.GetRelease,
	)

	// =========================
	// CREATE RELEASE
	// =========================

	mux.Handle(
		"POST /api/v1/releases",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						releaseHandler.CreateRelease,
					),
				),
			),
		),
	)

	// =========================
	// UPDATE RELEASE
	// =========================

	mux.Handle(
		"PUT /api/v1/releases/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						releaseHandler.UpdateRelease,
					),
				),
			),
		),
	)

	// =========================
	// DELETE RELEASE
	// =========================

	mux.Handle(
		"DELETE /api/v1/releases/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						releaseHandler.DeleteRelease,
					),
				),
			),
		),
	)

	// =========================
	// ADD TRACK TO RELEASE
	// =========================

	mux.Handle(
		"POST /api/v1/releases/{id}/tracks/{musicID}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						releaseHandler.AddTrack,
					),
				),
			),
		),
	)

	// =========================
	// REMOVE TRACK FROM RELEASE
	// =========================

	mux.Handle(
		"DELETE /api/v1/releases/{id}/tracks/{musicID}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						releaseHandler.RemoveTrack,
					),
				),
			),
		),
	)

	// =========================
	// PUBLIC ARTIST FOLLOW COUNT
	// =========================

	mux.HandleFunc(
		"GET /api/v1/artists/{id}/followers/count",
		artistFollowHandler.GetFollowerCount,
	)

	// =========================
	// ARTIST FOLLOW
	// =========================

	mux.Handle(
		"POST /api/v1/artists/{id}/follow",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					artistFollowHandler.Follow,
				),
			),
		),
	)

	mux.Handle(
		"DELETE /api/v1/artists/{id}/follow",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					artistFollowHandler.Unfollow,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/artists/{id}/follow",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					artistFollowHandler.GetStatus,
				),
			),
		),
	)

	// =========================
	// PUBLIC SEARCH
	// =========================

	mux.HandleFunc(
		"GET /api/v1/search",
		searchHandler.Search,
	)

	// =========================
	// PUBLIC DISCOVERY
	// =========================

	mux.HandleFunc(
		"GET /api/v1/discover",
		discoveryHandler.Discover,
	)

	mux.HandleFunc(
		"GET /api/v1/discovery/hero-artists",
		discoveryHandler.HeroArtists,
	)

	// =========================
	// PUBLIC MUSIC
	// =========================

	mux.HandleFunc(
		"GET /api/v1/music",
		musicHandler.GetAllMusic,
	)

	mux.HandleFunc(
		"GET /api/v1/music/{id}",
		musicHandler.GetMusic,
	)

	// =========================
	// PUBLIC LYRICS
	// =========================

	mux.HandleFunc(
		"GET /api/v1/music/{id}/lyrics",
		lyricsHandler.GetLyrics,
	)

	// =========================
	// CREATE MUSIC
	// =========================

	mux.Handle(
		"POST /api/v1/music",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						musicHandler.CreateMusic,
					),
				),
			),
		),
	)

	// =========================
	// UPDATE MUSIC
	// =========================

	mux.Handle(
		"PUT /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						musicHandler.UpdateMusic,
					),
				),
			),
		),
	)

	// =========================
	// PATCH MUSIC
	// =========================

	mux.Handle(
		"PATCH /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						musicHandler.PatchMusic,
					),
				),
			),
		),
	)

	// =========================
	// DELETE MUSIC
	// =========================

	mux.Handle(
		"DELETE /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						musicHandler.DeleteMusic,
					),
				),
			),
		),
	)

	// =========================
	// CREATE / UPDATE LYRICS
	// =========================

	mux.Handle(
		"PUT /api/v1/music/{id}/lyrics",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						lyricsHandler.UpsertLyrics,
					),
				),
			),
		),
	)

	// =========================
	// DELETE LYRICS
	// =========================

	mux.Handle(
		"DELETE /api/v1/music/{id}/lyrics",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				middleware.RequireArtist(
					artistRepo,
					http.HandlerFunc(
						lyricsHandler.DeleteLyrics,
					),
				),
			),
		),
	)

	// =========================
	// LIKE MUSIC
	// =========================

	mux.Handle(
		"POST /api/v1/music/{id}/like",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					musicHandler.LikeMusic,
				),
			),
		),
	)

	// =========================
	// UNLIKE MUSIC
	// =========================

	mux.Handle(
		"DELETE /api/v1/music/{id}/like",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					musicHandler.UnlikeMusic,
				),
			),
		),
	)

	// =========================
	// PLAYLISTS
	// =========================

	mux.Handle(
		"POST /api/v1/playlists",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playlistHandler.CreatePlaylist,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/me/playlists",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playlistHandler.GetMyPlaylists,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/playlists/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playlistHandler.GetPlaylist,
				),
			),
		),
	)

	mux.Handle(
		"PUT /api/v1/playlists/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playlistHandler.UpdatePlaylist,
				),
			),
		),
	)

	mux.Handle(
		"DELETE /api/v1/playlists/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playlistHandler.DeletePlaylist,
				),
			),
		),
	)

	mux.Handle(
		"POST /api/v1/playlists/{id}/tracks/{musicID}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playlistHandler.AddTrack,
				),
			),
		),
	)

	mux.Handle(
		"DELETE /api/v1/playlists/{id}/tracks/{musicID}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playlistHandler.RemoveTrack,
				),
			),
		),
	)

	// =========================
	// PLAYBACK TRACKING
	// =========================
	//
	// Playback endpoints are for normal authenticated
	// listeners: JWT -> verified user -> handler. No
	// artist profile requirement.

	mux.Handle(
		"POST /api/v1/playback/sessions",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playbackHandler.CreateSession,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/playback/sessions/{session_id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playbackHandler.GetSession,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/playback/sessions/{session_id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playbackHandler.UpdateProgress,
				),
			),
		),
	)

	mux.Handle(
		"POST /api/v1/playback/sessions/{session_id}/complete",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playbackHandler.CompleteSession,
				),
			),
		),
	)

	// =========================
	// LISTENING HISTORY
	// =========================

	mux.Handle(
		"GET /api/v1/me/listening-history",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					playbackHandler.GetListeningHistory,
				),
			),
		),
	)

	// =========================
	// PODCAST PLAYBACK TRACKING
	// (Phase 4.1)
	//
	// Dedicated podcast playback endpoints. Podcast episode
	// ids must never be submitted to the music playback
	// session routes above.
	// =========================

	mux.Handle(
		"POST /api/v1/podcast-playback/sessions",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastPlaybackHandler.CreateSession,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/podcast-playback/sessions/{session_id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastPlaybackHandler.GetSession,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/podcast-playback/sessions/{session_id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastPlaybackHandler.UpdateProgress,
				),
			),
		),
	)

	mux.Handle(
		"POST /api/v1/podcast-playback/sessions/{session_id}/complete",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastPlaybackHandler.CompleteSession,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/me/podcast-listening-history",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastPlaybackHandler.GetPodcastListeningHistory,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/me/podcast-continue-listening",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastPlaybackHandler.GetContinueListening,
				),
			),
		),
	)

	// =========================
	// PODCAST LIVE (creator)
	// =========================
	//
	// Any verified user owning a podcast can host. No artist
	// role is required (unified user architecture).

	mux.Handle(
		"POST /api/v1/me/podcast-episodes/{id}/live/schedule",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastLiveHandler.ScheduleLiveEpisode,
				),
			),
		),
	)

	mux.Handle(
		"GET /api/v1/me/podcast-episodes/{id}/live",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastLiveHandler.GetLiveByEpisode,
				),
			),
		),
	)

	mux.Handle(
		"POST /api/v1/me/podcast-episodes/{id}/live/start",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastLiveHandler.StartLiveEpisode,
				),
			),
		),
	)

	mux.Handle(
		"POST /api/v1/me/podcast-episodes/{id}/live/end",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastLiveHandler.EndLiveEpisode,
				),
			),
		),
	)

	mux.Handle(
		"POST /api/v1/me/podcast-episodes/{id}/live/cancel",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastLiveHandler.CancelLiveEpisode,
				),
			),
		),
	)

	mux.Handle(
		"POST /api/v1/podcast-live-sessions/{id}/host-token",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastLiveHandler.GetHostToken,
				),
			),
		),
	)

	// Recording/replay handoff: publishing the finished
	// recording turns the ENDED episode into a normal
	// on-demand episode. Owner confirmation is required, so
	// this lives with the creator routes.
	mux.Handle(
		"POST /api/v1/me/podcast-live-sessions/{id}/publish-recording",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireVerifiedUser(
				userRepo,
				http.HandlerFunc(
					podcastLiveHandler.PublishLiveRecording,
				),
			),
		),
	)

	// =========================
	// PODCAST LIVE (public)
	// =========================
	//
	// Discovery and listener tokens work without an account.
	// The listener-token route parses an optional Bearer
	// header itself so signed-in users get stable identities
	// while anonymous listeners stay fully anonymous.

	mux.HandleFunc(
		"POST /api/v1/podcast-live-sessions/{id}/listener-token",
		podcastLiveHandler.GetListenerToken,
	)

	mux.HandleFunc(
		"GET /api/v1/podcast-live/upcoming",
		podcastLiveHandler.ListUpcomingLive,
	)

	mux.HandleFunc(
		"GET /api/v1/podcast-live/current",
		podcastLiveHandler.ListCurrentlyLive,
	)

	mux.HandleFunc(
		"GET /api/v1/podcast-live/{id}",
		podcastLiveHandler.GetPublicLive,
	)

	// =========================
	// LEGACY PUBLIC MUSIC
	// =========================

	mux.HandleFunc(
		"GET /music",
		musicHandler.GetAllMusic,
	)

	mux.HandleFunc(
		"GET /music/{id}",
		musicHandler.GetMusic,
	)

	// =========================
	// CORS
	// =========================

	corsConfig :=
		middleware.CORSConfig{
			AllowedOrigins: cfg.CORSAllowedOrigins,

			AllowedMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
			},

			AllowedHeaders: []string{
				"Authorization",
				"Content-Type",
				"X-Request-ID",
			},
		}

	// =========================
	// GLOBAL MIDDLEWARE
	// =========================

	var rootHandler http.Handler = mux

	rootHandler = middleware.CORS(
		corsConfig,
		rootHandler,
	)

	rootHandler =
		middleware.RequestLogger(
			rootHandler,
		)

	// =========================
	// HTTP SERVER
	// =========================

	server := &http.Server{
		Addr: ":" + cfg.Port,

		Handler: rootHandler,

		ReadTimeout: cfg.ReadTimeout,

		WriteTimeout: cfg.WriteTimeout,

		IdleTimeout: cfg.IdleTimeout,
	}

	// =========================
	// START SERVER
	// =========================

	serverErrors := make(
		chan error,
		1,
	)

	go func() {
		log.Printf(
			"music API listening on port %s",
			cfg.Port,
		)

		serverErrors <- server.ListenAndServe()
	}()

	// =========================
	// SHUTDOWN
	// =========================

	shutdownSignals := make(
		chan os.Signal,
		1,
	)

	signal.Notify(
		shutdownSignals,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	select {
	case err := <-serverErrors:
		if err != nil &&
			!errors.Is(
				err,
				http.ErrServerClosed,
			) {

			log.Fatalf(
				"server error: %v",
				err,
			)
		}

	case sig := <-shutdownSignals:
		log.Printf(
			"shutdown signal received: %s",
			sig,
		)

		shutdownCtx, cancel :=
			context.WithTimeout(
				context.Background(),
				cfg.ShutdownTimeout,
			)
		defer cancel()

		if err :=
			server.Shutdown(
				shutdownCtx,
			); err != nil {

			log.Printf(
				"graceful shutdown failed: %v",
				err,
			)

			if closeErr :=
				server.Close(); closeErr != nil {

				log.Printf(
					"forced server close failed: %v",
					closeErr,
				)
			}
		}
	}

	time.Sleep(
		100 * time.Millisecond,
	)

	log.Println(
		"music API stopped",
	)
}
