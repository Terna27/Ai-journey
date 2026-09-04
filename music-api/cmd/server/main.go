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
		log.Fatalf("failed to load configuration: %v", err)
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
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// =========================
	// REPOSITORIES
	// =========================

	musicRepo := repository.NewMusicRepository(
		db.Pool,
	)

	artistRepo := repository.NewArtistRepository(
		db.Pool,
	)

	userRepo := repository.NewUserRepository(
		db.Pool,
	)

	// =========================
	// SERVICES
	// =========================

	musicService := services.NewMusicService(
		musicRepo,
	)

	artistService := services.NewArtistService(
		artistRepo,
	)

	userService := services.NewUserService(
		userRepo,
	)

	jwtService := services.NewJWTService(
		cfg.JWTSecret,
	)

	cloudinaryService, err := services.NewCloudinaryService()
	if err != nil {
		log.Fatalf(
			"failed to initialize Cloudinary: %v",
			err,
		)
	}

	// =========================
	// HANDLERS
	// =========================

	musicHandler := handler.NewMusicHandler(
		musicService,
		artistService,
		userService,
		jwtService,
		cloudinaryService,
	)

	healthHandler := handler.NewHealthHandler(
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
	// API V1 - AUTH
	// =========================
	//
	// Unified authentication.
	//
	// Every person registers as a user first.
	//

	mux.HandleFunc(
		"POST /api/v1/auth/register",
		musicHandler.RegisterUser,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		musicHandler.LoginUser,
	)

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
	// API V1 - ARTIST PROFILE
	// =========================
	//
	// Authenticated users can upgrade their existing
	// account by creating an artist profile.
	//
	// RequireArtist is intentionally NOT used here.
	//

	mux.Handle(
		"POST /api/v1/artists/profile",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(
				musicHandler.CreateArtistProfile,
			),
		),
	)

	// =========================
	// API V1 - PUBLIC MUSIC
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
	// API V1 - CREATE MUSIC
	// =========================
	//
	// Requires:
	//
	// 1. Valid user JWT
	// 2. Artist profile
	//

	mux.Handle(
		"POST /api/v1/music",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireArtist(
				artistRepo,
				http.HandlerFunc(
					musicHandler.CreateMusic,
				),
			),
		),
	)

	// =========================
	// API V1 - UPDATE MUSIC
	// =========================

	mux.Handle(
		"PUT /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireArtist(
				artistRepo,
				http.HandlerFunc(
					musicHandler.UpdateMusic,
				),
			),
		),
	)

	// =========================
	// API V1 - PATCH MUSIC
	// =========================

	mux.Handle(
		"PATCH /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireArtist(
				artistRepo,
				http.HandlerFunc(
					musicHandler.PatchMusic,
				),
			),
		),
	)

	// =========================
	// API V1 - DELETE MUSIC
	// =========================

	mux.Handle(
		"DELETE /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			middleware.RequireArtist(
				artistRepo,
				http.HandlerFunc(
					musicHandler.DeleteMusic,
				),
			),
		),
	)

	// =========================
	// API V1 - LIKE MUSIC
	// =========================
	//
	// Any authenticated user should eventually be
	// able to like music.
	//
	// Artist capability is NOT required.
	//

	mux.Handle(
		"POST /api/v1/music/{id}/like",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(
				musicHandler.LikeMusic,
			),
		),
	)

	// =========================
	// LEGACY PUBLIC MUSIC
	// =========================
	//
	// Retained temporarily for compatibility.
	//

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

	corsConfig := middleware.CORSConfig{
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

	rootHandler = middleware.RequestLogger(
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
	// SHUTDOWN SIGNALS
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

		shutdownCtx, cancel := context.WithTimeout(
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

			if closeErr := server.Close(); closeErr != nil {
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
