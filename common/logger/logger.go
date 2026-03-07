package logger

import (
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	atom     = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	instance *zap.Logger
)

// Init builds the logger on the shared atom. Call once at process startup.
// levelStr: "error"|"warn"|"info"|"debug"|"trace"
func Init(levelStr string) *zap.Logger {
	atom.SetLevel(ParseLevel(levelStr))

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(os.Stdout),
		atom,
	)
	instance = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(instance)
	return instance
}

// UpdateInstance must be called after bridge.AttachToZapLogger wraps the logger.
func UpdateInstance(zl *zap.Logger) {
	instance = zl
	zap.ReplaceGlobals(instance)
}

func Logger() *zap.Logger          { return instance }
func AtomicLevel() zap.AtomicLevel { return atom }

// SetLevel sets the global log level. Empty string is a no-op (keeps current level).
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

// OdigosLogger is a slog-style logger (Info(msg, k, v...)) for backward compatibility.
// Use LoggerCompat() to get it; Logger() returns raw *zap.Logger per plan.
type OdigosLogger struct {
	sugared *zap.SugaredLogger
}

// LoggerCompat returns a logger with slog-style API. Use for existing .Info(msg, k, v) call sites.
func LoggerCompat() *OdigosLogger {
	if instance != nil {
		return &OdigosLogger{sugared: instance.Sugar()}
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
