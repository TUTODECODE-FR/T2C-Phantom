package utils

import (
	"log/slog"
	"os"
)

var SecurityLogger *slog.Logger

// InitSecurityLogger initializes a JSON-based structured logger for security events.
func InitSecurityLogger(logPath string) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// Fallback to standard output if file can't be opened
		SecurityLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
		SecurityLogger.Warn("Failed to open security log file, falling back to stdout", "error", err)
		return
	}
	SecurityLogger = slog.New(slog.NewJSONHandler(f, nil))
}

// LogSecurityEvent is a helper wrapper to log standard security anomalies.
func LogSecurityEvent(eventType, details, peer string) {
	if SecurityLogger == nil {
		slog.Default().Warn("Security Anomaly Detected",
			"type", eventType,
			"details", details,
			"peer", peer,
		)
		return
	}
	SecurityLogger.Warn("Security Anomaly Detected",
		"type", eventType,
		"details", details,
		"peer", peer,
	)
}
