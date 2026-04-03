package db

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"database/sql"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

/**
* Create the connection pool
 */
func OpenPostgres() (*sql.DB, error) {

	poolSize, _ := strconv.Atoi(os.Getenv("DB_POOLSIZE"))
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))

	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), port, os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	db, err := sql.Open("postgres", psqlconn)

	db.SetMaxOpenConns(poolSize)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return db, err
}

func CheckPostgres() (bool, error) {
	conn, err := OpenPostgres()
	defer Close(conn)

	if err != nil {
		return false, err
	}

	err = conn.Ping()

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "3D000": // database does not exist
				return false, nil
			default:
				return false, err
			}
		}

		return false, err
	}

	return true, nil
}

// You can stop the worker by closing the quit channel: close(quit)
func StartKeepAlive() {

	heartbeat, _ := strconv.Atoi(os.Getenv("DB_HEARTBEAT"))

	ticker := time.NewTicker(time.Duration(heartbeat) * time.Second)

	quit := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				//log.Printf("Ping...\n")

				if !DB_Ping() {
					fmt.Println("Error: Database unavailable")
					_connection = nil
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func ConnectAndKeepAlive() (*sql.DB, error) {

	conn, err := OpenPostgres()

	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	err = conn.Ping()

	if err == nil {

		_connection = conn

		//log.Printf("Starting keep alive function...\n")
		StartKeepAlive()

	} else {
		fmt.Printf("Error: %s\n", err.Error())
		_connection = nil
	}

	return _connection, err
}
