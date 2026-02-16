package internal

import (
	"go.uber.org/fx"

	"nurmed/internal/auth"
	"nurmed/internal/users"
)

var Module = fx.Options(
	auth.Module,
	users.Module,
)
