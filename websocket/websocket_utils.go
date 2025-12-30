package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketConnection struct {
	Conn             *websocket.Conn `json:"conn"`
	SessionId        string          `json:"sessionId"`
	Email            string          `json:"email"`
	Created          time.Time       `json:"created"`
	LastActivity     string          `json:"lastActivity"`
	LastActivityTime time.Time       `json:"lastActivityTime"`
}

var (
	connections []WebsocketConnection
	mu          sync.Mutex
)

func GetConnections() []WebsocketConnection {
	mu.Lock()
	defer mu.Unlock()

	// copy to avoid extern updates
	result := make([]WebsocketConnection, len(connections))
	copy(result, connections)
	//fmt.Println(result)
	return result
}

func GetConnectionsCount() int32 {
	mu.Lock()
	defer mu.Unlock()
	return int32(len(connections))
}

func AddConnection(conn *websocket.Conn, sessionId string, userData map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()

	connections = append(connections, WebsocketConnection{
		Conn:      conn,
		SessionId: sessionId,
		Email:     userData["email"].(string),
		Created:   time.Now().UTC(),
	})

	//fmt.Println(connections)
}

func RemoveConnection(sessionId string) {
	mu.Lock()
	defer mu.Unlock()

	for i, c := range connections {
		if c.SessionId == sessionId {
			connections = append(connections[:i], connections[i+1:]...)
			return
		}
	}
}

func UpdateConnection(sessionId string, activity string) {
	mu.Lock()
	defer mu.Unlock()

	for i := range connections {
		if connections[i].SessionId == sessionId {
			connections[i].LastActivity = activity
			connections[i].LastActivityTime = time.Now().UTC()
			return
		}
	}
}
