package utils

import (
	"net/http"

	"github.com/rs/cors"

	"nurmed/pkg/config"
)

func AddCors(router http.Handler, cnf config.Config) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:4200",
			"https://app.nurfarm.ru",
			"https://swag.nurfarm.ru",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"X-Refresh-Token",
		},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler(router)
}
