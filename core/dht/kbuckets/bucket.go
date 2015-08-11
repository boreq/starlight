package kbuckets

import (
	"container/list"
	"errors"
	"github.com/boreq/netblog/network/node"
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
	if b.find(id) != nil {
		return true
	} else {
		return false
	}
}

// Update updates the address of an entry and moves it to front of the bucket.
func (b *bucket) Update(k int, id node.ID, address string) {
	el := b.find(id)
	if el != nil {
		b.entries.Remove(el)
	}
	en := Entry{id, address}
	b.entries.PushFront(en)
}

// DropLast removes the last entry from the bucket and returns it.
func (b *bucket) DropLast() (Entry, error) {
	el := b.entries.Back()
	if el == nil {
		return Entry{}, errors.New("Bucket is empty")
	}
	return b.entries.Remove(el).(Entry), nil
}

// Find returns a list element with stores an Entry with a given id.
func (b *bucket) find(id node.ID) *list.Element {
	for el := b.entries.Front(); el != nil; el = el.Next() {
		en := el.Value.(Entry)
		if node.CompareId(en.Id, id) {
			return el
		}
	}
	return nil
}
