package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"database/sql"

	_ "github.com/lib/pq"

	"github.com/redis/go-redis/v9"
)

var _connection *sql.DB

var (
	_redisConn *redis.Client
	once       sync.Once
)

/**
* Create the connection pool
 */
func DB_Open() (*sql.DB, error) {

	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", conf.DB.Host, conf.DB.Port, conf.DB.User, conf.DB.Password, conf.DB.Name)
	db, err := sql.Open("postgres", psqlconn)

	db.SetMaxOpenConns(conf.DB.PoolSize)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	/*
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", conf.DB.User, conf.DB.Password, conf.DB.Host, conf.DB.Port, conf.DB.Name)
		fmt.Println(dsn)


		config, err := pgxpool.ParseConfig(psqlconn)
		if err != nil {
			log.Fatal(err)
		}
		config.MaxConns = 10

		pool, err := pgxpool.NewWithConfig(ctx, config)
	*/

	return db, err
}

func DB_Close(db *sql.DB) {

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

// You can stop the worker by closing the quit channel: close(quit)
func DB_KeepAlive() {

	ticker := time.NewTicker(time.Duration(conf.DB.Heartbeat) * time.Second)

	quit := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				//LogWrite("Ping...\n")

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

func DB_ConnectKeepAlive() *sql.DB {

	LogWrite("Connecting to database %s:%d...\n", conf.DB.Host, conf.DB.Port)

	conn, err := DB_Open()

	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	err = conn.Ping()

	if err == nil {

		_connection = conn

		LogWrite("Starting keep alive function...\n")
		DB_KeepAlive()

	} else {
		fmt.Printf("Error: %s\n", err.Error())
		_connection = nil
	}

	return _connection
}

func RedisConnect() *redis.Client {
	once.Do(func() {
		addr := fmt.Sprintf("%s:%d", conf.Redis.Host, conf.Redis.Port)

		LogWrite("Connecting to Redis %s ...\n", addr)

		_redisConn = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: conf.Redis.Password,
			DB:       0,
			PoolSize: conf.Redis.PoolSize,
		})

		_, err := _redisConn.Ping(context.Background()).Result()
		if err != nil {
			LogWrite("Redis connection failed: %v\n", err)
		} else {
			LogWrite("Redis connected\n")
		}
	})

	return _redisConn
}

func RedisGetConnection() *redis.Client {
	return RedisConnect()
}
