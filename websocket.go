package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"log"

	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var ctx = context.Background()

var activeConnections int32

type Message struct {
	Type string
	Text string
}

// Struttura per il WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permetti connessioni da qualsiasi origine
		return true
	},
}

func setSessionActive(key string, active bool) map[string]interface{} {

	rdb := RedisGetConnection()

	// Get session

	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		fmt.Println("Chiave non trovata")
		return nil
	} else if err != nil {
		log.Fatalf("Errore nel GET: %v", err)
		return nil
	} else {
		//fmt.Printf("Valore corrente: %s\n", val)
	}

	// Fase 1: Parsing del JSON
	var data map[string]interface{}
	err = json.Unmarshal([]byte(val), &data)
	if err != nil {
		panic(err)
	}

	// Fase 2: Modifica del JSON
	if active {
		data["status"] = "online"
	} else {
		data["status"] = "idle"
	}

	now := time.Now().UTC()
	isoString := now.Format(time.RFC3339)
	data["updated"] = isoString

	// Fase 3: Conversione di nuovo in stringa
	modifiedJSON, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	// Update session

	//err = rdb.Set(ctx, key, modifiedJSON, 0).Err()
	err = rdb.SetArgs(ctx, key, modifiedJSON, redis.SetArgs{
		KeepTTL: true,
	}).Err()

	if err != nil {
		log.Fatalf("Errore nella modifica: %v", err)
	}

	return data
}

func updateLastAccess(userId string) {

	//LogWrite("Updating last access for %s\n", userId)

	db := DB_GetConnection()

	if db != nil {

		//now := time.Now().UTC()

		_, err := db.Exec("update api.users set last_access = now(), updated = now() where id = $1", userId)

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
			return nil, fmt.Errorf("algoritmo di firma non valido: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return "", fmt.Errorf("token non valido: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("claims non validi")
	}

	// ✅ Qui cambiamo da "sub" a "sessionId"
	sessionID, ok := claims["sessionId"].(string)
	if !ok {
		return "", fmt.Errorf("claim 'sessionId' non trovato o non è una stringa")
	}

	// Verifica scadenza se presente
	if expRaw, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(expRaw) {
			return "", fmt.Errorf("token scaduto")
		}
	}

	return sessionID, nil
}

func handleConnection(w http.ResponseWriter, r *http.Request) {

	// Read the temporary token
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	//fmt.Println("Token:", token)

	sessionID, err := verifyJWT(token)

	if err != nil {
		log.Println("Can't decode token:", err)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Errore nell'upgrade WebSocket:", err)
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

	session := setSessionActive(sessionID, true)
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

		switch msg.Type {
		case "ping":
			now := time.Now().UTC()
			isoString := now.Format(time.RFC3339)
			reply = Message{Type: "pong", Text: isoString}
		default:
			reply = Message{Type: "default", Text: "Hello from websocket server"}
		}

		jsonStr, _ := json.Marshal(reply)

		if err := conn.WriteMessage(websocket.TextMessage, []byte(jsonStr)); err != nil {
			log.Println("Error writing message:", err)
			break
		}
	}

	fmt.Println("Session disconnected:", sessionID)
	setSessionActive(sessionID, false)
	updateLastAccess(user["id"].(string))
}
