package herenow

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/go-chi/chi/v5"

	"ekhoes-server/common"
	"ekhoes-server/module"
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

var thisModule common.Module

func Register() {
	thisModule = common.Module{
		Id:        "herenow",
		Name:      "HereNow",
		InitFunc:  Init,
		Install:   Install,
		WsHandler: WsHandler,
	}
	module.Register(thisModule)
}

func Init(r *chi.Mux) error {

	//utils.Log(thisModule, "Initializing...")

	root := fmt.Sprintf("/%s", thisModule.Id)

	r.Route(root, func(r chi.Router) {
		r.Post("/welcome", Welcome)
		r.Post("/login", Login)

		r.Route("/hotspot", func(r chi.Router) {
			// GET /hotspot
			r.Get("/", GetHotspot)

			// POST /hotspot
			r.Post("/", PostHotspot)

			// Routes with /hotspot/{id}
			r.Route("/{id}", func(r chi.Router) {
				// GET /hotspot/{id}
				r.Get("/", GetHotspot)

				// PUT /hotspot/{id}
				r.Put("/", PutHotspot)

				// DELETE /hotspot/{id}
				r.Delete("/", DeleteHotspot)

				// POST/DELETE /hotspot/{id}/like
				r.Post("/like", LikeHotspot)
				r.Delete("/like", LikeHotspot)

				// POST /hotspot/{id}/clone
				r.Post("/clone", CloneHotspotHandler)

				// POST/DELETE /hotspot/{id}/subscription
				r.Post("/subscription", SubscribeUnsubscribeHandler)
				r.Delete("/subscription", SubscribeUnsubscribeHandler)

				// POST /hotspot/{id}/comment
				r.Get("/comments", GetCommentsHandler)

				// POST /hotspot/{id}/comment
				r.Post("/comment", PostHotspotCommentHandler)

				// DELETE /hotspot/{id}/comment/{commentId}
				r.Delete("/comment/{commentId}", DeleteHotspotCommentHandler)
			})
		})

		r.Get("/categories", GetCategoriesHandler)
		r.Get("/mysubscriptions", GetMySubscriptions)
		r.Get("/search", SearchHandler)
	})

	return nil
}
