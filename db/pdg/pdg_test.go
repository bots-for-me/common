package pdg

import (
	"testing"
)

var testPath = "test"

type testStructure struct {
	A string
	B *string
}

// go test -vet=off -test.v ./...

func TestWork(t *testing.T) {
	// defer os.RemoveAll(testPath)
	db, err := New(testPath, &testStructure{})
	if err != nil {
		t.Fatal(err)
	}
	a := testStructure{A: "hellp"}
	a.B = &a.A
	t.Logf("a: %#v", a)
	id := "123456"
	if err = db.Put(id, &a); err != nil {
		t.Fatal(err)
	}
	var b testStructure
	var found bool
	if found, err = db.Get(id, &b); err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("id %q not found", id)
	}
	t.Logf("b: %#v", b)
	if a.A != b.A {
		t.Fatal("a.A != b.A")
	}
	if b.B == nil {
		t.Fatal("b.B == nil")
	}
	if *a.B != *b.B {
		t.Fatal("*a.B != *b.A")
	}
	db.Close()
}
