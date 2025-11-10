package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFilesCmd_Success(t *testing.T) {
	// Create temporary directories
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create test files in source
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, name := range testFiles {
		sourcePath := filepath.Join(sourceDir, name)
		if err := os.WriteFile(sourcePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create symlinks for some files in target
	linkedFiles := []string{"file1.txt", "file2.txt"}
	for _, name := range linkedFiles {
		sourcePath := filepath.Join(sourceDir, name)
		targetPath := filepath.Join(targetDir, name)
		if err := os.Symlink(sourcePath, targetPath); err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}
	}

	// Execute command synchronously
	cmd := loadFilesCmd(sourceDir, targetDir)
	msg := cmd()

	// Type assert the message
	loadedMsg, ok := msg.(filesLoadedMsg)
	if !ok {
		t.Fatalf("Expected filesLoadedMsg, got %T", msg)
	}

	// Check for errors
	if loadedMsg.err != nil {
		t.Errorf("Expected no error, got %v", loadedMsg.err)
	}

	// Check available files count
	if len(loadedMsg.availableFiles) != len(testFiles) {
		t.Errorf("Expected %d available files, got %d", len(testFiles), len(loadedMsg.availableFiles))
	}

	// Check enabled files count
	if len(loadedMsg.enabledFiles) != len(linkedFiles) {
		t.Errorf("Expected %d enabled files, got %d", len(linkedFiles), len(loadedMsg.enabledFiles))
	}

	// Verify all available files are present
	availableMap := make(map[string]bool)
	for _, f := range loadedMsg.availableFiles {
		availableMap[f] = true
	}
	for _, expected := range testFiles {
		if !availableMap[expected] {
			t.Errorf("Expected available file %q not found in results", expected)
		}
	}

	// Verify all enabled files are present
	enabledMap := make(map[string]bool)
	for _, f := range loadedMsg.enabledFiles {
		enabledMap[f] = true
	}
	for _, expected := range linkedFiles {
		if !enabledMap[expected] {
			t.Errorf("Expected enabled file %q not found in results", expected)
		}
	}
}

func TestLoadFilesCmd_NonExistentSourceDir(t *testing.T) {
	// Use a source directory that doesn't exist
	nonExistentSource := "/this/directory/does/not/exist/source"
	targetDir := t.TempDir()

	// Execute command synchronously
	cmd := loadFilesCmd(nonExistentSource, targetDir)
	msg := cmd()

	// Type assert the message
	loadedMsg, ok := msg.(filesLoadedMsg)
	if !ok {
		t.Fatalf("Expected filesLoadedMsg, got %T", msg)
	}

	// Should have an error
	if loadedMsg.err == nil {
		t.Error("Expected error for non-existent source directory, got nil")
	}
}

func TestLoadFilesCmd_NonExistentTargetDir(t *testing.T) {
	// Create source but use non-existent target
	sourceDir := t.TempDir()
	nonExistentTarget := "/this/directory/does/not/exist/target"

	// Create a test file in source
	sourcePath := filepath.Join(sourceDir, "file1.txt")
	if err := os.WriteFile(sourcePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Execute command synchronously
	cmd := loadFilesCmd(sourceDir, nonExistentTarget)
	msg := cmd()

	// Type assert the message
	loadedMsg, ok := msg.(filesLoadedMsg)
	if !ok {
		t.Fatalf("Expected filesLoadedMsg, got %T", msg)
	}

	// Should have an error
	if loadedMsg.err == nil {
		t.Error("Expected error for non-existent target directory, got nil")
	}

	// Should still have available files loaded
	if len(loadedMsg.availableFiles) == 0 {
		t.Error("Expected available files to be loaded before error")
	}
}

func TestLoadFilesCmd_EmptyDirs(t *testing.T) {
	// Create temporary directories with no files
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Execute command synchronously
	cmd := loadFilesCmd(sourceDir, targetDir)
	msg := cmd()

	// Type assert the message
	loadedMsg, ok := msg.(filesLoadedMsg)
	if !ok {
		t.Fatalf("Expected filesLoadedMsg, got %T", msg)
	}

	// Check for errors
	if loadedMsg.err != nil {
		t.Errorf("Expected no error, got %v", loadedMsg.err)
	}

	// Should return empty lists
	if len(loadedMsg.availableFiles) != 0 {
		t.Errorf("Expected 0 available files, got %d", len(loadedMsg.availableFiles))
	}

	if len(loadedMsg.enabledFiles) != 0 {
		t.Errorf("Expected 0 enabled files, got %d", len(loadedMsg.enabledFiles))
	}
}
