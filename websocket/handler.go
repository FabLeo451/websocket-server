package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	"log"

	"ekhoes-server/auth"
	"ekhoes-server/common"
	"ekhoes-server/module"

	//"websocket-server/herenow"

	"github.com/gorilla/websocket"
)

// Struttura per il WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permetti connessioni da qualsiasi origine
		return true
	},
}

func closeOnError(conn *websocket.Conn, code int, message string) {
	_ = conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(
			code,
			message,
		),
	)
	conn.Close()
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

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading WebSocket:", err)
		return
	}

	token := ""
	sessionId := ""
	userName := "Unknown"
	userEmail := ""

	// Check if user has a token (cookie or query parameter)

	cookie, err := r.Cookie("cookie-ekhoes")
	if err == nil {
		token = cookie.Value
	} else {
		token = r.URL.Query().Get("token")
	}

	//fmt.Println("token:", token)

	if token != "" {
		sessionId, err = auth.VerifyJWT(token)

		if err != nil {
			log.Println(err)
			closeOnError(conn, websocket.ClosePolicyViolation /* 1008 */, err.Error())
			return
		}

		sess, found := auth.SetSessionActive(sessionId, true)

		if !found {
			log.Printf("Session not found in websocket connection handler: %s\n", sessionId)
			closeOnError(conn, websocket.ClosePolicyViolation /* 1008 */, "Session not found")
			return
		}

		userEmail = sess.User.Email
	}

	log.Printf("%s %s connected\n", userName, userEmail)

	AddConnection(conn, sessionId, userEmail)

	defer func() {
		RemoveConnection(sessionId)
		conn.Close()
		log.Printf("%s disconnected\n", userEmail)
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

		var msg common.Message

		err = json.Unmarshal(p, &msg)

		if err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}

		// Process message

		var reply common.Message

		m, ok := module.GetModule(msg.AppId)

		if ok {
			// Module handler

			err := m.WsHandler(msg, reply)

			if err != nil {
				log.Printf("[%s] Error processing websocket message: %s", m.Name, err)
			}
		} else {
			// Fallback

			if msg.Type == "ping" {
				now := time.Now().UTC()
				isoString := now.Format(time.RFC3339)
				payload, _ := json.Marshal(isoString)
				reply = common.Message{Type: "pong", Payload: payload}

				jsonStr, _ := json.Marshal(reply)

				if err := conn.WriteMessage(websocket.TextMessage, []byte(jsonStr)); err != nil {
					log.Println("Error writing message:", err)
					break
				}
			} else {
				log.Printf("Unhandled message from app '%s' %v\n", msg.AppId, msg)
				//reply = Message{Type: "default", Text: "Hello from websocket server"}
			}
		}

		replyBytes, err := json.Marshal(reply)

		if err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, replyBytes); err != nil {
			log.Println("Error writing message:", err)
			break
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
