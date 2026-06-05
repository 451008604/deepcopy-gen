package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_DryRun(t *testing.T) {
	dir := filepath.Join("testdata", "simple")
	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", dir, "-dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "package simple") {
		t.Error("expected output to contain 'package simple'")
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
	os.Remove(filepath.Join(dir, "structinfo.go"))
}

func TestRun_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "types.go"), []byte("package test\ntype Foo struct { X int }\n"), 0644)

	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	outPath := filepath.Join(dir, "structinfo.go")
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
	for _, pkg := range []string{"simple", "complex", "nested", "empty"} {
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
	if !strings.Contains(stderr.String(), "-no-reflect") {
		t.Error("expected help to show -no-reflect flag")
	}
}

func TestRun_NoReflect(t *testing.T) {
	dir := filepath.Join("testdata", "iface")
	var stdout, stderr bytes.Buffer
	err := run([]string{"-dir", dir, "-dry-run", "-no-reflect"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	out := stdout.String()
	if strings.Contains(out, "DeepCopyAny") {
		t.Error("expected no DeepCopyAny calls with -no-reflect")
	}
	if strings.Contains(out, "dc \"github.com/451008604/deepcopy-gen/deepcopy\"") {
		t.Error("expected no deepcopy import with -no-reflect")
	}
	if !strings.Contains(out, "*out = *in") {
		t.Error("expected shallow copy via *out = *in")
	}
}
