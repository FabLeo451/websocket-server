package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"ekhoes-server/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"log"
)

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

func CreateSession(session Session) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	sessionID := uuid.New().String()

	err = db.Set(sessionID, data)

	if err != nil {
		log.Fatalf("Error creating session: %v", err)
	}

	return sessionID, nil
}

func DeleteSession(rdb *redis.Client, sessionId string) error {
	deleted, err := db.DeleteKey(sessionId)
	if err != nil {
		return fmt.Errorf("unable to remove key: %w", err)
	}

	if !deleted {
		msg := fmt.Sprintf("session key not found for deletion: %s", sessionId)
		return errors.New(msg)
	}

	return nil
}

func DeleteAllSessions(rdb *redis.Client) error {
	err := db.DeleteByPattern("*")
	if err != nil {
		return fmt.Errorf("unable to remove key: %w", err)
	}

	return nil
}

func SetSessionActive(sessionId string, active bool) (Session, bool) {
	sessionStr, err := db.Get(sessionId)
	if err != nil {
		panic(err)
	}

	var session Session
	err = json.Unmarshal([]byte(sessionStr), &session)
	if err != nil {
		panic(err)
	}

	if active {
		session.Status = "online"
	} else {
		session.Status = "idle"
	}

	session.Updated = time.Now().UTC()

	modifiedJSON, err := json.Marshal(session)
	if err != nil {
		panic(err)
	}

	err = db.Update(sessionId, modifiedJSON)

	if err != nil {
		log.Fatalf("Error updating: %v", err)
	}

	return session, true
}

func GetSessions(rdb *redis.Client) ([]Session, error) {
	var sessions []Session

	keys, err := db.GetKeysByPattern("*")
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		val, err := db.Get(key)
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
