package websocket

import (
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"log"

	"websocket-server/auth"
	"websocket-server/db"
	"websocket-server/herenow"

	"github.com/gorilla/websocket"
)

var activeConnections int32

type Message struct {
	AppId   string
	Type    string
	Subtype string
	Token   string
	Text    string
}

// Struttura per il WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permetti connessioni da qualsiasi origine
		return true
	},
}

func GetActiveConnectionsCount() int32 {
	return atomic.LoadInt32(&activeConnections)
}

func updateLastAccess(userId string) {

	//log.Printf("Updating last access for %s\n", userId)

	db := db.DB_GetConnection()

	if db != nil {

		//now := time.Now().UTC()

		_, err := db.Exec("update "+os.Getenv("DB_SCHEMA")+".users set last_access = now(), updated = now() where id = $1", userId)

		if err != nil {
			log.Printf("%s\n", err.Error())
		}

	} else {
		log.Printf("Database unavailable\n")
	}

}

func HandleConnection(w http.ResponseWriter, r *http.Request) {

	// Read the temporary token
	token := r.URL.Query().Get("token")

	//fmt.Println("Token:", token)

	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	sessionId, err := auth.VerifyJWT(token)

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

	sess := auth.SetSessionActive(db.RedisGetConnection(), sessionId, true)

	if sess == nil {
		log.Printf("Session not found in websocket connection handler: %s\n", sessionId)
		return
	}

	user := sess["user"].(map[string]interface{})
	updateLastAccess(user["id"].(string))

	log.Printf("%s connected\n", user["name"])

	// Incrementa contatore connessioni
	atomic.AddInt32(&activeConnections, 1)
	defer func() {
		// Decrementa quando la connessione si chiude
		atomic.AddInt32(&activeConnections, -1)
		//fmt.Println("Connessioni attive:", atomic.LoadInt32(&activeConnections))
	}()

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
		case "here-now":
			jsonStr, err := herenow.MessageHandler(user["id"].(string), msg.Type, msg.Subtype, msg.Text)
							if err != nil {
					log.Println(err)
					break
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(jsonStr)); err != nil {
					log.Println("Error writing message:", err)
					break
				}

		default:
			if msg.Type == "ping" {
				now := time.Now().UTC()
				isoString := now.Format(time.RFC3339)
				reply = Message{Type: "pong", Text: isoString}

				jsonStr, _ := json.Marshal(reply)

				if err := conn.WriteMessage(websocket.TextMessage, []byte(jsonStr)); err != nil {
					log.Println("Error writing message:", err)
					break
				}
			} else {
				log.Printf("Unhandled message from app '%s' of type '%s': %s\n", msg.AppId, msg.Type, msg.Text)
				//reply = Message{Type: "default", Text: "Hello from websocket server"}
			}
		}

	}

	log.Printf("%s disconnected\n", user["name"])

	auth.SetSessionActive(db.RedisGetConnection(), sessionId, false)
	updateLastAccess(user["id"].(string))
}
