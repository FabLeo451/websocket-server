package websocket

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"log"

	"ekhoes-server/auth"
	"ekhoes-server/module/cli"

	//"websocket-server/herenow"

	"github.com/gorilla/websocket"
)

type Message struct {
	AppId     string `json:"appId"`
	MessageId string `json:"messageId"`
	Payload   string `json:"payload"`
}

// Struttura per il WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permetti connessioni da qualsiasi origine
		return true
	},
}

func HandleConnection(w http.ResponseWriter, r *http.Request) {
	/*
		dump, err := httputil.DumpRequest(r, true) // true = include il body
		if err != nil {
			fmt.Println("Errore DumpRequest:", err)
			return
		}

		fmt.Println("===== HTTP REQUEST DUMP =====")
		fmt.Println(string(dump))
		fmt.Println("===== END REQUEST =====")
	*/
	token := ""

	// Check cookie
	//token = r.Header.Get("cookie-ekhoes")
	cookie, err := r.Cookie("cookie-ekhoes")
	if err == nil {
		token = cookie.Value
	} else {
		token = r.URL.Query().Get("token")
	}
	//fmt.Println("token:", token)

	// Read the temporary token
	if token == "" {
		token = r.URL.Query().Get("token")
	}

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

	sess, found := auth.SetSessionActive(sessionId, true)

	if !found {
		log.Printf("Session not found in websocket connection handler: %s\n", sessionId)
		_ = conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(
				websocket.ClosePolicyViolation, // 1008
				"Session not found",
			),
		)
		//conn.Close()
		return
	}

	log.Printf("%s connected\n", sess.User.Email)

	AddConnection(conn, sessionId, sess.User.Email)

	defer func() {
		RemoveConnection(sessionId)
		conn.Close()
		log.Printf("%s disconnected\n", sess.User.Email)
		auth.SetSessionActive(sessionId, false)
	}()

	for {

		// Read message (messageType is an int with value websocket.BinaryMessage or websocket.TextMessage)

		_, p, err := conn.ReadMessage()

		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
		//fmt.Println(string(p))

		// Unmarshal message

		var msg Message

		err = json.Unmarshal(p, &msg)

		if err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}

		bytes, err := base64.StdEncoding.DecodeString(msg.Payload)
		if err != nil {
			log.Println("Unable to decode payload from bas64:", err)
			continue
		}

		payload := string(bytes)

		// Process message

		var reply Message

		switch msg.AppId {
		/*
			case "here-now":
				resultPayloadStr, err := herenow.MessageHandler(user["id"].(string), msg.Type, msg.Subtype, payload)

				if err != nil {
					log.Println(err)
				} else {
					var reply Message

					encoded := base64.StdEncoding.EncodeToString([]byte(resultPayloadStr))
					reply = Message{AppId: msg.AppId, Type: msg.Type, Subtype: msg.Subtype, Payload: encoded}

					jsonStr, _ := json.Marshal(reply)

					if err := conn.WriteMessage(websocket.TextMessage, []byte(jsonStr)); err != nil {
						log.Println("Error writing message:", err)
					}
				}
		*/

		case "cli":
			_ = cli.MessageHandler(conn, "", payload)

		default:
			if msg.MessageId == "ping" {
				now := time.Now().UTC()
				isoString := now.Format(time.RFC3339)
				reply = Message{MessageId: "pong", Payload: isoString}

				jsonStr, _ := json.Marshal(reply)

				if err := conn.WriteMessage(websocket.TextMessage, []byte(jsonStr)); err != nil {
					log.Println("Error writing message:", err)
					break
				}
			} else {
				log.Printf("Unhandled message from app '%s' message id '%s': %s\n", msg.AppId, msg.MessageId, payload)
				//reply = Message{Type: "default", Text: "Hello from websocket server"}
			}
		}
	}

	log.Printf("Disonnected\n")
}

/**
 * GET /ws
 */
func GetConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if auth.HasPrivilege(claims["privileges"].(string), "ek_read_websocket") == false {
		http.Error(w, "missing required privileges", http.StatusUnauthorized)
		return
	}

	connections := GetConnections()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(connections)
	w.WriteHeader(http.StatusOK)
}
