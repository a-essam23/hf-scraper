// Path: internal/service/storage.go
package service

import (
	"context"

	"hf-scraper/internal/domain"
)

// ModelStorage defines the interface for persisting HuggingFaceModel data.
type ModelStorage interface {
	// Upsert inserts a new model or updates an existing one, identified by its ID.
	Upsert(ctx context.Context, model domain.HuggingFaceModel) error

	// FindByID retrieves a single model by its unique ID.
	FindByID(ctx context.Context, id string) (*domain.HuggingFaceModel, error)

	// FindMostRecentlyModified finds the model with the latest `lastModified` timestamp.
	// This is crucial for the "Watch Mode" logic.
	FindMostRecentlyModified(ctx context.Context) (*domain.HuggingFaceModel, error)
}

// StatusStorage defines the interface for persisting the service's operational state.
type StatusStorage interface {
	GetStatusDocument(ctx context.Context) (*domain.StatusDocument, error)
	UpdateStatus(ctx context.Context, status domain.ServiceStatus) error
	UpdateBackfillCursor(ctx context.Context, cursorURL string) error
}
