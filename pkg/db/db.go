package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"nurmed/internal/interfaces"

	"nurmed/pkg/config"
	"nurmed/pkg/logger"
)

var Module = fx.Options(
	fx.Provide(NewDBConn),
)

type Params struct {
	fx.In
	Config config.Config
	Logger logger.ILogger
}

type dbConn struct {
	config    config.Config
	dbPool    *pgxpool.Pool
	dbReplica *pgxpool.Pool
	logger    logger.ILogger
}

func NewDBConn(params Params) (interfaces.Querier, error) {

	var (
		dns    = params.Config.GetString("database.dns")
		dbPool *pgxpool.Pool
		err    error
	)

	dbPool, err = pgxpool.New(context.Background(), dns)
	if err != nil {
		params.Logger.Error(nil, fmt.Sprintf("Err on pgxpool.Connect(fraud_db_tj): %v", err))
		return nil, err
	}

	return &dbConn{
		dbPool: dbPool,
		logger: params.Logger,
		config: params.Config,
	}, nil
}

func (db *dbConn) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return db.dbPool.Exec(ctx, sql, arguments...)
}

func (db *dbConn) Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error) {
	return db.dbPool.Query(ctx, sql, optionsAndArgs...)
}

func (db *dbConn) QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row {
	return db.dbPool.QueryRow(ctx, sql, optionsAndArgs...)
}

func (db *dbConn) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.dbPool.Begin(ctx)
}

func (db *dbConn) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return db.dbPool.SendBatch(ctx, batch)
}
