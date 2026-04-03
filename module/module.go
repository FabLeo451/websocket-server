package module

import (
	"log"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	Id          string
	Name        string
	InitFunc    func(*chi.Mux) error
	Install     func() error
	PostInstall func(...interface{}) error
}

// var modules map[string]Module
var modules = make(map[string]Module)
var loaded []string

/*
func init() {
		modules = map[string]Module{
			"herenow": {
				Name:     "Ekhoes",
				InitFunc: herenow.Init,
			},
		}
}
*/

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

		if m.InitFunc != nil {
			if err := m.InitFunc(r); err != nil {
				panic(err)
			}
		}

		loaded = append(loaded, m.Name)
	}
}

func GetLoadedModules() string {
	return strings.Join(loaded, ",")
}

func Register(m Module) {
	modules[m.Id] = m
}

func GetModule(id string) Module {
	return modules[id]
}
