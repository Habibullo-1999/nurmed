package migration

import (
	"errors"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"nurmed/pkg/config"
	"nurmed/pkg/logger"
)

var Module = fx.Options(
	fx.Invoke(
		New,
		NewDevice,
	),
)

type Params struct {
	fx.In
	Logger logger.ILogger
	Config config.Config
}

const (
	migrationFilesPath = "file://migrations/tj"
)

func New(p Params) {
	m, err := migrate.New(migrationFilesPath, p.Config.GetString("database.migration"))
	if err != nil {
		p.Logger.Error(nil, "err from migration.New", zap.Error(err))
		os.Exit(1)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		p.Logger.Error(nil, "err from up migration", zap.Error(err))
		os.Exit(1)
	}
}

func NewDevice(p Params) {
	m, err := migrate.New("file://migrations/device", p.Config.GetString("database.device.migration"))
	if err != nil {
		p.Logger.Error(nil, "err from migration.New", zap.Error(err))
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		p.Logger.Error(nil, "err from up migration", zap.Error(err))
		os.Exit(1)
	}
}
