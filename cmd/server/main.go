package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"auction-site-go/internal/domain"
	"auction-site-go/internal/persistence"
	"auction-site-go/internal/web"
)

const eventsFile = "tmp/events.jsonl"

func main() {
	// Ensure directory exists
	dir := filepath.Dir(eventsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	// Read events
	events, err := persistence.ReadEvents(eventsFile)
	if err != nil {
		log.Fatalf("Failed to read events: %v", err)
	}

	// Initialize repository
	repo := domain.EventsToAuctionStates(events)

	// Event handler
	onEvent := func(event domain.Event) error {
		return persistence.WriteEvents(eventsFile, []domain.Event{event})
	}

	// Get current time
	getCurrentTime := time.Now

	// Create web application
	app := web.NewApp(repo, onEvent, getCurrentTime)

	// Start server
	log.Fatal(app.Run(":8080"))
}
