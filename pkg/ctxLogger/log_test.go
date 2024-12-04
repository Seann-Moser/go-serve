package ctxLogger

import (
	"context"
	"go.uber.org/zap"
	"testing"
)

func TestGetLogger(t *testing.T) {
	// Mock global logger
	globalLogger = zap.NewNop() // A no-op logger for testing

	t.Run("Nil Context", func(t *testing.T) {
		logger := GetLogger(nil)
		if logger != globalLogger {
			t.Errorf("Expected globalLogger, got %v", logger)
		}
	})

	t.Run("Closed Context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Close the context

		logger := GetLogger(ctx)
		if logger != globalLogger {
			t.Errorf("Expected globalLogger, got %v", logger)
		}
	})

	t.Run("Context With Logger", func(t *testing.T) {
		expectedLogger := zap.NewNop() // Mock logger
		ctx := context.WithValue(context.Background(), CTX_LOGGER, expectedLogger)

		logger := GetLogger(ctx)
		if logger != expectedLogger {
			t.Errorf("Expected context logger, got %v", logger)
		}
	})

	t.Run("Context Without Logger", func(t *testing.T) {
		ctx := context.Background()

		logger := GetLogger(ctx)
		if logger != globalLogger {
			t.Errorf("Expected globalLogger, got %v", logger)
		}
	})

	t.Run("Global Logger Nil", func(t *testing.T) {
		globalLogger = nil // Simulate nil global logger
		logger := GetLogger(nil)
		if logger != zap.L() {
			t.Errorf("Expected zap.L(), got %v", logger)
		}
	})
}
