package utils

import (
	"net/http"

	"github.com/rs/cors"

	"nurmed/pkg/config"
)

func AddCors(router http.Handler, cnf config.Config) http.Handler {
	// return cors.New(cors.Options{
	//	AllowedOrigins:   cnf.GetStringSlice("cors.allowedOrigins"),
	//	AllowedMethods:   []string{http.MethodDelete, http.MethodGet, http.MethodPost, http.MethodPut},
	//	AllowedHeaders:   []string{"*"},
	//	MaxAge:           10,
	//	AllowCredentials: true,
	// }).Handler(router)

	return cors.AllowAll().Handler(router)
}
