package herenow

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Boundaries struct {
	NorthEast Location `json:"northEast"`
	SouthWest Location `json:"southWest"`
}

type Hotspot struct {
	Id        string   `json:"id"`
	Name      string   `json:"name"`
	Owner     string   `json:"owner"`
	Enabled   bool     `json:"enabled"`
	Position  Location `json:"position"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
	Created   string   `json:"created"`
	Updated   string   `json:"updated"`
}

func Init() bool {
	conn := DB_ConnectKeepAlive()

	if conn == nil {
		return false
	}

	RedisConnect()

	return true
}

func hnMessageHandler(socket *websocket.Conn, message Message) {
	/*
		claims, err := decodeJWT(message.Token)

		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("Received message from '%s' of type '%s': %s\n", claims["name"], message.Type, message.Text)
	*/
	log.Printf("Received message of type '%s/%s': %s\n", message.Type, message.Subtype, message.Text)

	var reply Message

	switch message.Type {
	case "hotspots":

		var hotspots []Hotspot

		switch message.Subtype {
		case "byPosition":

			var loc Location

			err := json.Unmarshal([]byte(message.Text), &loc)

			if err != nil {
				log.Println("Error parsing location string:", err)
				return
			}

			hotspots = getNearbyHotspot(loc.Latitude, loc.Longitude)

		case "byBoundaries":

			var boundaries Boundaries

			err := json.Unmarshal([]byte(message.Text), &boundaries)

			//fmt.Println(boundaries)

			if err != nil {
				log.Println("Error parsing boundaries string:", err)
				return
			}

			hotspots = getHotspotsInBoundaries(boundaries)

		default:
			log.Printf("Unespected subtype: %s\n", message.Subtype)
			return
		}

		//fmt.Printf("Hotspots found: %d\n", len(hotspots))

		jsonBytes, err := json.Marshal(hotspots)
		if err != nil {
			log.Println("Error converting in JSON:", err)
			return
		}

		jsonString := string(jsonBytes)

		reply = Message{Type: message.Type, Text: jsonString}

		jsonStr, _ := json.Marshal(reply)

		if err := socket.WriteMessage(websocket.TextMessage, []byte(jsonStr)); err != nil {
			log.Println("Error writing message:", err)
			break
		}
	}
}

/**
 * Return nearby hotspots
 */
func getNearbyHotspot(latitude float64, longitude float64) []Hotspot {

	var hotspots []Hotspot

	db := DB_GetConnection()

	if db != nil {

		rows, err := db.Query(`SELECT id, name, owner, enabled, ST_Y(position::geometry) AS latitude, ST_X(position::geometry) AS longitude
			FROM hn.HOTSPOTS 
			WHERE ST_DWithin(
				position,
				ST_MakePoint($1, $2)::geography,
				5000  -- meters
			)
			AND NOW() BETWEEN start_time AND end_time
			AND ENABLED = true`, latitude, longitude)

		if err != nil {
			log.Println(err.Error())
			return nil
		}
		defer rows.Close()

		for rows.Next() {
			var h Hotspot
			err := rows.Scan(
				&h.Id, &h.Name, &h.Owner, &h.Enabled,
				&h.Position.Latitude, &h.Position.Longitude,
			)
			if err != nil {
				log.Println("Error reading rows: " + err.Error())
				return nil
			}
			hotspots = append(hotspots, h)
		}

		//fmt.Println(hotspots)

	} else {
		log.Println("Error: database not available")
		return nil
	}

	return hotspots
}

/**
 * Return hotspots in the given boundaries
 */
func getHotspotsInBoundaries(boundaries Boundaries) []Hotspot {
	var hotspots []Hotspot

	db := DB_GetConnection()

	if db == nil {
		log.Println("Error: database not available")
		return nil
	}

	query := `
		SELECT id, name, owner, enabled, 
		       ST_Y(position::geometry) AS latitude, 
		       ST_X(position::geometry) AS longitude
		FROM hn.HOTSPOTS 
		WHERE ST_Contains(
			ST_MakeEnvelope(
				$1, $2,  -- SW.lon, SW.lat
				$3, $4,  -- NE.lon, NE.lat
				4326     -- SRID
			),
			position::geometry
		)
		AND NOW() BETWEEN start_time AND end_time
		AND enabled = true;
	`

	rows, err := db.Query(query,
		boundaries.SouthWest.Longitude,
		boundaries.SouthWest.Latitude,
		boundaries.NorthEast.Longitude,
		boundaries.NorthEast.Latitude,
	)

	if err != nil {
		log.Println("Query error:", err.Error())
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var h Hotspot
		err := rows.Scan(
			&h.Id, &h.Name, &h.Owner, &h.Enabled,
			&h.Position.Latitude, &h.Position.Longitude,
		)
		if err != nil {
			log.Println("Error reading rows:", err.Error())
			return nil
		}
		hotspots = append(hotspots, h)
	}

	return hotspots
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

func HotspotHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("%s %s\n", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodOptions:
		optionsPreflight(w, r)
	case http.MethodPost:
		postHotspot(w, r)
	case http.MethodPut:
		putHotspot(w, r)
	case http.MethodGet:
		getHotspot(w, r)
	case http.MethodDelete:
		deleteHotspot(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

/**
 * get /hotspot/[id]
 * If id is missing, return all hotspots owned by logged user
 */
func getHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	whereCond := ""
	whereVal := ""

	hotspotId := ""

	// Check if asking for a specific hotspot
	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 3 {

		if claims["userId"].(string) == "" {
			log.Println("Error: Missing user id in token")
			http.Error(w, "Missing user id in token", http.StatusUnauthorized)
			return
		}

		userId := claims["userId"].(string)

		whereCond = "OWNER = $1"
		whereVal = userId

	} else {

		hotspotId = parts[2]
		whereCond = "id = $1"
		whereVal = hotspotId

	}

	var hotspots []Hotspot

	db := DB_GetConnection()

	if db != nil {

		rows, err := db.Query(`SELECT id, name, owner, enabled, ST_Y(position::geometry) AS latitude, ST_X(position::geometry) AS longitude, start_time, end_time, created, updated
			FROM hn.HOTSPOTS WHERE `+whereCond+` ORDER BY CREATED`, whereVal)

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

		//fmt.Println(hotspots)

	} else {
		http.Error(w, "database not available", http.StatusInternalServerError)
		return
	}

	// If searching for a specific hotspot and not found send the correct code
	if hotspotId != "" && len(hotspots) == 0 {
		w.WriteHeader(http.StatusNotFound)
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
	INSERT INTO hn.HOTSPOTS (
		id, name, owner, enabled, position, start_time, end_time
	) VALUES (
		$1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326), $7, $8
	)
	RETURNING created, updated
`
	/*
		now := time.Now().UTC()
		isoString := now.Format(time.RFC3339)

		hotspot.StartTime = isoString
		hotspot.EndTime = isoString
	*/

	var created, updated time.Time

	err := db.QueryRow(query,
		id, hotspot.Name, hotspot.Owner, true,
		hotspot.Position.Longitude, hotspot.Position.Latitude,
		hotspot.StartTime, hotspot.EndTime,
	).Scan(&created, &updated)

	hotspot.Created = created.Format(time.RFC3339)
	hotspot.Updated = updated.Format(time.RFC3339)

	if err != nil {
		log.Println(err.Error())
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

	//fmt.Println(claims)

	hotspot.Owner = claims["userId"].(string)

	log.Printf("Creating hotspot %v\n", hotspot)

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
 * PUT /hotspot/id
 */
func putHotspot(w http.ResponseWriter, r *http.Request) {

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

	//userId := claims["userId"].(string)

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "Missing hotspot Id in URL", http.StatusBadRequest)
		return
	}
	hotspotId := parts[2]

	var hotspot Hotspot

	err = json.NewDecoder(r.Body).Decode(&hotspot)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Updating hotspot %v\n", hotspot)

	db := DB_GetConnection()
	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	query := `update hn.HOTSPOTS set name=$1, position = ST_SetSRID(ST_MakePoint($2, $3), 4326), start_time = $4, end_time = $5 WHERE id = $6`

	_, err = db.Exec(query, hotspot.Name, hotspot.Position.Longitude, hotspot.Position.Latitude, hotspot.StartTime, hotspot.EndTime, hotspotId)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
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

	log.Printf("Deleting hotspot %s...\n", hotspotId)

	db := DB_GetConnection()
	if db == nil {
		log.Println("Database not available")
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	query := `DELETE FROM hn.HOTSPOTS WHERE id = $1 AND owner = $2`

	_, err = db.Exec(query, hotspotId, userId)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
