package middleware

import (
	"context"
	"errors"

	"connectrpc.com/connect"
)

// ErrorInterceptor normalizes errors into standard connect errors
func ErrorInterceptor() connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			res, err := next(ctx, req)
			if err != nil {
				// Prevent exposing internal panic or unhandled error text directly.
				// In a real app we might check for specific custom error types here.
				var connectErr *connect.Error
				if !errors.As(err, &connectErr) {
					err = connect.NewError(connect.CodeInternal, errors.New("internal system error"))
				}
				return nil, err
			}
			return res, nil
		})
	})
}
