package channel

import (
	"testing"
	"time"
)

// TestInsertRemove checks if an id can be inserted and then removed.
func TestInsertRemove(t *testing.T) {
	id := []byte("test")
	ti := time.Now().UTC()

	b := &bucket{}
	b.Insert(id, ti)
	if b.Len() != 1 {
		t.Fatal("Id was not inserted")
	}

	b.Remove(id)
	if b.Len() != 0 {
		t.Fatal("Id was not removed")
	}
}

// TestUpdate checks if the time parameter of an entry is updated when an entry
// with a time which is placed further into the future is inserted.
func TestUpdate(t *testing.T) {
	id := []byte("test")
	t1 := time.Now().UTC()
	t2 := t1.AddDate(0, 0, 1)

	b := &bucket{}
	b.Insert(id, t1)
	if b.Len() != 1 {
		t.Fatal("Id was not inserted")
	}

	b.Insert(id, t2)
	if e := b.find(id); e != nil {
		entry := e.Value.(*entry)
		if !entry.time.Equal(t2) {
			t.Fatal("Time was not updated")
		}
	} else {
		t.Fatal("Entry not found")
	}
}

// TestNoUpdate checks that the time of an entry is not updated by an entry with
// a time which is placed further into the past.
func TestNoUpdate(t *testing.T) {
	id := []byte("test")
	t1 := time.Now().UTC()
	t2 := t1.AddDate(0, 0, -1)

	b := &bucket{}
	b.Insert(id, t1)
	if b.Len() != 1 {
		t.Fatal("Id was not inserted")
	}

	b.Insert(id, t2)
	if e := b.find(id); e != nil {
		entry := e.Value.(*entry)
		if entry.time.Equal(t2) {
			t.Fatal("Time was updated")
		}
	} else {
		t.Fatal("Entry not found")
	}
}

// TestCleanup checks if old entries are removed.
func TestCleanup(t *testing.T) {
	id := []byte("test")
	t1 := time.Now().UTC().AddDate(0, 0, -1)

	b := &bucket{}
	b.Insert(id, t1)
	if b.Len() != 1 {
		t.Fatal("Id was not inserted")
	}

	b.Cleanup()
	if b.Len() != 0 {
		t.Fatal("Id was not removed")
	}
}

// TestNoCleanup checks if fresh entries are preserved during cleanup.
func TestNoCleanup(t *testing.T) {
	id := []byte("test")
	t1 := time.Now().UTC().AddDate(0, 0, 1)

	b := &bucket{}
	b.Insert(id, t1)
	if b.Len() != 1 {
		t.Fatal("Id was not inserted")
	}

	b.Cleanup()
	if b.Len() != 1 {
		t.Fatal("Id was removed")
	}
}

// TestGet checks if random ids are returned by the Get method.
func TestGet(t *testing.T) {
	t1 := time.Now().UTC().AddDate(0, 0, -1)

	b := &bucket{}
	var i byte
	for i = 0; i < 10; i++ {
		b.Insert([]byte{i}, t1)
	}

	nodes := b.Get(1)
	if len(nodes) == 0 {
		t.Fatal("Got 0 nodes")
	}
}
