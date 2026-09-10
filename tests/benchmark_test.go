package tests

import (
	"strconv"
	"testing"

	"github.com/451008604/deepcopy-gen/testdata"
)

// 保留结果，测量调用方实际持有副本时的分配。
var (
	pointResult  *testdata.Point
	mapResult    *testdata.WithMap
	matrixResult *testdata.WithSlice
)

func BenchmarkDeepCopy_Point(b *testing.B) {
	in := &testdata.Point{X: 1, Y: 2}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pointResult = in.DeepCopy()
	}
}

func BenchmarkDeepCopy_Map(b *testing.B) {
	in := &testdata.WithMap{Scores: make(map[string]int, 1024)}
	for i := 0; i < 1024; i++ {
		in.Scores[strconv.Itoa(i)] = i
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapResult = in.DeepCopy()
	}
}

func BenchmarkDeepCopy_Matrix(b *testing.B) {
	in := &testdata.WithSlice{Matrix: make([][]float64, 32)}
	for i := range in.Matrix {
		in.Matrix[i] = make([]float64, 32)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matrixResult = in.DeepCopy()
	}
}
