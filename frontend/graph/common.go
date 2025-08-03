package graph

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	gqlHandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	gqlTransport "github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/frontend/middlewares"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
)

func GetGQLHandler(ctx context.Context, logger logr.Logger, odigosMetrics *collectormetrics.OdigosMetricsConsumer) http.Handler {
	// Use "New" instead of "NewDefaultServer" for production grade.
	srv := gqlHandler.New(NewExecutableSchema(Config{
		Resolvers: &Resolver{
			MetricsConsumer: odigosMetrics,
			Logger:          logger,
		},
	}))

	srv.AddTransport(gqlTransport.GET{})
	srv.AddTransport(gqlTransport.POST{})
	srv.AddTransport(gqlTransport.Websocket{KeepAlivePingInterval: 10 * time.Second})
	srv.Use(middlewares.OperationInterceptor())
	srv.Use(extension.Introspection{}) // allows us to see documentation for the schema in the gql playground

	return srv
}

func extractJWTPayload(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	// Decode the payload (second part of the JWT)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse the payload as JSON
	var payload map[string]interface{}
	err = json.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT payload: %w", err)
	}

	return payload, nil
}
