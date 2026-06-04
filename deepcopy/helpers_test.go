package deepcopy

import (
	"reflect"
	"testing"
)

func TestCopyPtr(t *testing.T) {
	if CopyPtr[int](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	v := 42
	cp := CopyPtr(&v)
	if *cp != 42 {
		t.Fatalf("expected 42, got %d", *cp)
	}
	*cp = 99
	if v != 42 {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopyDoublePtr(t *testing.T) {
	if CopyDoublePtr[int](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	v := 42
	p := &v
	pp := &p
	cp := CopyDoublePtr(pp)
	if **cp != 42 {
		t.Fatalf("expected 42, got %d", **cp)
	}
	**cp = 99
	if v != 42 {
		t.Fatal("mutation leaked to original")
	}

	var nilP *int
	pp2 := &nilP
	cp2 := CopyDoublePtr(pp2)
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
	if CopySlicePtr[int](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	a, b := 1, 2
	orig := []*int{&a, nil, &b}
	cp := CopySlicePtr(orig)
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
	if CopySliceSlice[int](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := [][]int{{1, 2}, nil, {3, 4}}
	cp := CopySliceSlice(orig)
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
	if CopySliceMap[string, int](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := []map[string]int{{"a": 1}, nil, {"b": 2}}
	cp := CopySliceMap(orig)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy does not match original")
	}
	cp[0]["a"] = 99
	if orig[0]["a"] == 99 {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopyMap(t *testing.T) {
	if CopyMap[string, int](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := map[string]int{"a": 1, "b": 2}
	cp := CopyMap(orig)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy does not match original")
	}
	cp["a"] = 99
	if orig["a"] == 99 {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopyMapPtr(t *testing.T) {
	if CopyMapPtr[string, bool](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	a, b := true, false
	orig := map[string]*bool{"x": &a, "y": &b, "z": nil}
	cp := CopyMapPtr(orig)
	if *cp["x"] != true || *cp["y"] != false || cp["z"] != nil {
		t.Fatal("values mismatch")
	}
	*cp["x"] = false
	if !a {
		t.Fatal("mutation leaked to original")
	}
}

func TestCopyMapSlice(t *testing.T) {
	if CopyMapSlice[string, int](nil) != nil {
		t.Fatal("nil input should return nil")
	}
	orig := map[string][]int{"a": {1, 2}, "b": nil}
	cp := CopyMapSlice(orig)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy does not match original")
	}
	cp["a"][0] = 99
	if orig["a"][0] == 99 {
		t.Fatal("mutation leaked to original")
	}
	if cp["b"] != nil {
		t.Fatal("nil value should stay nil")
	}
}

func TestDeepCopyAny_Nil(t *testing.T) {
	if DeepCopyAny(nil) != nil {
		t.Fatal("nil should return nil")
	}
}

func TestDeepCopyAny_BasicTypes(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"int", 42},
		{"string", "hello"},
		{"bool", true},
		{"float64", 3.14},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := DeepCopyAny(tt.val)
			if !reflect.DeepEqual(cp, tt.val) {
				t.Fatalf("expected %v, got %v", tt.val, cp)
			}
		})
	}
}

func TestDeepCopyAny_Pointer(t *testing.T) {
	v := 42
	cp := DeepCopyAny(&v).(*int)
	if *cp != 42 {
		t.Fatalf("expected 42, got %d", *cp)
	}
	*cp = 99
	if v != 42 {
		t.Fatal("mutation leaked to original")
	}
}

func TestDeepCopyAny_Slice(t *testing.T) {
	orig := []int{1, 2, 3}
	cp := DeepCopyAny(orig).([]int)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy mismatch")
	}
	cp[0] = 99
	if orig[0] == 99 {
		t.Fatal("mutation leaked")
	}
}

func TestDeepCopyAny_Map(t *testing.T) {
	orig := map[string]int{"a": 1, "b": 2}
	cp := DeepCopyAny(orig).(map[string]int)
	if !reflect.DeepEqual(orig, cp) {
		t.Fatal("copy mismatch")
	}
	cp["a"] = 99
	if orig["a"] == 99 {
		t.Fatal("mutation leaked")
	}
}

func TestDeepCopyAny_Struct(t *testing.T) {
	type Foo struct {
		Name string
		Val  int
	}
	orig := Foo{Name: "test", Val: 42}
	cp := DeepCopyAny(orig).(Foo)
	if cp.Name != "test" || cp.Val != 42 {
		t.Fatalf("expected {test 42}, got %v", cp)
	}
}

func TestDeepCopyAny_NestedPointer(t *testing.T) {
	v := 42
	p := &v
	cp := DeepCopyAny(&p).(**int)
	if **cp != 42 {
		t.Fatalf("expected 42, got %d", **cp)
	}
	**cp = 99
	if v != 42 {
		t.Fatal("mutation leaked")
	}
}

func TestDeepCopyAny_NilPointer(t *testing.T) {
	var p *int
	cp := DeepCopyAny(p).(*int)
	if cp != nil {
		t.Fatal("nil pointer should stay nil")
	}
}

func TestDeepCopyAny_NilSlice(t *testing.T) {
	var s []int
	cp := DeepCopyAny(&s).(*[]int)
	if *cp != nil {
		t.Fatal("nil slice should stay nil")
	}
}

func TestDeepCopyAny_NilMap(t *testing.T) {
	var m map[string]int
	cp := DeepCopyAny(&m).(*map[string]int)
	if *cp != nil {
		t.Fatal("nil map should stay nil")
	}
}


