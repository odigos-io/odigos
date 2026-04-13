package logger

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	rtlog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	atom     = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	instance *zap.Logger
)

// Init builds the logger on the shared atom. Call once at process startup.
// levelStr: "error"|"warn"|"info"|"debug". component is the process name (e.g. "scheduler", "autoscaler")
// and is added to every log line as the "component" field.
//
// Options (all optional, backward-compatible):
//   - WithRotation(cfg): write to a rotated file instead of stdout (off by default).
//   - WithStdout(true): keep stdout when rotation is enabled (for tee-style logging).
//
// For non-Kubernetes binaries (e.g. vm-agent): use Init then LoggerCompat() or ToLogr()/WrapLogr for logging.
// Do not use FromContext (it requires controller-runtime context); SetLevel still applies to the shared atom.
func Init(levelStr string, component string, opts ...Option) *zap.Logger {
	atom.SetLevel(ParseLevel(levelStr))

	cfg := initConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderCfg)

	ws := buildWriteSyncer(cfg)

	core := zapcore.NewCore(encoder, ws, atom)
	instance = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if component != "" {
		instance = instance.With(zap.String("component", component))
	}
	zap.ReplaceGlobals(instance)
	return instance
}

// buildWriteSyncer returns the appropriate write syncer(s) based on initConfig.
func buildWriteSyncer(cfg initConfig) zapcore.WriteSyncer {
	if cfg.rotation == nil {
		return zapcore.AddSync(os.Stdout)
	}

	fileSyncer := zapcore.AddSync(cfg.rotation.toLumberjack())
	if cfg.stdout {
		return zapcore.NewMultiWriteSyncer(fileSyncer, zapcore.AddSync(os.Stdout))
	}
	return fileSyncer
}

// UpdateInstance must be called after bridge.AttachToZapLogger wraps the logger.
func UpdateInstance(zl *zap.Logger) {
	instance = zl
	zap.ReplaceGlobals(instance)
}

func Logger() *zap.Logger          { return instance }
func AtomicLevel() zap.AtomicLevel { return atom }

// SetLevel sets the global log level at runtime. Empty string is a no-op (keeps current level).
// It updates the shared AtomicLevel, so all existing loggers (FromContext, LoggerCompat, ToLogr)
// immediately respect the new level without process restart.
func SetLevel(lvl string) {
	if lvl != "" {
		atom.SetLevel(ParseLevel(lvl))
	}
}

func CurrentLevel() string { return atom.Level().String() }

// ToLogr returns a logr.Logger for ctrl.SetLogger / klog.SetLogger.
func ToLogr() logr.Logger {
	if instance != nil {
		return zapr.NewLogger(instance)
	}
	return logr.Discard()
}

// ContextLogger wraps logr.Logger with a level-based API (Info, Debug, Error).
// Use FromContext(ctx) in reconcilers so you can call logger.Info/Debug/Error
// instead of logr's V(n).Info. Matches the style of LoggerCompat().
//
// All ContextLoggers that come from FromContext(ctx) or WrapLogr(l) use the same
// underlying logr as ctrl.SetLogger(commonlogger.ToLogr()), so .Debug() respects
// the process log level (ODIGOS_LOG_LEVEL / SetLevel). Pass a logr from context
// or from mgr.GetLogger() so it is our level-controlled zap; do not wrap logr.Discard()
// in production code.
type ContextLogger struct {
	logr.Logger
}

// FromContext returns a logger from the controller-runtime context. Use in Reconcile:
//
//	logger := commonlogger.FromContext(ctx)
//	logger.Info("reconciling")
//	logger.Debug("detail", "key", value)
//	logger.Error(err, "failed")
//
// The logger is the one set with ctrl.SetLogger(commonlogger.ToLogr()), so levels are respected.
// When the process calls SetLevel (e.g. from UI or effective config), this logger reflects the new
// level immediately because it uses the shared AtomicLevel; no restart is required.
// For request-scoped identity, add .WithValues("namespace", req.Namespace, "name", req.Name).
func FromContext(ctx context.Context) *ContextLogger {
	return &ContextLogger{Logger: rtlog.FromContext(ctx).WithCallDepth(1)}
}

// WrapLogr wraps a logr.Logger (e.g. ctrl.Log.WithName("...") or mgr.GetLogger().WithName("..."))
// so you can use .Debug(). Use a logr that was set via ctrl.SetLogger(commonlogger.ToLogr())
// so the underlying zap level (atom) is used; e.g. mgr.GetLogger() is that logger when the
// manager was created after SetLogger.
func WrapLogr(l logr.Logger) *ContextLogger {
	return &ContextLogger{Logger: l.WithCallDepth(1)}
}

// Logr returns the underlying logr.Logger for use with APIs that require logr.Logger (e.g. some libraries).
func (c *ContextLogger) Logr() logr.Logger {
	if c == nil {
		return logr.Discard()
	}
	return c.Logger
}

// Info logs at info level.
func (c *ContextLogger) Info(msg string, keysAndValues ...interface{}) {
	if c != nil {
		c.Logger.Info(msg, keysAndValues...)
	}
}

// Error logs at error level (logr-style: error first, then message).
func (c *ContextLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if c != nil {
		c.Logger.Error(err, msg, keysAndValues...)
	}
}

// Debug logs at debug level. Only emitted when log level is debug.
func (c *ContextLogger) Debug(msg string, keysAndValues ...interface{}) {
	if c != nil {
		c.Logger.V(1).Info(msg, keysAndValues...)
	}
}

// WithName returns a new ContextLogger with the given name (subsystem).
func (c *ContextLogger) WithName(name string) *ContextLogger {
	if c == nil {
		return &ContextLogger{Logger: logr.Discard()}
	}
	return &ContextLogger{Logger: c.Logger.WithName(name)}
}

// WithValues returns a new ContextLogger with the given key/value pairs.
func (c *ContextLogger) WithValues(keysAndValues ...interface{}) *ContextLogger {
	if c == nil {
		return &ContextLogger{Logger: logr.Discard()}
	}
	return &ContextLogger{Logger: c.Logger.WithValues(keysAndValues...)}
}

// OdigosLogger is a slog-style logger (Info(msg, k, v...)) for backward compatibility.
// Use LoggerCompat() to get it; Logger() returns raw *zap.Logger per plan.
type OdigosLogger struct {
	sugared *zap.SugaredLogger
}

// LoggerCompat returns a logger with slog-style API. Use for existing .Info(msg, k, v) call sites.
// AddCallerSkip(1) skips the OdigosLogger wrapper frame so the caller field points to the
// actual call site instead of logger/logger.go.
func LoggerCompat() *OdigosLogger {
	if instance != nil {
		return &OdigosLogger{sugared: instance.WithOptions(zap.AddCallerSkip(1)).Sugar()}
	}
	return &OdigosLogger{sugared: zap.NewNop().Sugar()}
}

func (l *OdigosLogger) With(keysAndValues ...interface{}) *OdigosLogger {
	if l == nil || l.sugared == nil {
		return LoggerCompat()
	}
	return &OdigosLogger{sugared: l.sugared.With(keysAndValues...)}
}

func (l *OdigosLogger) Info(msg string, keysAndValues ...interface{}) {
	if l != nil && l.sugared != nil {
		l.sugared.Infow(msg, keysAndValues...)
	}
}

func (l *OdigosLogger) Error(msg string, keysAndValues ...interface{}) {
	if l != nil && l.sugared != nil {
		l.sugared.Errorw(msg, keysAndValues...)
	}
}

func (l *OdigosLogger) Warn(msg string, keysAndValues ...interface{}) {
	if l != nil && l.sugared != nil {
		l.sugared.Warnw(msg, keysAndValues...)
	}
}

func (l *OdigosLogger) Debug(msg string, keysAndValues ...interface{}) {
	if l != nil && l.sugared != nil {
		l.sugared.Debugw(msg, keysAndValues...)
	}
}
