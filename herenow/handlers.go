package herenow

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"websocket-server/auth"
	"websocket-server/db"

	"github.com/go-chi/chi/v5"
)

/**
 * Handler for websocket messages
 */
func MessageHandler(userId string, msgType string, subtype string, payload string) (string, error) {

	//log.Printf("Received message of type '%s/%s': %s\n", message.Type, message.Subtype, message.Text)

	switch msgType {
	case "map":

		var hotspots []Hotspot

		switch subtype {
		case "byPosition":

			var loc Location

			err := json.Unmarshal([]byte(payload), &loc)

			if err != nil {
				e := fmt.Sprintf("Error parsing location string: %v\n", err)
				return "", errors.New(e)
			}

			hotspots = getNearbyHotspot(loc.Latitude, loc.Longitude)

		case "getHotspotsByBoundaries":

			var boundaries Boundaries

			err := json.Unmarshal([]byte(payload), &boundaries)

			//fmt.Printf("%+v\n", boundaries)

			if err != nil {
				e := fmt.Sprintf("Error parsing boundaries string: %v\n", err)
				return "", errors.New(e)
			}

			hotspots = getHotspotsInBoundaries(userId, boundaries)

		default:
			e := fmt.Sprintf("Unespected subtype: %s\n", subtype)
			return "", errors.New(e)
		}

		//fmt.Printf("Hotspots found: %d\n", len(hotspots))

		jsonBytes, err := json.Marshal(hotspots)
		if err != nil {
			return "", err
		}

		jsonString := string(jsonBytes)

		return jsonString, nil

	default:
		e := fmt.Sprintf("Unespected type: %s\n", subtype)
		return "", errors.New(e)
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

/*
func checkAuthorization(r *http.Request) (jwt.MapClaims, error) {

	token := r.Header.Get("Authorization")

	//fmt.Printf("[createHotSpot] Authorization: %s\n", token)

	if token == "" {
		return nil, errors.New("missing Authorization header")
	}

	claims, err := auth.DecodeJWT(token)

	if err != nil {
		return nil, errors.New("invalid token")
	}

	if claims["userId"].(string) == "" {
		return nil, errors.New("missing user id in token")
	}

	return claims, nil
}
*/
/**
 * get /hotspot/[id]
 * If id is missing, return all hotspots owned by logged user
 */
func GetHotspot(w http.ResponseWriter, r *http.Request) {

	userId := ""
	whereCond := ""
	whereVal := ""

	//hotspotId := ""

	// Check if asking for a specific hotspot
	hotspotId := chi.URLParam(r, "id")

	//if len(parts) < 3 {
	if hotspotId == "" { // All user's hotspots
		claims, err := auth.CheckAuthorization(r)

		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		userId = claims["userId"].(string)

		whereCond = "OWNER = $2"
		whereVal = userId

	} else { // Specific hotspot

		//hotspotId = parts[2]
		whereCond = "h.id = $2"
		whereVal = hotspotId

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

	claims, err := auth.CheckAuthorization(r)

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

	_, err := auth.CheckAuthorization(r)

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

	claims, err := auth.CheckAuthorization(r)

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

	claims, err := auth.CheckAuthorization(r)

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

	_, err := auth.CheckAuthorization(r)

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

	claims, err := auth.CheckAuthorization(r)

	if err != nil {
		log.Println(err)
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
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

/**
 * GET /mysubscriptions[?count]
 */
func GetMySubscriptions(w http.ResponseWriter, r *http.Request) {

	//addCorsHeaders(w, r)

	claims, err := auth.CheckAuthorization(r)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	db := db.DB_GetConnection()

	if db == nil {
		log.Println("Error: database not available")
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	userId := claims["userId"].(string)
	//countFlag := r.URL.Query().Has("count")
	var count int16

	rows, err := db.Query(`SELECT count(1) from hn.subscriptions where user_id = $1`, userId)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		//var c Category
		err := rows.Scan(&count)

		if err != nil {
			log.Println(err)
			http.Error(w, "error reading rows: "+err.Error(), http.StatusInternalServerError)
			return
		}

		//categories = append(categories, c)
	}

	w.Write([]byte(fmt.Sprintf(`{"count":%d }`, count)))
	w.WriteHeader(http.StatusOK)
}

/**
 * GET /search?q=infinity%20hotel%20munich&format=json&limit=1
 */
func SearchHandler(w http.ResponseWriter, r *http.Request) {

	_, err := auth.CheckAuthorization(r)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	q := r.URL.Query().Get("q")

	if q == "" {
		log.Println("missing query")
		http.Error(w, "missing query", http.StatusBadRequest)
		return
	}

	result, err := Search(q)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(result)
	w.WriteHeader(http.StatusOK)
}

/**
 * POST /hotspot/{id}/comment
 */
func PostHotspotCommentHandler(w http.ResponseWriter, r *http.Request) {

	_, err := auth.CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var comment Comment

	err = json.NewDecoder(r.Body).Decode(&comment)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	comment.HotspotId = chi.URLParam(r, "id")

	insertedComment, err := AddComment(comment)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insertedComment)
	w.WriteHeader(http.StatusCreated)
}

/**
 * DELETE /hotspot/{id}/comment/{commentId}
 */
func DeleteHotspotCommentHandler(w http.ResponseWriter, r *http.Request) {

	_, err := auth.CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	commentId := chi.URLParam(r, "commentId")

	err = DeleteComment(commentId)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

/**
 * GET /hotspot/{id}/comments
 */
func GetCommentsHandler(w http.ResponseWriter, r *http.Request) {

	hotspotId := chi.URLParam(r, "id")

	limitStr := "1000000"
	offsetStr := "-1"

	if r.URL.Query().Has("limit") {
		limitStr = r.URL.Query().Get("limit")
	}

	if r.URL.Query().Has("offset") {
		offsetStr = r.URL.Query().Get("offset")
	}

	limit64, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	offset64, err := strconv.ParseInt(offsetStr, 10, 32)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit := int(limit64)
	offset := int32(offset64)

	comments, err := getComments(hotspotId, limit, offset)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comments)
}
