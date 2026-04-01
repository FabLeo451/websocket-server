package server

import (
	"ekhoes-server/herenow"
	"log"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	Id       string
	Name     string
	InitFunc func(*chi.Mux) error
}

var modules []Module

func init() {
	modules = []Module{
		{
			Id:       "herenow",
			Name:     "HereNow",
			InitFunc: herenow.Init,
		},
	}
}

func InitModules(r *chi.Mux) {
	for _, m := range modules {

		log.Printf("Initializing module %s...", m.Name)

		if m.InitFunc == nil {
			continue;
		}

		if err := m.InitFunc(r); err != nil {
			panic(err)
		}
	}
}
