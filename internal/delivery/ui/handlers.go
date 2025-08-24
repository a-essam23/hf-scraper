// path: internal/delivery/ui/handlers.go
package ui

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"hf-scraper/internal/domain"
	"hf-scraper/internal/service"
)

// dataService defines the interface required by the UI handlers.
type dataService interface {
	GetModelByID(ctx context.Context, id string) (*domain.HuggingFaceModel, error)
	SearchModels(ctx context.Context, opts service.SearchOptions) ([]domain.HuggingFaceModel, int64, error)
}

// Handlers holds dependencies for UI handlers.
type Handlers struct {
	service   dataService
	templates *template.Template
}

// NewHandlers creates a new UI handler struct.
func NewHandlers(s dataService) *Handlers {
	tpl := template.Must(template.ParseGlob("web/template/*.html"))
	tpl = template.Must(tpl.ParseGlob("web/template/fragments/*.html"))
	// Debug: Print all template names
	fmt.Println("Loaded templates:")
	for _, t := range tpl.Templates() {
		fmt.Printf("  - %s\n", t.Name())
	}

	return &Handlers{
		service:   s,
		templates: tpl,
	}
}

// RegisterRoutes registers all UI routes on the given ServeMux.
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	// Register most specific routes first.

	// 1. Static files: Handles "/static/..."
	fileServer := http.FileServer(http.Dir("./web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	// 2. API-like endpoints for HTMX
	mux.HandleFunc("/search", h.handleSearch)

	// 3. Model detail pages: Handles "/model/author/name"
	mux.HandleFunc("/models/", h.handleShowModel)

	// 4. Root/Index page: This is the catch-all and MUST be last.
	mux.HandleFunc("/", h.handleShowIndex)
}

// handleShowIndex serves the main search page.
func (h *Handlers) handleShowIndex(w http.ResponseWriter, r *http.Request) {
	// This ensures that only the exact path "/" is handled here.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	models, total, _ := h.service.SearchModels(r.Context(), service.SearchOptions{
		Page:      1,
		Limit:     20,
		SortBy:    "likes",
		SortOrder: -1,
	})

	data := h.buildTemplateData(r, models, total)
	fmt.Printf("Executing template: index.html\n")
	fmt.Printf("Data keys: %v\n", reflect.ValueOf(data).MapKeys())

	err := h.templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		fmt.Printf("Template execution error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleSearch is an HTMX endpoint that returns the search results.
func (h *Handlers) handleSearch(w http.ResponseWriter, r *http.Request) {
	// *** FIX 2: Correctly render a single response for HTMX ***
	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	if page == 0 {
		page = 1
	}

	opts := service.SearchOptions{
		Query:     r.URL.Query().Get("q"),
		SortBy:    r.URL.Query().Get("sort"),
		SortOrder: -1, // Default desc
		Page:      page,
		Limit:     20,
	}
	if r.URL.Query().Get("order") == "1" {
		opts.SortOrder = 1
	}

	models, total, err := h.service.SearchModels(r.Context(), opts)
	if err != nil {
		log.Printf("Error searching models: %v", err)
		http.Error(w, "Failed to search models", http.StatusInternalServerError)
		return
	}

	data := h.buildTemplateData(r, models, total)
	// Render the new wrapper template which contains both the table and pagination.
	h.templates.ExecuteTemplate(w, "search_results.html", data)
}

// handleShowModel serves the model details page.
func (h *Handlers) handleShowModel(w http.ResponseWriter, r *http.Request) {
	modelID := strings.TrimPrefix(r.URL.Path, "/models/")
	model, err := h.service.GetModelByID(r.Context(), modelID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if model == nil {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Model": model,
	}
	h.templates.ExecuteTemplate(w, "model.html", data)
}

// buildTemplateData is a helper to construct the data map for templates.
func (h *Handlers) buildTemplateData(r *http.Request, models []domain.HuggingFaceModel, total int64) map[string]interface{} {
	const pageSize = 20
	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	if page == 0 {
		page = 1
	}
	sortOrder, _ := strconv.Atoi(r.URL.Query().Get("order"))
	if sortOrder == 0 {
		sortOrder = -1 // Default to descending if not specified
	}

	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "likes" // Default sort
	}

	return map[string]any{
		"Models":      models,
		"Query":       r.URL.Query().Get("q"),
		"SortBy":      sortBy,
		"SortOrder":   sortOrder,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int64(math.Ceil(float64(total) / float64(pageSize))),
		"NextPage":    page + 1,
		"PrevPage":    page - 1,
	}
}
