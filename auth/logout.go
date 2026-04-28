package auth

import (
	"encoding/json"
	"log"
	"net/http"
)

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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//sessionId, err := verifyJWT(payload.Token)
	claims, _, err := DecodeJWT(payload.Token)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		Delete(sessionId)
	}

	w.WriteHeader(http.StatusOK)
}
