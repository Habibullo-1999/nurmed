package rest

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	authhandler "nurmed/cmd/nurmed/handlers/auth"
	restmiddleware "nurmed/cmd/nurmed/handlers/middleware"
	"nurmed/cmd/nurmed/handlers/users"
	"nurmed/internal/auth"
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
	AuthService auth.Service
	AuthHandler authhandler.Handler
	UserHandler users.Handler
}

func NewRouter(params Params) {

	ginRouter := gin.New()
	registerSwaggerRoutes(ginRouter)
	mw := restmiddleware.New(params.AuthService, params.Logger, params.Config)

	baseUrlV2, version := "/api/", params.Config.GetString("server.version")

	api := ginRouter.Group(baseUrlV2 + version)
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"version": version,
			})
		})
		authAPI := api.Group("/auth")
		{
			authAPI.POST("/login", mw.LoginRateLimitMiddleware(), params.AuthHandler.Login)
			authAPI.POST("/refresh", params.AuthHandler.Refresh)
			authAPI.POST("/logout", params.AuthHandler.Logout)
		}
		usersAPI := api.Group("/users")
		{
			usersAPI.Use(
				mw.AuthMiddleware(),
				mw.ScopeMiddleware(),
			)
			usersAPI.GET("", mw.PermissionMiddleware("users.read"), params.UserHandler.GetUsers)
			usersAPI.POST("", mw.PermissionMiddleware("users.create"), params.UserHandler.CreateUser)
		}
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
