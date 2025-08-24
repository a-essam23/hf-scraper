// Path: internal/domain/models.go
package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// --- Enum and Custom Type for the "gated" field ---

// GatedStatus represents the possible states for a gated model.
type GatedStatus string

const (
	GatedStatusTrue   GatedStatus = "true"
	GatedStatusFalse  GatedStatus = "false"
	GatedStatusAuto   GatedStatus = "auto"
	GatedStatusManual GatedStatus = "manual"
)

// UnmarshalJSON implements the json.Unmarshaler interface for GatedStatus.
// It can handle JSON booleans (true/false) or JSON strings ("true", "false", "auto","auto").
func (gs *GatedStatus) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal as a string.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch GatedStatus(s) {
		case GatedStatusTrue, GatedStatusFalse, GatedStatusAuto, GatedStatusManual:
			*gs = GatedStatus(s)
			return nil
		default:
			return fmt.Errorf("unknown GatedStatus string: %s", s)
		}
	}

	// If it's not a string, try to unmarshal as a boolean.
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if b {
			*gs = GatedStatusTrue
		} else {
			*gs = GatedStatusFalse
		}
		return nil
	}

	return fmt.Errorf("gated field is not a recognizable string or boolean")
}

// --- Custom Type for the "private" field ---

// FlexibleBool is a custom boolean type that can be unmarshaled from
// a JSON boolean (true/false) or a JSON string ("true"/"false").
type FlexibleBool bool

// UnmarshalJSON implements the json.Unmarshaler interface for FlexibleBool.
func (fb *FlexibleBool) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*fb = FlexibleBool(b)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsedBool, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*fb = FlexibleBool(parsedBool)
	return nil
}

// ServiceStatus represents the operational state of the daemon.
type ServiceStatus string

type Sibling struct {
	Rfilename string `json:"rfilename" bson:"rfilename"`
}

const (
	// StatusNeedsBackfill indicates the service needs to perform the initial historical scrape.
	StatusNeedsBackfill ServiceStatus = "NEEDS_BACKFILL"
	// StatusWatching indicates the service is in its normal, continuous update-watching mode.
	StatusWatching ServiceStatus = "WATCHING"
)

// HuggingFaceModel represents the metadata for a single model from the Hugging Face Hub.
// It includes struct tags for JSON serialization and BSON mapping for MongoDB.
type HuggingFaceModel struct {
	ID           string       `json:"id" bson:"_id"`
	Author       string       `json:"author" bson:"author"`
	SHA          string       `json:"sha" bson:"sha"`
	LastModified time.Time    `json:"lastModified" bson:"lastModified"`
	CreatedAt    time.Time    `json:"createdAt" bson:"createdAt"`
	Private      FlexibleBool `json:"private" bson:"private"`
	Gated        GatedStatus  `json:"gated" bson:"gated"`
	Likes        int          `json:"likes" bson:"likes"`
	Downloads    int          `json:"downloads" bson:"downloads"`
	Tags         []string     `json:"tags" bson:"tags"`
	PipelineTag  string       `json:"pipeline_tag" bson:"pipeline_tag"`
	Siblings     []Sibling    `json:"siblings" bson:"siblings"`
}

// StatusDocument represents the state of the service, stored in the database.
// This allows the daemon to be stateful and resilient across restarts.
type StatusDocument struct {
	ID        string        `bson:"_id"` // A constant key, e.g., "service_status"
	Status    ServiceStatus `bson:"status"`
	UpdatedAt time.Time     `bson:"updatedAt"`
	// BackfillCursor stores the 'NextURL' to resume scraping from.
	BackfillCursor string `bson:"backfillCursor,omitempty"`
}
