package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCreateSymlink_MixedPaths tests all combinations of absolute and relative paths
func TestCreateSymlink_MixedPaths(t *testing.T) {
	// Create temp dir structure
	tempDir, err := os.MkdirTemp("", "lnka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "services-available")
	targetDir := filepath.Join(tempDir, "services-enabled")

	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Remember original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory for relative path tests
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	testCases := []struct {
		name      string
		sourceDir string
		targetDir string
		testFile  string
	}{
		{
			name:      "both absolute",
			sourceDir: sourceDir,
			targetDir: targetDir,
			testFile:  "test-absolute.yml",
		},
		{
			name:      "both relative",
			sourceDir: "services-available",
			targetDir: "services-enabled",
			testFile:  "test-relative.yml",
		},
		{
			name:      "source absolute, target relative",
			sourceDir: sourceDir,
			targetDir: "services-enabled",
			testFile:  "test-abs-rel.yml",
		},
		{
			name:      "source relative, target absolute",
			sourceDir: "services-available",
			targetDir: targetDir,
			testFile:  "test-rel-abs.yml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test file
			var sourceFile string
			if filepath.IsAbs(tc.sourceDir) {
				sourceFile = filepath.Join(tc.sourceDir, tc.testFile)
			} else {
				sourceFile = filepath.Join(tempDir, tc.sourceDir, tc.testFile)
			}

			if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			defer os.Remove(sourceFile)

			// Create symlink
			if err := CreateSymlink(tc.sourceDir, tc.targetDir, tc.testFile); err != nil {
				t.Fatalf("CreateSymlink failed: %v", err)
			}

			// Determine link path
			var linkPath string
			if filepath.IsAbs(tc.targetDir) {
				linkPath = filepath.Join(tc.targetDir, tc.testFile)
			} else {
				linkPath = filepath.Join(tempDir, tc.targetDir, tc.testFile)
			}
			defer os.Remove(linkPath)

			// Verify symlink was created
			linkTarget, err := os.Readlink(linkPath)
			if err != nil {
				t.Fatalf("Failed to read symlink: %v", err)
			}

			t.Logf("Symlink target: %s", linkTarget)

			// Verify we can read through the symlink
			content, err := os.ReadFile(linkPath)
			if err != nil {
				t.Errorf("Failed to read through symlink: %v", err)
			}
			if string(content) != "test content" {
				t.Errorf("Content mismatch: got %q, want %q", string(content), "test content")
			}

			// Verify the symlink resolves to the correct source file
			var linkDir string
			if filepath.IsAbs(tc.targetDir) {
				linkDir = tc.targetDir
			} else {
				linkDir = filepath.Join(tempDir, tc.targetDir)
			}

			resolvedPath := linkTarget
			if !filepath.IsAbs(linkTarget) {
				resolvedPath = filepath.Join(linkDir, linkTarget)
			}

			resolvedAbs, err := filepath.Abs(resolvedPath)
			if err != nil {
				t.Fatalf("Failed to resolve symlink path: %v", err)
			}

			sourceAbs, err := filepath.Abs(sourceFile)
			if err != nil {
				t.Fatalf("Failed to get absolute source path: %v", err)
			}

			// On macOS, /var is a symlink to /private/var, so we need to evaluate symlinks
			resolvedAbs, _ = filepath.EvalSymlinks(resolvedAbs)
			sourceAbs, _ = filepath.EvalSymlinks(sourceAbs)

			if resolvedAbs != sourceAbs {
				t.Errorf("Symlink doesn't resolve to source file:\n  resolved: %s\n  source:   %s", resolvedAbs, sourceAbs)
			}

			// Verify the symlink is relative (preferred)
			if filepath.IsAbs(linkTarget) {
				t.Logf("Warning: symlink is absolute (relative would be better for portability)")
			} else {
				t.Logf("Good: symlink is relative (%s)", linkTarget)
			}
		})
	}
}

// TestCreateSymlink_EdgeCases tests edge cases with different directory structures
func TestCreateSymlink_EdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lnka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Remember and restore working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	testCases := []struct {
		name           string
		setupDirs      func() (sourceDir, targetDir string)
		testFile       string
		expectedPrefix string // expected prefix in symlink target
	}{
		{
			name: "sibling directories",
			setupDirs: func() (string, string) {
				os.Mkdir("source", 0755)
				os.Mkdir("target", 0755)
				return "source", "target"
			},
			testFile:       "file.txt",
			expectedPrefix: "../source/",
		},
		{
			name: "deeply nested source",
			setupDirs: func() (string, string) {
				os.MkdirAll("a/b/c/source", 0755)
				os.Mkdir("target", 0755)
				return "a/b/c/source", "target"
			},
			testFile:       "file.txt",
			expectedPrefix: "../a/b/c/source/",
		},
		{
			name: "deeply nested target",
			setupDirs: func() (string, string) {
				os.Mkdir("source", 0755)
				os.MkdirAll("x/y/z/target", 0755)
				return "source", "x/y/z/target"
			},
			testFile:       "file.txt",
			expectedPrefix: "../../../../source/",
		},
		{
			name: "same directory (source equals target)",
			setupDirs: func() (string, string) {
				os.Mkdir("same", 0755)
				return "same", "same"
			},
			testFile:       "file.txt",
			expectedPrefix: "file.txt", // Should just be the filename, no path
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create subdirectories for this test
			testDir := filepath.Join(tempDir, tc.name)
			os.Mkdir(testDir, 0755)
			os.Chdir(testDir)
			defer os.Chdir(tempDir)

			sourceDir, targetDir := tc.setupDirs()

			// Create test file
			sourceFile := filepath.Join(sourceDir, tc.testFile)
			if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Skip if source and target are the same (symlink to itself doesn't make sense)
			if sourceDir == targetDir {
				t.Skip("Skipping same directory test (symlink to self)")
				return
			}

			// Create symlink
			if err := CreateSymlink(sourceDir, targetDir, tc.testFile); err != nil {
				t.Fatalf("CreateSymlink failed: %v", err)
			}

			// Read symlink
			linkPath := filepath.Join(targetDir, tc.testFile)
			linkTarget, err := os.Readlink(linkPath)
			if err != nil {
				t.Fatalf("Failed to read symlink: %v", err)
			}

			t.Logf("Created symlink: %s -> %s", linkPath, linkTarget)

			// Verify prefix
			if tc.expectedPrefix != "" && linkTarget != tc.expectedPrefix && !filepath.HasPrefix(linkTarget, tc.expectedPrefix) {
				t.Errorf("Unexpected symlink target:\n  got:      %s\n  expected prefix: %s", linkTarget, tc.expectedPrefix)
			}

			// Verify the symlink works
			content, err := os.ReadFile(linkPath)
			if err != nil {
				t.Errorf("Failed to read through symlink %s: %v", linkPath, err)
			}
			if string(content) != "test" {
				t.Errorf("Content mismatch")
			}
		})
	}
}
