package pkg

import (
	"go.uber.org/fx"

	"nurmed/pkg/config"
	"nurmed/pkg/db"
	"nurmed/pkg/logger"
	"nurmed/pkg/migration"
	"nurmed/pkg/repositories"
)

var Module = fx.Options(
	config.Module,
	logger.Module,
	db.Module,
	migration.Module,
	repositories.Module,
)
