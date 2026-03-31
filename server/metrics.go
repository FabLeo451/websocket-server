package server

import (
	"ekhoes-server/websocket"
	"encoding/json"
	"net/http"
)

func GetMetrics(w http.ResponseWriter, r *http.Request) {

	metrics := map[string]interface{}{
		"count": websocket.GetConnectionsCount(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}
