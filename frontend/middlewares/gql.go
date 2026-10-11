package middlewares

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/vektah/gqlparser/v2/ast"
)

type operationInterceptor struct{}

func OperationInterceptor() graphql.HandlerExtension {
	return &operationInterceptor{}
}

func (o *operationInterceptor) ExtensionName() string {
	return "OperationInterceptor"
}

func (o *operationInterceptor) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (o *operationInterceptor) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	operationCtx := graphql.GetOperationContext(ctx)

	//  Check if the operation is a mutation, then check if the UI is in readonly mode
	if operationCtx.Operation.Operation == ast.Mutation {
		if services.IsReadonlyMode(ctx) && !AdminOverrideFromContext(ctx) {
			return func(ctx context.Context) *graphql.Response {
				return graphql.ErrorResponse(ctx, "%s", services.ErrorIsReadonly.Error())
			}
		}
		// Note: CSRF validation is handled by the Gin middleware before reaching GraphQL
	}

	return next(ctx)
}
