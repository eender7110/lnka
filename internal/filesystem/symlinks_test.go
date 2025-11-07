package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCreateSymlink_SiblingDirectories tests that symlinks are created correctly
// when source and target directories are siblings (in the same parent directory)
func TestCreateSymlink_SiblingDirectories(t *testing.T) {
	// Create a temporary directory structure:
	// temp/
	//   ├── services-available/
	//   │   └── test-file.yml
	//   └── services-enabled/
	//       └── test-file.yml -> ../services-available/test-file.yml

	tempDir, err := os.MkdirTemp("", "lnka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "services-available")
	targetDir := filepath.Join(tempDir, "services-enabled")

	// Create directories
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create a test file in source directory
	testFile := "test-file.yml"
	sourceFile := filepath.Join(sourceDir, testFile)
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create the symlink
	if err := CreateSymlink(sourceDir, targetDir, testFile); err != nil {
		t.Fatalf("CreateSymlink failed: %v", err)
	}

	// Verify the symlink was created
	linkPath := filepath.Join(targetDir, testFile)
	linkTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	// The symlink should be relative and start with ../
	expectedTarget := filepath.Join("..", "services-available", testFile)
	if linkTarget != expectedTarget {
		t.Errorf("Symlink target incorrect:\n  got:  %q\n  want: %q", linkTarget, expectedTarget)
	}

	// Verify the symlink actually works (can resolve to the source file)
	resolvedPath := filepath.Join(targetDir, linkTarget)
	resolvedAbs, err := filepath.Abs(resolvedPath)
	if err != nil {
		t.Fatalf("Failed to resolve symlink path: %v", err)
	}

	sourceAbs, err := filepath.Abs(sourceFile)
	if err != nil {
		t.Fatalf("Failed to get absolute source path: %v", err)
	}

	if resolvedAbs != sourceAbs {
		t.Errorf("Symlink doesn't resolve to source file:\n  resolved: %s\n  source:   %s", resolvedAbs, sourceAbs)
	}

	// Verify we can actually read the file through the symlink
	content, err := os.ReadFile(linkPath)
	if err != nil {
		t.Errorf("Failed to read through symlink: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Content mismatch: got %q, want %q", string(content), "test content")
	}
}

// TestCreateSymlink_NestedDirectories tests symlink creation with nested directories
func TestCreateSymlink_NestedDirectories(t *testing.T) {
	// Create a more complex directory structure:
	// temp/
	//   ├── config/
	//   │   └── available/
	//   │       └── test.conf
	//   └── active/
	//       └── test.conf -> ../config/available/test.conf

	tempDir, err := os.MkdirTemp("", "lnka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "config", "available")
	targetDir := filepath.Join(tempDir, "active")

	// Create directories
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create a test file
	testFile := "test.conf"
	sourceFile := filepath.Join(sourceDir, testFile)
	if err := os.WriteFile(sourceFile, []byte("config data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create the symlink
	if err := CreateSymlink(sourceDir, targetDir, testFile); err != nil {
		t.Fatalf("CreateSymlink failed: %v", err)
	}

	// Verify the symlink
	linkPath := filepath.Join(targetDir, testFile)
	linkTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	// Should be a relative path
	expectedTarget := filepath.Join("..", "config", "available", testFile)
	if linkTarget != expectedTarget {
		t.Errorf("Symlink target incorrect:\n  got:  %q\n  want: %q", linkTarget, expectedTarget)
	}

	// Verify the symlink resolves correctly
	content, err := os.ReadFile(linkPath)
	if err != nil {
		t.Errorf("Failed to read through symlink: %v", err)
	}
	if string(content) != "config data" {
		t.Errorf("Content mismatch: got %q, want %q", string(content), "config data")
	}
}
