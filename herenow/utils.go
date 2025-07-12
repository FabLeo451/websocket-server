package herenow

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"websocket-server/db"
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
	conn := db.DB_ConnectKeepAlive()

	if conn == nil {
		return false
	}

	db.RedisConnect()

	return true
}

/**
 * Return nearby hotspots
 */
func getNearbyHotspot(latitude float64, longitude float64) []Hotspot {

	var hotspots []Hotspot

	db := db.DB_GetConnection()

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

	db := db.DB_GetConnection()

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

	db := db.DB_GetConnection()

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
	db := db.DB_GetConnection()

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
