package channel

import (
	"container/list"
	"errors"
	"github.com/boreq/lainnet/network/node"
	"math/rand"
	"time"
)

type entry struct {
	id   node.ID
	time time.Time
}

type bucket struct {
	entries list.List
}

// Insert inserts a new node into the bucket or updates the time of an existing
// entry.
func (b *bucket) Insert(id node.ID, t time.Time) {
	if e := b.find(id); e != nil {
		entry := e.Value.(*entry)
		if entry.time.Before(t) {
			entry.time = t
		}
	} else {
		b.insert(id, t)
	}
}

// insert inserts a new entry into the bucket.
func (b *bucket) insert(id node.ID, t time.Time) {
	e := &entry{
		id:   id,
		time: t,
	}
	b.entries.PushBack(e)
}

// Remove removes a node from the bucket and returns an error if the entry
// with the given id was not present.
func (b *bucket) Remove(id node.ID) error {
	for e := b.entries.Front(); e != nil; e = e.Next() {
		entry := e.Value.(*entry)
		if node.CompareId(entry.id, id) {
			b.entries.Remove(e)
			return nil
		}
	}
	return errors.New("Entry not found")
}

// Contains checks if an entry already exists in the bucket.
func (b *bucket) Contains(id node.ID) bool {
	return b.find(id) != nil
}

// find attempts to locate an element corresponding to the id and returns nil
// when the element doesn't exist.
func (b *bucket) find(id node.ID) *list.Element {
	for e := b.entries.Front(); e != nil; e = e.Next() {
		entry := e.Value.(*entry)
		if node.CompareId(entry.id, id) {
			return e
		}
	}
	return nil
}

// Len returns the number of nodes in this bucket.
func (b *bucket) Len() int {
	return b.entries.Len()
}

// Get returns i random nodes from the bucket.
func (b *bucket) Get(i int) []node.ID {
	var l int
	ints := rand.Perm(b.entries.Len())

	// Make sure that i is not greater than the number of nodes in the bucket.
	if len(ints) > i {
		l = i
	} else {
		l = len(ints)
	}

	rv := make([]node.ID, l)
	// Iterate l times to pick l elements.
	for i := 0; i < l; i++ {
		// Find and append ints[i]-th element.
		j := 0
		for e := b.entries.Front(); e != nil; e = e.Next() {
			if j == ints[i] {
				entry := e.Value.(*entry)
				rv[i] = entry.id
			}
			j++
		}
	}
	return rv
}

// Cleanup removes all entries with time property in the past.
func (b *bucket) Cleanup() {
	now := time.Now().UTC()
	var next *list.Element
	for e := b.entries.Front(); e != nil; e = next {
		next = e.Next()
		entry := e.Value.(*entry)
		if now.After(entry.time) {
			b.entries.Remove(e)
		}
	}
}
