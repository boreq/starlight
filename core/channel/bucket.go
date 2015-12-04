package channel

import (
	"container/list"
	"errors"
	"github.com/boreq/lainnet/network/node"
	"time"
)

type entry struct {
	id   node.ID
	time time.Time
}

type bucket struct {
	entries list.List
}

// Insert inserts an id into the bucket.
func (b *bucket) Insert(id node.ID) {
	e := entry{
		id:   id,
		time: time.Now(),
	}
	b.entries.PushBack(e)
}

// Remove removes the entry from the bucket and returns an error if the entry
// with the given id is not present.
func (b *bucket) Remove(id node.ID) error {
	for e := b.entries.Front(); e != nil; e = e.Next() {
		entry := e.Value.(entry)
		if node.CompareId(entry.id, id) {
			b.entries.Remove(e)
			return nil
		}
	}
	return errors.New("Entry not found")
}

// Contains checks if an entry already exists in a bucket.
func (b *bucket) Contains(id node.ID) bool {
	for e := b.entries.Front(); e != nil; e = e.Next() {
		entry := e.Value.(entry)
		if node.CompareId(entry.id, id) {
			return true
		}
	}
	return false
}

func (b *bucket) Len() int {
	return b.entries.Len()
}
