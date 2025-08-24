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
func (s *MongoStatusStorage) GetStatusDocument(ctx context.Context) (*domain.StatusDocument, error) {
	var doc domain.StatusDocument
	filter := bson.M{"_id": statusDocumentID}
	err := s.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Return a default, "first run" document.
			return &domain.StatusDocument{
				ID:     statusDocumentID,
				Status: domain.StatusNeedsBackfill,
			}, nil
		}
		return nil, err
	}
	return &doc, nil
}
func (s *MongoStatusStorage) UpdateStatus(ctx context.Context, status domain.ServiceStatus) error {
	filter := bson.M{"_id": statusDocumentID}
	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now().UTC(),
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := s.collection.UpdateOne(ctx, filter, update, opts)
	return err
}
func (s *MongoStatusStorage) UpdateBackfillCursor(ctx context.Context, cursorURL string) error {
	filter := bson.M{"_id": statusDocumentID}
	update := bson.M{
		"$set": bson.M{
			"backfillCursor": cursorURL,
			"updatedAt":      time.Now().UTC(),
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := s.collection.UpdateOne(ctx, filter, update, opts)
	return err
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
