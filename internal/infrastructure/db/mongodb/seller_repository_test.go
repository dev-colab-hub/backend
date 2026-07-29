package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sklinkert/go-ddd/internal/domain/entities"
	"github.com/sklinkert/go-ddd/internal/infrastructure/db/mongodb"
	"github.com/sklinkert/go-ddd/internal/testhelpers"
)

func TestSellerRepository_Create(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	seller := entities.NewSeller("Test Seller")
	validatedSeller, err := entities.NewValidatedSeller(seller)
	require.NoError(t, err)

	createdSeller, err := repo.Create(context.Background(), validatedSeller)

	require.NoError(t, err)
	require.NotNil(t, createdSeller)
	assert.Equal(t, validatedSeller.Name, createdSeller.Name)
	assert.NotEqual(t, uuid.Nil, createdSeller.Id)
	assert.False(t, createdSeller.CreatedAt.IsZero())
	assert.False(t, createdSeller.UpdatedAt.IsZero())
}

func TestSellerRepository_FindById(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	seller := entities.NewSeller("Test Seller")
	validatedSeller, err := entities.NewValidatedSeller(seller)
	require.NoError(t, err)

	createdSeller, err := repo.Create(context.Background(), validatedSeller)
	require.NoError(t, err)

	foundSeller, err := repo.FindById(context.Background(), createdSeller.Id)

	require.NoError(t, err)
	require.NotNil(t, foundSeller)
	assert.Equal(t, createdSeller.Id, foundSeller.Id)
	assert.Equal(t, createdSeller.Name, foundSeller.Name)
	assert.Equal(t, createdSeller.CreatedAt.Unix(), foundSeller.CreatedAt.Unix())
	assert.Equal(t, createdSeller.UpdatedAt.Unix(), foundSeller.UpdatedAt.Unix())
}

func TestSellerRepository_FindById_NotFound(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	foundSeller, err := repo.FindById(context.Background(), uuid.New())

	assert.NoError(t, err)
	assert.Nil(t, foundSeller)
}

func TestSellerRepository_FindAll(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	seller1 := entities.NewSeller("Seller One")
	validatedSeller1, err := entities.NewValidatedSeller(seller1)
	require.NoError(t, err)

	seller2 := entities.NewSeller("Seller Two")
	validatedSeller2, err := entities.NewValidatedSeller(seller2)
	require.NoError(t, err)

	createdSeller1, err := repo.Create(context.Background(), validatedSeller1)
	require.NoError(t, err)

	createdSeller2, err := repo.Create(context.Background(), validatedSeller2)
	require.NoError(t, err)

	sellers, err := repo.FindAll(context.Background())

	require.NoError(t, err)
	require.Len(t, sellers, 2)

	var foundSeller1, foundSeller2 *entities.Seller
	for _, s := range sellers {
		if s.Id == createdSeller1.Id {
			foundSeller1 = s
		} else if s.Id == createdSeller2.Id {
			foundSeller2 = s
		}
	}

	require.NotNil(t, foundSeller1)
	require.NotNil(t, foundSeller2)
	assert.Equal(t, "Seller One", foundSeller1.Name)
	assert.Equal(t, "Seller Two", foundSeller2.Name)
}

func TestSellerRepository_FindAll_Empty(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	sellers, err := repo.FindAll(context.Background())

	require.NoError(t, err)
	assert.Empty(t, sellers)
}

func TestSellerRepository_Update(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	seller := entities.NewSeller("Original Seller")
	validatedSeller, err := entities.NewValidatedSeller(seller)
	require.NoError(t, err)

	createdSeller, err := repo.Create(context.Background(), validatedSeller)
	require.NoError(t, err)

	updatedSeller := &entities.Seller{
		Id:        createdSeller.Id,
		Name:      "Updated Seller",
		CreatedAt: createdSeller.CreatedAt,
		UpdatedAt: time.Now(),
	}

	validatedUpdatedSeller, err := entities.NewValidatedSeller(updatedSeller)
	require.NoError(t, err)

	result, err := repo.Update(context.Background(), validatedUpdatedSeller)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Updated Seller", result.Name)
	assert.Equal(t, createdSeller.Id, result.Id)
	assert.True(t, result.UpdatedAt.After(createdSeller.UpdatedAt))
}

func TestSellerRepository_Update_NotFound(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	nonExistentSeller := &entities.Seller{
		Id:        uuid.New(),
		Name:      "Non-existent Seller",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	validatedNonExistentSeller, err := entities.NewValidatedSeller(nonExistentSeller)
	require.NoError(t, err)

	result, err := repo.Update(context.Background(), validatedNonExistentSeller)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSellerRepository_Delete(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	seller := entities.NewSeller("Test Seller")
	validatedSeller, err := entities.NewValidatedSeller(seller)
	require.NoError(t, err)

	createdSeller, err := repo.Create(context.Background(), validatedSeller)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), createdSeller.Id)
	require.NoError(t, err)

	deletedSeller, err := repo.FindById(context.Background(), createdSeller.Id)
	assert.NoError(t, err)
	assert.Nil(t, deletedSeller)
}

func TestSellerRepository_Delete_NotFound(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	// Mongo's UpdateOne (used for soft delete) doesn't fail if nothing
	// matches, matching the previous Postgres UPDATE behavior.
	err := repo.Delete(context.Background(), uuid.New())

	assert.NoError(t, err)
}

func TestSellerRepository_Delete_WithExistingProducts(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	sellerRepo := mongodb.NewSellerRepository(testDB.Database)
	productRepo := mongodb.NewProductRepository(testDB.Client, testDB.Database)

	seller := entities.NewSeller("Test Seller")
	validatedSeller, err := entities.NewValidatedSeller(seller)
	require.NoError(t, err)

	createdSeller, err := sellerRepo.Create(context.Background(), validatedSeller)
	require.NoError(t, err)

	product := entities.NewProduct("Test Product", mustMoney(t, 9999, entities.USD), *validatedSeller)
	validatedProduct, err := entities.NewValidatedProduct(product)
	require.NoError(t, err)

	_, err = productRepo.Create(context.Background(), validatedProduct)
	require.NoError(t, err)

	// Soft delete should succeed even with related products: there is no
	// referential-integrity/cascade check across the products and sellers
	// collections (no foreign keys in Mongo).
	err = sellerRepo.Delete(context.Background(), createdSeller.Id)
	assert.NoError(t, err)

	softDeletedSeller, err := sellerRepo.FindById(context.Background(), createdSeller.Id)
	assert.NoError(t, err)
	assert.Nil(t, softDeletedSeller)
}

func TestSellerRepository_Create_EmptyName(t *testing.T) {
	// Create a seller with empty name
	seller := &entities.Seller{
		Id:        uuid.New(),
		Name:      "", // Empty name
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// This should fail at the validation level, before any repository call.
	_, err := entities.NewValidatedSeller(seller)
	assert.Error(t, err)
}

func TestSellerRepository_Create_LongName(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close(t)

	repo := mongodb.NewSellerRepository(testDB.Database)

	longName := make([]byte, 1000)
	for i := range longName {
		longName[i] = 'A'
	}

	seller := entities.NewSeller(string(longName))
	validatedSeller, err := entities.NewValidatedSeller(seller)
	require.NoError(t, err)

	createdSeller, err := repo.Create(context.Background(), validatedSeller)

	require.NoError(t, err)
	require.NotNil(t, createdSeller)
	assert.Equal(t, string(longName), createdSeller.Name)
}
