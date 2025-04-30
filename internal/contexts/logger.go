package contexts

import (
	"context"
	"log/slog"
)

type contextKey int

const (
	slogKey contextKey = iota
	metadataKey
)

func WithSlog(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, slogKey, logger)
}

func SlogFrom(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(slogKey).(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}
