package ctxLogger

import (
	"context"
	"errors"
	"go.uber.org/zap/zapcore"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type CtxLogger string

const (
	CTX_LOGGER = "logger"
)

var ignoreContextCanceled = false

var globalLogger *zap.Logger
var skip = []zap.Option{zap.AddCallerSkip(1)}

func Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("ctx_logger", pflag.ExitOnError)
	fs.String("log-level", "debug", "")
	fs.BoolP("is-prod", "p", false, "")
	fs.BoolP("ignore-context-canceled", "c", false, "")
	return fs
}

func ConfigureCtx(logger *zap.Logger, ctx context.Context) context.Context {
	return context.WithValue(ctx, CTX_LOGGER, logger) //nolint:staticcheck
}

func With(ctx context.Context, fields ...zapcore.Field) context.Context {
	if len(fields) == 0 {
		return ctx
	}
	return ConfigureCtx(GetLogger(ctx).With(fields...), ctx)
}

func GetLogger(ctx context.Context) *zap.Logger {
	if ctx == nil || ctx.Err() != nil { // Check if context is nil or closed
		if globalLogger != nil {
			return globalLogger
		}
		return zap.L()
	}
	logger := ctx.Value(CTX_LOGGER)
	if logger == nil {
		if globalLogger != nil {
			return globalLogger
		}
		return zap.L()
	}
	return logger.(*zap.Logger)
}

func NewLoggerFromFlags() (*zap.Logger, error) {
	logger, err := NewLogger(viper.GetBool("is-prod"), viper.GetString("log-level"), viper.GetBool("ignore-context-canceled"))
	if err != nil {
		return nil, err
	}
	globalLogger = logger
	return logger, nil
}

func NewLogger(production bool, level string, icc bool) (*zap.Logger, error) {
	var conf zap.Config
	if production {
		conf = zap.NewProductionConfig()
	} else {
		conf = zap.NewDevelopmentConfig()
	}

	if err := conf.Level.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}
	ignoreContextCanceled = icc
	logger, err := conf.Build()
	if err != nil {
		return nil, err
	}
	globalLogger = logger
	return logger, nil
}

func Check(ctx context.Context, lvl zapcore.Level, msg string) *zapcore.CheckedEntry {
	return GetLogger(ctx).WithOptions(skip...).Check(lvl, msg)
}

func Debug(ctx context.Context, message string, fields ...zap.Field) {
	if ignoreContextCanceled && ContextCanceled(fields...) {
		return
	}
	GetLogger(ctx).WithOptions(skip...).Debug(message, fields...)
}

func Info(ctx context.Context, message string, fields ...zap.Field) {
	if ignoreContextCanceled && ContextCanceled(fields...) {
		return
	}
	GetLogger(ctx).WithOptions(skip...).Info(message, fields...)
}

func Warn(ctx context.Context, message string, fields ...zap.Field) {
	if ignoreContextCanceled && ContextCanceled(fields...) {
		return
	}
	GetLogger(ctx).WithOptions(skip...).Warn(message, fields...)
}

func Error(ctx context.Context, message string, fields ...zap.Field) {
	if ignoreContextCanceled && ContextCanceled(fields...) {
		return
	}
	GetLogger(ctx).WithOptions(skip...).Error(message, fields...)
}

func Fatal(ctx context.Context, message string, fields ...zap.Field) {
	GetLogger(ctx).WithOptions(skip...).Fatal(message, fields...)
}

func Panic(ctx context.Context, msg string, fields ...zapcore.Field) {
	GetLogger(ctx).WithOptions(skip...).Panic(msg, fields...)
}
func WithOptions(ctx context.Context, opts ...zap.Option) context.Context {
	return ConfigureCtx(GetLogger(ctx).WithOptions(opts...), ctx)
}

func ContextCanceled(fields ...zap.Field) bool {
	for _, field := range fields {
		if field.Type != zapcore.ErrorType {
			continue
		}
		if field.Interface == nil {
			continue
		}
		err := field.Interface.(error)
		if errors.Is(err, context.Canceled) {
			return true
		}
	}
	return false
}
