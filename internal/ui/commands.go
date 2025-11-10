package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marco-arnold/lnka/internal/filesystem"
)

// loadFilesCmd creates a command that asynchronously loads both
// available files and enabled files. This ensures both operations
// complete before returning a single message.
// Returns filesLoadedMsg when complete.
func loadFilesCmd(sourceDir, targetDir string) tea.Cmd {
	return func() tea.Msg {
		// Load available files
		availableFiles, err := filesystem.ListAvailableFiles(sourceDir)
		if err != nil {
			return filesLoadedMsg{
				availableFiles: nil,
				enabledFiles:   nil,
				err:            err,
			}
		}

		// Load enabled files
		enabledFiles, err := filesystem.GetEnabledFiles(sourceDir, targetDir)
		if err != nil {
			return filesLoadedMsg{
				availableFiles: availableFiles,
				enabledFiles:   nil,
				err:            err,
			}
		}

		return filesLoadedMsg{
			availableFiles: availableFiles,
			enabledFiles:   enabledFiles,
			err:            nil,
		}
	}
}
