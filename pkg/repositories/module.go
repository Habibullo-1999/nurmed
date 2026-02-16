package repositories

import (
	"go.uber.org/fx"

	"nurmed/pkg/repositories/auth"
	"nurmed/pkg/repositories/users"
)

var Module = fx.Options(
	auth.Module,
	users.Module,
)
