package jogger

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ContextKey string

type Span struct {
	logger *zap.Logger
	start  time.Time
	fields []zap.Field
	mu     sync.Mutex
}

const (
	RequestIDKey ContextKey = "requestID"
	SpanKey      ContextKey = "currentSpan"
	LoggerKey    ContextKey = "currentLogger"
)

var baseLogger *zap.Logger

func init() {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)

	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel)

	baseLogger = zap.New(core)
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func FromContext(ctx context.Context) *zap.Logger {
	fields := []zap.Field{}

	if rid, ok := ctx.Value(RequestIDKey).(string); ok {
		fields = append(fields, zap.String("requestID", rid))
	}
	if span, ok := ctx.Value(SpanKey).(string); ok {
		fields = append(fields, zap.String("span", span))
	}

	if l, ok := ctx.Value(LoggerKey).(*zap.Logger); ok {
		return l.With(fields...)
	}

	return baseLogger.With(fields...)
}

func StartSpan(ctx context.Context, name string) (Span, context.Context) {
	requestID, _ := ctx.Value(RequestIDKey).(string)
	spanID := uuid.New().String()

	fields := []zap.Field{
		zap.String("span", name),
		zap.String("spanID", spanID),
	}

	if requestID != "" {
		fields = append(fields, zap.String("requestID", requestID))
	}

	l := baseLogger.With(fields...)

	ctx = context.WithValue(ctx, SpanKey, spanID)

	return Span{
		logger: l,
		start:  time.Now(),
	}, ctx
}

func (s *Span) SetTag(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fields = append(s.fields, zap.Any(key, value))
}

func (s *Span) Finish(err *error) {
	s.mu.Lock()
	fieldsCopy := make([]zap.Field, len(s.fields))
	copy(fieldsCopy, s.fields)
	s.mu.Unlock()

	elapsed := time.Since(s.start)
	fieldsCopy = append(fieldsCopy, zap.Duration("duration", elapsed))

	if err != nil && *err != nil {
		fieldsCopy = append(fieldsCopy, zap.Error(*err))
		s.logger.Error("span finished with error", fieldsCopy...)
	} else if elapsed > 1*time.Second {
		s.logger.Warn("span finished slowly", fieldsCopy...)
	} else {
		s.logger.Info("span finished successfully", fieldsCopy...)
	}
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Info(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Warn(msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Error(msg, fields...)
}
