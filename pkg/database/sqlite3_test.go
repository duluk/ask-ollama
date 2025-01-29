package database

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

const dbPath = "./test.db"
const dbTable = "conversations_test"

func TestMain(m *testing.M) {
	code := m.Run()

	os.Remove(dbPath)

	os.Exit(code)
}

func TestNewDB(t *testing.T) {
	db, err := NewDB(dbPath, dbTable)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	db.Close()
}

func TestInsertConversation(t *testing.T) {
	db, err := NewDB(dbPath, dbTable)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	err = db.InsertConversation("prompt", "response", "model_name", 0.5, 10, 20, 1)
	assert.Nil(t, err)

	db.Close()
	RemoveDB()
}

func TestClose(t *testing.T) {
	db, err := NewDB(dbPath, dbTable)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	db.Close()
	RemoveDB()
}

func TestInsertConversationWithError(t *testing.T) {
	db, err := NewDB(dbPath, dbTable)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	// Insert a conversation with invalid data and assert there are errors
	// TODO: This won't do much at this point because InsertConversation doesn't do
	// much validation of iput, and Go's type system won't let me enter an
	// invalid argument. InsertConversation should, however, do some
	// validation. For instance, there are restrictions about temperature - eg,
	// 0.123 is technically invalid.
	// err = db.InsertConversation("prompt", "response", "", 0.0)
	// assert.NotNil(t, err)

	db.Close()
	RemoveDB()
}

func TestLoadConversations(t *testing.T) {
	db, err := NewDB(dbPath, dbTable)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	err = db.InsertConversation("prompt", "response", "model_name", 0.5, 10, 20, 1)
	assert.Nil(t, err)
	err = db.InsertConversation("prompt2", "response2", "model_name2", 0.5, 10, 20, 2)
	assert.Nil(t, err)

	conversations, err := db.LoadConversationFromDB(1)
	assert.Nil(t, err)
	// Len 2 because one prompt/response turn in the DB is one row; however,
	// it's two LLMConversations, which LoadConversationFromDB does.
	assert.Len(t, conversations, 2)

	db.Close()
	RemoveDB()
}

func TestSearchForConversation(t *testing.T) {
	db, err := NewDB(dbPath, dbTable)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	err = db.InsertConversation("prompt", "response", "model_name", 0.5, 10, 20, 1)
	assert.Nil(t, err)
	err = db.InsertConversation("prompt2", "response2", "model_name2", 0.5, 10, 20, 2)
	assert.Nil(t, err)

	ids, err := db.SearchForConversation("response")
	assert.Nil(t, err)
	assert.Len(t, ids, 2)
	assert.Equal(t, 1, ids[0])

	ids, err = db.SearchForConversation("marklar")
	assert.Nil(t, err)
	assert.Len(t, ids, 0)
	// assert.Equal(t, 1, ids[0])

	db.Close()
	RemoveDB()
}

func RemoveDB() {
	os.Remove(dbPath)
}
