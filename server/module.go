package server

import (
	"ekhoes-server/herenow"
	"log"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	Name     string
	InitFunc func(*chi.Mux) error
}

var modules map[string]Module

func init() {
	modules = map[string]Module{
		"herenow": {
			Name:     "HereNow",
			InitFunc: herenow.Init,
		},
	}
}

func InitModules(r *chi.Mux) {
	modulesEnv := os.Getenv("MODULES")

	if modulesEnv == "" {
		return
	}

	ids := strings.Split(modulesEnv, ",")

	for _, id := range ids {
		id = strings.TrimSpace(id)

		m, ok := modules[id]

		if !ok {
			log.Printf("Module not found: %s", id)
			continue
		}

		log.Printf("Initializing module %s...", m.Name)

		if err := m.InitFunc(r); err != nil {
			panic(err)
		}
	}
}
