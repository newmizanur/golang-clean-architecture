package resolver

import (
	"context"

	pb "golang-clean-architecture/internal/delivery/grpc/pb"
)

// Resolver is the root resolver. Holds the gRPC client (not the usecase directly)
// so GraphQL talks to the gRPC service over the network — the BFF/gateway topology.
type Resolver struct {
	ItemClient pb.ItemServiceClient
}

type contextKeyJWT struct{}

// ContextWithJWT stores the JWT in context so resolvers can forward it to gRPC.
func ContextWithJWT(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, contextKeyJWT{}, token)
}

func jwtFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyJWT{}).(string)
	return v
}
