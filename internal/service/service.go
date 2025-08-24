// Path: internal/service/service.go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"hf-scraper/internal/config"
	"hf-scraper/internal/domain"
	"hf-scraper/internal/events"
	"hf-scraper/internal/scraper"
)

const (
	// The initial URL for the backfill process.
	backfillStartURL = "https://huggingface.co/api/models?sort=createdAt&full=true&page=1&limit=100"
	// The URL for the watch mode process.
	watchStartURL = "https://huggingface.co/api/models?sort=lastModified&page=1&limit=100&full=true"

	// Event topics
	EventModeChange = "status:mode_change"
)

// Scraper defines the interface for a component that can fetch models.
// This allows for mocking in tests.
// type Scraper interface {
// 	FetchModels(ctx context.Context, url string) (*ScrapeResult, error)
// }

// Service is the central orchestrator of the daemon's logic.
type Service struct {
	cfg           config.WatcherConfig
	scraper       scraper.Scraper
	modelStorage  ModelStorage
	statusStorage StatusStorage
	broker        *events.Broker
	stopChan      chan struct{} // Used for graceful shutdown
}

// NewService creates a new core application service.
func NewService(
	cfg config.WatcherConfig,
	scraper scraper.Scraper,
	modelStorage ModelStorage,
	statusStorage StatusStorage,
	broker *events.Broker,
) *Service {
	return &Service{
		cfg:           cfg,
		scraper:       scraper,
		modelStorage:  modelStorage,
		statusStorage: statusStorage,
		broker:        broker,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the main operational loop of the service.
// It is a long-running, blocking method.
func (s *Service) Start(ctx context.Context) error {
	log.Println("Service starting...")
	initialStatus, err := s.statusStorage.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("could not determine initial service status: %w", err)
	}

	log.Printf("Initial status is: %s", initialStatus)

	if initialStatus == domain.StatusNeedsBackfill {
		err := s.runBackfill(ctx)
		if err != nil {
			return fmt.Errorf("backfill process failed: %w", err)
		}
		// After backfill, the status is updated, so we proceed directly to watching.
	}

	s.startWatcher(ctx)
	return nil
}

// Stop gracefully shuts down the service's background processes.
func (s *Service) Stop() {
	log.Println("Service stopping...")
	close(s.stopChan)
}

// runBackfill executes the one-time, historical data scrape.
func (s *Service) runBackfill(ctx context.Context) error {
	log.Println("Starting Backfill Mode...")
	page := 0 // Start with the first page

	for { // Loop indefinitely until we get an empty page
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Construct the URL with the current page number
			currentURL := fmt.Sprintf("%s&page=%d", backfillStartURL, page)

			log.Printf("Backfill: Fetching page %d (%s)", page, currentURL)
			result, err := s.scraper.FetchModels(ctx, currentURL)
			if err != nil {
				log.Printf("Error fetching page, will retry after 10s: %v", err)
				time.Sleep(10 * time.Second) // Simple retry logic
				continue
			}

			// The stop condition: if a page has no models, we're done.
			if len(result.Models) == 0 {
				log.Println("Backfill: Received an empty page. Assuming end of results.")
				goto completion // Break out of the outer loop
			}

			log.Printf("Backfill: Storing %d models...", len(result.Models))
			for _, model := range result.Models {
				if err := s.modelStorage.Upsert(ctx, model); err != nil {
					log.Printf("Warning: failed to upsert model %s: %v", model.ID, err)
				}
			}

			page++ // Move to the next page
		}
	}

completion:
	log.Println("Backfill Mode completed.")
	log.Println("Updating service status to WATCHING.")
	if err := s.statusStorage.SetStatus(ctx, domain.StatusWatching); err != nil {
		return fmt.Errorf("failed to update status to WATCHING after backfill: %w", err)
	}

	// Announce the mode change to the rest of the application.
	s.broker.Publish(EventModeChange, domain.StatusWatching)

	return nil
}

// startWatcher begins the permanent, periodic watch for updates.
func (s *Service) startWatcher(ctx context.Context) {
	log.Printf("Starting Watch Mode. Checking for updates every %d minutes.", s.cfg.IntervalMinutes)
	ticker := time.NewTicker(time.Duration(s.cfg.IntervalMinutes) * time.Minute)
	defer ticker.Stop()

	// Run the first cycle immediately on startup.
	s.runWatchCycle(ctx)

	for {
		select {
		case <-ticker.C:
			s.runWatchCycle(ctx)
		case <-s.stopChan:
			log.Println("Watch Mode stopped.")
			return
		case <-ctx.Done():
			log.Println("Watch Mode context cancelled.")
			return
		}
	}
}

// runWatchCycle performs a single check for new or updated models.
func (s *Service) runWatchCycle(ctx context.Context) {
	log.Println("Watch Cycle: Starting check for latest models.")

	// 1. Establish the benchmark from our own database.
	latestModel, err := s.modelStorage.FindMostRecentlyModified(ctx)
	if err != nil {
		log.Printf("Watch Cycle Error: could not get latest model from DB: %v", err)
		return
	}
	// If the DB is empty, use a zero time. Any model will be newer.
	latestKnownUpdate := time.Time{}
	if latestModel != nil {
		latestKnownUpdate = latestModel.LastModified
		log.Printf("Watch Cycle: Latest known update timestamp is %s (from model %s)", latestKnownUpdate.Format(time.RFC3339), latestModel.ID)
	} else {
		log.Println("Watch Cycle: No existing models found. Will fetch all new models.")
	}

	// 2. Fetch the first page of the latest models from the API.
	result, err := s.scraper.FetchModels(ctx, watchStartURL)
	if err != nil {
		log.Printf("Watch Cycle Error: failed to fetch from API: %v", err)
		return
	}

	// 3. Iterate, compare, and stop when we see a model that is not new.
	updateCount := 0
	for _, model := range result.Models {
		if model.LastModified.After(latestKnownUpdate) {
			if err := s.modelStorage.Upsert(ctx, model); err != nil {
				log.Printf("Watch Cycle Warning: failed to upsert model %s: %v", model.ID, err)
				continue
			}
			updateCount++
		} else {
			// This is the key to efficiency: stop as soon as we see a model we already know about.
			log.Println("Watch Cycle: Reached a model that is not new. Stopping check.")
			break
		}
	}

	if updateCount > 0 {
		log.Printf("Watch Cycle: Finished. Upserted %d new or updated models.", updateCount)
	} else {
		log.Printf("Watch Cycle: Finished. No new updates found.")
	}
}

// GetModelByID provides a simple data-retrieval method for the Delivery Layer.
func (s *Service) GetModelByID(ctx context.Context, id string) (*domain.HuggingFaceModel, error) {
	return s.modelStorage.FindByID(ctx, id)
}