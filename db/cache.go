package db

import (
	"log"
	"os"

	"github.com/TwiN/gocache/v2"

	"websocket-server/config"
)

var _cache *gocache.Cache

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
		_cache = c
	}

	return nil
}

/*
func Set(key string, value string) {
	_cache.Set(key, value, cache.DefaultExpiration)
}


func Get(key string) (string, bool) {
    value, found := c.Get("chiave")
    if found {
        fmt.Println("Valore:", value)
    }
}
*/
