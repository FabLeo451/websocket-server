package admin

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/go-chi/chi/v5"

	"ekhoes-server/module"
	"ekhoes-server/websocket"
)

//go:embed sql
var all embed.FS

var SqlFS fs.FS

func init() {
	sub, err := fs.Sub(all, "sql")
	if err != nil {
		panic(err)
	}
	SqlFS = sub
}

var thisModule module.Module

func Register() {
	thisModule = module.Module{
		Id:          "admin",
		Name:        "Admin",
		InitFunc:    Init,
		Install:     Install,
		PostInstall: CreateAdmin,
	}
	module.Register(thisModule)
}

func Init(r *chi.Mux) error {

	root := fmt.Sprintf("/%s", thisModule.Id)

	r.Route(root, func(r chi.Router) {
		r.Post("/login", Login)

		r.Route("/ctl", func(r chi.Router) {
			r.Get("/sessions", GetSessionsHandler)
			r.Delete("/session/{id}", DeleteSessionHandler)
			r.Delete("/sessions", DeleteAllSessionsHandler)

			r.Get("/ws", websocket.GetConnectionsHandler)

			r.Get("/system", GetSystemInfo)
			r.Get("/top", TopCpuProcesses)
		})
	})

	return nil
}
