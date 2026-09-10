package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_DryRun(t *testing.T) {
	dir := "testdata"
	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", dir, "-dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "package testdata") {
		t.Error("expected output to contain 'package testdata'")
	}
	if !strings.Contains(stdout.String(), "DeepCopy") {
		t.Error("expected output to contain 'DeepCopy'")
	}
	if !strings.Contains(stderr.String(), "processed") {
		t.Error("expected stderr to contain 'processed'")
	}
}

func TestRun_Verbose(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "types.go"), []byte("package test\ntype Foo struct { X int }\n"), 0644)

	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", dir, "-v"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "wrote") {
		t.Error("expected verbose output to contain 'wrote'")
	}
	os.Remove(filepath.Join(dir, "deepcopy.dc.go"))
}

func TestRun_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "types.go"), []byte("package test\ntype Foo struct { X int }\n"), 0644)

	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	outPath := filepath.Join(dir, "deepcopy.dc.go")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("generated file not found: %v", err)
	}
	if !strings.Contains(string(data), "DeepCopy") {
		t.Error("generated file should contain DeepCopy")
	}
}

func TestRun_NonExistentDir(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", "/nonexistent/path/xyz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestRun_FileNotDir(t *testing.T) {
	f, _ := os.CreateTemp("", "test")
	f.Close()
	defer os.Remove(f.Name())

	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", f.Name()}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for file instead of directory")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' error, got: %v", err)
	}
}

func TestRun_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", dir, "-v"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-unknown-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid flag")
	}
}

func TestRun_MultiplePackages(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", "testdata", "-dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	out := stdout.String()
	for _, pkg := range []string{"testdata", "external", "gamecore", "multifile"} {
		if !strings.Contains(out, "package "+pkg) {
			t.Errorf("expected output to contain 'package %s'", pkg)
		}
	}
}

func TestRun_NoArgsShowsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Error("expected stderr to contain 'Usage:'")
	}
}

// TestTestDataCompiles is a regression guard: every checked-in testdata
// package (including its generated deepcopy.dc.go) must compile. This catches
// generator bugs that parse-only validation (assertValidGo) misses, such as
// references to undefined methods or calls to unexported methods across
// package boundaries.
func TestTestDataCompiles(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	dirs := []string{"testdata"}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirs = append(dirs, filepath.Join("testdata", e.Name()))
	}
	for _, sub := range dirs {
		t.Run(sub, func(t *testing.T) {
			cmd := exec.Command("go", "build", "./...")
			cmd.Dir = sub
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("go build %s failed:\n%s", sub, out)
			}
		})
	}
}

func TestRun_PositionalArgumentAndStaleCleanup(t *testing.T) {
	dir := t.TempDir()
	source := []byte("package fixture\ntype Item struct { Value *int }\n")
	if err := os.WriteFile(filepath.Join(dir, "types.go"), source, 0644); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(dir, "orphan")
	if err := os.Mkdir(orphan, 0755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(orphan, "deepcopy.dc.go")
	if err := os.WriteFile(stale, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-dry-run", dir}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatal("dry-run removed stale output")
	}
	if _, err := os.Stat(filepath.Join(dir, "deepcopy.dc.go")); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote generated output")
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"-v", dir}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale output still exists")
	}
	if !strings.Contains(stdout.String(), "removed stale") {
		t.Fatal("missing stale cleanup message")
	}
	generated, err := os.ReadFile(filepath.Join(dir, "deepcopy.dc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(generated, []byte("func (in *Item) DeepCopy() *Item")) {
		t.Fatal("positional path did not generate Item")
	}
	actual, err := os.ReadFile(filepath.Join(dir, "types.go"))
	if err != nil || !bytes.Equal(actual, source) {
		t.Fatal("source file changed")
	}
}

func TestRun_InvalidSourcePreservesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte("invalid Go"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "deepcopy.dc.go")
	if err := os.WriteFile(path, []byte("old output"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{dir}, &stdout, &stderr); err == nil {
		t.Fatal("invalid source accepted")
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != "old output" {
		t.Fatal("failed scan destroyed existing output")
	}
}

func TestRun_FlagsWithoutDirectory(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-v"}, &stdout, &stderr); err == nil {
		t.Fatal("missing directory accepted")
	}
}
