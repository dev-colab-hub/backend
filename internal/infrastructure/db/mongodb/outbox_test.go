package mongodb_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/sklinkert/go-ddd/internal/domain/entities"
	"github.com/sklinkert/go-ddd/internal/infrastructure/db/mongodb"
	"github.com/sklinkert/go-ddd/internal/testhelpers"
)

// outboxEventDoc mirrors the unexported document shape in the mongodb
// package, just for decoding in this external test package.
type outboxEventDoc struct {
	ID          string     `bson:"_id"`
	AggregateID string     `bson:"aggregate_id"`
	EventName   string     `bson:"event_name"`
	Payload     []byte     `bson:"payload"`
	PublishedAt *time.Time `bson:"published_at"`
}

func TestProductRepository_Create_WritesOutboxEvent(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewProductRepository(testDB.Client, testDB.Database)
	validatedSeller := createTestSeller(t, testDB, "Outbox Seller")

	product := entities.NewProduct("Outbox Product", mustMoney(t, 1299, entities.EUR), *validatedSeller)
	validatedProduct, err := entities.NewValidatedProduct(product)
	require.NoError(t, err)

	createdProduct, err := repo.Create(context.Background(), validatedProduct)
	require.NoError(t, err)

	// The ProductCreated event must be committed together with the product.
	collection := testDB.Database.Collection(mongodb.OutboxEventsCollection)

	cursor, err := collection.Find(context.Background(), bson.D{{Key: "published_at", Value: nil}})
	require.NoError(t, err)

	var events []outboxEventDoc
	require.NoError(t, cursor.All(context.Background(), &events))
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, "product.created", event.EventName)
	assert.Equal(t, createdProduct.Id.String(), event.AggregateID)
	assert.Nil(t, event.PublishedAt)

	var payload struct {
		Name            string `json:"Name"`
		PriceMinorUnits int64  `json:"PriceMinorUnits"`
		Currency        string `json:"Currency"`
	}
	require.NoError(t, json.Unmarshal(event.Payload, &payload))
	assert.Equal(t, "Outbox Product", payload.Name)
	assert.Equal(t, int64(1299), payload.PriceMinorUnits)
	assert.Equal(t, "EUR", payload.Currency)

	// Marking published removes it from the unpublished set (relay behavior).
	_, err = collection.UpdateOne(context.Background(),
		bson.D{{Key: "_id", Value: event.ID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "published_at", Value: time.Now()}}}},
	)
	require.NoError(t, err)

	cursor, err = collection.Find(context.Background(), bson.D{{Key: "published_at", Value: nil}})
	require.NoError(t, err)

	var remaining []outboxEventDoc
	require.NoError(t, cursor.All(context.Background(), &remaining))
	assert.Empty(t, remaining)
}
