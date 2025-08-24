// Path: cmd/daemon/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hf-scraper/internal/config"
	"hf-scraper/internal/delivery/rest"
	"hf-scraper/internal/events"
	"hf-scraper/internal/scraper"
	"hf-scraper/internal/service"
	"hf-scraper/internal/storage"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Setup Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Initialize Database Connection
	log.Println("Connecting to MongoDB...")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Database.URI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	db := mongoClient.Database(cfg.Database.Name)

	// 4. Initialize Components (Layers 2 & Cross-cutting)
	log.Println("Initializing components...")
	broker := events.NewBroker()
	modelStore := storage.NewMongoModelStorage(db, cfg.Database.Collection)
	statusStore := storage.NewMongoStatusStorage(db, "_status") // Use a dedicated collection
	hfScraper := scraper.NewScraper(cfg.Scraper)

	// 5. Initialize The Engine (Layer 3)
	coreService := service.NewService(cfg.Watcher, *hfScraper, modelStore, statusStore, broker)

	// 6. Start the Engine in the background
	go func() {
		if err := coreService.Start(ctx); err != nil {
			log.Printf("Core service error: %v", err)
			cancel() // Trigger shutdown on critical service error
		}
	}()

	// 7. Initialize and Start The API Server (Layer 4)
	apiServer := rest.NewServer(cfg.Server.Port, coreService)
	go func() {
		log.Printf("API server starting on port %s", cfg.Server.Port)
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	// 8. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received. Shutting down gracefully...")

	// Cancel the main context to signal background processes to stop
	cancel()

	// Give background processes time to stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop the API server
	if err := apiServer.Stop(shutdownCtx); err != nil {
		log.Printf("Error during API server shutdown: %v", err)
	}

	// The core service stops gracefully via the cancelled context.
	coreService.Stop()

	log.Println("Server shut down successfully.")
}