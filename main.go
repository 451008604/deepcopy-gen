package main

import (
	"bufio"
	"crypto/rand"
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
			if err := writeGeneratedFile(outPath, func(w io.Writer) error {
				_, err := io.WriteString(w, code)
				return err
			}); err != nil {
				return fmt.Errorf("writing %s: %w", outPath, err)
			}
			generatedFiles[outPath] = true
		}
	}

	if *dryRun {
		fmt.Fprintf(stderr, "processed %d packages, %d structs\n", len(packages), totalStructs)
		return nil
	}

	// A failed generation can leave earlier packages updated. Only clean up
	// stale files once all packages have been written successfully.
	return removeStaleGeneratedFiles(dir, generatedFiles)
}

func writeGeneratedFile(path string, write func(io.Writer) error) error {
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("output path is not a regular file")
		}
		owned, err := hasGeneratedHeader(path)
		if err != nil {
			return err
		}
		if !owned {
			return errors.New("output file is missing the deepcopy-gen header")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Open exclusively with the same umask-sensitive mode as the old writer.
	var suffix [16]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return err
	}
	tempPath := filepath.Join(filepath.Dir(path), fmt.Sprintf(".deepcopy-gen-%x.tmp", suffix))
	f, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer os.Remove(tempPath)

	err = write(f)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tempPath, path)
}

func hasGeneratedHeader(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	line, err := bufio.NewReader(f).ReadSlice('\n')
	if errors.Is(err, io.EOF) || errors.Is(err, bufio.ErrBufferFull) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimRight(string(line), "\r\n") == generator.GeneratedHeader, nil
}

func removeStaleGeneratedFiles(dir string, generatedFiles map[string]bool) error {
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// Match the scanner's directory exclusions.
			if path != dir && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if path != generator.OutputPath(filepath.Dir(path)) || generatedFiles[path] || !entry.Type().IsRegular() {
			return nil
		}
		owned, err := hasGeneratedHeader(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking stale %s: %w", path, err)
		}
		if owned {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("removing stale %s: %w", path, err)
			}
		}
		return nil
	})
}
