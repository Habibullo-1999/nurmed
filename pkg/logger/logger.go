package logger

import (
	"context"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"nurmed/pkg/config"
	"nurmed/pkg/ctxman"
)

var Module = fx.Provide(NewLogger)

type Params struct {
	fx.In
	fx.Lifecycle
	Config config.Config
}

type ILogger interface {
	// logs with context
	Debug(ctx context.Context, log string, fields ...zapcore.Field)
	Info(ctx context.Context, log string, fields ...zapcore.Field)
	Warning(ctx context.Context, log string, fields ...zapcore.Field)
	Error(ctx context.Context, log string, fields ...zapcore.Field)
}

// NewLogger constructs a new logger.
func NewLogger(params Params) ILogger {

	level := getLevel(params.Config)

	// write syncers
	stdoutSyncer := zapcore.Lock(os.Stdout)

	// tee core
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			stdoutSyncer,
			level,
		),
		zapcore.NewCore(getEncoder(), getWriter(params.Config), level),
	)

	// get log core
	// core := zapcore.NewCore(getEncoder(), getWriter(params.Config), level)

	// create log instance with AddCaller option.
	// AddCallerSkip option - skips stack trace where log called
	log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	params.Lifecycle.Append(
		fx.Hook{
			OnStop: func(ctx context.Context) error {
				_ = log.Sync()
				return nil
			},
		},
	)

	return &logger{lg: log.Sugar(), config: params.Config}
}

type logger struct {
	lg     *zap.SugaredLogger
	config config.Config
}

// getEncoder returns Encoder
func getEncoder() zapcore.Encoder {

	var encoderConfig = zapcore.EncoderConfig{
		MessageKey: "message",

		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder,

		TimeKey:    "time",
		EncodeTime: zapcore.ISO8601TimeEncoder,

		CallerKey:    "caller",
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	return zapcore.NewConsoleEncoder(encoderConfig)
}

// getWriter returns WriteSyncer
func getWriter(config config.Config) zapcore.WriteSyncer {

	filename := config.GetString("logger.filename")
	if filename == "" {
		filename = "./app.log"
	}

	maxSize := config.GetInt("logger.maxSize")
	if maxSize == 0 {
		maxSize = 200
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename, // location of log file
		MaxSize:    maxSize,  // maximum size of log file in MBs, before it is rotated
		MaxBackups: 10,       // maximum no. of old files to retain
		MaxAge:     30,       // maximum number of days it will retain old files
		Compress:   false,    // whether to compress/archive old files
	}

	return zapcore.AddSync(lumberJackLogger)
}

func getLevel(config config.Config) zapcore.Level {
	switch config.GetString("logger.level") {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.DebugLevel
	}
}

func (l *logger) Debug(ctx context.Context, log string, fields ...zapcore.Field) {
	if ctx != nil {
		if ctxValue := ctx.Value(ctxman.RequestID); ctxValue != nil {
			fields = append(fields, zap.String(ctxman.RequestID.String(), ctxValue.(string)))
		}
	}
	l.lg.Desugar().Debug(log, fields...)
}

func (l *logger) Info(ctx context.Context, log string, fields ...zapcore.Field) {
	if ctx != nil {
		if ctxValue := ctx.Value(ctxman.RequestID); ctxValue != nil {
			fields = append(fields, zap.String(ctxman.RequestID.String(), ctxValue.(string)))
		}
	}
	l.lg.Desugar().Info(log, fields...)
}

func (l *logger) Warning(ctx context.Context, log string, fields ...zapcore.Field) {
	if ctx != nil {
		if ctxValue := ctx.Value(ctxman.RequestID); ctxValue != nil {
			fields = append(fields, zap.String(ctxman.RequestID.String(), ctxValue.(string)))
		}
	}
	l.lg.Desugar().Warn(log, fields...)
}

func (l *logger) Error(ctx context.Context, log string, fields ...zapcore.Field) {
	if ctx != nil {
		if ctxValue := ctx.Value(ctxman.RequestID); ctxValue != nil {
			fields = append(fields, zap.String(ctxman.RequestID.String(), ctxValue.(string)))
		}
	}
	l.lg.Desugar().Error(log, fields...)
}
