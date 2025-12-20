package websocket

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

type WebsocketConnection struct {
	conn      *websocket.Conn
	sessionId string
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
	fmt.Println(result)
	return result
}

func GetConnectionsCount() int32 {
	mu.Lock()
	defer mu.Unlock()
	return int32(len(connections))
}

func AddConnection(conn *websocket.Conn, sessionId string) {
	mu.Lock()
	defer mu.Unlock()

	connections = append(connections, WebsocketConnection{
		conn:      conn,
		sessionId: sessionId,
	})

	fmt.Println(connections)
}

func RemoveConnection(sessionId string) {
	mu.Lock()
	defer mu.Unlock()

	for i, c := range connections {
		if c.sessionId == sessionId {
			connections = append(connections[:i], connections[i+1:]...)
			return
		}
	}
}
