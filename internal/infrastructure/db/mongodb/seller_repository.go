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

// SellerRepository never touches the outbox (sellers raise no domain events
// today), so unlike ProductRepository it needs no session/transaction.
type SellerRepository struct {
	db *mongo.Database
}

func NewSellerRepository(db *mongo.Database) repositories.SellerRepository {
	return &SellerRepository{db: db}
}

func (repo *SellerRepository) collection() *mongo.Collection {
	return repo.db.Collection(SellersCollection)
}

func (repo *SellerRepository) Create(ctx context.Context, seller *entities.ValidatedSeller) (*entities.Seller, error) {
	if _, err := repo.collection().InsertOne(ctx, sellerToDoc(seller)); err != nil {
		return nil, err
	}

	return repo.FindById(ctx, seller.Id)
}

func (repo *SellerRepository) FindById(ctx context.Context, id uuid.UUID) (*entities.Seller, error) {
	var doc sellerDoc
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

	return sellerFromDoc(doc)
}

func (repo *SellerRepository) FindAll(ctx context.Context) ([]*entities.Seller, error) {
	cursor, err := repo.collection().Find(ctx,
		bson.D{{Key: "deleted_at", Value: nil}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}

	var docs []sellerDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	sellers := make([]*entities.Seller, len(docs))
	for i, doc := range docs {
		seller, err := sellerFromDoc(doc)
		if err != nil {
			return nil, err
		}
		sellers[i] = seller
	}

	return sellers, nil
}

func (repo *SellerRepository) Update(ctx context.Context, seller *entities.ValidatedSeller) (*entities.Seller, error) {
	filter := bson.D{
		{Key: "_id", Value: seller.Id.String()},
		{Key: "deleted_at", Value: nil},
	}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: seller.Name},
		{Key: "updated_at", Value: seller.UpdatedAt},
	}}}

	result, err := repo.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		// Nothing matched: the seller does not exist (or is soft-deleted).
		return nil, entities.ErrSellerNotFound
	}

	return repo.FindById(ctx, seller.Id)
}

func (repo *SellerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := repo.collection().UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id.String()}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "deleted_at", Value: now}}}},
	)
	return err
}
