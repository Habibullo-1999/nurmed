package handlers

import (
	"go.uber.org/fx"

	"nurmed/cmd/nurmed/handlers/auth"
	"nurmed/cmd/nurmed/handlers/purchases"
	"nurmed/cmd/nurmed/handlers/sales"
	"nurmed/cmd/nurmed/handlers/users"
)

var Module = fx.Options(
	auth.Module,
	purchases.Module,
	sales.Module,
	users.Module,
)
