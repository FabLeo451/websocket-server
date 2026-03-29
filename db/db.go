package db

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"database/sql"

	_ "github.com/lib/pq"

	"ekhoes-server/config"
)

//go:embed sql
var all embed.FS

var SqlFS fs.FS

var _connection *sql.DB

// Builtin module init
func init() {
	sub, err := fs.Sub(all, "sql")
	if err != nil {
		panic(err)
	}
	SqlFS = sub
}

// Exported custom init
func Init(module string) error {
	log.Printf("Initializing database (%s)...", config.Runtime.Database)

	script, err := LoadSQL("init.sql")

	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(script)

	_, err = DB_GetConnection().Exec(script)

	if err != nil {
		return err
	}
	/*
		if flagCreateAdmin {
			log.Print("Creating Administrator user...")

			if err := CreateAdmin("admin"); err != nil {
				return err
			}
		}
	*/

	return nil
}

func LoadSQL(filename string) (string, error) {
	folder := "postgres"
	if config.Local() {
		folder = "local"
	}

	path := fmt.Sprintf("%s/%s", folder, filename)

	content, err := fs.ReadFile(SqlFS, path)

	if err != nil {
		return "", err
	}

	script := string(content)

	return script, nil
}

func ExecuteSQL(filename string, args ...any) error {
	script, err := LoadSQL(filename)
	if err != nil {
		return err
	}

	_, err = DB_GetConnection().Exec(script, args...)
	if err != nil {
		return err
	}

	return nil
}

func Close(db *sql.DB) {
	if db != nil {
		db.Close()
	}
}

func DB_GetConnection() *sql.DB {
	return _connection
}

func DB_Ping() bool {
	conn := DB_GetConnection()

	err := conn.Ping()

	return err == nil
}

func OpenStaff(app string) error {
	if config.Local() {
		config.Runtime.Database = "Local"
		config.Runtime.Local = true

		if app == "" {
			app = "ekhoes"
		}

		log.Printf("Opening local database '%s'... ", app)

		conn, err := openLocal(app)
		_connection = conn

		if err != nil {
			return err
		}
	} else {
		if config.PosgresEnabled() {
			config.Runtime.Database = "PostgreSQL"

			log.Printf("Connecting to database %s:%s...\n", os.Getenv("DB_HOST"), os.Getenv("DB_PORT"))

			_, err := ConnectAndKeepAlive()
			if err != nil {
				return err
			}
		}
	}

	err := OpenCache()

	if err != nil {
		return err
	}

	return nil
}

func CloseStuff() {

	if _connection != nil {
		log.Println("Closing database connection...")
		Close(_connection)
	}

	if config.RedisEnabled() {
		log.Println("Closing Redis connection...")
		RedisClose()
	}
}

func OpenAndInit(app string) error {
	err := OpenStaff(app)

	if err != nil {
		return err
	}

	err = Init(app)

	return err
}

func CheckDatabaseExists() (bool, error) {
	exists := false

	if config.PosgresEnabled() {
		e, err := CheckPostgres()

		if err != nil {
			return false, err
		}

		exists = e
	} else if config.Local() {
		exists = CheckLocal("ekhoes")
	}

	return exists, nil
}

func CreateDatabase() error {
	if config.PosgresEnabled() {
		return ExecuteSQL("create_db.sql")
	} else if config.Local() {
		dbPath := fmt.Sprintf("%s/ekhoes.db", dbFolder)

		dir := filepath.Dir(dbPath)
		err := os.MkdirAll(dir, 0755)

		return err
	}

	return nil
}

func CreateUser(id string, name string, email string, password string, status string, role string) error {
	err := ExecuteSQL("create_user.sql", id, name, email, password, status)

	if err == nil {
		return ExecuteSQL("add_role.sql", id, role)
	}

	return nil
}

func CreateAdmin(email string) error {
	return CreateUser("1000", "Administrator", email, "admin", "enabled", "ADMIN")
}
