package herenow

import (
	"database/sql"
	"ekhoes-server/auth"
	"ekhoes-server/db"
	"ekhoes-server/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

/**
 * POST /login
 * -H "x-user-agent: Radar/1.0.0" -H "x-platform: Android" -d '{ email: "admin@hal9k.net", password: "admin" }'
 */
func Login(w http.ResponseWriter, r *http.Request) {
	var (
		credentials    auth.Credentials
		user           auth.User
		password_match bool = false
	)

	err := json.NewDecoder(r.Body).Decode(&credentials)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn := db.DB_GetConnection()

	if conn != nil {
		query, err := db.LoadSQL(SqlFS, "authenticate.sql")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rows, err := conn.Query(query, credentials.Password, credentials.Email)

		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for rows.Next() {
			_ = rows.Scan(&user.Id, &user.Name, &password_match)

			if !password_match {
				http.Error(w, "Wrong password", http.StatusUnauthorized)
				return
			}
		}

		if user.Id == "" {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		// Create session

		user.Email = credentials.Email
		sessionId, err := auth.CreateSession("hnw", credentials, user, r.RemoteAddr)

		if err != nil {
			log.Println(err)
			http.Error(w, "Error creating session", http.StatusInternalServerError)
			return
		}

		// Create token

		var expiresAt *time.Time = nil

		token, err := auth.GenerateJWT(sessionId, user.Id, credentials.Email, user.Name, "", "", expiresAt)

		if err != nil {
			log.Println(err)
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"token":"%s", "name":"%s", "id":"%s" }`, token, user.Name, user.Id)))

		utils.Log(thisModule, "%s successfully authenticated\n", credentials.Email)
	} else {
		utils.LogErr(thisModule, errors.New("Database unavailable"))
		http.Error(w, "Database unavailable", http.StatusBadRequest)
		return
	}
}
