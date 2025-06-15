package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Hotspot struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Owner    string
	Enabled  bool `json:"enabled"`
	Position struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}

func hnMessageHandler(message Message) {

	claims, err := decodeJWT(message.Token)

	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Received message from '%s' of type '%s': %s\n", claims["name"], message.Type, message.Text)
}

func addCorsHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	if origin != "" {
		// Imposta l'origine della richiesta come origine consentita
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin") // Importante per caching corretto
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func checkAuthorization(r *http.Request) (jwt.MapClaims, error) {

	token := r.Header.Get("Authorization")

	//fmt.Printf("[createHotSpot] Authorization: %s\n", token)

	if token == "" {
		return nil, errors.New("missing Authorization header")
	}

	claims, err := decodeJWT(token)

	if err != nil {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func createHotspot(hotspot Hotspot) (string, error) {

	log.Printf("User '%s' creating hotspot '%s'\n", hotspot.Owner, hotspot.Name)

	id := uuid.New().String()

	db := DB_GetConnection()

	if db != nil {
		//fmt.Println("Creating")
		//fmt.Println(hotspot)

		query := `
			INSERT INTO hn.HOT_SPOTS (
				id, name, owner, enabled, position, start_time, end_time
			) VALUES (
				$1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326), $7, $8
			)`

		now := time.Now().UTC()
		isoString := now.Format(time.RFC3339)

		_, err := db.Query(query,
			id, hotspot.Name, hotspot.Owner, true, hotspot.Position.Latitude, hotspot.Position.Longitude, isoString, isoString,
		)

		if err != nil {
			return "", err
		}

	} else {
		return "", errors.New("database not available")
	}

	return id, nil
}

/**
 * POST /hotspot
 */
func postHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if claims["userId"].(string) == "" {
		http.Error(w, "Missing user id in token", http.StatusBadRequest)
		return
	}

	var hotspot Hotspot

	err = json.NewDecoder(r.Body).Decode(&hotspot)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(claims)

	hotspot.Owner = claims["userId"].(string)

	_, err = createHotspot(hotspot)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//w.Write([]byte(fmt.Sprintf(`{"token":"%s"}`, token)))
}
