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
	"hf-scraper/internal/delivery/ui"
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

	// 4. Initialize Components
	log.Println("Initializing components...")
	broker := events.NewBroker()
	modelStore := storage.NewMongoModelStorage(db, cfg.Database.Collection)
	statusStore := storage.NewMongoStatusStorage(db, cfg.Database.StatusCollection)
	hfScraper := scraper.NewScraper(cfg.Scraper)
	coreService := service.NewService(cfg.Watcher, cfg.Scraper, *hfScraper, modelStore, statusStore, broker)

	// 5. Initialize and Start The Server (API and UI)
	uiHandlers := ui.NewHandlers(coreService)
	mux := http.NewServeMux()
	uiHandlers.RegisterRoutes(mux) // Register all UI routes and static files

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 6. Start the Engine
	go func() {
		if err := coreService.Start(ctx); err != nil {
			log.Printf("Core service error: %v", err)
			cancel()
		}
	}()

	// 7. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received. Shutting down gracefully...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("Server shut down successfully.")
}
