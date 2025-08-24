// Path: internal/domain/models.go
package domain

import "time"

// ServiceStatus represents the operational state of the daemon.
type ServiceStatus string

const (
	// StatusNeedsBackfill indicates the service needs to perform the initial historical scrape.
	StatusNeedsBackfill ServiceStatus = "NEEDS_BACKFILL"
	// StatusWatching indicates the service is in its normal, continuous update-watching mode.
	StatusWatching ServiceStatus = "WATCHING"
)

// HuggingFaceModel represents the metadata for a single model from the Hugging Face Hub.
// It includes struct tags for JSON serialization and BSON mapping for MongoDB.
type HuggingFaceModel struct {
	ID           string    `json:"id" bson:"_id"`
	Author       string    `json:"author" bson:"author"`
	SHA          string    `json:"sha" bson:"sha"`
	LastModified time.Time `json:"lastModified" bson:"lastModified"`
	CreatedAt    time.Time `json:"hf_createdAt" bson:"hf_createdAt"`
	Private      bool      `json:"private" bson:"private"`
	Gated        bool      `json:"gated" bson:"gated"`
	Likes        int       `json:"likes" bson:"likes"`
	Downloads    int       `json:"downloads" bson:"downloads"`
	Tags         []string  `json:"tags" bson:"tags"`
	PipelineTag  string    `json:"pipeline_tag" bson:"pipeline_tag"`
}

// StatusDocument represents the state of the service, stored in the database.
// This allows the daemon to be stateful and resilient across restarts.
type StatusDocument struct {
	ID        string        `bson:"_id"` // A constant key, e.g., "service_status"
	Status    ServiceStatus `bson:"status"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}