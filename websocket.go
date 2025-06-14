package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"log"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var activeConnections int32

type Message struct {
	AppId string
	Type  string
	Text  string
}

// Struttura per il WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permetti connessioni da qualsiasi origine
		return true
	},
}

func updateLastAccess(userId string) {

	//LogWrite("Updating last access for %s\n", userId)

	db := DB_GetConnection()

	if db != nil {

		//now := time.Now().UTC()

		_, err := db.Exec("update "+conf.DB.Schema+".users set last_access = now(), updated = now() where id = $1", userId)

		if err != nil {
			LogWrite("%s\n", err.Error())
		}

	} else {
		LogWrite("Database unavailable\n")
	}

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

func handleConnection(w http.ResponseWriter, r *http.Request) {

	// Read the temporary token
	token := r.URL.Query().Get("token")

	//fmt.Println("Token:", token)

	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	sessionId, err := verifyJWT(token)

	if err != nil {
		log.Println("Can't decode token:", err)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading WebSocket:", err)
		return
	}
	defer conn.Close()

	// Incrementa contatore connessioni
	atomic.AddInt32(&activeConnections, 1)
	defer func() {
		// Decrementa quando la connessione si chiude
		atomic.AddInt32(&activeConnections, -1)
		//fmt.Println("Connessioni attive:", atomic.LoadInt32(&activeConnections))
	}()

	session := setSessionActive(sessionId, true)
	user := session["user"].(map[string]interface{})
	updateLastAccess(user["id"].(string))

	for {

		// Read message (messageType is an int with value websocket.BinaryMessage or websocket.TextMessage)

		_, p, err := conn.ReadMessage()

		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// Unmarshal message

		var msg Message

		err = json.Unmarshal(p, &msg)

		if err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}

		// Process message

		var reply Message

		switch msg.AppId {

		default:
			if msg.Type == "ping" {
				now := time.Now().UTC()
				isoString := now.Format(time.RFC3339)
				reply = Message{Type: "pong", Text: isoString}
			} else {
				log.Printf("Unhandled message from app '%s' of type '%s': %s\n", msg.AppId, msg.Type, msg.Text)
				//reply = Message{Type: "default", Text: "Hello from websocket server"}
			}
		}

		jsonStr, _ := json.Marshal(reply)

		if err := conn.WriteMessage(websocket.TextMessage, []byte(jsonStr)); err != nil {
			log.Println("Error writing message:", err)
			break
		}
	}

	fmt.Println("Session disconnected:", sessionId)
	setSessionActive(sessionId, false)
	updateLastAccess(user["id"].(string))
}
