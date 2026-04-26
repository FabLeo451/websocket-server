package common

import (
	"encoding/json"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	Id          string
	Name        string
	InitFunc    func(*chi.Mux) error
	Install     func() error
	PostInstall func(...interface{}) error
	WsHandler   func(Message, Message) error
}

type Message struct {
	AppId   string          `json:"appId"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
