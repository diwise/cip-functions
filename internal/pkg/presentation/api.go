package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
)

type API interface {
	Router() *chi.Mux
}

func New(ctx context.Context) API {
	api_ := &api{
		router: chi.NewRouter(),
	}

	api_.router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	api_.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})

	return api_
}

type api struct {
	router *chi.Mux
}

func (a *api) Router() *chi.Mux {
	return a.router
}
