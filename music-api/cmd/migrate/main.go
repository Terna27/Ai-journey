package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"music-api/internal/database"

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

	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		log.Fatalf("failed to find migrations: %v", err)
	}

	sort.Strings(files)

	if len(files) == 0 {
		log.Fatal("no migration files found")
	}

	for _, file := range files {
		log.Printf("applying migration: %s", file)

		data, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read migration %s: %v", file, err)
		}

		sql := strings.TrimSpace(string(data))
		if sql == "" {
			log.Printf("skipping empty migration: %s", file)
			continue
		}

		if _, err := db.Pool.Exec(context.Background(), sql); err != nil {
			log.Fatalf("migration failed (%s): %v", file, err)
		}

		log.Printf("migration applied: %s", file)
	}

	log.Println("all migrations applied successfully")
}