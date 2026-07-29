package mongodb

import (
	"time"

	"github.com/google/uuid"

	"github.com/sklinkert/go-ddd/internal/domain/entities"
)

// Document structs mirror sqlc's generated models.go: they are the
// persistence-layer shape, kept separate from the domain entities. UUIDs are
// stored as strings for mongosh-readability; Money becomes an embedded
// sub-document.

type moneyDoc struct {
	MinorUnits int64  `bson:"minor_units"`
	Currency   string `bson:"currency"`
}

type productDoc struct {
	ID        string     `bson:"_id"`
	Name      string     `bson:"name"`
	Price     moneyDoc   `bson:"price"`
	SellerID  string     `bson:"seller_id"`
	CreatedAt time.Time  `bson:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at"`
}

func productToDoc(product *entities.ValidatedProduct) productDoc {
	return productDoc{
		ID:   product.Id.String(),
		Name: product.Name,
		Price: moneyDoc{
			MinorUnits: product.Price.MinorUnits(),
			Currency:   string(product.Price.Currency()),
		},
		SellerID:  product.SellerId.String(),
		CreatedAt: product.CreatedAt,
		UpdatedAt: product.UpdatedAt,
	}
}

func productFromDoc(doc productDoc) (*entities.Product, error) {
	id, err := uuid.Parse(doc.ID)
	if err != nil {
		return nil, err
	}

	sellerID, err := uuid.Parse(doc.SellerID)
	if err != nil {
		return nil, err
	}

	price, err := entities.NewMoney(doc.Price.MinorUnits, entities.Currency(doc.Price.Currency))
	if err != nil {
		return nil, err
	}

	return &entities.Product{
		Id:        id,
		Name:      doc.Name,
		Price:     price,
		SellerId:  sellerID,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}

type sellerDoc struct {
	ID        string     `bson:"_id"`
	Name      string     `bson:"name"`
	CreatedAt time.Time  `bson:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at"`
}

func sellerToDoc(seller *entities.ValidatedSeller) sellerDoc {
	return sellerDoc{
		ID:        seller.Id.String(),
		Name:      seller.Name,
		CreatedAt: seller.CreatedAt,
		UpdatedAt: seller.UpdatedAt,
	}
}

func sellerFromDoc(doc sellerDoc) (*entities.Seller, error) {
	id, err := uuid.Parse(doc.ID)
	if err != nil {
		return nil, err
	}

	return &entities.Seller{
		Id:        id,
		Name:      doc.Name,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}

type idempotencyDoc struct {
	ID         string    `bson:"_id"`
	Key        string    `bson:"key"`
	Request    string    `bson:"request"`
	Response   string    `bson:"response"`
	StatusCode int       `bson:"status_code"`
	CreatedAt  time.Time `bson:"created_at"`
}

type outboxEventDoc struct {
	ID          string     `bson:"_id"`
	AggregateID string     `bson:"aggregate_id"`
	EventName   string     `bson:"event_name"`
	Payload     []byte     `bson:"payload"`
	OccurredAt  time.Time  `bson:"occurred_at"`
	PublishedAt *time.Time `bson:"published_at"`
}
