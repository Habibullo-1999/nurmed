package rest

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	authhandler "nurmed/cmd/nurmed/handlers/auth"
	restmiddleware "nurmed/cmd/nurmed/handlers/middleware"
	productshandler "nurmed/cmd/nurmed/handlers/products"
	purchaseshandler "nurmed/cmd/nurmed/handlers/purchases"
	saleshandler "nurmed/cmd/nurmed/handlers/sales"
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
	Lifecycle    fx.Lifecycle
	Config       config.Config
	Logger       logger.ILogger
	AuthService  auth.Service
	AuthHandler  authhandler.Handler
	Products     productshandler.Handler
	Purchases    purchaseshandler.Handler
	UserHandler  users.Handler
	SalesHandler saleshandler.Handler
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

		salesAPI := api.Group("/sales")
		{
			salesAPI.Use(
				mw.AuthMiddleware(),
				mw.ScopeMiddleware(),
			)
			salesAPI.GET("/realizations", mw.PermissionMiddleware("sales.realization.read"), params.SalesHandler.GetRealizations)
			salesAPI.POST("/realizations", mw.PermissionMiddleware("sales.realization.create"), params.SalesHandler.CreateRealization)
			salesAPI.GET("/registry", mw.PermissionMiddleware("sales.registry.read"), params.SalesHandler.GetRegistry)
			salesAPI.GET("/mobile", mw.PermissionMiddleware("sales.mobile.read"), params.SalesHandler.GetMobileSales)
			salesAPI.POST("/mobile", mw.PermissionMiddleware("sales.mobile.create"), params.SalesHandler.CreateMobileSale)
			salesAPI.GET("/pos", mw.PermissionMiddleware("sales.pos.read"), params.SalesHandler.GetPOSSales)
			salesAPI.POST("/pos", mw.PermissionMiddleware("sales.pos.create"), params.SalesHandler.CreatePOSSale)
			salesAPI.GET("/returns", mw.PermissionMiddleware("sales.return.read"), params.SalesHandler.GetReturns)
			salesAPI.POST("/returns", mw.PermissionMiddleware("sales.return.create"), params.SalesHandler.CreateReturn)
		}

		purchasesAPI := api.Group("/purchases")
		{
			purchasesAPI.Use(
				mw.AuthMiddleware(),
				mw.ScopeMiddleware(),
			)
			purchasesAPI.GET("/acquisitions", mw.PermissionMiddleware("purchases.acquisition.read"), params.Purchases.GetAcquisitions)
			purchasesAPI.POST("/acquisitions", mw.PermissionMiddleware("purchases.acquisition.create"), params.Purchases.CreateAcquisition)
			purchasesAPI.GET("/registry", mw.PermissionMiddleware("purchases.registry.read"), params.Purchases.GetRegistry)
			purchasesAPI.GET("/returns", mw.PermissionMiddleware("purchases.return.read"), params.Purchases.GetReturns)
			purchasesAPI.POST("/returns", mw.PermissionMiddleware("purchases.return.create"), params.Purchases.CreateReturn)
		}

		productsAPI := api.Group("/products")
		{
			productsAPI.Use(
				mw.AuthMiddleware(),
				mw.ScopeMiddleware(),
			)
			productsAPI.GET("", mw.PermissionMiddleware("products.read"), params.Products.GetProducts)
			productsAPI.POST("", mw.PermissionMiddleware("products.create"), params.Products.CreateProduct)
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
