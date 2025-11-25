package middlewares

import "context"

const AdminOverrideHeader = "X-Odigos-Admin-Allow"

type ctxKey string

const ctxKeyAdminOverride ctxKey = "odigos_admin_override"

func WithAdminOverride(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKeyAdminOverride, true)
}

func AdminOverrideFromContext(ctx context.Context) bool {
	v, ok := ctx.Value(ctxKeyAdminOverride).(bool)
	return ok && v
}


