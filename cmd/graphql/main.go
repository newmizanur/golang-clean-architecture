package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"golang-clean-architecture/internal/config"
	"golang-clean-architecture/internal/delivery/graphql/dataloader"
	"golang-clean-architecture/internal/delivery/graphql/graph"
	"golang-clean-architecture/internal/delivery/graphql/resolver"
	pb "golang-clean-architecture/internal/delivery/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// graphiqlHTML is a self-contained GraphiQL page that loads its assets from
// jsDelivr CDN. Unlike gqlgen's built-in playground.Handler (Apollo Sandbox),
// this pre-configures the endpoint and Authorization header so the editor is
// ready to use immediately on open.
const graphiqlHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>GraphiQL</title>
  <style>
    body { margin: 0; height: 100vh; overflow: hidden; }
    #graphiql { height: 100vh; }
  </style>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphiql@3/graphiql.min.css" />
  <script src="https://cdn.jsdelivr.net/npm/react@18/umd/react.production.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/react-dom@18/umd/react-dom.production.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/graphiql@3/graphiql.min.js"></script>
</head>
<body>
  <div id="graphiql"></div>
  <script>
    const root = ReactDOM.createRoot(document.getElementById('graphiql'));
    root.render(
      React.createElement(GraphiQL, {
        fetcher: GraphiQL.createFetcher({ url: '/query' }),
        defaultEditorToolsVisibility: true,
      })
    );
  </script>
</body>
</html>`

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)

	grpcAddr := fmt.Sprintf("%s:%d", viperConfig.GetString("grpc.host"), viperConfig.GetInt("grpc.port"))
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
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, graphiqlHTML)
	})
	mux.Handle("/query", jwtMiddleware(loaderMiddleware(srv)))

	graphqlHost := viperConfig.GetString("graphql.host")
	port := viperConfig.GetInt("graphql.port")
	addr := fmt.Sprintf("%s:%d", graphqlHost, port)
	log.Infof("GraphQL server listening on %s (GraphiQL at http://localhost:%d/)", addr, port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("GraphQL server failed: %v", err)
	}
}
