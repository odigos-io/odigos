package logger

import (
	"context"

	rtlog "sigs.k8s.io/controller-runtime/pkg/log"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

// FromContext returns a logger from the controller-runtime context. Use in Reconcile:
//
//	logger := k8slogger.FromContext(ctx)
//	logger.Info("reconciling")
//	logger.Debug("detail", "key", value)
//	logger.Error(err, "failed")
//
// The logger is the one set with ctrl.SetLogger(commonlogger.ToLogr()), so levels are respected.
// When the process calls SetLevel (e.g. from UI or effective config), this logger reflects the new
// level immediately because it uses the shared AtomicLevel; no restart is required.
// For request-scoped identity, add .WithValues("namespace", req.Namespace, "name", req.Name).
func FromContext(ctx context.Context) *commonlogger.ContextLogger {
	return commonlogger.WrapLogr(rtlog.FromContext(ctx))
}
