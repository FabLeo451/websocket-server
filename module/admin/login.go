package admin

import (
	"ekhoes-server/auth"
	"ekhoes-server/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

/**
 * POST /login
 * -H "x-user-agent: Radar/1.0.0" -H "x-platform: Android" -d '{ email: "admin@hal9k.net", password: "admin" }'
 */
func Login(w http.ResponseWriter, r *http.Request) {

	nosession := r.URL.Query().Has("nosession") // Used by cli

	var credentials auth.Credentials

	err := json.NewDecoder(r.Body).Decode(&credentials)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authRes, err := Authenticate(credentials.Email, credentials.Password)

	if err != nil {
		utils.LogErr(thisModule, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !authRes.Success {
		http.Error(w, authRes.Message, http.StatusUnauthorized)
		return
	}

	sessionId := ""

	if !nosession {
		session := auth.Session{
			User:       authRes.User,
			Agent:      credentials.Agent,
			Platform:   credentials.Platform,
			Model:      credentials.Model,
			DeviceName: credentials.DeviceName,
			DeviceType: credentials.DeviceType,
			Ip:         r.RemoteAddr,
		}
		sessionId, err = auth.CreateSession(thisModule.Id, session, 0)

		if err != nil {
			log.Println(err)
			http.Error(w, "Error creating session", http.StatusInternalServerError)
			return
		}
	}

	var expiresAt *time.Time = nil

	token, err := auth.GenerateJWT(sessionId, authRes.User.Id, credentials.Email, authRes.User.Name, authRes.User.Roles, authRes.User.Privileges, expiresAt)

	if err != nil {
		log.Println(err)
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	hostname, _ := os.Hostname()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"token":"%s", "name":"%s", "id":"%s", "hostname":"%s" }`, token, authRes.User.Name, authRes.User.Id, hostname)))

	//fmt.Println(token)

	utils.Log(thisModule, "%s successfully authenticated\n", authRes.User.Email)
}
