package herenow

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"websocket-server/db"

	"github.com/go-chi/chi/v5"
)

/**
 * Handler for websocket messages
 */
func hnMessageHandler(socket *websocket.Conn, userId string, message Message) {

	//log.Printf("Received message of type '%s/%s': %s\n", message.Type, message.Subtype, message.Text)

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

			hotspots = getHotspotsInBoundaries(userId, boundaries)

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

	if claims["userId"].(string) == "" {
		return nil, errors.New("missing user id in token")
	}

	return claims, nil
}

/*
func HotspotHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("%s %s\n", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodOptions:
		optionsPreflight(w, r)
	case http.MethodPost:
		PostHotspot(w, r)
	case http.MethodPut:
		putHotspot(w, r)
	case http.MethodGet:
		GetHotspot(w, r)
	case http.MethodDelete:
		deleteHotspot(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
*/

/**
 * get /hotspot/[id]
 * If id is missing, return all hotspots owned by logged user
 */
func GetHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	userId := claims["userId"].(string)

	whereCond := ""
	whereVal := ""

	hotspotId := ""

	// Check if asking for a specific hotspot
	//parts := strings.Split(r.URL.Path, "/")
	id := chi.URLParam(r, "id")

	//if len(parts) < 3 {
	if id == "" { // All user's hotspots

		whereCond = "OWNER = $2"
		whereVal = userId

	} else { // Specific hotspot

		//hotspotId = parts[2]
		whereCond = "h.id = $2"
		whereVal = id

	}

	var hotspots []Hotspot

	db := db.DB_GetConnection()

	if db != nil {

		rows, err := db.Query(
			`SELECT h.id, h.name, h.description, h.category, u.name as owner, enabled, private, ST_Y(position::geometry) AS latitude, ST_X(position::geometry) AS longitude, 
			start_time, end_time, h.created, h.updated,

			COALESCE(like_counts.total_likes, 0) AS likes,

			EXISTS (
				SELECT 1
				FROM hn.LIKES l2
				WHERE l2.hotspot_id = h.id AND l2.user_id = $1
			) AS liked_by_me,

			COALESCE(subs_counts.total_subs, 0) AS subscriptions,

			EXISTS (
				SELECT 1
				FROM hn.SUBSCRIPTIONS sub
				WHERE sub.hotspot_id = h.id AND sub.user_id = $1
			) AS subscribed,

			(h.owner = $1) AS owned_by_me

			FROM hn.HOTSPOTS h
			JOIN ekhoes.users u ON h.owner = u.id

			-- Join to count likes
			LEFT JOIN (
				SELECT hotspot_id, COUNT(*) AS total_likes
				FROM hn.LIKES
				GROUP BY hotspot_id
			) AS like_counts ON like_counts.hotspot_id = h.id

			-- Join to count subscriptions
			LEFT JOIN (
				SELECT hotspot_id, COUNT(*) AS total_subs
				FROM hn.SUBSCRIPTIONS
				GROUP BY hotspot_id
			) AS subs_counts ON subs_counts.hotspot_id = h.id

			WHERE `+whereCond+`
			AND h.owner = u.id
			ORDER BY CREATED`, userId, whereVal)

		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var h Hotspot
			err := rows.Scan(
				&h.Id, &h.Name, &h.Description, &h.Category, &h.Owner, &h.Enabled, &h.Private,
				&h.Position.Latitude, &h.Position.Longitude,
				&h.StartTime, &h.EndTime, &h.Created, &h.Updated,
				&h.Likes, &h.LikedByMe, &h.Subscriptions, &h.Subscribed, &h.OwnedByMe,
			)
			if err != nil {
				log.Println(err)
				http.Error(w, "error reading rows: "+err.Error(), http.StatusInternalServerError)
				return
			}
			hotspots = append(hotspots, h)
		}

		//fmt.Println(hotspots)

	} else {
		log.Println("Error: database not available")
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

/**
 * POST /hotspot
 */
func PostHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
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
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newHotspot)
}

/**
 * PUT /hotspot/id
 */
func PutHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	_, err := checkAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	hotspotId := chi.URLParam(r, "id")

	var hotspot Hotspot

	err = json.NewDecoder(r.Body).Decode(&hotspot)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Updating hotspot %v\n", hotspot)

	db := db.DB_GetConnection()
	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	query := `update hn.HOTSPOTS set name=$1, description=$2, category=$3, position = ST_SetSRID(ST_MakePoint($4, $5), 4326), start_time = $6, end_time = $7, enabled = $8, private = $9 WHERE id = $10`

	_, err = db.Exec(query, hotspot.Name, hotspot.Description, hotspot.Category, hotspot.Position.Longitude, hotspot.Position.Latitude, hotspot.StartTime, hotspot.EndTime, hotspot.Enabled, hotspot.Private, hotspotId)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

/**
 * DELETE /hotspot/id
 */
func DeleteHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
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

	db := db.DB_GetConnection()
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

/**
 * POST/DELETE /hotspot/{id}/like
 */
func LikeHotspot(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	hotspotId := chi.URLParam(r, "id")
	userId := claims["userId"].(string)
	IlikeIt := false

	if r.Method == http.MethodPost {
		IlikeIt = true
	}

	err = Like(hotspotId, userId, IlikeIt)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

/**
 * POST /hotspot/{id}/clone
 */
func CloneHotspotHandler(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	_, err := checkAuthorization(r)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	hotspotId := chi.URLParam(r, "id")
	//userId := claims["userId"].(string)

	err = CloneHotspot(hotspotId)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

/**
 * GET /categories
 */
func GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {

	addCorsHeaders(w, r)

	var categories []Category

	db := db.DB_GetConnection()

	if db != nil {

		rows, err := db.Query(`SELECT id, label from hn.categories order by id`)

		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var c Category
			err := rows.Scan(&c.Id, &c.Label)

			if err != nil {
				log.Println(err)
				http.Error(w, "error reading rows: "+err.Error(), http.StatusInternalServerError)
				return
			}

			categories = append(categories, c)
		}

	} else {
		log.Println("Error: database not available")
		http.Error(w, "database not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(categories)
}

/**
 * POST/DELETE /hotspot/{id}/subscription
 */
func SubscribeUnsubscribeHandler(w http.ResponseWriter, r *http.Request) {

	//addCorsHeaders(w, r)

	claims, err := checkAuthorization(r)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	hotspotId := chi.URLParam(r, "id")
	userId := claims["userId"].(string)
	subscriptionFlag := false

	if r.Method == http.MethodPost {
		subscriptionFlag = true
	}

	err = Subscribe(hotspotId, userId, subscriptionFlag)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
