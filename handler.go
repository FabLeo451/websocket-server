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

type CustomClaims struct {
	SessionId string `json:"sessionId"`
	UserId    string `json:"userId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
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

func generateJWT(sessionId, userId, email, name string) (string, error) {
	claims := CustomClaims{
		SessionId: sessionId,
		UserId:    userId,
		Email:     email,
		Name:      name,
		RegisteredClaims: jwt.RegisteredClaims{
			//ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "websocket-server",
			Subject:  userId,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(conf.JwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func decodeJWT(tokenString string) (jwt.MapClaims, error) {

	var jwtSecret = []byte(conf.JwtSecret)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid sign algoritm: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

func verifyJWT(tokenString string) (string, error) {

	var jwtSecret = []byte(conf.JwtSecret)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid sign algoritm: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}

	sessionId, ok := claims["sessionId"].(string)
	if !ok {
		return "", fmt.Errorf("claim 'sessionId' not found or not a string")
	}

	// Check expiration
	if expRaw, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(expRaw) {
			return "", fmt.Errorf("token expired")
		}
	}

	return sessionId, nil
}

func optionsPreflight(w http.ResponseWriter, r *http.Request) {
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

	// Qui puoi continuare con il normale flusso della tua applicazione
	w.Write([]byte("Hello from server"))
}

/**
 * POST /login
 * -H "x-user-agent: Radar/1.0.0" -H "x-platform: Android" -d '{ email: "admin@hal9k.net", password: "admin" }'
 */
func login(w http.ResponseWriter, r *http.Request) {

	origin := r.Header.Get("Origin")

	if origin != "" {
		// Imposta l'origine della richiesta come origine consentita
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin") // Importante per caching corretto
	}

	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	var credentials Credentials

	//reqDump, _ := httputil.DumpRequest(r, true)
	//fmt.Printf("Request:\n%s\n", string(reqDump))

	id, name := "", ""

	isGuest := r.URL.Query().Has("guest")

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
	w.Write([]byte(fmt.Sprintf(`{"token":"%s"}`, token)))

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
