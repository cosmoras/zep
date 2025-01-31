package postgres

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/getzep/zep/pkg/llms"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
)

func TestEnsurePostgresSchemaSetup(t *testing.T) {
	CleanDB(t, testDB)

	t.Run("should succeed when all schema setup is successful", func(t *testing.T) {
		err := CreateSchema(testCtx, appState, testDB)
		assert.NoError(t, err)

		checkForTable(t, testDB, &SessionSchema{})
		checkForTable(t, testDB, &MessageStoreSchema{})
		checkForTable(t, testDB, &SummaryStoreSchema{})
		checkForTable(t, testDB, &MessageVectorStoreSchema{})
	})
	t.Run("should not fail on second run", func(t *testing.T) {
		err := CreateSchema(testCtx, appState, testDB)
		assert.NoError(t, err)
	})
}

func TestCreateDocumentTable(t *testing.T) {
	collection := NewTestCollectionDAO(3)

	tableName, err := generateDocumentTableName(&collection)
	assert.NoError(t, err)

	err = createDocumentTable(testCtx, appState, testDB, tableName, collection.EmbeddingDimensions)
	assert.NoError(t, err)
}

func TestUpdatedAtIsSetAfterUpdate(t *testing.T) {
	// Define a list of all schemas
	schemas := []bun.BeforeAppendModelHook{
		&SessionSchema{},
		&MessageStoreSchema{},
		&SummaryStoreSchema{},
		&MessageVectorStoreSchema{},
		&UserSchema{},
		&DocumentCollectionSchema{},
	}

	// Iterate over all schemas
	for _, schema := range schemas {
		// Create a new instance of the schema
		instance := reflect.New(reflect.TypeOf(schema).Elem()).Interface().(bun.BeforeAppendModelHook)

		// Set the UpdatedAt field to a time far in the past
		reflect.ValueOf(instance).
			Elem().
			FieldByName("UpdatedAt").
			Set(reflect.ValueOf(time.Unix(0, 0)))

		// Create a dummy UpdateQuery
		updateQuery := &bun.UpdateQuery{}

		// Call the BeforeAppendModel method, which should update the UpdatedAt field
		err := instance.BeforeAppendModel(context.Background(), updateQuery)
		assert.NoError(t, err)

		// Check that the UpdatedAt field was updated
		assert.True(
			t,
			reflect.ValueOf(instance).Elem().FieldByName("UpdatedAt").Interface().(time.Time).After(
				time.Now().Add(-time.Minute),
			),
		)
	}
}

func TestCheckMessageEmbeddingDims(t *testing.T) {
	// Create a new DB
	CleanDB(t, testDB)
	err := CreateSchema(testCtx, appState, testDB)
	assert.NoError(t, err)

	// Get the embedding model
	model, err := llms.GetEmbeddingModel(appState, "message")
	assert.NoError(t, err)

	testWidth := model.Dimensions + 1
	// Set the embedding column to a specific width
	err = MigrateMessageEmbeddingDims(testCtx, testDB, testWidth)
	assert.NoError(t, err)

	width, err := getEmbeddingColumnWidth(testCtx, "message_embedding", testDB)
	assert.NoError(t, err)

	assert.Equal(t, width, testWidth)

	// Clean the DB
	CleanDB(t, testDB)
	err = CreateSchema(testCtx, appState, testDB)
	assert.NoError(t, err)

}
