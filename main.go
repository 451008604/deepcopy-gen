package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/451008604/deepcopy-gen/generator"
	"github.com/451008604/deepcopy-gen/scanner"
)

// 发布构建可通过 -ldflags "-X main.version=vX.Y.Z" 指定版本。
var version string

func versionString(info *debug.BuildInfo) string {
	if version != "" {
		return version
	}
	if info == nil {
		return "dev"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	revision, dirty := "", ""
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if revision != "" {
		return "dev (" + revision + dirty + ")"
	}
	return "dev"
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("deepcopy-gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "print generated code to stdout instead of writing files")
	showVersion := fs.Bool("version", false, "print tool version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: deepcopy-gen [options] [directory]\n\n")
		fmt.Fprintf(stderr, "A Go code generation tool for generating type-safe deep copy methods.\n\n")
		fmt.Fprintf(stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nExamples:\n")
		fmt.Fprintf(stderr, "  deepcopy-gen ./models              # Generate deepcopy.dc.go files\n")
		fmt.Fprintf(stderr, "  deepcopy-gen -version               # Print tool version\n")
		fmt.Fprintf(stderr, "  deepcopy-gen -dry-run ./models      # Preview generated code\n")
		fmt.Fprintf(stderr, "\nOptions must precede the directory.\n")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *showVersion {
		if fs.NArg() != 0 || *dryRun {
			return fmt.Errorf("-version must be used without a directory or -dry-run")
		}
		info, _ := debug.ReadBuildInfo()
		_, err := fmt.Fprintf(stdout, "deepcopy-gen %s\n", versionString(info))
		return err
	}
	if len(args) == 0 {
		fs.Usage()
		return nil
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("exactly one directory is required; options must precede the directory")
	}
	dir := fs.Arg(0)

	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	existingFiles := collectExistingDCFiles(dir)

	packages, err := scanner.ScanDir(dir)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	generatedFiles := make(map[string]bool)
	totalStructs := 0
	for _, pkg := range packages {
		if len(pkg.Structs) == 0 {
			continue
		}
		totalStructs += len(pkg.Structs)

		code, genErr := generator.GenerateWithOptions(pkg, generator.Options{})
		if genErr != nil {
			return fmt.Errorf("generating for package %s: %w", pkg.Name, genErr)
		}

		if *dryRun {
			fmt.Fprintf(stdout, "// === Package: %s (%s) ===\n", pkg.Name, pkg.Dir)
			fmt.Fprintln(stdout, code)
		} else {
			outPath := generator.OutputPath(pkg.Dir)
			if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing old %s: %w", outPath, err)
			}
			if err := os.WriteFile(outPath, []byte(code), 0644); err != nil {
				return fmt.Errorf("writing %s: %w", outPath, err)
			}
			generatedFiles[outPath] = true
		}
	}

	if !*dryRun {
		for _, oldFile := range existingFiles {
			if !generatedFiles[oldFile] {
				if err := os.Remove(oldFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("removing stale %s: %w", oldFile, err)
				}
			}
		}
	}

	if *dryRun {
		fmt.Fprintf(stderr, "processed %d packages, %d structs\n", len(packages), totalStructs)
	}

	return nil
}

func collectExistingDCFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".dc.go") {
			files = append(files, path)
		}
		return nil
	})
	return files
}
