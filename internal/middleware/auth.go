package middleware

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
)

type userContextKey struct{}
type roleContextKey struct{}

// WithUser adds a user ID to the context.
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userContextKey{}, userID)
}

// UserFromContext retrieves the user ID from the context.
func UserFromContext(ctx context.Context) (string, bool) {
	usr, ok := ctx.Value(userContextKey{}).(string)
	return usr, ok
}

// WithRole adds a role to the context.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleContextKey{}, role)
}

// RoleFromContext retrieves the role from the context.
func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleContextKey{}).(string)
	return role, ok
}

// AuthInterceptor acts as a simplistic auth layer for Connect handlers.
// It extracts an "authorization" header and strips "Bearer ".
func AuthInterceptor() connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token := req.Header().Get("Authorization")
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing authorization header"))
			}

			// Very naive token parsing for demonstration logic.
			// Format expected: "Bearer user_id:role"
			token = strings.TrimPrefix(token, "Bearer ")
			parts := strings.Split(token, ":")
			if len(parts) != 2 {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid authorization format"))
			}

			ctx = WithUser(ctx, parts[0])
			ctx = WithRole(ctx, parts[1])

			return next(ctx, req)
		})
	})
}
