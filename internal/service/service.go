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
	// Event topics
	EventModeChange = "status:mode_change"
)

// Service is the central orchestrator of the daemon's logic.
type Service struct {
	cfg           config.WatcherConfig
	scraperCfg    config.ScraperConfig // Added for base URL
	scraper       scraper.Scraper
	modelStorage  ModelStorage
	statusStorage StatusStorage
	broker        *events.Broker
}

// NewService creates a new core application service.
func NewService(
	cfg config.WatcherConfig,
	scraperCfg config.ScraperConfig, // Added
	scraper scraper.Scraper,
	modelStorage ModelStorage,
	statusStorage StatusStorage,
	broker *events.Broker,
) *Service {
	return &Service{
		cfg:           cfg,
		scraperCfg:    scraperCfg, // Added
		scraper:       scraper,
		modelStorage:  modelStorage,
		statusStorage: statusStorage,
		broker:        broker,
	}
}

// Start begins the main operational loop of the service.
// It is a long-running, blocking method.
func (s *Service) Start(ctx context.Context) error {
	log.Println("Service starting...")
	statusDoc, err := s.statusStorage.GetStatusDocument(ctx)
	if err != nil {
		return fmt.Errorf("could not determine initial service status: %w", err)
	}

	log.Printf("Initial status is: %s", statusDoc.Status)

	if statusDoc.Status == domain.StatusNeedsBackfill {
		// Pass the cursor to the backfill process.
		err := s.runBackfill(ctx, statusDoc.BackfillCursor)
		if err != nil {
			// If context was cancelled, it's a graceful shutdown, not an error.
			if ctx.Err() == context.Canceled {
				log.Println("Backfill process cancelled gracefully.")
				return nil
			}
			return fmt.Errorf("backfill process failed: %w", err)
		}
	}

	s.startWatcher(ctx)
	return nil
}

// runBackfill executes the one-time, historical data scrape.
func (s *Service) runBackfill(ctx context.Context, initialCursor string) error {
	log.Println("Starting Backfill Mode...")
	backfillStartURL := fmt.Sprintf("%s/api/models?sort=createdAt&direction=1&full=true", s.scraperCfg.BaseURL)

	currentURL := backfillStartURL
	if initialCursor != "" {
		log.Printf("Resuming backfill from saved cursor: %s", initialCursor)
		currentURL = initialCursor
	} else {
		log.Println("Starting a fresh backfill. Saving initial state.")
		if err := s.statusStorage.UpdateStatus(ctx, domain.StatusNeedsBackfill); err != nil {
			log.Printf("Warning: failed to save initial status: %v", err)
		}
	}

	for currentURL != "" {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Printf("Backfill: Fetching %s", currentURL)
			result, err := s.scraper.FetchModels(ctx, currentURL)
			if err != nil {
				log.Printf("Error fetching page, will retry after 10s: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			if len(result.Models) > 0 {
				log.Printf("Backfill: Storing %d models...", len(result.Models))
				if err := s.modelStorage.BulkUpsert(ctx, result.Models); err != nil {
					log.Printf("CRITICAL: FAILED TO BULK UPSERT MODELS. Error: %v", err)
					// We add a small sleep to avoid a rapid failure loop on DB issues.
					time.Sleep(10 * time.Second)
					continue // Retry the same page after a delay
				}
			}

			// *** RESILIENCY FIX ***
			// Update the cursor bookmark ONLY AFTER the page is processed successfully.
			if err := s.statusStorage.UpdateBackfillCursor(ctx, result.NextURL); err != nil {
				log.Printf("CRITICAL: FAILED TO SAVE BACKFILL CURSOR. Error: %v", err)
				time.Sleep(10 * time.Second)
			}

			currentURL = result.NextURL
		}
	}

	log.Println("Backfill Mode completed.")
	log.Println("Updating service status to WATCHING.")
	if err := s.statusStorage.UpdateStatus(ctx, domain.StatusWatching); err != nil {
		return fmt.Errorf("failed to update status to WATCHING after backfill: %w", err)
	}

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
		case <-ctx.Done():
			log.Println("Watch Mode stopped.")
			return
		}
	}
}

// runWatchCycle performs a single check for new or updated models.
func (s *Service) runWatchCycle(ctx context.Context) {
	log.Println("Watch Cycle: Starting check for latest models.")
	watchStartURL := fmt.Sprintf("%s/api/models?sort=lastModified&direction=-1&full=true", s.scraperCfg.BaseURL)

	latestModel, err := s.modelStorage.FindMostRecentlyModified(ctx)
	if err != nil {
		log.Printf("Watch Cycle Error: could not get latest model from DB: %v", err)
		return
	}

	latestKnownUpdate := time.Time{}
	if latestModel != nil {
		latestKnownUpdate = latestModel.LastModified
		log.Printf("Watch Cycle: Latest known update timestamp is %s (from model %s)", latestKnownUpdate.Format(time.RFC3339), latestModel.ID)
	} else {
		log.Println("Watch Cycle: No existing models found. Will fetch all new models.")
	}

	result, err := s.scraper.FetchModels(ctx, watchStartURL)
	if err != nil {
		log.Printf("Watch Cycle Error: failed to fetch from API: %v", err)
		return
	}

	modelsToUpdate := make([]domain.HuggingFaceModel, 0)
	for _, model := range result.Models {
		if model.LastModified.After(latestKnownUpdate) {
			modelsToUpdate = append(modelsToUpdate, model)
		} else {
			log.Println("Watch Cycle: Reached a model that is not new. Stopping check.")
			break
		}
	}

	if len(modelsToUpdate) > 0 {
		log.Printf("Watch Cycle: Found %d new/updated models. Storing...", len(modelsToUpdate))
		if err := s.modelStorage.BulkUpsert(ctx, modelsToUpdate); err != nil {
			log.Printf("Watch Cycle Error: failed to bulk upsert models: %v", err)
		} else {
			log.Printf("Watch Cycle: Finished. Stored %d models.", len(modelsToUpdate))
		}
	} else {
		log.Printf("Watch Cycle: Finished. No new updates found.")
	}
}

// GetModelByID provides a simple data-retrieval method for the Delivery Layer.
func (s *Service) GetModelByID(ctx context.Context, id string) (*domain.HuggingFaceModel, error) {
	return s.modelStorage.FindByID(ctx, id)
}
