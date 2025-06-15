package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	w.Header().Set("Access-Control-Allow-Credentials", "true")
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

func hotspotHandler(w http.ResponseWriter, r *http.Request) {
	/*
		addCorsHeaders(w, r)

		claims, err := checkAuthorization(r)

		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if claims["userId"].(string) == "" {
			http.Error(w, "Missing user id in token", http.StatusUnauthorized)
			return
		}
	*/

	//reqDump, _ := httputil.DumpRequest(r, true)
	//fmt.Printf("Request:\n%s\n", string(reqDump))

	switch r.Method {
	case http.MethodOptions:
		//fmt.Println("OPTIONS /hotspot")
		optionsPreflight(w, r)
	case http.MethodPost:
		postHotspot(w, r)
	case http.MethodGet:
		getHotspot(w, r)
	case http.MethodDelete:
		deleteHotspot(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

/**
 * get /hotspots
 */
func getHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if claims["userId"].(string) == "" {
		http.Error(w, "Missing user id in token", http.StatusUnauthorized)
		return
	}

	userId := claims["userId"].(string)

	var hotspots []Hotspot

	db := DB_GetConnection()

	if db != nil {

		rows, err := db.Query(`SELECT id, name, owner, enabled, ST_Y(position::geometry) AS latitude, ST_X(position::geometry) AS longitude, start_time, end_time, created, updated
			FROM hn.HOT_SPOTS WHERE OWNER = $1 ORDER BY CREATED`, userId)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var h Hotspot
			err := rows.Scan(
				&h.Id, &h.Name, &h.Owner, &h.Enabled,
				&h.Position.Latitude, &h.Position.Longitude,
				&h.StartTime, &h.EndTime, &h.Created, &h.Updated,
			)
			if err != nil {
				http.Error(w, "error reading rows: "+err.Error(), http.StatusInternalServerError)
				return
			}
			hotspots = append(hotspots, h)
		}

	} else {
		http.Error(w, "database not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hotspots)
}

func createHotspot(hotspot Hotspot) (*Hotspot, error) {
	log.Printf("User '%s' creating hotspot '%s'\n", hotspot.Owner, hotspot.Name)

	id := uuid.New().String()
	db := DB_GetConnection()

	if db == nil {
		return nil, errors.New("database not available")
	}

	query := `
		INSERT INTO hn.HOT_SPOTS (
			id, name, owner, enabled, position, start_time, end_time
		) VALUES (
			$1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326), $7, $8
		)
		RETURNING created, updated
	`

	now := time.Now().UTC()
	isoString := now.Format(time.RFC3339)

	hotspot.StartTime = isoString
	hotspot.EndTime = isoString

	var created, updated time.Time

	err := db.QueryRow(query,
		id, hotspot.Name, hotspot.Owner, true,
		hotspot.Position.Latitude, hotspot.Position.Longitude,
		hotspot.StartTime, hotspot.EndTime,
	).Scan(&created, &updated)

	hotspot.Created = created.Format(time.RFC3339)
	hotspot.Updated = updated.Format(time.RFC3339)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &hotspot, nil
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
		http.Error(w, "Missing user id in token", http.StatusUnauthorized)
		return
	}

	var hotspot Hotspot

	err = json.NewDecoder(r.Body).Decode(&hotspot)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(claims)

	hotspot.Owner = claims["userId"].(string)

	newHotspot, err := createHotspot(hotspot)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newHotspot)
}

/**
 * DELETE /hotspot/id
 */
func deleteHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if claims["userId"].(string) == "" {
		http.Error(w, "Missing user id in token", http.StatusUnauthorized)
		return
	}

	userId := claims["userId"].(string)

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "Missing hotspot Id in URL", http.StatusBadRequest)
		return
	}
	hotspotId := parts[2]

	fmt.Printf("Deleting hotspot %s...\n", hotspotId)

	db := DB_GetConnection()
	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	query := `DELETE FROM hn.HOT_SPOTS WHERE id = $1 AND owner = $2`

	_, err = db.Exec(query, hotspotId, userId)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
