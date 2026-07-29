package mongodb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sklinkert/go-ddd/internal/infrastructure/db/mongodb"
	"github.com/sklinkert/go-ddd/internal/testhelpers"
)

func TestNewClient(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	// testDB.Client was built through NewClient already; verify it's usable.
	ctx := context.Background()
	err := testDB.Client.Ping(ctx, nil)
	assert.NoError(t, err)
}

func TestNewClient_InvalidURI(t *testing.T) {
	ctx := context.Background()

	client, err := mongodb.NewClient(ctx, "not-a-mongo-uri")
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_UnreachableHost(t *testing.T) {
	ctx := context.Background()

	client, err := mongodb.NewClient(ctx, "mongodb://127.0.0.1:1/no-such-host?serverSelectionTimeoutMS=1000&connectTimeoutMS=1000")
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestEnsureIndexes(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	// SetupTestDB already calls EnsureIndexes once; calling it again must be
	// a no-op (idempotent), the same way it's safe to run on every startup.
	require.NoError(t, mongodb.EnsureIndexes(context.Background(), testDB.Database))
}
