package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/TwiN/gocache/v2"
	"github.com/redis/go-redis/v9"

	"ekhoes-server/config"
)

var (
	cache *gocache.Cache
	ctx   = context.Background()
)

func OpenCache() error {
	if config.RedisEnabled() {

		log.Printf("Connecting to Redis %s:%s...\n", os.Getenv("EKHOES_REDIS_HOST"), os.Getenv("EKHOES_REDIS_PORT"))

		_, err := RedisConnect()
		if err != nil {
			return err
		}

		info, err := RedisGetConnection().Info(ctx).Result()

		if err != nil {
			log.Println(err)
			config.Runtime.Cache = "Redis - " + err.Error()
		} else {
			//fmt.Println(info)
			lines := strings.Split(info, "\n")

			version, os := "", ""

			for _, line := range lines {
				if strings.HasPrefix(line, "redis_version:") {
					version = strings.TrimPrefix(line, "redis_version:")
				} else if strings.HasPrefix(line, "os:") {
					os = strings.TrimPrefix(line, "os:")
				}

				if version != "" && os != "" {
					break
				}
			}

			config.Runtime.Cache = "Redis " + version + " " + os
		}

	} else {
		config.Runtime.Cache = "Internal"

		log.Println("Creating cache...")

		c := gocache.NewCache().WithMaxSize(1000).WithEvictionPolicy(gocache.LeastRecentlyUsed)
		cache = c
	}

	return nil
}

func SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	var err error

	if config.RedisEnabled() {
		err = RedisGetConnection().Set(ctx, key, value, ttl).Err()
	} else {
		if ttl == 0 {
			cache.Set(key, value)
		} else {
			cache.SetWithTTL(key, value, ttl)
		}
	}

	return err
}

func Set(key string, value interface{}) error {
	/*
		var err error

		if config.RedisEnabled() {
			err = RedisGetConnection().Set(ctx, key, value, 0).Err()
		} else {
			cache.Set(key, value)
		}

		return err
	*/
	return SetWithTTL(key, value, 0)
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
