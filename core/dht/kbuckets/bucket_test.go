package kbuckets

import "testing"

// TestBucket ensures that the most basic functionality is in order by inserting
// a single <id, addr> pair into the bucket.
func TestBucket(t *testing.T) {
	id := []byte{0}
	addr := "addr"

	b := &bucket{}
	if b.Len() != 0 {
		t.Fatal("Bucket should be empty")
	}
	if b.Contains(id) {
		t.Fatal("Bucket should not contain the id")
	}

	b.Update(id, addr)
	if b.Len() != 1 {
		t.Fatal("Bucket should contain one entry")
	}
	if !b.Contains(id) {
		t.Fatal("Bucket should contain the id")
	}
}

// TestTryReplace tests the replacement cache mechanism.
func TestTryReplace(t *testing.T) {
	b := &bucket{}
	b.Update([]byte{0}, "addr0")
	b.Update([]byte{1}, "addr1")

	c := &bucket{}
	c.Update([]byte{2}, "addr2")

	// Fail, the entry in the bucket is not stale.
	err := b.TryReplaceLast(c)
	if err == nil {
		t.Fatal("This should fail because the entry is not stale")
	}
	if b.Len() != 2 || c.Len() != 1 {
		t.Fatal("Invalid lengths", b.Len(), c.Len())
	}

	// Succeed after marking the entry as stale.
	b.Unresponsive([]byte{0}, "addr0")
	err = b.TryReplaceLast(c)
	if err != nil {
		t.Fatal("The entry is stale, this should succeed, error:", err)
	}
	if b.Len() != 2 || c.Len() != 0 {
		t.Fatal("Invalid lengths", b.Len(), c.Len())
	}
}

// TestTryReplaceEmpty tests the replacement cache mechanism with an empty bucket.
func TestTryReplaceEmpty(t *testing.T) {
	b := &bucket{}
	c := &bucket{}

	// Fail, the entry in the bucket is not stale.
	err := b.TryReplaceLast(c)
	if err == nil {
		t.Fatal("This should fail because the bucket and the cache are empty")
	}
}
