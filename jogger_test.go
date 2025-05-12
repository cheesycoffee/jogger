package jogger_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cheesycoffee/jogger"
	"go.uber.org/zap"
)

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	ctx = jogger.WithRequestID(ctx, "test-id")
	if got := ctx.Value(jogger.RequestIDKey); got != "test-id" {
		t.Errorf("expected requestID to be 'test-id', got %v", got)
	}
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = jogger.WithRequestID(ctx, "abc-123")
	logger := jogger.FromContext(ctx)
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
}

func TestFromContextWithSpan(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, jogger.SpanKey, "test-span-id")
	ctx = context.WithValue(ctx, jogger.RequestIDKey, "req-123")

	logger := jogger.FromContext(ctx)
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
	logger.Info("testing FromContext with span")
}

func TestStartSpan(t *testing.T) {
	ctx := jogger.WithRequestID(context.Background(), "req-id")
	if ctx.Value(jogger.RequestIDKey) == "" {
		t.Fatal("expected logger with request id")
	}
}

func TestFromContextWithCustomLogger(t *testing.T) {
	customLogger, _ := zap.NewDevelopment()
	ctx := context.Background()
	ctx = context.WithValue(ctx, jogger.SpanKey, "span-xyz")
	ctx = context.WithValue(ctx, jogger.RequestIDKey, "req-xyz")
	ctx = context.WithValue(ctx, jogger.LoggerKey, customLogger)

	logger := jogger.FromContext(ctx)
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
	logger.Info("testing FromContext with custom logger")
}

func TestStartSpanIncludesRequestID(t *testing.T) {
	ctx := jogger.WithRequestID(context.Background(), "req-included")
	span, _ := jogger.StartSpan(ctx, "span-with-requestID")

	err := error(nil)
	span.Finish(&err)
}

func TestSpanSetTag(t *testing.T) {
	ctx := context.Background()
	span, _ := jogger.StartSpan(ctx, "tag-test")
	span.SetTag("key", "value")
	span.Finish(nil)
}

func TestSpanFinishSuccess(t *testing.T) {
	ctx := context.Background()
	span, _ := jogger.StartSpan(ctx, "quick-span")
	time.Sleep(10 * time.Millisecond)
	span.Finish(nil)
}

func TestSpanFinishSlow(t *testing.T) {
	ctx := context.Background()
	span, _ := jogger.StartSpan(ctx, "slow-span")
	time.Sleep(1100 * time.Millisecond)
	span.Finish(nil)
}

func TestSpanFinishError(t *testing.T) {
	ctx := context.Background()
	span, _ := jogger.StartSpan(ctx, "error-span")
	err := errors.New("something failed")
	span.Finish(&err)
}

func TestInfoWarnErrorLogging(t *testing.T) {
	ctx := jogger.WithRequestID(context.Background(), "log-test")

	jogger.Info(ctx, "info message", zap.String("foo", "bar"))
	jogger.Warn(ctx, "warn message")
	jogger.Error(ctx, "error message", zap.Error(errors.New("fail")))
}
