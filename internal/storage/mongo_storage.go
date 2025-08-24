// Path: internal/storage/mongo_storage.go
package storage

import (
	"context"
	"errors"

	"hf-scraper/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoModelStorage is the MongoDB implementation of the ModelStorage interface.
type MongoModelStorage struct {
	collection *mongo.Collection
}

// NewMongoModelStorage creates a new storage adapter for models.
func NewMongoModelStorage(db *mongo.Database, collectionName string) *MongoModelStorage {
	return &MongoModelStorage{
		collection: db.Collection(collectionName),
	}
}

// Upsert implements the ModelStorage interface.
func (s *MongoModelStorage) Upsert(ctx context.Context, model domain.HuggingFaceModel) error {
	opts := options.Replace().SetUpsert(true)
	filter := bson.M{"_id": model.ID}
	_, err := s.collection.ReplaceOne(ctx, filter, model, opts)
	return err
}

// BulkUpsert implements the ModelStorage interface.
func (s *MongoModelStorage) BulkUpsert(ctx context.Context, models []domain.HuggingFaceModel) error {
	if len(models) == 0 {
		return nil
	}

	writeModels := make([]mongo.WriteModel, len(models))
	for i, model := range models {
		filter := bson.M{"_id": model.ID}
		replacement := model
		writeModels[i] = mongo.NewReplaceOneModel().SetFilter(filter).SetReplacement(replacement).SetUpsert(true)
	}

	// SetOrdered(false) allows MongoDB to process the operations in parallel, which is faster.
	opts := options.BulkWrite().SetOrdered(false)
	_, err := s.collection.BulkWrite(ctx, writeModels, opts)
	return err
}

// FindByID implements the ModelStorage interface.
func (s *MongoModelStorage) FindByID(ctx context.Context, id string) (*domain.HuggingFaceModel, error) {
	var model domain.HuggingFaceModel
	filter := bson.M{"_id": id}
	err := s.collection.FindOne(ctx, filter).Decode(&model)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // Return nil, nil if not found
		}
		return nil, err
	}
	return &model, nil
}

// FindMostRecentlyModified implements the ModelStorage interface.
func (s *MongoModelStorage) FindMostRecentlyModified(ctx context.Context) (*domain.HuggingFaceModel, error) {
	var model domain.HuggingFaceModel
	opts := options.FindOne().SetSort(bson.D{{Key: "lastModified", Value: -1}})
	err := s.collection.FindOne(ctx, bson.D{}, opts).Decode(&model)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // No models in DB yet
		}
		return nil, err
	}
	return &model, nil
}
