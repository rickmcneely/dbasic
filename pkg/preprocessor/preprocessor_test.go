package preprocessor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a small helper for the tests below.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// TestIncludeWithinProject verifies that ordinary relative includes (including
// ones in subdirectories) are expanded normally.
func TestIncludeWithinProject(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "lib", "greet.dbas"), "SUB Greet()\nEND SUB\n")
	main := filepath.Join(root, "main.dbas")
	writeFile(t, main, "INCLUDE \"lib/greet.dbas\"\nSUB Main()\nEND SUB\n")

	res, err := New(root).Process(main)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Source, "SUB Greet()") {
		t.Fatalf("included content missing from output:\n%s", res.Source)
	}
}

// TestIncludeTraversalBlocked verifies that ../ traversal and absolute paths
// outside the project are rejected by default.
func TestIncludeTraversalBlocked(t *testing.T) {
	root := t.TempDir()
	// A secret file that lives outside the project root.
	secret := filepath.Join(filepath.Dir(root), "secret.txt")
	writeFile(t, secret, "TOP SECRET\n")
	defer os.Remove(secret)

	cases := []struct{ name, include string }{
		{"parent traversal", "../secret.txt"},
		{"absolute path", secret},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			main := filepath.Join(root, "main.dbas")
			writeFile(t, main, "INCLUDE \""+tc.include+"\"\nSUB Main()\nEND SUB\n")

			res, err := New(root).Process(main)
			if err == nil {
				t.Fatalf("expected traversal to be rejected, got source:\n%s", res.Source)
			}
			if !strings.Contains(err.Error(), "escapes the project directory") {
				t.Fatalf("expected 'escapes the project directory' error, got: %v", err)
			}
		})
	}
}

// TestIncludeTraversalAllowedWithFlag verifies SetAllowExternal re-enables
// out-of-project includes for callers that opt in.
func TestIncludeTraversalAllowedWithFlag(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(filepath.Dir(root), "shared.dbas")
	writeFile(t, external, "SUB Shared()\nEND SUB\n")
	defer os.Remove(external)

	main := filepath.Join(root, "main.dbas")
	writeFile(t, main, "INCLUDE \"../shared.dbas\"\nSUB Main()\nEND SUB\n")

	pp := New(root)
	pp.SetAllowExternal(true)
	res, err := pp.Process(main)
	if err != nil {
		t.Fatalf("unexpected error with allowExternal=true: %v", err)
	}
	if !strings.Contains(res.Source, "SUB Shared()") {
		t.Fatalf("external include not expanded:\n%s", res.Source)
	}
}
