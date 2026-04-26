package module

import (
	"ekhoes-server/common"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
)

// var modules map[string]Module
var modules = make(map[string]common.Module)
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
	modulesEnv := os.Getenv("EKHOES_MODULES")

	if modulesEnv == "" {
		return
	}

	log.Printf("Initializing modules...")

	ids := strings.Split(modulesEnv, ",")

	for _, id := range ids {
		id = strings.TrimSpace(id)

		m, ok := modules[id]

		if !ok {
			log.Printf("Module not found: %s", id)
			continue
		}

		fmt.Printf("\t%-10s... ", m.Name)
		success := true

		if m.InitFunc != nil {
			if err := m.InitFunc(r); err != nil {
				fmt.Println(err.Error())
				success = false
			}
		}

		if success {
			fmt.Println("OK")
			loaded = append(loaded, m.Name)
		}
	}
}

func GetLoadedModules() string {
	return strings.Join(loaded, ",")
}

func Register(m common.Module) {
	modules[m.Id] = m
}

func GetModule(id string) (common.Module, bool) {
	module, ok := modules[id]
	return module, ok
}
