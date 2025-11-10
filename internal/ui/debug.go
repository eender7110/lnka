package ui

import (
	"log"
)

// logDebug writes a debug message to the log if DEBUG mode is enabled.
// The log file is configured in main.go via tea.LogToFile().
func logDebug(format string, args ...interface{}) {
	log.Printf(format, args...)
}
