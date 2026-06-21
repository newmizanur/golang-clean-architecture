package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"golang-clean-architecture/internal/config"
	"golang-clean-architecture/internal/delivery/graphql/dataloader"
	"golang-clean-architecture/internal/delivery/graphql/graph"
	"golang-clean-architecture/internal/delivery/graphql/resolver"
	pb "golang-clean-architecture/internal/delivery/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)

	grpcAddr := fmt.Sprintf("localhost:%d", viperConfig.GetInt("grpc.port"))
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	itemClient := pb.NewItemServiceClient(conn)

	rootResolver := &resolver.Resolver{
		ItemClient: itemClient,
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: rootResolver}))

	jwtMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")
			ctx := resolver.ContextWithJWT(r.Context(), token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	loaderMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := dataloader.WithLoader(r.Context())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	mux.Handle("/query", jwtMiddleware(loaderMiddleware(srv)))

	port := viperConfig.GetInt("graphql.port")
	log.Infof("GraphQL server listening on :%d (playground at http://localhost:%d/)", port, port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		log.Fatalf("GraphQL server failed: %v", err)
	}
}
