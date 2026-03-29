package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"ekhoes-server/db"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

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

	sessions, err := GetSessions(db.RedisGetConnection())

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

	if err == nil {
		log.Printf("Session deleted: %s\n", sessionId)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

/**
 * DELETE /sessions
 */
func DeleteAllSessionsHandler(w http.ResponseWriter, r *http.Request) {

	claims, err := CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if HasPrivilege(claims["privileges"].(string), "ek_delete_session") == false {
		http.Error(w, "missing required privileges", http.StatusUnauthorized)
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
