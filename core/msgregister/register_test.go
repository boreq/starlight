package msgregister

import (
	"testing"
	"time"
)

func TestInsert(t *testing.T) {
	t1 := time.Now().Add(10 * time.Second)
	r := New()
	err := r.Insert([]byte("a"), t1)
	if err != nil {
		t.Fatal(err)
	}
	err = r.Insert([]byte("b"), t1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInsertTwice(t *testing.T) {
	t1 := time.Now().Add(10 * time.Second)
	r := New()
	err := r.Insert([]byte("a"), t1)
	if err != nil {
		t.Fatal(err)
	}
	err = r.Insert([]byte("a"), t1)
	if err == nil {
		t.Fatal("No error returned")
	}
}

func TestInsertTimeout(t *testing.T) {
	t1 := time.Now().Add(-10 * time.Second)
	t2 := time.Now().Add(10 * time.Second)
	r := New()
	err := r.Insert([]byte("a"), t1)
	if err != nil {
		t.Fatal(err)
	}
	err = r.Insert([]byte("a"), t2)
	if err != nil {
		t.Fatal(err)
	}
}
