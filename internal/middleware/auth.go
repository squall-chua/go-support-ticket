package middleware

import (
	"context"
	"strings"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	pb "github.com/squall-chua/go-support-ticket/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*TokenInfo, error)
}

type TokenInfo struct {
	UserID string
	Scopes []string
	Roles  []string
}

type userContextKey struct{}
type roleContextKey struct{}
type tokenInfoKey struct{}

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

// ContextWithTokenInfo stores the TokenInfo in the context.
func ContextWithTokenInfo(ctx context.Context, info *TokenInfo) context.Context {
	return context.WithValue(ctx, tokenInfoKey{}, info)
}

// TokenInfoFromContext retrieves the TokenInfo from the context.
func TokenInfoFromContext(ctx context.Context) (*TokenInfo, bool) {
	info, ok := ctx.Value(tokenInfoKey{}).(*TokenInfo)
	return info, ok
}

// gRPC Authentication Interceptor
func UnaryAuthInterceptor(validator TokenValidator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract method descriptor
		methodName := strings.TrimPrefix(info.FullMethod, "/")
		parts := strings.Split(methodName, "/")
		if len(parts) != 2 {
			return handler(ctx, req)
		}

		fullName := protoreflect.FullName(parts[0] + "." + parts[1])
		desc, err := protoregistry.GlobalFiles.FindDescriptorByName(fullName)
		if err != nil {
			return handler(ctx, req)
		}

		methodDesc, ok := desc.(protoreflect.MethodDescriptor)
		if !ok {
			return handler(ctx, req)
		}

		ext := proto.GetExtension(methodDesc.Options(), pb.E_Rule)
		rule, ok := ext.(*pb.AuthRule)

		if !ok || rule == nil || (len(rule.RequiredScopes) == 0 && len(rule.RequiredRoles) == 0) {
			return handler(ctx, req)
		}

		tokenStr, err := grpcauth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
		}

		tokenInfo, err := validator.ValidateToken(ctx, tokenStr)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "token validation failed")
		}

		// Validate Scopes
		if len(rule.RequiredScopes) > 0 {
			hasScope := false
			for _, required := range rule.RequiredScopes {
				for _, provided := range tokenInfo.Scopes {
					if provided == required {
						hasScope = true
						break
					}
				}
				if hasScope {
					break
				}
			}
			if !hasScope {
				return nil, status.Errorf(codes.PermissionDenied, "missing required scope")
			}
		}

		// Validate Roles
		if len(rule.RequiredRoles) > 0 {
			hasRole := false
			for _, required := range rule.RequiredRoles {
				for _, provided := range tokenInfo.Roles {
					if provided == required {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}
			if !hasRole {
				return nil, status.Errorf(codes.PermissionDenied, "missing required role")
			}
		}

		ctx = ContextWithTokenInfo(ctx, tokenInfo)
		ctx = WithUser(ctx, tokenInfo.UserID)
		if len(tokenInfo.Roles) > 0 {
			ctx = WithRole(ctx, tokenInfo.Roles[0]) // Legacy support for single role
		}
		return handler(ctx, req)
	}
}
