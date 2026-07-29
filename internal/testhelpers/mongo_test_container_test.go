package testhelpers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	dbmongo "github.com/sklinkert/go-ddd/internal/infrastructure/db/mongodb"
	"github.com/sklinkert/go-ddd/internal/testhelpers"
)

func TestSetupTestDB(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	require.NotNil(t, testDB)
	require.NotNil(t, testDB.Container)
	require.NotNil(t, testDB.Client)
	require.NotNil(t, testDB.Database)

	defer testDB.Close(t)

	ctx := context.Background()
	err := testDB.Client.Ping(ctx, nil)
	assert.NoError(t, err)
}

func TestMongoTestContainer_TruncateTables(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	ctx := context.Background()

	_, err := testDB.Database.Collection(dbmongo.SellersCollection).InsertOne(ctx, bson.D{
		{Key: "_id", Value: "test-seller"},
		{Key: "name", Value: "Test Seller"},
	})
	require.NoError(t, err)

	_, err = testDB.Database.Collection(dbmongo.ProductsCollection).InsertOne(ctx, bson.D{
		{Key: "_id", Value: "test-product"},
		{Key: "name", Value: "Test Product"},
		{Key: "seller_id", Value: "test-seller"},
	})
	require.NoError(t, err)

	_, err = testDB.Database.Collection(dbmongo.IdempotencyCollection).InsertOne(ctx, bson.D{
		{Key: "_id", Value: "test-idempotency"},
		{Key: "key", Value: "test-key"},
	})
	require.NoError(t, err)

	for _, collection := range []string{dbmongo.SellersCollection, dbmongo.ProductsCollection, dbmongo.IdempotencyCollection} {
		count, err := testDB.Database.Collection(collection).CountDocuments(ctx, bson.D{})
		require.NoError(t, err)
		assert.Greater(t, count, int64(0), "collection %s should have data before truncation", collection)
	}

	testDB.TruncateTables(t)

	for _, collection := range []string{dbmongo.SellersCollection, dbmongo.ProductsCollection, dbmongo.IdempotencyCollection} {
		count, err := testDB.Database.Collection(collection).CountDocuments(ctx, bson.D{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "collection %s should be empty after truncation", collection)
	}
}

func TestMongoTestContainer_Close(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	require.NotNil(t, testDB)

	ctx := context.Background()
	err := testDB.Client.Ping(ctx, nil)
	assert.NoError(t, err)

	assert.NotPanics(t, func() {
		testDB.Close(t)
	})
}

func TestMultipleTestContainers(t *testing.T) {
	testDB1 := testhelpers.SetupTestDB(t)
	testDB2 := testhelpers.SetupTestDB(t)

	defer testDB1.Close(t)
	defer testDB2.Close(t)

	ctx := context.Background()
	err := testDB1.Client.Ping(ctx, nil)
	assert.NoError(t, err)

	err = testDB2.Client.Ping(ctx, nil)
	assert.NoError(t, err)

	assert.NotEqual(t, testDB1.Container, testDB2.Container)
	assert.NotEqual(t, testDB1.Client, testDB2.Client)
}
