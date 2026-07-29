package mongodb

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sklinkert/go-ddd/internal/domain/entities"
	"github.com/sklinkert/go-ddd/internal/domain/repositories"
)

type IdempotencyRepository struct {
	db *mongo.Database
}

func NewIdempotencyRepository(db *mongo.Database) repositories.IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

func (repo *IdempotencyRepository) collection() *mongo.Collection {
	return repo.db.Collection(IdempotencyCollection)
}

// Reserve relies on the unique index on `key` (see EnsureIndexes): a
// duplicate-key error means another request already holds the reservation.
// This is a single atomic insert-if-absent, the same guarantee the Postgres
// `INSERT ... ON CONFLICT (key) DO NOTHING` gave, just surfaced as an error
// instead of a zero-rows-affected result.
func (repo *IdempotencyRepository) Reserve(ctx context.Context, record *entities.IdempotencyRecord) (bool, error) {
	doc := idempotencyDoc{
		ID:        record.Id.String(),
		Key:       record.Key,
		Request:   record.Request,
		CreatedAt: record.CreatedAt,
	}

	if _, err := repo.collection().InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (repo *IdempotencyRepository) FindByKey(ctx context.Context, key string) (*entities.IdempotencyRecord, error) {
	var doc idempotencyDoc
	err := repo.collection().FindOne(ctx, bson.D{{Key: "key", Value: key}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	id, err := uuid.Parse(doc.ID)
	if err != nil {
		return nil, err
	}

	return &entities.IdempotencyRecord{
		Id:         id,
		Key:        doc.Key,
		Request:    doc.Request,
		Response:   doc.Response,
		StatusCode: doc.StatusCode,
		CreatedAt:  doc.CreatedAt,
	}, nil
}

func (repo *IdempotencyRepository) SetResponse(ctx context.Context, key string, response string, statusCode int) error {
	_, err := repo.collection().UpdateOne(ctx,
		bson.D{{Key: "key", Value: key}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "response", Value: response},
			{Key: "status_code", Value: statusCode},
		}}},
	)
	return err
}

func (repo *IdempotencyRepository) Delete(ctx context.Context, key string) error {
	_, err := repo.collection().DeleteOne(ctx, bson.D{{Key: "key", Value: key}})
	return err
}
