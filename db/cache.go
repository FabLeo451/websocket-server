package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/TwiN/gocache/v2"

	"websocket-server/config"
)

var (
	cache *gocache.Cache
	ctx   = context.Background()
)

func OpenCache() error {
	if config.RedisEnabled() {
		log.Printf("Connecting to Redis %s:%s...\n", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))

		_, err := RedisConnect()
		if err != nil {
			return err
		}
	} else {
		log.Println("Creating cache...")
		c := gocache.NewCache().WithMaxSize(1000).WithEvictionPolicy(gocache.LeastRecentlyUsed)
		cache = c
	}

	return nil
}

func Set(key string, value interface{}) error {
	var err error

	if config.RedisEnabled() {
		err = RedisGetConnection().Set(ctx, key, value, 0).Err()
	} else {
		cache.Set(key, value)
	}

	return err
}

func GetKeysByPattern(pattern string) ([]string, error) {
	var err error
	var keys []string

	if config.RedisEnabled() {
		keys, err = RedisGetConnection().Keys(ctx, "*").Result()
	} else {
		keys = cache.GetKeysByPattern(pattern, 0)
	}

	return keys, err
}

func Get(key string) (string, error) {
	var (
		err error
		val string
	)

	if config.RedisEnabled() {
		val, err = RedisGetConnection().Get(ctx, key).Result()
	} else {
		if i, found := cache.Get(key); found {
			val = fmt.Sprintf("%s", i)
		}
	}

	return val, err
}

/*
func Get(key string) (string, bool) {
    value, found := c.Get("chiave")
    if found {
        fmt.Println("Valore:", value)
    }
}
*/
