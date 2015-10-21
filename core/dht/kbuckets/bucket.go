package kbuckets

import (
	"container/list"
	"errors"
	"github.com/boreq/lainnet/network/node"
)

type bucket struct {
	entries list.List
}

// Len returns the number of elements in a bucket.
func (b *bucket) Len() int {
	return b.entries.Len()
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
		rw[i] = el.Value.(node.NodeInfo)
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
	en := node.NodeInfo{id, address}
	b.entries.PushFront(en)
}

// DropLast removes the last entry from the bucket and returns it.
func (b *bucket) DropLast() (*node.NodeInfo, error) {
	el := b.entries.Back()
	if el == nil {
		return nil, errors.New("Bucket is empty")
	}
	entry := b.entries.Remove(el).(node.NodeInfo)
	return &entry, nil
}

// Find returns a list element which stores an entry with the given id.
func (b *bucket) find(id node.ID) *list.Element {
	for el := b.entries.Front(); el != nil; el = el.Next() {
		en := el.Value.(node.NodeInfo)
		if node.CompareId(en.Id, id) {
			return el
		}
	}
	return nil
}
