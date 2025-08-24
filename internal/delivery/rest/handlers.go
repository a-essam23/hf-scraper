// Path: internal/delivery/rest/handlers.go
package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"hf-scraper/internal/domain"
)

// dataService defines the interface required by the handlers from the core service.
// This keeps the delivery layer decoupled from the full service implementation.
type dataService interface {
	GetModelByID(ctx context.Context, id string) (*domain.HuggingFaceModel, error)
}

// ModelHandlers holds dependencies for model-related HTTP handlers.
type ModelHandlers struct {
	service dataService
}

// NewModelHandlers creates a new handler struct.
func NewModelHandlers(s dataService) *ModelHandlers {
	return &ModelHandlers{service: s}
}

// GetModelByID handles the request for a single model.
// Path: /models/{author}/{modelName}
func (h *ModelHandlers) GetModelByID(w http.ResponseWriter, r *http.Request) {
	// Re-construct the model ID from the path segments.
	// Example: /models/google-bert/bert-base-uncased -> "google-bert/bert-base-uncased"
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/models/"), "/")
	if len(parts) != 2 {
		http.Error(w, "Invalid model ID format. Expected {author}/{modelName}", http.StatusBadRequest)
		return
	}
	modelID := strings.Join(parts, "/")

	model, err := h.service.GetModelByID(r.Context(), modelID)
	if err != nil {
		// Log the internal error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if model == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}