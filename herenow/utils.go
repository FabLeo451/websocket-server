package herenow

import (
	"database/sql"
	"errors"
	"log"
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
	Private   bool     `json:"private"`
	Position  Location `json:"position"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
	Likes     int64    `json:"likes"`
	LikedByMe bool     `json:"likedByMe"`
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
 * Return hotspot with the given id
 */
func GetHotspotById(id string) *Hotspot {

	db := db.DB_GetConnection()

	if db == nil {
		log.Println("Error: database not available")
		return nil
	}

	const query = `
		SELECT id, name, owner, enabled, private,
		       ST_Y(position::geometry) AS latitude, 
		       ST_X(position::geometry) AS longitude,
			   start_time, end_time
		FROM hn.HOTSPOTS 
		WHERE id = $1;
	`

	var hotspot Hotspot

	err := db.QueryRow(query, id).Scan(
		&hotspot.Id,
		&hotspot.Name,
		&hotspot.Owner,
		&hotspot.Enabled,
		&hotspot.Private,
		&hotspot.Position.Latitude,
		&hotspot.Position.Longitude,
		&hotspot.StartTime,
		&hotspot.EndTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("No hotspot found with id: %s", id)
		} else {
			log.Println("Query error:", err)
		}
		return nil
	}

	return &hotspot
}

/**
 * Return nearby hotspots
 */
func getNearbyHotspot(latitude float64, longitude float64) []Hotspot {

	var hotspots []Hotspot

	db := db.DB_GetConnection()

	if db != nil {

		rows, err := db.Query(`SELECT id, name, owner, enabled, private, ST_Y(position::geometry) AS latitude, ST_X(position::geometry) AS longitude
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
				&h.Id, &h.Name, &h.Owner, &h.Enabled, &h.Private,
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
		SELECT id, name, owner, enabled, private,
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
			&h.Id, &h.Name, &h.Owner, &h.Enabled, &h.Private,
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

func createHotspot(hotspot Hotspot) (*Hotspot, error) {
	log.Printf("User '%s' creating hotspot '%s'\n", hotspot.Owner, hotspot.Name)

	id := uuid.New().String()
	db := db.DB_GetConnection()

	if db == nil {
		return nil, errors.New("database not available")
	}

	query := `
	INSERT INTO hn.HOTSPOTS (
		id, name, owner, enabled, position, start_time, end_time, private
	) VALUES (
		$1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326), $7, $8, $9
	)
	RETURNING created, updated`

	var created, updated time.Time

	err := db.QueryRow(query,
		id, hotspot.Name, hotspot.Owner, hotspot.Enabled,
		hotspot.Position.Longitude, hotspot.Position.Latitude,
		hotspot.StartTime, hotspot.EndTime, hotspot.Private,
	).Scan(&created, &updated)

	hotspot.Created = created.Format(time.RFC3339)
	hotspot.Updated = updated.Format(time.RFC3339)

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return &hotspot, nil
}

func Like(hotspotId string, userId string, like bool) error {

	db := db.DB_GetConnection()

	if db == nil {
		log.Println("Error: database not available")
		return errors.New("database not available")
	}

	var query string

	if like {
		query = `
			INSERT INTO hn.LIKES (hotspot_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`
	} else {
		query = `DELETE FROM hn.LIKES WHERE hotspot_id = $1 AND user_id = $2`
	}

	_, err := db.Exec(query, hotspotId, userId)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func CloneHotspot(id string) error {

	hotspot := GetHotspotById(id)

	if hotspot != nil {

		log.Printf("Duplicating %v\n", hotspot)

		hotspot.Name = "Copy of " + hotspot.Name
		createHotspot(*hotspot)

		return nil

	} else {
		return errors.New("hotspot not found")
	}
}
