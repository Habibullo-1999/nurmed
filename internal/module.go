package internal

import (
	"go.uber.org/fx"

	"nurmed/internal/auth"
	"nurmed/internal/sales"
	"nurmed/internal/users"
)

var Module = fx.Options(
	auth.Module,
	sales.Module,
	users.Module,
)
