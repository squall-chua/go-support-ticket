package middleware

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

// LoggingInterceptor logs basic information about each unary RPC request.
func LoggingInterceptor() connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			
			// User tracking
			usr, _ := UserFromContext(ctx)
			
			res, err := next(ctx, req)
			
			duration := time.Since(start)

			if err != nil {
				slog.Error("RPC Error",
					slog.String("procedure", req.Spec().Procedure),
					slog.String("user", usr),
					slog.String("error", err.Error()),
					slog.Duration("latency", duration),
				)
			} else {
				slog.Info("RPC Success",
					slog.String("procedure", req.Spec().Procedure),
					slog.String("user", usr),
					slog.Duration("latency", duration),
				)
			}
			
			return res, err
		})
	})
}
