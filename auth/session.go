package auth

import (
	"ekhoes-server/db"
	"ekhoes-server/utils"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"log"
)

type User struct {
	Id         string `json:"id" bson:"Id"`
	Name       string `json:"name" bson:"Name"`
	Email      string `json:"email" bson:"Email"`
	Roles      string `json:"roles" bson:"Roles"`
	Privileges string `json:"privileges" bson:"Privileges"`
	IsGuest    bool   `json:"isGuest" bson:"IsGuest"`
	IsUSer     bool   `json:"isUSer" bson:"IsUSer"`
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
	Created    time.Time `json:"created"`
	Updated    time.Time `json:"updated"`
}

var SessionNotFound = errors.New("session not found")

func CreateSession(appId string, session Session, ttl time.Duration) (string, error) {
	session.Created = time.Now().UTC()
	session.Updated = time.Now().UTC()

	if session.Status == "" {
		session.Status = "idle"
	}

	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	sessionId := fmt.Sprintf("ses:%s:%s", appId, utils.ULID())

	err = db.SetWithTTL(sessionId, data, ttl)

	if err != nil {
		log.Fatalf("Error creating session: %v", err)
	}

	return sessionId, nil
}

func Delete(sessionId string) error {
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

func DeleteAllSessions() error {
	err := db.DeleteByPattern("*")
	if err != nil {
		return fmt.Errorf("unable to remove key: %w", err)
	}

	return nil
}

func SetSessionActive(sessionId string, active bool) (Session, bool) {
	var session Session

	sessionStr, err := db.Get(sessionId)
	if err != nil {
		return session, false
	}

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

func GetSessions() ([]Session, error) {
	var sessions []Session

	keys, err := db.GetKeysByPattern("ses:*")
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

func GetSession(id string) (Session, error) {
	val, err := db.Get(id)

	var sess Session

	if err == db.KeyNotFound {
		return sess, SessionNotFound
	}

	err = json.Unmarshal([]byte(val), &sess)

	return sess, err
}
