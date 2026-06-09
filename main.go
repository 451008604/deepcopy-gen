package main

import (
	"flag"
	"fmt"
	"github.com/451008604/deepcopy-gen/generator"
	"github.com/451008604/deepcopy-gen/scanner"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("deepcopy-gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "", "directory to scan for Go structs")
	dryRun := fs.Bool("dry-run", false, "print generated code to stdout instead of writing files")
	verbose := fs.Bool("v", false, "verbose output")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: deepcopy-gen [options] [directory]\n\n")
		fmt.Fprintf(stderr, "A Go code generation tool for generating type-safe deep copy methods.\n\n")
		fmt.Fprintf(stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nExamples:\n")
		fmt.Fprintf(stderr, "  deepcopy-gen ./models              # Generate deepcopy.dc.go files\n")
		fmt.Fprintf(stderr, "  deepcopy-gen -dir ./models          # Same as above\n")
		fmt.Fprintf(stderr, "  deepcopy-gen . -dry-run             # Preview generated code\n")
		fmt.Fprintf(stderr, "  deepcopy-gen . -v                   # Verbose output\n")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dir == "" {
		if fs.NArg() > 0 {
			*dir = fs.Arg(0)
		} else if len(args) == 0 {
			fs.Usage()
			return nil
		} else {
			return fmt.Errorf("-dir or directory argument is required")
		}
	}

	info, err := os.Stat(*dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", *dir)
	}

	packages, err := scanner.ScanDir(*dir)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	totalStructs := 0
	for _, pkg := range packages {
		if len(pkg.Structs) == 0 {
			if !*dryRun {
				outPath := generator.OutputPath(pkg.Dir)
				if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("removing old %s: %w", outPath, err)
				}
			}
			continue
		}
		totalStructs += len(pkg.Structs)

		code, genErr := generator.GenerateWithOptions(pkg, generator.Options{})
		if genErr != nil {
			return fmt.Errorf("generating for package %s: %w", pkg.Name, genErr)
		}

		if err := generator.ValidateGenerated(code); err != nil {
			return fmt.Errorf("generated code is invalid for package %s: %w", pkg.Name, err)
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
			if *verbose {
				fmt.Fprintf(stdout, "wrote %s (%d structs)\n", outPath, len(pkg.Structs))
			}
		}
	}

	if *verbose || *dryRun {
		fmt.Fprintf(stderr, "processed %d packages, %d structs\n", len(packages), totalStructs)
	}

	return nil
}
