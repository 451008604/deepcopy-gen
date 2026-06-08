package deepcopy

import (
	"reflect"
	"testing"
)

func TestCopyPtr(t *testing.T) {
	v := NewVisited()
	if CopyPtr[int](v, nil) != nil {
		t.Fatal("nil input should return nil")
	}
	p := 42
	cp := CopyPtr(v, &p)
	if *cp != 42 {
		t.Fatalf("expected 42, got %d", *cp)
	}
	*cp = 99
	if p != 42 {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopyDoublePtr(t *testing.T) {
	v := NewVisited()
	if CopyDoublePtr[int](v, nil) != nil {
		t.Fatal("nil input should return nil")
	}
	val := 42
	p := &val
	pp := &p
	cp := CopyDoublePtr(v, pp)
	if **cp != 42 {
		t.Fatalf("expected 42, got %d", **cp)
	}
	**cp = 99
	if val != 42 {
		t.Fatal("mutation leaked to original")
	}

	var nilP *int
	pp2 := &nilP
	cp2 := CopyDoublePtr(v, pp2)
	if *cp2 != nil {
		t.Fatal("inner nil should stay nil")
	}
}

func TestCopySlice(t *testing.T) {
	if CopySlice[string](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := []string{"a", "b", "c"}
	cp := CopySlice(orig)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy does not match original")
	}
	cp[0] = "modified"
	if orig[0] == "modified" {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopySlice_Empty(t *testing.T) {
	orig := []int{}
	cp := CopySlice(orig)
	if len(cp) != 0 {
		t.Fatal("empty slice should produce empty copy")
	}
}

func TestCopySlicePtr(t *testing.T) {
	v := NewVisited()
	if CopySlicePtr[int](v, nil) != nil {
		t.Fatal("nil input should return nil")
	}
	a, b := 1, 2
	orig := []*int{&a, nil, &b}
	cp := CopySlicePtr(v, orig)
	if len(cp) != 3 {
		t.Fatalf("expected len 3, got %d", len(cp))
	}
	if cp[1] != nil {
		t.Fatal("nil element should stay nil")
	}
	if *cp[0] != 1 || *cp[2] != 2 {
		t.Fatal("values mismatch")
	}
	*cp[0] = 99
	if a != 1 {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopySliceSlice(t *testing.T) {
	v := NewVisited()
	if CopySliceSlice[int](v, nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := [][]int{{1, 2}, nil, {3, 4}}
	cp := CopySliceSlice(v, orig)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy does not match original")
	}
	cp[0][0] = 99
	if orig[0][0] == 99 {
		t.Fatal("mutation leaked to original")
	}
	if cp[1] != nil {
		t.Fatal("nil inner slice should stay nil")
	}
}

func TestCopySliceMap(t *testing.T) {
	v := NewVisited()
	if CopySliceMap[string, int](v, nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := []map[string]int{{"a": 1}, nil, {"b": 2}}
	cp := CopySliceMap(v, orig)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy does not match original")
	}
	cp[0]["a"] = 99
	if orig[0]["a"] == 99 {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopyMap(t *testing.T) {
	v := NewVisited()
	if CopyMap[string, int](v, nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := map[string]int{"a": 1, "b": 2}
	cp := CopyMap(v, orig)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy does not match original")
	}
	cp["a"] = 99
	if orig["a"] == 99 {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopyMapPtr(t *testing.T) {
	v := NewVisited()
	if CopyMapPtr[string, bool](v, nil) != nil {
		t.Fatal("nil input should return nil")
	}
	a, b := true, false
	orig := map[string]*bool{"x": &a, "y": &b, "z": nil}
	cp := CopyMapPtr(v, orig)
	if *cp["x"] != true || *cp["y"] != false || cp["z"] != nil {
		t.Fatal("values mismatch")
	}
	*cp["x"] = false
	if !a {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopyMapSlice(t *testing.T) {
	v := NewVisited()
	if CopyMapSlice[string, int](v, nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := map[string][]int{"a": {1, 2}, "b": nil}
	cp := CopyMapSlice(v, orig)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatalf("copy does not match original\norig: %v\ncp:   %v", orig, cp)
	}
	cp["a"][0] = 99
	if orig["a"][0] == 99 {
		t.Fatal("mutation leaked to original")
	}
	if cp["b"] != nil {
		t.Fatal("nil value should stay nil")
	}
}

type cycleA struct {
	Name string
	B    *cycleB
}

type cycleB struct {
	Value int
	A     *cycleA
}

func (a *cycleA) deepcopy(v Visited) *cycleA {
	if a == nil {
		return nil
	}
	if out, ok := v[a]; ok {
		return out.(*cycleA)
	}
	out := new(cycleA)
	v[a] = out
	*out = *a
	out.B = CopyPtr(v, a.B)
	return out
}

func (b *cycleB) deepcopy(v Visited) *cycleB {
	if b == nil {
		return nil
	}
	if out, ok := v[b]; ok {
		return out.(*cycleB)
	}
	out := new(cycleB)
	v[b] = out
	*out = *b
	out.A = CopyPtr(v, b.A)
	return out
}

func TestCopyPtr_CrossPackageCycle(t *testing.T) {
	a := &cycleA{Name: "root"}
	b := &cycleB{Value: 42}
	a.B = b
	b.A = a

	v := NewVisited()
	cp := CopyPtr(v, a)
	if cp == nil {
		t.Fatal("copy should not be nil")
	}
	if cp.Name != "root" {
		t.Fatalf("expected Name=root, got %q", cp.Name)
	}
	if cp.B == nil {
		t.Fatal("B should not be nil")
	}
	if cp.B.Value != 42 {
		t.Fatalf("expected B.Value=42, got %d", cp.B.Value)
	}
	if cp.B.A != cp {
		t.Fatal("cycle should be preserved: B.A should point to the copy of A")
	}
	if cp.B.A == a {
		t.Fatal("B.A should not point to original A")
	}
}

type cycleSliceA struct {
	Name string
	Bs   []*cycleSliceB
}

type cycleSliceB struct {
	Value int
	As    []*cycleSliceA
}

func (a *cycleSliceA) deepcopy(v Visited) *cycleSliceA {
	if a == nil {
		return nil
	}
	if out, ok := v[a]; ok {
		return out.(*cycleSliceA)
	}
	out := new(cycleSliceA)
	v[a] = out
	*out = *a
	out.Bs = CopySlicePtr(v, a.Bs)
	return out
}

func (b *cycleSliceB) deepcopy(v Visited) *cycleSliceB {
	if b == nil {
		return nil
	}
	if out, ok := v[b]; ok {
		return out.(*cycleSliceB)
	}
	out := new(cycleSliceB)
	v[b] = out
	*out = *b
	out.As = CopySlicePtr(v, b.As)
	return out
}

func TestCopySlicePtr_CrossPackageCycle(t *testing.T) {
	a := &cycleSliceA{Name: "root"}
	b := &cycleSliceB{Value: 42}
	a.Bs = []*cycleSliceB{b}
	b.As = []*cycleSliceA{a}

	v := NewVisited()
	cp := CopyPtr(v, a)
	if cp == nil {
		t.Fatal("copy should not be nil")
	}
	if cp.Name != "root" {
		t.Fatalf("expected Name=root, got %q", cp.Name)
	}
	if len(cp.Bs) != 1 {
		t.Fatalf("expected 1 B, got %d", len(cp.Bs))
	}
	if cp.Bs[0].Value != 42 {
		t.Fatalf("expected Bs[0].Value=42, got %d", cp.Bs[0].Value)
	}
	if len(cp.Bs[0].As) != 1 {
		t.Fatalf("expected 1 A in Bs[0].As, got %d", len(cp.Bs[0].As))
	}
	if cp.Bs[0].As[0] != cp {
		t.Fatal("cycle should be preserved: Bs[0].As[0] should point to copy of A")
	}
}
