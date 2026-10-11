package graph

import (
	"context"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	gqlHandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	gqlTransport "github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/odigos-io/odigos/frontend/middlewares"
)

func GetGQLHandler(ctx context.Context, gqlExecutableSchema graphql.ExecutableSchema) http.Handler {
	// Use "New" instead of "NewDefaultServer" for production grade.
	srv := gqlHandler.New(gqlExecutableSchema)

	srv.AddTransport(gqlTransport.GET{})
	srv.AddTransport(gqlTransport.POST{})
	srv.AddTransport(gqlTransport.Websocket{KeepAlivePingInterval: 10 * time.Second})
	srv.Use(middlewares.OperationInterceptor())
	srv.Use(extension.Introspection{}) // allows us to see documentation for the schema in the gql playground

	return srv
}
