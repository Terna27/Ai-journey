
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"music-api/internal/database"
	"music-api/internal/handler"
	"music-api/internal/middleware"
	"music-api/internal/repository"
	"music-api/internal/services"

	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("error loading .env file")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := database.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// =========================
	// SERVICES
	// =========================

	repo := repository.NewMusicRepository(db.Pool)
	svc := services.NewMusicService(repo)

	artistRepo := repository.NewArtistRepository(db.Pool)
	artistService := services.NewArtistService(artistRepo)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	jwtService := services.NewJWTService(jwtSecret)

	cloudinaryService, err := services.NewCloudinaryService()
	if err != nil {
		log.Fatal(err)
	}

	musicHandler := handler.NewMusicHandler(
		svc,
		artistService,
		jwtService,
		cloudinaryService,
	)

	// =========================
	// ROUTER
	// =========================

	mux := http.NewServeMux()

	// =========================
	// ARTIST AUTHENTICATION
	// =========================

	mux.HandleFunc("POST /artists/register", musicHandler.RegisterArtist)
	mux.HandleFunc("POST /artists/login", musicHandler.LoginArtist)

	// =========================
	// FRONTEND
	// =========================

	staticFS := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, "web/index.html")
	})

	// =========================
	// PUBLIC API ROUTES
	// =========================

	mux.HandleFunc("GET /music", musicHandler.GetAllMusic)
	mux.HandleFunc("GET /music/{id}", musicHandler.GetMusic)

	// =========================
	// PROTECTED MUSIC ROUTES
	// =========================

	protectedMux := http.NewServeMux()

	protectedMux.HandleFunc("POST /music", musicHandler.CreateMusic)
	protectedMux.HandleFunc("PUT /music/{id}", musicHandler.UpdateMusic)
	protectedMux.HandleFunc("PATCH /music/{id}", musicHandler.PatchMusic)
	protectedMux.HandleFunc("DELETE /music/{id}", musicHandler.DeleteMusic)
	protectedMux.HandleFunc("POST /music/{id}/like", musicHandler.LikeMusic)

	protectedHandler := middleware.APIKey(protectedMux)

	mux.Handle("POST /music", protectedHandler)
	mux.Handle("PUT /music/{id}", protectedHandler)
	mux.Handle("PATCH /music/{id}", protectedHandler)
	mux.Handle("DELETE /music/{id}", protectedHandler)
	mux.Handle("POST /music/{id}/like", protectedHandler)

	// =========================
	// REQUEST LOGGING
	// =========================

	handler := middleware.RequestLogger(mux)

	// =========================
	// SERVER
	// =========================

	port := os.Getenv("PORT")
	if port == "" {
		port = "7000"
	}

	log.Printf("Server running on http://localhost:%s", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

