package generator

import (
	"testing"

	"github.com/451008604/deepcopy-gen/scanner"
)

var generatedResult string

func BenchmarkGenerate(b *testing.B) {
	packages, err := scanner.ScanDir("../testdata")
	if err != nil {
		b.Fatal(err)
	}
	for _, pkg := range packages {
		if pkg.Name != "testdata" {
			continue
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			generatedResult, err = Generate(pkg)
			if err != nil {
				b.Fatal(err)
			}
		}
		return
	}
	b.Fatal("testdata package not found")
}
