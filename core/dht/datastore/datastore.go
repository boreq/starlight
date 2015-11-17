package datastore

import (
	"errors"
	"fmt"
	"time"
)

// New creates a new datastore. Threshold is the duration after which the data
// is removed from the datastore.
func New(threshold time.Duration) *Datastore {
	rv := &Datastore{
		items:     make(map[string]item),
		threshold: threshold,
	}
	return rv
}

type item struct {
	data []byte
	time time.Time
}

// Datastore stores <key, value> pairs for a certain amount of time.
type Datastore struct {
	items     map[string]item
	threshold time.Duration
}

// Store removes the stale data and stores a new entry.
func (d *Datastore) Store(key, data []byte) error {
	d.cleanup()
	sKey := fmt.Sprintf("%x", key)
	d.items[sKey] = item{data, time.Now()}
	return nil
}

// Get removes an entry from the datastore. A stale entry can be removed.
func (d *Datastore) Get(key []byte) ([]byte, error) {
	sKey := fmt.Sprintf("%x", key)
	item, ok := d.items[sKey]
	if !ok {
		return nil, errors.New("Not found")
	} else {
		return item.data, nil
	}
}

// cleanup removes the stale data.
func (d *Datastore) cleanup() {
	n := time.Now()
	for key, item := range d.items {
		if item.time.Add(d.threshold).Before(n) {
			delete(d.items, key)
		}
	}
}
