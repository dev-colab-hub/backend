package mongodb

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sklinkert/go-ddd/internal/domain/entities"
	"github.com/sklinkert/go-ddd/internal/domain/repositories"
)

// ProductRepository has no soft-deleted-seller join with FindAll/FindById
// (the Postgres version joined sellers to hide products of a soft-deleted
// seller); listing/reading a product no longer depends on its seller's
// liveness. Flagged as a deliberate simplification, not an oversight.
type ProductRepository struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewProductRepository(client *mongo.Client, db *mongo.Database) repositories.ProductRepository {
	return &ProductRepository{client: client, db: db}
}

func (r *ProductRepository) collection() *mongo.Collection {
	return r.db.Collection(ProductsCollection)
}

// Create persists the product and its recorded domain events in one
// multi-document transaction (transactional outbox): either both are
// committed or neither. Requires Mongo to run as a replica set.
//
// Domain events are pulled once, before the transaction starts: Mongo may
// retry the callback on a transient transaction error, and PullEvents
// clears the aggregate's in-memory event slice, so pulling inside the
// callback would silently drop events on a retry.
func (repo *ProductRepository) Create(ctx context.Context, product *entities.ValidatedProduct) (*entities.Product, error) {
	domainEvents := product.PullEvents()
	doc := productToDoc(product)

	session, err := repo.client.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, func(sc context.Context) (any, error) {
		if _, err := repo.collection().InsertOne(sc, doc); err != nil {
			return nil, err
		}

		if err := insertOutboxEvents(sc, repo.db.Collection(OutboxEventsCollection), domainEvents); err != nil {
			return nil, err
		}

		// Read-after-write inside the same transaction, so a transient
		// failure cannot surface after the commit already succeeded.
		var created productDoc
		if err := repo.collection().FindOne(sc, bson.D{{Key: "_id", Value: doc.ID}}).Decode(&created); err != nil {
			return nil, err
		}

		return created, nil
	})
	if err != nil {
		return nil, err
	}

	return productFromDoc(result.(productDoc))
}

func (repo *ProductRepository) FindById(ctx context.Context, id uuid.UUID) (*entities.Product, error) {
	var doc productDoc
	err := repo.collection().FindOne(ctx, bson.D{
		{Key: "_id", Value: id.String()},
		{Key: "deleted_at", Value: nil},
	}).Decode(&doc)
	if err != nil {
		// A missing document is not an error: return (nil, nil) so callers
		// can translate it into a 404 instead of a 500.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return productFromDoc(doc)
}

func (repo *ProductRepository) FindAll(ctx context.Context) ([]*entities.Product, error) {
	cursor, err := repo.collection().Find(ctx,
		bson.D{{Key: "deleted_at", Value: nil}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}

	var docs []productDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	products := make([]*entities.Product, len(docs))
	for i, doc := range docs {
		product, err := productFromDoc(doc)
		if err != nil {
			return nil, err
		}
		products[i] = product
	}

	return products, nil
}

func (repo *ProductRepository) Update(ctx context.Context, product *entities.ValidatedProduct) (*entities.Product, error) {
	filter := bson.D{
		{Key: "_id", Value: product.Id.String()},
		{Key: "deleted_at", Value: nil},
	}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: product.Name},
		{Key: "price", Value: moneyDoc{
			MinorUnits: product.Price.MinorUnits(),
			Currency:   string(product.Price.Currency()),
		}},
		{Key: "seller_id", Value: product.SellerId.String()},
		{Key: "updated_at", Value: product.UpdatedAt},
	}}}

	result, err := repo.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		// Nothing matched: the product does not exist (or is soft-deleted).
		return nil, entities.ErrProductNotFound
	}

	return repo.FindById(ctx, product.Id)
}

func (repo *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := repo.collection().UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id.String()}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "deleted_at", Value: now}}}},
	)
	return err
}
