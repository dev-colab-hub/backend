package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	ProductsCollection     = "products"
	SellersCollection      = "sellers"
	IdempotencyCollection  = "idempotency_records"
	OutboxEventsCollection = "outbox_events"
)

// EnsureIndexes creates the indexes the repositories rely on. CreateOne is
// idempotent (a matching index is a no-op), so this is safe to call on every
// startup instead of running versioned schema migrations.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if _, err := db.Collection(IdempotencyCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// Partial index over unpublished events only, mirroring the Postgres
	// partial index used for the same relay-polling query.
	if _, err := db.Collection(OutboxEventsCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "occurred_at", Value: 1}},
		Options: options.Index().
			SetName("unpublished_events").
			SetPartialFilterExpression(bson.D{{Key: "published_at", Value: bson.D{{Key: "$eq", Value: nil}}}}),
	}); err != nil {
		return err
	}

	for _, collection := range []string{ProductsCollection, SellersCollection} {
		if _, err := db.Collection(collection).Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{{Key: "deleted_at", Value: 1}},
		}); err != nil {
			return err
		}
	}

	if _, err := db.Collection(ProductsCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "seller_id", Value: 1}},
	}); err != nil {
		return err
	}

	return nil
}
