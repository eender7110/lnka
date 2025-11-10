package ui

import (
	"testing"
)

func TestFileItemFilterValue(t *testing.T) {
	tests := []struct {
		name     string
		fileItem fileItem
		expected string
	}{
		{
			name:     "simple filename",
			fileItem: fileItem{name: "config.yaml", isEnabled: false},
			expected: "config.yaml",
		},
		{
			name:     "filename with path",
			fileItem: fileItem{name: ".bashrc", isEnabled: true},
			expected: ".bashrc",
		},
		{
			name:     "empty filename",
			fileItem: fileItem{name: "", isEnabled: false},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fileItem.FilterValue()
			if result != tt.expected {
				t.Errorf("FilterValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilesLoadedMsg(t *testing.T) {
	// Test message creation with both available and enabled files
	msg := filesLoadedMsg{
		availableFiles: []string{"file1.txt", "file2.txt", "file3.txt"},
		enabledFiles:   []string{"file1.txt"},
		err:            nil,
	}

	if len(msg.availableFiles) != 3 {
		t.Errorf("Expected 3 available files, got %d", len(msg.availableFiles))
	}

	if len(msg.enabledFiles) != 1 {
		t.Errorf("Expected 1 enabled file, got %d", len(msg.enabledFiles))
	}

	if msg.err != nil {
		t.Errorf("Expected no error, got %v", msg.err)
	}
}
