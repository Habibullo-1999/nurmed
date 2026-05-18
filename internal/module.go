package internal

import (
	"go.uber.org/fx"

	"nurmed/internal/auth"
	"nurmed/internal/company"
	"nurmed/internal/products"
	"nurmed/internal/purchases"
	"nurmed/internal/sales"
	"nurmed/internal/users"
	"nurmed/internal/warehouse"
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
