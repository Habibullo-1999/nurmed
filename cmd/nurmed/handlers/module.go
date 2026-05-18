package handlers

import (
	"go.uber.org/fx"

	"nurmed/cmd/nurmed/handlers/auth"
	"nurmed/cmd/nurmed/handlers/company"
	"nurmed/cmd/nurmed/handlers/products"
	"nurmed/cmd/nurmed/handlers/purchases"
	"nurmed/cmd/nurmed/handlers/sales"
	"nurmed/cmd/nurmed/handlers/users"
	"nurmed/cmd/nurmed/handlers/warehouse"
)

var Module = fx.Options(
	auth.Module,
	company.Module,
	products.Module,
	purchases.Module,
	sales.Module,
	users.Module,
	warehouse.Module,
)
