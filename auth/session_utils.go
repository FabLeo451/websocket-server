package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"log"
)

var ctx = context.Background()

type User struct {
	Id   string `json:"id" bson:"Id"`
	Name string `json:"name" bson:"Name"`
	//Password string `json:"password" bson:"Password"`
	Email string `json:"email" bson:"Email"`
}

type Session struct {
	Id         string    `json:"id"`
	User       User      `json:"user"`
	Agent      string    `json:"agent"`
	Platform   string    `json:"platform"`
	Model      string    `json:"model"`
	DeviceName string    `json:"deviceName"`
	DeviceType string    `json:"deviceType"`
	Ip         string    `json:"ip"`
	Status     string    `json:"status"`
	Updated    time.Time `json:"updated"`
}

func CreateSession(rdb *redis.Client, session Session) (string, error) {

	//rdb := RedisGetConnection()

	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	sessionID := uuid.New().String()

	err = rdb.Set(ctx, sessionID, data, 0).Err()

	if err != nil {
		log.Fatalf("Error creating session: %v", err)
	}

	return sessionID, nil
}

func DeleteSession(rdb *redis.Client, sessionId string) error {

	result, err := rdb.Del(ctx, sessionId).Result()
	if err != nil {
		return fmt.Errorf("unable to remove key: %w", err)
	}

	if result == 0 {
		msg := fmt.Sprintf("deleting session key not found: %s", sessionId)
		return errors.New(msg)
	}

	return nil
}

func DeleteSessionsByPattern(rdb *redis.Client, pattern string) error {
	var cursor uint64
	ctx := context.Background()

	for {
		keys, newCursor, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan error: %w", err)
		}

		if len(keys) > 0 {
			if err := rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete error: %w", err)
			}
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func DeleteAllSessions(rdb *redis.Client) error {

	err := DeleteSessionsByPattern(rdb, "*")
	if err != nil {
		return fmt.Errorf("unable to remove key: %w", err)
	}

	return nil
}

func SetSessionActive(rdb *redis.Client, key string, active bool) map[string]interface{} {

	//rdb := RedisGetConnection()

	// Get session

	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		fmt.Println("Key not found")
		return nil
	} else if err != nil {
		log.Fatalf("Error in GET: %v", err)
		return nil
	} else {
		//fmt.Printf("Valore corrente: %s\n", val)
	}

	// Fase 1: Parsing del JSON
	var data map[string]interface{}
	err = json.Unmarshal([]byte(val), &data)
	if err != nil {
		panic(err)
	}

	// Fase 2: Modifica del JSON
	if active {
		data["status"] = "online"
	} else {
		data["status"] = "idle"
	}

	now := time.Now().UTC()
	isoString := now.Format(time.RFC3339)
	data["updated"] = isoString

	// Fase 3: Conversione di nuovo in stringa
	modifiedJSON, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	// Update session

	//err = rdb.Set(ctx, key, modifiedJSON, 0).Err()
	err = rdb.SetArgs(ctx, key, modifiedJSON, redis.SetArgs{
		KeepTTL: true,
	}).Err()

	if err != nil {
		log.Fatalf("Error updating: %v", err)
	}

	return data
}

func GetSessions(rdb *redis.Client, pattern string) ([]Session, error) {
	ctx := context.Background()

	var sessions []Session

	// Recupera tutte le chiavi corrispondenti al pattern
	keys, err := rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			log.Printf("Errore nel leggere chiave %s: %v", key, err)
			continue
		}

		var sess Session
		if err := json.Unmarshal([]byte(val), &sess); err != nil {
			log.Printf("Errore nel parsing JSON della chiave %s: %v", key, err)
			continue
		}

		sess.Id = key

		sessions = append(sessions, sess)
	}

	return sessions, nil
}
