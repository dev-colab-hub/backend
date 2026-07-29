package testhelpers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"

	dbmongo "github.com/sklinkert/go-ddd/internal/infrastructure/db/mongodb"
)

const testDatabaseName = "testdb"

// MongoTestContainer manages a test database container. Started as a
// single-node replica set so tests exercise the same multi-document
// transactions (product create + outbox insert) production uses.
type MongoTestContainer struct {
	Container *mongodb.MongoDBContainer
	Client    *mongodriver.Client
	Database  *mongodriver.Database
}

// SetupTestDB creates a new MongoDB test container and ensures the same
// indexes production relies on.
func SetupTestDB(t *testing.T) *MongoTestContainer {
	ctx := context.Background()

	container, err := mongodb.Run(ctx, "mongo:7", mongodb.WithReplicaSet("rs0"))
	require.NoError(t, err, "Failed to start mongodb container")

	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err, "Failed to get connection string")

	// The container advertises its internal Docker bridge IP as the replica
	// set member host for rs.initiate(), which isn't routable from the host
	// process on Docker Desktop (Windows/Mac). directConnection bypasses
	// topology discovery and talks to the single mapped-port connection
	// directly instead of trying to dial that internal IP.
	uri += "&directConnection=true"

	client, err := dbmongo.NewClient(ctx, uri)
	require.NoError(t, err, "Failed to connect to test database")

	database := client.Database(testDatabaseName)

	err = dbmongo.EnsureIndexes(ctx, database)
	require.NoError(t, err, "Failed to ensure indexes")

	return &MongoTestContainer{
		Container: container,
		Client:    client,
		Database:  database,
	}
}

// Close cleans up the test database and container.
func (m *MongoTestContainer) Close(t *testing.T) {
	ctx := context.Background()

	if m.Client != nil {
		_ = m.Client.Disconnect(ctx)
	}

	if m.Container != nil {
		err := m.Container.Terminate(ctx)
		require.NoError(t, err, "Failed to terminate container")
	}
}

// TruncateTables removes all documents from every collection under test,
// used for per-test cleanup instead of restarting the container each test.
func (m *MongoTestContainer) TruncateTables(t *testing.T) {
	ctx := context.Background()

	collections := []string{
		dbmongo.ProductsCollection,
		dbmongo.IdempotencyCollection,
		dbmongo.OutboxEventsCollection,
		dbmongo.SellersCollection,
	}

	for _, collection := range collections {
		_, err := m.Database.Collection(collection).DeleteMany(ctx, bson.D{})
		require.NoError(t, err, "Failed to truncate collection %s", collection)
	}
}
