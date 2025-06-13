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

	"github.com/golang-jwt/jwt/v5"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type CustomClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

type Host struct {
	Mem  *mem.VirtualMemoryStat
	Disk *disk.UsageStat
}

func getMetrics(w http.ResponseWriter, r *http.Request) {

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
	fmt.Printf("%s: %s %s %s\n", r.RemoteAddr, r.UserAgent(), r.Method, r.URL)
	//io.WriteString(w, "This is my website!\n")

	response, _ := json.Marshal(conf.Package)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func generateJWT(userID, email, name string) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		Email:  email,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			//ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "websocket-server",
			Subject:  userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(conf.JwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

/**
 * POST /login
 * -d '{ email: "admin@hal9k.net", password: "admin" }'
 */
func login(w http.ResponseWriter, r *http.Request) {
	var user User

	//reqDump, _ := httputil.DumpRequest(r, true)
	//fmt.Printf("REQUEST:\n%s\n", string(reqDump))

	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//db, err := DB_Connect()
	db := DB_GetConnection()

	//if err == nil {
	if db != nil {

		query := "SELECT ID, NAME, (PASSWORD = crypt($1, PASSWORD)) AS password_match FROM " + conf.DB.Schema + ".users WHERE LOWER(EMAIL) = LOWER($2) AND status = 'enabled'"

		rows, err := db.Query(query, user.Password, user.Email)

		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		id, name := "", ""
		password_match := false

		for rows.Next() {
			_ = rows.Scan(&id, &name, &password_match)

			if !password_match {
				http.Error(w, "Wrong password", http.StatusUnauthorized)
				return
			}

			token, err := generateJWT(id, user.Email, name)

			if err != nil {
				log.Println(err)
				http.Error(w, "Error generating token", http.StatusInternalServerError)
				return
			}

			user := User{
				Id:    id,
				Name:  name,
				Email: user.Email,
			}

			agent := r.Header.Get("User-Agent")
			platform := "Device" // oppure potresti dedurlo
			ip := r.RemoteAddr
			status := "idle"
			updated := time.Now().Format(time.RFC3339)

			session := Session{
				User:     user,
				Agent:    agent,
				Platform: platform,
				Ip:       ip,
				Status:   status,
				Updated:  updated,
			}

			createSession(session)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"token":"%s"}`, token)))
		}

		if id == "" {
			http.Error(w, "User not found", http.StatusUnauthorized)
		}

	} else {
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
	}

}
