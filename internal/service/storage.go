// Path: internal/service/storage.go
package service

import (
	"context"

	"hf-scraper/internal/domain"
)

// SearchOptions holds parameters for searching and sorting models.
type SearchOptions struct {
	Query     string
	SortBy    string // e.g., "likes", "downloads", "lastModified"
	SortOrder int    // 1 for ascending, -1 for descending
	Limit     int64
	Page      int64
}

// ModelStorage defines the interface for persisting HuggingFaceModel data.
type ModelStorage interface {
	// Upsert inserts a new model or updates an existing one, identified by its ID.
	Upsert(ctx context.Context, model domain.HuggingFaceModel) error

	// BulkUpsert efficiently inserts or updates multiple models in a single operation.
	BulkUpsert(ctx context.Context, models []domain.HuggingFaceModel) error

	// FindByID retrieves a single model by its unique ID.
	FindByID(ctx context.Context, id string) (*domain.HuggingFaceModel, error)

	// FindMostRecentlyModified finds the model with the latest `lastModified` timestamp.
	// This is crucial for the "Watch Mode" logic.
	FindMostRecentlyModified(ctx context.Context) (*domain.HuggingFaceModel, error)

	SearchModels(ctx context.Context, opts SearchOptions) ([]domain.HuggingFaceModel, int64, error)
}

// StatusStorage defines the interface for persisting the service's operational state.
type StatusStorage interface {
	GetStatusDocument(ctx context.Context) (*domain.StatusDocument, error)
	UpdateStatus(ctx context.Context, status domain.ServiceStatus) error
	UpdateBackfillCursor(ctx context.Context, cursorURL string) error
}
