package rest

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"nurmed/pkg/config"
	"nurmed/pkg/logger"
	"nurmed/pkg/utils"
)

var Module = fx.Options(
	fx.Invoke(
		NewRouter,
	),
)

type Params struct {
	fx.In
	Lifecycle fx.Lifecycle
	Config    config.Config
	Logger    logger.ILogger
}

func NewRouter(params Params) {

	ginRouter := gin.New()

	baseUrlV2, version := "/api/", params.Config.GetString("server.version")

	api := ginRouter.Group(baseUrlV2 + version)
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"version": version,
			})
		})
	}

	server := http.Server{
		Addr:    params.Config.GetString("server.port"),
		Handler: utils.AddCors(ginRouter, params.Config),
	}

	params.Lifecycle.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				params.Logger.Info(nil, "Application started")
				go server.ListenAndServe()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				params.Logger.Info(nil, "Application stopped")
				return server.Shutdown(ctx)
			},
		},
	)
}
