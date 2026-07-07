package tests

import (
	"testing"

	"github.com/451008604/deepcopy-gen/testdata/external"
	"github.com/451008604/deepcopy-gen/testdata/namedtypes"
)

// These tests verify runtime deep-copy isolation for field/type patterns that
// previously produced compile errors or silent shallow copies. They exercise
// the generated code in testdata/namedtypes/deepcopy.dc.go.

func TestNamedPtrFields_DeepCopy(t *testing.T) {
	s := namedtypes.Score(5)
	m := namedtypes.Matrix{1, 2, 3}
	inv := namedtypes.Inventory{"k": 1}
	in := &namedtypes.NamedPtrFields{S: &s, M: &m, I: &inv}
	cp := in.DeepCopy()

	*cp.S = 999
	(*cp.M)[0] = 999
	(*cp.I)["k"] = 999

	if s != 5 {
		t.Error("*Score leaked:", s)
	}
	if m[0] != 1 {
		t.Error("*Matrix leaked:", m)
	}
	if inv["k"] != 1 {
		t.Error("*Inventory leaked:", inv)
	}
}

func TestNamedValueFields_DeepCopy(t *testing.T) {
	a, b, c := 1, 2, 3
	in := &namedtypes.NamedValueFields{T: namedtypes.Tags{"x"}, M: namedtypes.Meta{"k": 1}, C: namedtypes.Coords{&a, &b, &c}}
	cp := in.DeepCopy()

	cp.T[0] = "y"
	cp.M["k"] = 999
	*cp.C[0] = 999
	cp.C[1] = nil

	if in.T[0] == "y" {
		t.Error("Tags value field leaked")
	}
	if in.M["k"] == 999 {
		t.Error("Meta value field leaked")
	}
	if a == 999 || in.C[1] == nil {
		t.Error("Coords value field leaked")
	}
}

func TestPtrToComposite_DeepCopy(t *testing.T) {
	basic := []string{"x"}
	items := []*external.GameItem{{Name: "sword", Count: 1}}
	m := map[string]int{"k": 1}
	mi := map[string]*external.GameItem{"k": {Name: "potion", Count: 2}}
	in := &namedtypes.PtrToComposite{Basic: &basic, Items: &items, M: &m, MI: &mi}
	cp := in.DeepCopy()

	(*cp.Basic)[0] = "y"
	(*cp.Items)[0].Name = "mut"
	(*cp.Items)[0].Count = 999
	(*cp.M)["k"] = 999
	(*cp.MI)["k"].Name = "mut"
	(*cp.MI)["k"].Count = 999

	if basic[0] == "y" {
		t.Error("*[]string leaked")
	}
	if items[0].Name == "mut" || items[0].Count == 999 {
		t.Error("*[]*external.GameItem leaked")
	}
	if m["k"] == 999 {
		t.Error("*map[string]int leaked")
	}
	if mi["k"].Name == "mut" || mi["k"].Count == 999 {
		t.Error("*map[string]*external.GameItem leaked")
	}
}

func TestMapOfStruct_DeepCopy(t *testing.T) {
	in := &namedtypes.MapOfStruct{
		Same:  map[string]namedtypes.Item{"k": {Tags: []string{"a"}}},
		Cross: map[string]external.GameItem{"k": {Name: "n", Count: 1}},
	}
	cp := in.DeepCopy()

	same := cp.Same["k"]
	same.Tags[0] = "mut"
	cp.Same["k"] = same

	cross := cp.Cross["k"]
	cross.Name = "mut"
	cross.Count = 999
	cp.Cross["k"] = cross

	if in.Same["k"].Tags[0] == "mut" {
		t.Error("map[string]Item (same-pkg) leaked")
	}
	if in.Cross["k"].Name == "mut" || in.Cross["k"].Count == 999 {
		t.Error("map[string]external.GameItem (cross-pkg) leaked")
	}
}

func TestAnonHolder_DeepCopy(t *testing.T) {
	n := 5
	p := "hi"
	in := &namedtypes.AnonHolder{}
	in.S.N = &n
	in.S.Inner.P = &p
	cp := in.DeepCopy()

	*cp.S.N = 999
	*cp.S.Inner.P = "mut"

	if n != 5 {
		t.Error("anon struct *int leaked:", n)
	}
	if p != "hi" {
		t.Error("nested anon struct *string leaked:", p)
	}
}

func TestHasStructValue_DeepCopy(t *testing.T) {
	in := &namedtypes.HasStructValue{Home: namedtypes.Addr{Tags: []string{"x"}}}
	cp := in.DeepCopy()
	cp.Home.Tags[0] = "y"
	if in.Home.Tags[0] == "y" {
		t.Error("same-pkg struct value field leaked")
	}
}

func TestHasCrossValue_DeepCopy(t *testing.T) {
	in := &namedtypes.HasCrossValue{Owner: external.AccountInfo{ID: 7, Username: "u"}}
	cp := in.DeepCopy()
	cp.Owner.ID = 999
	cp.Owner.Username = "mut"
	if in.Owner.ID == 999 || in.Owner.Username == "mut" {
		t.Error("cross-pkg struct value field leaked")
	}
}
