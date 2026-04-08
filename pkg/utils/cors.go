package utils

import (
	"net/http"

	"github.com/rs/cors"

	"nurmed/pkg/config"
)

func AddCors(router http.Handler, cnf config.Config) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}).Handler(router)
}
