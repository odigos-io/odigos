package middlewares

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/vektah/gqlparser/v2/ast"
)

type operationInterceptor struct {
	marketplace bool
}

func OperationInterceptor() graphql.HandlerExtension {
	return &operationInterceptor{marketplace: services.CurrentLicenseProvider() == services.MarketplaceLicenseProvider}
}

func (o *operationInterceptor) InterceptField(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	if o.marketplace {
		field := graphql.GetFieldContext(ctx)
		if field != nil {
			if field.Object == "ComputePlatform" && field.Field.Name == "apiTokens" {
				return []*model.APIToken{}, nil
			}
			if field.Object == "Mutation" && field.Field.Name == "updateApiToken" {
				return nil, errors.New("this installation uses AWS Marketplace contract entitlements; an Odigos token cannot replace Marketplace activation")
			}
		}
	}
	return next(ctx)
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
