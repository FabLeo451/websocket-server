package websocket

import (
	"ekhoes-server/auth"
	"ekhoes-server/utils"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketConnection struct {
	Conn         *websocket.Conn `json:"conn"`
	ConnectionId string          `json:"connectionId"`
	SessionId    string          `json:"sessionId"`
	Name         string          `json:"name"`
	Email        string          `json:"email"`
	Created      time.Time       `json:"created"`
}

var (
	connections = make(map[string]map[string]*WebsocketConnection)
	mu          sync.RWMutex
)

func GetConnections() []*WebsocketConnection {
	mu.RLock()
	defer mu.RUnlock()

	var result []*WebsocketConnection

	for _, sessionMap := range connections {
		for _, conn := range sessionMap {
			result = append(result, conn)
		}
	}

	return result
}

func GetConnectionsCount() int32 {
	mu.RLock()
	defer mu.RUnlock()

	var count int32

	for _, sessionMap := range connections {
		count += int32(len(sessionMap))
	}

	return count
}

func AddConnection(wsConn *WebsocketConnection) {
	mu.Lock()
	defer mu.Unlock()

	wsConn.ConnectionId = utils.ULID()
	wsConn.Created = time.Now().UTC()

	if _, ok := connections[wsConn.SessionId]; !ok {
		connections[wsConn.SessionId] = make(map[string]*WebsocketConnection)
	}

	connections[wsConn.SessionId][wsConn.ConnectionId] = wsConn
}

func RemoveConnection(sessionId, connectionId string) {
	mu.Lock()
	defer mu.Unlock()

	if sessionMap, ok := connections[sessionId]; ok {
		delete(sessionMap, connectionId)

		if len(sessionMap) == 0 {
			delete(connections, sessionId)
		}
	}
}

func GetWebsocketConnection(sessionId, connectionId string) *WebsocketConnection {
	mu.RLock()
	defer mu.RUnlock()

	if sessionMap, ok := connections[sessionId]; ok {
		if conn, ok := sessionMap[connectionId]; ok {
			return conn
		}
	}

	return nil
}

func onDisconnect(wsConn *WebsocketConnection) {
	RemoveConnection(wsConn.SessionId, wsConn.ConnectionId)
	wsConn.Conn.Close()
	auth.SetSessionActive(wsConn.SessionId, false)
}

func disconnect(wsConn *WebsocketConnection) {
	_ = wsConn.Conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(
			websocket.CloseServiceRestart,
			"Connection closed server side",
		),
	)
	onDisconnect(wsConn)
}

func DisconnectAll() {
	for _, sessionMap := range connections {
		for _, conn := range sessionMap {
			disconnect(conn)
		}
	}
}
