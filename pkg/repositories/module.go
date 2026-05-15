package repositories

import (
	"go.uber.org/fx"

	"nurmed/pkg/repositories/auth"
	"nurmed/pkg/repositories/products"
	"nurmed/pkg/repositories/purchases"
	"nurmed/pkg/repositories/sales"
	"nurmed/pkg/repositories/users"
	"nurmed/pkg/repositories/warehouse"
)

var Module = fx.Options(
	auth.Module,
	products.Module,
	purchases.Module,
	sales.Module,
	users.Module,
	warehouse.Module,
)
