package kbuckets

import (
	"container/list"
	"errors"
	"github.com/boreq/lainnet/network/node"
)

type bucketEntry struct {
	Node  node.NodeInfo
	Stale bool
}

type bucket struct {
	entries list.List
}

// Len returns the number of elements in a bucket.
func (b *bucket) Len() int {
	return b.entries.Len()
}

// Clear removes all entries from the bucket.
func (b *bucket) Clear() {
	b.entries.Init()
}

// Contains checks if an entry already exists in a bucket.
func (b *bucket) Contains(id node.ID) bool {
	return b.find(id) != nil
}

// Entries returns a slice with all entries in this bucket.
func (b *bucket) Entries() []node.NodeInfo {
	rw := make([]node.NodeInfo, b.Len())
	i := 0
	for el := b.entries.Front(); el != nil; el = el.Next() {
		rw[i] = el.Value.(*bucketEntry).Node
		i++
	}
	return rw
}

// Update adds a new entry at the front of the bucket or updates the address of
// an already existing entry and moves it to front of the bucket.
func (b *bucket) Update(id node.ID, address string) {
	el := b.find(id)
	if el != nil {
		b.entries.Remove(el)
	}
	en := &bucketEntry{node.NodeInfo{id, address}, false}
	b.entries.PushFront(en)
}

// Unresponsive marks an entry as unresponsive.
func (b *bucket) Unresponsive(id node.ID, address string) {
	el := b.find(id)
	if el != nil {
		en := el.Value.(*bucketEntry)
		if en.Node.Address == address {
			en.Stale = true
		}
	}
}

// TryReplaceLast checks if the last entry in this bucket is marked as stale
// and if it is the case removes it, pops the first entry from c (which is
// presumed to be a replacement cache) and inserts it into this bucket.
func (b *bucket) TryReplaceLast(c *bucket) error {
	el := b.entries.Back()
	entry := el.Value.(*bucketEntry)
	if !entry.Stale {
		return errors.New("The last entry is not stale")
	}
	nd, err := c.DropFirst()
	if err != nil {
		return err
	}
	b.DropLast()
	b.Update(nd.Id, nd.Address)
	return nil
}

// DropLast removes the last entry from the bucket and returns it.
func (b *bucket) DropLast() (*node.NodeInfo, error) {
	el := b.entries.Back()
	if el == nil {
		return nil, errors.New("Bucket is empty")
	}
	entry := b.entries.Remove(el).(*bucketEntry).Node
	return &entry, nil
}

// DropFirst removes the first entry from the bucket and returns it.
func (b *bucket) DropFirst() (*node.NodeInfo, error) {
	el := b.entries.Front()
	if el == nil {
		return nil, errors.New("Bucket is empty")
	}
	entry := b.entries.Remove(el).(*bucketEntry).Node
	return &entry, nil
}

// Find returns a list element which stores an entry with the given id.
func (b *bucket) find(id node.ID) *list.Element {
	for el := b.entries.Front(); el != nil; el = el.Next() {
		en := el.Value.(*bucketEntry).Node
		if node.CompareId(en.Id, id) {
			return el
		}
	}
	return nil
}
