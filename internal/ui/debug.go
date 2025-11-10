package ui

import (
	"log"
)

// debugEnabled controls whether debug logging is active
var debugEnabled = false

// SetDebugEnabled enables or disables debug logging.
// This should be called from main.go when the --debug flag is set.
func SetDebugEnabled(enabled bool) {
	debugEnabled = enabled
}

// logDebug writes a debug message to the log if debug mode is enabled.
// The log file is configured in main.go via tea.LogToFile().
// Debug mode must be explicitly enabled via SetDebugEnabled(true).
func logDebug(format string, args ...any) {
	if debugEnabled {
		log.Printf(format, args...)
	}
}
