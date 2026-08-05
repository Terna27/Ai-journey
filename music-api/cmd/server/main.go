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

	repo := repository.NewMusicRepository(db.Pool)

	musicHandler := handler.NewMusicHandler(repo)

	mux := http.NewServeMux()

	handler := middleware.APIKey(mux)

	mux.HandleFunc("POST /music", musicHandler.CreateMusic)
	mux.HandleFunc("GET /music", musicHandler.GetAllMusic)
	mux.HandleFunc("GET /music/{id}", musicHandler.GetMusic)
	mux.HandleFunc("PUT /music/{id}", musicHandler.UpdateMusic)
	mux.HandleFunc("PATCH /music/{id}", musicHandler.PatchMusic)
	mux.HandleFunc("DELETE /music/{id}", musicHandler.DeleteMusic)
	mux.HandleFunc("POST /music/{id}/like", musicHandler.LikeMusic)

	port := os.Getenv("PORT")
	if port == "" {
		port = "7000"
	}

	log.Printf("Server running on http://localhost:%s", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
