package logging

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type Logger struct {
	inner *slog.Logger
}

func New(lvl slog.Level) *Logger {
	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: true,
		// Custom formatting
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Format layout: DD-MM-YY T HH:MM:SS
			if a.Key == slog.TimeKey {
				// Use reference date: 01 of February 15:04:05 2006
				const layout = "02-01-06T15:04:05"
				return slog.String(
					a.Key, a.Value.Time().Format(layout),
				)
			}

			// Format source: /build/history/saver.go -> saver.go
			if a.Key == slog.SourceKey {
				source, ok := a.Value.Any().(*slog.Source)
				if !ok {
					return a
				}
				// Use filepath.Base to get just the filename
				source.File = filepath.Base(source.File)
				return a
			}
			return a
		},
	}
	return &Logger{
		inner: slog.New(slog.NewTextHandler(os.Stdout, opts)),
	}
}

// With permanently adds any number of slog.Attr to logger.
func (l *Logger) With(attrs ...slog.Attr) *Logger {
	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}
	return &Logger{
		inner: l.inner.With(args...),
	}
}

// Logs at Info level
func (l *Logger) Info(msg string, attrs ...slog.Attr) {
	l.log(context.Background(), slog.LevelInfo, msg, attrs...)
}

// Logs at Debug level
func (l *Logger) Debug(msg string, attrs ...slog.Attr) {
	l.log(context.Background(), slog.LevelDebug, msg, attrs...)
}

// Logs at Error level
func (l *Logger) Error(msg string, attrs ...slog.Attr) {
	l.log(context.Background(), slog.LevelError, msg, attrs...)
}

// Logs at Error level and then panics
func (l *Logger) Panic(msg string, attrs ...slog.Attr) {
	l.log(context.Background(), slog.LevelError, msg, attrs...)
	panic(msg)
}

// Internal helper
func (l *Logger) log(
	ctx context.Context,
	level slog.Level,
	msg string,
	attrs ...slog.Attr,
) {
	if !l.inner.Enabled(ctx, level) {
		return
	}

	// Capture PC. We need to skip:
	// 0: runtime.Callers
	// 1: l.log
	// 2: l.Info / l.Debug / etc.
	// up to the actual caller:
	// 3: file.go
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	// Create the record manually
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])

	// Prepare attributes
	r.AddAttrs(attrs...)

	// Pass to handler
	_ = l.inner.Handler().Handle(ctx, r)
}
