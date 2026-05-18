package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey struct{}

var (
	base zerolog.Logger
	once sync.Once
)

// Init configures the global logger. Safe to call once at startup.
func Init(level, env string) {
	once.Do(func() {
		zerolog.TimeFieldFormat = time.RFC3339Nano

		var out io.Writer = os.Stdout
		if env == "development" {
			out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		}

		base = zerolog.New(out).
			Level(parseLevel(level)).
			With().
			Timestamp().
			Logger()
	})
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zerolog.DebugLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// With returns a logger derived from ctx, attaching trace correlation if a
// span is active.
func With(ctx context.Context) *zerolog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zerolog.Logger); ok {
		return l
	}
	l := base
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		sc := span.SpanContext()
		l = base.With().
			Str("trace_id", sc.TraceID().String()).
			Str("span_id", sc.SpanID().String()).
			Logger()
	}
	return &l
}

// Attach stores a pre-built logger on ctx so child handlers reuse the same
// field set.
func Attach(ctx context.Context, l *zerolog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// Convenience helpers — most call sites only want one of these.

func Info(ctx context.Context, msg string, kv ...any) {
	withFields(With(ctx).Info(), kv).Msg(msg)
}

func Warn(ctx context.Context, msg string, kv ...any) {
	withFields(With(ctx).Warn(), kv).Msg(msg)
}

func Error(ctx context.Context, err error, msg string, kv ...any) {
	ev := With(ctx).Error()
	if err != nil {
		ev = ev.Err(err)
	}
	withFields(ev, kv).Msg(msg)
}

func Debug(ctx context.Context, msg string, kv ...any) {
	withFields(With(ctx).Debug(), kv).Msg(msg)
}

func withFields(ev *zerolog.Event, kv []any) *zerolog.Event {
	for i := 0; i+1 < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			continue
		}
		ev = ev.Interface(k, kv[i+1])
	}
	return ev
}
