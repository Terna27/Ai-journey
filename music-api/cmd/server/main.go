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

	emailVerificationRepo :=
		repository.NewEmailVerificationRepository(
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

	artistService :=
		services.NewArtistService(
			artistRepo,
		)

	userService :=
		services.NewUserService(
			userRepo,
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

		if err := server.Shutdown(
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
