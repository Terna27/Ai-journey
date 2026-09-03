package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"music-api/internal/config"
	"music-api/internal/database"
	"music-api/internal/handler"
	"music-api/internal/middleware"
	"music-api/internal/repository"
	"music-api/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	// =========================
	// ENVIRONMENT
	// =========================

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: could not load .env file: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// =========================
	// DATABASE
	// =========================

	db, err := database.Connect(
		context.Background(),
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// =========================
	// SERVICES
	// =========================

	musicRepo := repository.NewMusicRepository(db.Pool)
	musicService := services.NewMusicService(musicRepo)

	artistRepo := repository.NewArtistRepository(db.Pool)
	artistService := services.NewArtistService(artistRepo)

	jwtService := services.NewJWTService(cfg.JWTSecret)

	cloudinaryService, err := services.NewCloudinaryService()
	if err != nil {
		log.Fatal(err)
	}

	// =========================
	// HANDLERS
	// =========================

	musicHandler := handler.NewMusicHandler(
		musicService,
		artistService,
		jwtService,
		cloudinaryService,
	)

	healthHandler := handler.NewHealthHandler(db.Pool)

	// =========================
	// ROUTER
	// =========================

	mux := http.NewServeMux()

	// =========================
	// HEALTH CHECKS
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
	// API V1 - ARTIST AUTH
	// =========================

	mux.Handle(
		"POST /api/v1/artists/register",
		middleware.LimitJSONBody(
			cfg.MaxJSONBodyBytes,
			http.HandlerFunc(musicHandler.RegisterArtist),
		),
	)

	mux.Handle(
		"POST /api/v1/artists/login",
		middleware.LimitJSONBody(
			cfg.MaxJSONBodyBytes,
			http.HandlerFunc(musicHandler.LoginArtist),
		),
	)

	// =========================
	// LEGACY ARTIST ROUTES
	// Temporary compatibility
	// =========================

	mux.Handle(
		"POST /artists/register",
		middleware.LimitJSONBody(
			cfg.MaxJSONBodyBytes,
			http.HandlerFunc(musicHandler.RegisterArtist),
		),
	)

	mux.Handle(
		"POST /artists/login",
		middleware.LimitJSONBody(
			cfg.MaxJSONBodyBytes,
			http.HandlerFunc(musicHandler.LoginArtist),
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
	// LEGACY PUBLIC MUSIC
	// Temporary compatibility
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
	// API V1 - PROTECTED MUSIC
	// JWT AUTHENTICATION
	// =========================

	mux.Handle(
		"POST /api/v1/music",
		middleware.LimitBody(
			cfg.MaxUploadBodyBytes,
			middleware.JWTAuth(
				jwtService,
				http.HandlerFunc(
					musicHandler.CreateMusic,
				),
			),
		),
	)

	mux.Handle(
		"PUT /api/v1/music/{id}",
		middleware.LimitJSONBody(
			cfg.MaxJSONBodyBytes,
			middleware.JWTAuth(
				jwtService,
				http.HandlerFunc(
					musicHandler.UpdateMusic,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/music/{id}",
		middleware.LimitJSONBody(
			cfg.MaxJSONBodyBytes,
			middleware.JWTAuth(
				jwtService,
				http.HandlerFunc(
					musicHandler.PatchMusic,
				),
			),
		),
	)

	mux.Handle(
		"DELETE /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(
				musicHandler.DeleteMusic,
			),
		),
	)

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
	// LEGACY PROTECTED MUSIC
	// JWT AUTHENTICATION
	//
	// Temporary compatibility.
	// These routes now follow the same
	// authentication model as API V1.
	// =========================

	mux.Handle(
		"POST /music",
		middleware.LimitBody(
			cfg.MaxUploadBodyBytes,
			middleware.JWTAuth(
				jwtService,
				http.HandlerFunc(
					musicHandler.CreateMusic,
				),
			),
		),
	)

	mux.Handle(
		"PUT /music/{id}",
		middleware.LimitJSONBody(
			cfg.MaxJSONBodyBytes,
			middleware.JWTAuth(
				jwtService,
				http.HandlerFunc(
					musicHandler.UpdateMusic,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /music/{id}",
		middleware.LimitJSONBody(
			cfg.MaxJSONBodyBytes,
			middleware.JWTAuth(
				jwtService,
				http.HandlerFunc(
					musicHandler.PatchMusic,
				),
			),
		),
	)

	mux.Handle(
		"DELETE /music/{id}",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(
				musicHandler.DeleteMusic,
			),
		),
	)

	mux.Handle(
		"POST /music/{id}/like",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(
				musicHandler.LikeMusic,
			),
		),
	)

	// =========================
	// FRONTEND
	// =========================

	staticFS := http.FileServer(
		http.Dir("web/static"),
	)

	mux.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			staticFS,
		),
	)

	mux.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				handler.WriteError(
					w,
					http.StatusNotFound,
					"NOT_FOUND",
					"resource not found",
				)
				return
			}

			http.ServeFile(
				w,
				r,
				"web/index.html",
			)
		},
	)

	// =========================
	// REQUEST LOGGING
	// =========================

	rootHandler := middleware.RequestLogger(mux)

	// =========================
	// SERVER
	// =========================

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      rootHandler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf(
			"Server running on http://localhost:%s",
			cfg.Port,
		)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// =========================
	// GRACEFUL SHUTDOWN
	// =========================

	shutdownSignal := make(chan os.Signal, 1)

	signal.Notify(
		shutdownSignal,
		os.Interrupt,
		syscall.SIGTERM,
	)

	select {
	case err := <-serverErrors:
		log.Fatalf(
			"server error: %v",
			err,
		)

	case sig := <-shutdownSignal:
		log.Printf(
			"shutdown signal received: %s",
			sig,
		)
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	log.Println("shutting down server...")

	if err := server.Shutdown(shutdownCtx); err != nil {
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

	log.Println("server stopped")
}
