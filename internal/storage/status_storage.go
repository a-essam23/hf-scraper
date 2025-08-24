// Path: internal/storage/status_storage.go
package storage

import (
	"context"
	"errors"
	"time"

	"hf-scraper/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const statusDocumentID = "service_status"

// MongoStatusStorage is the MongoDB implementation of the StatusStorage interface.
type MongoStatusStorage struct {
	collection *mongo.Collection
}

// NewMongoStatusStorage creates a new storage adapter for service status.
func NewMongoStatusStorage(db *mongo.Database, collectionName string) *MongoStatusStorage {
	return &MongoStatusStorage{
		collection: db.Collection(collectionName),
	}
}

// GetStatus implements the StatusStorage interface.
func (s *MongoStatusStorage) GetStatus(ctx context.Context) (domain.ServiceStatus, error) {
	var doc domain.StatusDocument
	filter := bson.M{"_id": statusDocumentID}
	err := s.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		// If no document is found, it means we've never run before.
		// Default to needing a backfill as per the design.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.StatusNeedsBackfill, nil
		}
		return "", err
	}
	return doc.Status, nil
}

// SetStatus implements the StatusStorage interface.
func (s *MongoStatusStorage) SetStatus(ctx context.Context, status domain.ServiceStatus) error {
	doc := domain.StatusDocument{
		ID:        statusDocumentID,
		Status:    status,
		UpdatedAt: time.Now().UTC(),
	}
	opts := options.Replace().SetUpsert(true)
	filter := bson.M{"_id": statusDocumentID}
	_, err := s.collection.ReplaceOne(ctx, filter, doc, opts)
	return err
}