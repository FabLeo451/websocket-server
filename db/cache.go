package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/TwiN/gocache/v2"
	"github.com/redis/go-redis/v9"

	"websocket-server/config"
)

var (
	cache *gocache.Cache
	ctx   = context.Background()
)

func OpenCache() error {
	if config.RedisEnabled() {
		config.Runtime.Cache = "Redis"

		log.Printf("Connecting to Redis %s:%s...\n", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))

		_, err := RedisConnect()
		if err != nil {
			return err
		}
	} else {
		config.Runtime.Cache = "Internal"

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

func Update(key string, value interface{}) error {
	var err error

	if config.RedisEnabled() {
		err = RedisGetConnection().SetArgs(ctx, key, value, redis.SetArgs{
			KeepTTL: true,
		}).Err()
	} else {
		ttl, err := cache.TTL(key)
		if err != nil {
			return fmt.Errorf("key not found: %s", key)
		}

		cache.SetWithTTL(key, value, ttl)
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

func DeleteKey(key string) (bool, error) {
	var (
		err     error
		n       int64
		deleted bool
	)

	if config.RedisEnabled() {
		n, err = RedisGetConnection().Del(ctx, key).Result()
		deleted = n > 0
	} else {
		deleted = cache.Delete(key)
	}

	return deleted, err
}

func DeleteByPattern(pattern string) error {
	if config.RedisEnabled() {
		var cursor uint64
		ctx := context.Background()

		for {
			keys, newCursor, err := RedisGetConnection().Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return fmt.Errorf("scan error: %w", err)
			}

			if len(keys) > 0 {
				if err := RedisGetConnection().Del(ctx, keys...).Err(); err != nil {
					return fmt.Errorf("delete error: %w", err)
				}
			}

			cursor = newCursor
			if cursor == 0 {
				break
			}
		}
	} else {
		keys := cache.GetKeysByPattern(pattern, 0)
		cache.DeleteAll(keys)
	}

	return nil
}
