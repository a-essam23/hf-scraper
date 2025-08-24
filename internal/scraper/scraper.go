// Path: internal/scraper/scraper.go
package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"hf-scraper/internal/config"
	"hf-scraper/internal/domain"

	"golang.org/x/time/rate"
)

const hfAPIBase = "https://huggingface.co"

// linkHeaderRegex is used to parse the 'Link' HTTP header for pagination.
var linkHeaderRegex = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// ScrapeResult holds the data returned from a single API call.
type ScrapeResult struct {
	Models  []domain.HuggingFaceModel
	NextURL string
}

// Scraper is a client for the Hugging Face API.
type Scraper struct {
	client  *http.Client
	limiter *rate.Limiter
}

// NewScraper creates and configures a new Scraper.
func NewScraper(cfg config.ScraperConfig) *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: rate.NewLimiter(
			rate.Limit(cfg.RequestsPerSecond),
			cfg.BurstLimit,
		),
	}
}

// FetchModels fetches a single page of models from the given URL.
// It respects the rate limit and parses the 'Link' header for the next page.
func (s *Scraper) FetchModels(ctx context.Context, url string) (*ScrapeResult, error) {
	// ... (rate limiting and request creation are the same)
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var models []domain.HuggingFaceModel
	if err := json.Unmarshal(body, &models); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json response: %w", err)
	}

	// Re-introducing the logic to parse the Link header for the next page URL.
	nextURL := ""
	linkHeader := resp.Header.Get("Link")
	if matches := linkHeaderRegex.FindStringSubmatch(linkHeader); len(matches) > 1 {
		nextURL = matches[1]
	}

	return &ScrapeResult{
		Models:  models,
		NextURL: nextURL,
	}, nil
}
