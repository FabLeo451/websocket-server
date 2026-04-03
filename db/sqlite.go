package db

import (
	"fmt"
	"os"
	"path/filepath"

	"database/sql"
	//"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var dbFolder string = "./data"

func CreateLocal(app string) error {
	dbPath := fmt.Sprintf("%s/%s.db", dbFolder, app)

	dir := filepath.Dir(dbPath)
	err := os.MkdirAll(dir, 0755)

	return err
}

func openLocal(app string) (*sql.DB, error) {
	dbPath := fmt.Sprintf("%s/%s.db", dbFolder, app)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Suggerito: WAL per concorrenza migliore (journal separato)
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return nil, err
	}

	// (opzionale) Foreign keys
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, err
	}

	return db, err
}

func CheckLocal(app string) bool {
	db, err := openLocal(app)

	result := err == nil

	if db != nil {
		Close(db)
	}

	return result
}

/*
func createAdminLocal() error {
	content, err := fs.ReadFile(DbSqlFS, "local/create_admin.sql")
	if err != nil {
		return err
	}

	sqlScript := string(content)

	//fmt.Println(sqlScript)

	//userID := uuid.New().String()

	_, err = DB_GetConnection().Exec(sqlScript, "admin")
	if err != nil {
		return err
	}

	return nil
}
*/
