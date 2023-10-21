package ctxLogger

import (
	"context"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type CtxLogger string

const (
	CTX_LOGGER = "logger"
)

func Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("ctx_logger", pflag.ExitOnError)
	fs.String("log-level", "debug", "")
	fs.BoolP("is-prod", "p", false, "")
	return fs
}

func ConfigureCtx(logger *zap.Logger, ctx context.Context) context.Context {
	return context.WithValue(ctx, CTX_LOGGER, logger) //nolint:staticcheck
}

func GetLogger(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.L()
	}
	logger := ctx.Value(CTX_LOGGER)
	if logger == nil {
		return zap.L()
	}
	return logger.(*zap.Logger)
}

func NewLoggerFromFlags() (*zap.Logger, error) {
	return NewLogger(viper.GetBool("is-prod"), viper.GetString("log-level"))
}

func NewLogger(production bool, level string) (*zap.Logger, error) {
	var conf zap.Config
	if production {
		conf = zap.NewProductionConfig()
	} else {
		conf = zap.NewDevelopmentConfig()
	}

	if err := conf.Level.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}

	return conf.Build()
}

func Debug(ctx context.Context, message string, fields ...zap.Field) {
	GetLogger(ctx).Debug(message, fields...)
}

func Info(ctx context.Context, message string, fields ...zap.Field) {
	GetLogger(ctx).Info(message, fields...)
}

func Warn(ctx context.Context, message string, fields ...zap.Field) {
	GetLogger(ctx).Warn(message, fields...)
}

func Error(ctx context.Context, message string, fields ...zap.Field) {
	GetLogger(ctx).Error(message, fields...)
}

func Fatal(ctx context.Context, message string, fields ...zap.Field) {
	GetLogger(ctx).Fatal(message, fields...)
}
