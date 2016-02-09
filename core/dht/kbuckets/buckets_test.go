package kbuckets

import (
	"bytes"
	"fmt"
	"testing"
)

func TestBuckets(t *testing.T) {
	selfId := []byte{0x0}

	buckets := New(selfId, 2)

	buckets.Update([]byte{0x8}, "addr1")
	buckets.Update([]byte{0xc}, "addr2")
	buckets.Update([]byte{0xe}, "addr3")

	fmt.Println("Add more distant ones")
	buckets.Update([]byte{0xf0}, "add")

	fmt.Println("Update")
	buckets.Update([]byte{0xe}, "addrnew")

	for _, entry := range buckets.GetClosest(selfId, 2) {
		fmt.Printf("%x %s\n", entry.Id, entry.Address)
	}
}

func TestReplace(t *testing.T) {
	const allOnes byte = 255

	b := &buckets{
		buckets: []*bucket{&bucket{}},
		cache:   []*bucket{&bucket{}},
		k:       2,
		self:    []byte{allOnes >> 8},
	}

	// Insert two entries into the first bucket.
	b.Update([]byte{allOnes}, "addr1")
	b.Update([]byte{allOnes - 1}, "addr2")
	if len(b.buckets) != 1 || b.buckets[0].Len() != 2 {
		t.Fatal("Invalid bucket len 1")
	}
	if len(b.cache) != 1 || b.cache[0].Len() != 0 {
		t.Fatal("Invalid cache len 1")
	}

	// Overflow the first bucket and split it.
	b.Update([]byte{allOnes - 2}, "addr3")
	if len(b.buckets) != 2 || b.buckets[0].Len() != 2 {
		t.Fatal("Invalid bucket len 2")
	}
	if len(b.cache) != 2 || b.cache[0].Len() != 1 {
		t.Fatal("Invalid cache len 2")
	}

	// Mark the last entry as stale.
	b.Unresponsive([]byte{allOnes}, "addr1")
	b.Unresponsive([]byte{allOnes - 1}, "addr2")
	b.Unresponsive([]byte{allOnes - 2}, "addr3")

	// Since the entry is marked as stale it should be replaced.
	if len(b.buckets) != 2 || !bytes.Equal(b.buckets[0].Entries()[0].Id, []byte{allOnes - 2}) {
		t.Fatal("Invalid bucket len 3")
	}
	if len(b.cache) != 2 || b.cache[0].Len() != 0 {
		t.Fatal("Invalid cache len 3")
	}
}
