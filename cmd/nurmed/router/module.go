package router

import (
	"go.uber.org/fx"

	"nurmed/cmd/nurmed/router/rest"
)

var Module = fx.Options(
	rest.Module,
)
