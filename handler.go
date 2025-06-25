package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
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

type Host struct {
	Mem  *mem.VirtualMemoryStat
	Disk *disk.UsageStat
}

func getMetrics(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	metrics := map[string]interface{}{
		"activeConnections": atomic.LoadInt32(&activeConnections),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

/*
func getSystemMetrics(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Content-Type", "application/json")

		var metrics Host

		// Get disk usage

		diskUsage, err := disk.Usage(conf.HostMountPoint)

		if err != nil {
			LogWrite("%s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		metrics.Disk = diskUsage

		// Get memory usage

		v, err := mem.VirtualMemory()
		if err != nil {
			panic(err)
		}

		metrics.Mem = v

		response, _ := json.Marshal(metrics)

		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
*/
func getRoot(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//fmt.Printf("%s: %s %s %s\n", r.RemoteAddr, r.UserAgent(), r.Method, r.URL)
	//io.WriteString(w, "This is my website!\n")

	response, _ := json.Marshal(conf.Package)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func optionsPreflight(w http.ResponseWriter, r *http.Request) {

	//reqDump, _ := httputil.DumpRequest(r, true)
	//fmt.Printf("Request:\n%s\n", string(reqDump))

	origin := r.Header.Get("Origin")
	if origin != "" {
		// Imposta l'origine della richiesta come origine consentita
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin") // Importante per caching corretto
	}

	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Se è una richiesta OPTIONS, rispondi subito
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/**
 * POST /login
 * -H "x-user-agent: Radar/1.0.0" -H "x-platform: Android" -d '{ email: "admin@hal9k.net", password: "admin" }'
 */
func login(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodOptions {
		fmt.Println("OPTIONS /login")
		optionsPreflight(w, r)
		return
	}

	addCorsHeaders(w, r)

	//reqDump, _ := httputil.DumpRequest(r, true)
	//fmt.Printf("Request:\n%s\n", string(reqDump))

	id, name := "", ""

	isGuest := r.URL.Query().Has("guest")

	var credentials Credentials

	err := json.NewDecoder(r.Body).Decode(&credentials)

	//fmt.Println(credentials)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if isGuest {
		name = credentials.Name
	} else {

		db := DB_GetConnection()

		if db != nil {

			query := "SELECT ID, NAME, (PASSWORD = crypt($1, PASSWORD)) AS password_match FROM " + conf.DB.Schema + ".users WHERE LOWER(EMAIL) = LOWER($2) AND status = 'enabled'"

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
				_ = rows.Scan(&id, &name, &password_match)

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

	user := User{
		Id:    id,
		Name:  name,
		Email: credentials.Email,
	}

	ip := r.RemoteAddr
	status := "idle"
	updated := time.Now().Format(time.RFC3339)

	session := Session{
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

	sessionId, err := createSession(session)

	if err != nil {
		log.Println(err)
		http.Error(w, "Error creating session", http.StatusInternalServerError)
		return
	}

	token, err := generateJWT(sessionId, id, credentials.Email, name)

	if err != nil {
		log.Println(err)
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"token":"%s", "name":"%s"}`, token, name)))

	if isGuest {
		log.Printf("Guest %s entered\n", session.User.Name)
	} else {
		log.Printf("User %s successfully authenticated\n", session.User.Name)
	}

}

/**
 * POST /logout
 * -d '{ "token": "12345" }'
 */
func logout(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodOptions {
		optionsPreflight(w, r)
		return
	}

	origin := r.Header.Get("Origin")

	if origin != "" {
		// Imposta l'origine della richiesta come origine consentita
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin") // Importante per caching corretto
	}

	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	type Payload struct {
		Token string `json:"token"`
	}

	var payload Payload

	//reqDump, _ := httputil.DumpRequest(r, true)
	//fmt.Printf("Request:\n%s\n", string(reqDump))

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		fmt.Println(err)
	}
	sessionId, err := verifyJWT(payload.Token)

	fmt.Printf("Deleting session: %s\n", sessionId)

	if err == nil {
		if sessionId != "" {
			deleteSession(sessionId)
		}
	}

}
