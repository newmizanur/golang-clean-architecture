package middleware

import (
	"context"
	"strings"

	"golang-clean-architecture/internal/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

var publicMethods = map[string]bool{}

// UnaryAuthInterceptor validates a JWT on every call (not just on connection
// setup) — gRPC connections are long-lived, so a token validated once at
// connect-time could outlive its expiry for the rest of the connection's life.
func UnaryAuthInterceptor(jwtSecret string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		tokenString := strings.TrimPrefix(values[0], "Bearer ")

		userID, err := auth.ParseToken(tokenString, jwtSecret)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		ctx = context.WithValue(ctx, UserIDContextKey, userID)
		return handler(ctx, req)
	}
}
