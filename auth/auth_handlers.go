package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"websocket-server/db"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type Credentials struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Agent      string `json:"agent"`
	Platform   string `json:"platform"`
	Model      string `json:"model"`
	DeviceName string `json:"deviceName"`
	DeviceType string `json:"deviceType"`
}

func CheckAuthorization(r *http.Request) (jwt.MapClaims, error) {

	token := r.Header.Get("Authorization")

	//fmt.Printf("[createHotSpot] Authorization: %s\n", token)

	if token == "" {
		return nil, errors.New("missing Authorization header")
	}

	claims, err := DecodeJWT(token)

	if err != nil {
		return nil, errors.New("invalid token")
	}

	if claims["userId"].(string) == "" {
		return nil, errors.New("missing user id in token")
	}

	return claims, nil
}

func contains(csv string, target string) bool {
	items := strings.Split(csv, ",")
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func HasPrivilege(privileges string, target string) bool {
	return contains(privileges, target) || contains(privileges, "ek_admin")
}

/**
 * POST /login
 * -H "x-user-agent: Radar/1.0.0" -H "x-platform: Android" -d '{ email: "admin@hal9k.net", password: "admin" }'
 */
func Login(w http.ResponseWriter, r *http.Request) {

	id, name, roles, privileges := "", "", "", ""

	isGuest := r.URL.Query().Has("guest")
	nosession := r.URL.Query().Has("nosession") // Used by cli

	var credentials Credentials

	err := json.NewDecoder(r.Body).Decode(&credentials)

	//fmt.Println(credentials)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if isGuest {
		id = "dummyGuestId"
		name = credentials.Name
	} else {

		db := db.DB_GetConnection()

		if db != nil {

			//query := "SELECT ID, NAME, (PASSWORD = crypt($1, PASSWORD)) AS password_match FROM " + os.Getenv("DB_SCHEMA") + ".users WHERE LOWER(EMAIL) = LOWER($2) AND status = 'enabled'"
			query := `SELECT 
						u.id,
						u.name,
						(u.password = crypt($1, u.password)) AS password_match,
						STRING_AGG(DISTINCT ur.roles, ', ') AS roles,
						STRING_AGG(DISTINCT rp.id_privilege, ', ') AS privileges
					FROM 
						` + os.Getenv("DB_SCHEMA") + `.users u
					JOIN 
						` + os.Getenv("DB_SCHEMA") + `.user_roles ur ON u.id = ur.user_id
					LEFT JOIN 
						` + os.Getenv("DB_SCHEMA") + `.roles_privileges rp ON ur.roles = rp.id_role
					WHERE 
						LOWER(u.email) = LOWER($2)
						AND u.status = 'enabled'
					GROUP BY 
						u.id`

			rows, err := db.Query(query, credentials.Password, credentials.Email)

			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			} else if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			password_match := false

			for rows.Next() {
				_ = rows.Scan(&id, &name, &password_match, &roles, &privileges)

				if !password_match {
					http.Error(w, "Wrong password", http.StatusUnauthorized)
					return
				}

			}

			if id == "" {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

		} else {
			http.Error(w, "Database unavailable", http.StatusInternalServerError)
			return
		}
	}

	// User authenticated or guest

	sessionId := ""

	user := User{
		Id:    id,
		Name:  name,
		Email: credentials.Email,
	}

	if !nosession {

		ip := r.RemoteAddr
		status := "idle"
		updated := time.Now() //.Format(time.RFC3339)

		sess := Session{
			User:       user,
			Agent:      credentials.Agent,
			Platform:   credentials.Platform,
			Model:      credentials.Model,
			DeviceName: credentials.DeviceName,
			DeviceType: credentials.DeviceType,
			Ip:         ip,
			Status:     status,
			Updated:    updated,
		}

		sessionId, err = CreateSession(db.RedisGetConnection(), sess)

		if err != nil {
			log.Println(err)
			http.Error(w, "Error creating session", http.StatusInternalServerError)
			return
		}
	}

	var expiresAt *time.Time = nil

	if isGuest {
		t := time.Now().Add(24 * time.Hour)
		expiresAt = &t
	}

	token, err := generateJWT(sessionId, id, credentials.Email, name, roles, privileges, expiresAt)

	if err != nil {
		log.Println(err)
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"token":"%s", "name":"%s", "id":"%s" }`, token, name, id)))

	//fmt.Println(token)

	if isGuest {
		log.Printf("Guest %s entered\n", user.Name)
	} else {
		log.Printf("User %s successfully authenticated\n", user.Name)
	}

}

/**
 * POST /logout
 * -d '{ "token": "12345" }'
 */
func Logout(w http.ResponseWriter, r *http.Request) {

	type Payload struct {
		Token string `json:"token"`
	}

	var payload Payload

	//reqDump, _ := httputil.DumpRequest(r, true)
	//fmt.Printf("Request:\n%s\n", string(reqDump))

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		log.Println(err)
		return
	}
	//sessionId, err := verifyJWT(payload.Token)
	claims, err := DecodeJWT(payload.Token)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionId, ok := claims["sessionId"].(string)

	if !ok {
		log.Println("claim 'sessionId' not found or not a string")
		http.Error(w, "claim 'sessionId' not found or not a string", http.StatusInternalServerError)
		return
	}

	log.Printf("Deleting session: %s\n", sessionId)

	if sessionId != "" {
		DeleteSession(db.RedisGetConnection(), sessionId)
	}

	w.WriteHeader(http.StatusOK)
}

/**
 * GET /sessions
 */
func GetSessionsHandler(w http.ResponseWriter, r *http.Request) {

	claims, err := CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if HasPrivilege(claims["privileges"].(string), "ek_read_session") == false {
		http.Error(w, "missing required privileges", http.StatusUnauthorized)
		return
	}

	sessions, err := GetSessions(db.RedisGetConnection(), "*")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
	w.WriteHeader(http.StatusOK)
}

/**
 * DELETE /session/[id]
 */
func DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {

	claims, err := CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if HasPrivilege(claims["privileges"].(string), "ek_delete_session") == false {
		http.Error(w, "missing required privileges", http.StatusUnauthorized)
		return
	}

	sessionId := chi.URLParam(r, "id")

	err = DeleteSession(db.RedisGetConnection(), sessionId)

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Session deleted: %s\n", sessionId)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

/**
 * DELETE /sessions
 */
func DeleteAllSessionsHandler(w http.ResponseWriter, r *http.Request) {

	_, err := CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	err = DeleteAllSessions(db.RedisGetConnection())

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("All sessions deleted")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
