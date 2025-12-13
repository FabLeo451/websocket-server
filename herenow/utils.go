package herenow

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	Id            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	Owner         string   `json:"owner"`
	OwnedByMe     bool     `json:"ownedByMe"`
	Enabled       bool     `json:"enabled"`
	Private       bool     `json:"private"`
	Position      Location `json:"position"`
	StartTime     string   `json:"startTime"`
	EndTime       string   `json:"endTime"`
	Likes         int64    `json:"likes"`
	LikedByMe     bool     `json:"likedByMe"`
	Subscriptions int64    `json:"subscriptions"`
	Subscribed    bool     `json:"subscribed"`
	Created       string   `json:"created"`
	Updated       string   `json:"updated"`
}

type Category struct {
	Id    string `json:"value"` // Expo select control needs "value"
	Label string `json:"label"`
}

type SearchResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

type Comment struct {
	Id        int32     `json:"id"`
	HotspotId string    `json:"hotspotId"`
	UserId    string    `json:"userId"`
	UserName  string    `json:"userName"`
	Message   string    `json:"message"`
	Created   time.Time `db:"created" json:"created"`
	Updated   time.Time `db:"updated" json:"updated"`
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
		SELECT id, name, description, category, owner, enabled, private,
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
		&hotspot.Description,
		&hotspot.Category,
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

		rows, err := db.Query(`SELECT id, name, description, category, owner, enabled, private, ST_Y(position::geometry) AS latitude, ST_X(position::geometry) AS longitude
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
				&h.Id, &h.Name, &h.Description, &h.Category, &h.Owner, &h.Enabled, &h.Private,
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
func getHotspotsInBoundaries(userId string, boundaries Boundaries) []Hotspot {
	var hotspots []Hotspot

	db := db.DB_GetConnection()

	if db == nil {
		log.Println("Error: database not available")
		return nil
	}

	query := `
		SELECT id, name, description, category, owner, enabled, private,
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
		AND enabled = true
		AND (private = false OR (private = true AND owner = $5));
	`

	rows, err := db.Query(query,
		boundaries.SouthWest.Longitude,
		boundaries.SouthWest.Latitude,
		boundaries.NorthEast.Longitude,
		boundaries.NorthEast.Latitude,
		userId,
	)

	if err != nil {
		log.Println("Query error:", err.Error())
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var h Hotspot
		err := rows.Scan(
			&h.Id, &h.Name, &h.Description, &h.Category, &h.Owner, &h.Enabled, &h.Private,
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
		id, name, description, category, owner, enabled, position, start_time, end_time, private
	) VALUES (
		$1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($7, $8), 4326), $9, $10, $11
	)
	RETURNING created, updated`

	var created, updated time.Time

	err := db.QueryRow(query,
		id, hotspot.Name, hotspot.Description, hotspot.Category, hotspot.Owner, hotspot.Enabled,
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

/**
 * Add or remove a like
 */
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

/**
 * Subscribe/unsubscribe
 */
func Subscribe(hotspotId string, userId string, subscribe bool) error {

	db := db.DB_GetConnection()

	if db == nil {
		log.Println("Error: database not available")
		return errors.New("database not available")
	}

	var query string

	if subscribe {
		query = `
			INSERT INTO hn.SUBSCRIPTIONS (hotspot_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`
	} else {
		query = `DELETE FROM hn.SUBSCRIPTIONS WHERE hotspot_id = $1 AND user_id = $2`
	}

	_, err := db.Exec(query, hotspotId, userId)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

/**
 * Search
 */
func Search(query string) (*SearchResult, error) {

	if query == "" {
		return nil, fmt.Errorf("missing query")
	}

	baseURL := "https://nominatim.openstreetmap.org/search"

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("limit", "1")

	endpoint := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "HereNow/1.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	return &results[0], nil
}

/**
 * AddComment
 */
func AddComment(comment Comment) (*Comment, error) {

	db := db.DB_GetConnection()

	if db == nil {
		log.Println("Error: database not available")
		return nil, errors.New("database not available")
	}

	query := `
		INSERT INTO hn.COMMENTS (hotspot_id, user_id, message)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING id, created, updated`

	err := db.QueryRow(query, comment.HotspotId, comment.UserId, comment.Message).Scan(&comment.Id, &comment.Created, &comment.Updated)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &comment, nil
}

/**
 * DeleteComment
 */
func DeleteComment(commentId string) error {

	db := db.DB_GetConnection()

	if db == nil {
		log.Println("Error: database not available")
		return errors.New("database not available")
	}

	query := `DELETE FROM hn.COMMENTS WHERE ID = $1`

	_, err := db.Exec(query, commentId)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

/**
 * Return comments for a given hotspots
 */
func getComments(hotspotId string, limit int, offset int32) ([]Comment, error) {

	var comments []Comment

	db := db.DB_GetConnection()

	if db != nil {

		condOffset := "AND c.id < $2"

		if offset < 0 {
			condOffset = "AND c.id != $2"
		}

		query := fmt.Sprintf(`
			SELECT c.id,
				c.hotspot_id,
				c.user_id,
				u.name,
				c.message,
				c.created,
				c.updated
			FROM hn.comments c
			JOIN ekhoes.users u ON c.user_id = u.id
			WHERE c.hotspot_id = $1
			%s
			ORDER BY c.id DESC
			LIMIT $3
		`, condOffset)

		rows, err := db.Query(query, hotspotId, offset, limit)

		if err != nil {
			log.Println(err.Error())
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var c Comment
			err := rows.Scan(
				&c.Id, &c.HotspotId, &c.UserId, &c.UserName, &c.Message, &c.Created, &c.Updated,
			)
			if err != nil {
				log.Println("Error reading rows: " + err.Error())
				return nil, err
			}
			comments = append(comments, c)
		}

		//fmt.Println(hotspots)

	} else {
		log.Println("Error: database not available")
		return nil, errors.New("database not available")
	}

	return comments, nil
}
