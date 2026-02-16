package pkg

import (
	"go.uber.org/fx"

	"nurmed/pkg/config"
	"nurmed/pkg/logger"
)

var Module = fx.Options(
	config.Module,
	logger.Module,
	//migration.Module,
)
