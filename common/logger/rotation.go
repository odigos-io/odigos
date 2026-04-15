package logger

import (
	"gopkg.in/natefinch/lumberjack.v2"
)

// RotationConfig controls log file rotation via lumberjack.
// All fields have sensible defaults; only Filename is required.
type RotationConfig struct {
	Filename   string // path to the log file (required)
	MaxSizeMB  int    // max megabytes before rotation (default: 100)
	MaxBackups int    // max number of old log files to keep (default: 3)
	MaxAgeDays int    // max days to retain old files (default: 7)
	Compress   bool   // gzip rotated files (default: true)
}

func (c RotationConfig) toLumberjack() *lumberjack.Logger {
	maxSize := c.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 100
	}
	maxBackups := c.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 3
	}
	maxAge := c.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 7
	}
	return &lumberjack.Logger{
		Filename:   c.Filename,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   c.Compress,
	}
}

// Option configures the logger created by Init.
type Option func(*initConfig)

type initConfig struct {
	rotation *RotationConfig
	stdout   bool // keep writing to stdout alongside the file (default: false when rotation is set)
}

// WithRotation enables log file rotation. When set, the logger writes to the
// specified file with automatic rotation instead of (or in addition to) stdout.
// By default stdout output is suppressed when rotation is enabled; use
// WithStdout(true) to keep it.
func WithRotation(cfg RotationConfig) Option {
	return func(c *initConfig) {
		c.rotation = &cfg
	}
}

// WithStdout controls whether stdout is included as a log sink.
// Without rotation this is always true regardless of this option.
// With rotation enabled, default is false; set to true to tee logs to both
// the rotated file and stdout.
func WithStdout(enabled bool) Option {
	return func(c *initConfig) {
		c.stdout = enabled
	}
}
