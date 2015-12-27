package channel

import (
	"testing"
	"time"
)

func TestBucketsGetCleanup(t *testing.T) {
	self := []byte{0}
	t1 := time.Now().UTC().Add(1 * time.Second)

	b := NewBuckets(self, 10)
	var i byte
	for i = 1; i < 10; i++ {
		b.Insert([]byte{i}, t1)
	}

	<-time.After(2 * time.Second)
	nodes := b.Get(1)
	if len(nodes) != 0 {
		t.Fatal("Cleanup didn't work")
	}

}

func TestBucketsGet(t *testing.T) {
	self := []byte{0}
	t1 := time.Now().UTC().Add(1 * time.Second)

	b := NewBuckets(self, 10)
	var i byte
	for i = 1; i < 10; i++ {
		b.Insert([]byte{i}, t1)
	}

	nodes := b.Get(1)
	if len(nodes) == 0 {
		t.Fatal("Got 0 nodes")
	}

}
