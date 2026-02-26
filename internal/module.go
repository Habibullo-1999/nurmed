package internal

import (
	"go.uber.org/fx"

	"nurmed/internal/auth"
	"nurmed/internal/products"
	"nurmed/internal/purchases"
	"nurmed/internal/sales"
	"nurmed/internal/users"
)

var Module = fx.Options(
	auth.Module,
	products.Module,
	purchases.Module,
	sales.Module,
	users.Module,
)
