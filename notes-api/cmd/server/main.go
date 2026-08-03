package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"notes-Api/internal/database"
	"notes-Api/internal/handler"
	"notes-Api/internal/repository"

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

	repo := repository.NewNoteRepository(db.Pool)

	noteHandler := handler.NewNoteHandler(repo)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /notes", noteHandler.CreateNote)
	mux.HandleFunc("GET /notes", noteHandler.GetAllNotes)
	mux.HandleFunc("GET /notes/{id}", noteHandler.GetNote)
	mux.HandleFunc("PUT /notes/{id}", noteHandler.UpdateNote)
	mux.HandleFunc("PATCH /notes/{id}", noteHandler.PatchNote)
	mux.HandleFunc("DELETE /notes/{id}", noteHandler.DeleteNote)

	port := os.Getenv("PORT")
	if port == "" {
		port = "7000"
	}

	log.Printf("Server running on http://localhost:%s", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
