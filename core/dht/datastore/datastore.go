// Package datastore provides a data structure used for storing <key, value>
// pairs for a certain amount of time.
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
	data interface{}
	time time.Time
}

// Datastore stores <key, value> pairs for a certain amount of time.
type Datastore struct {
	items     map[string]item
	threshold time.Duration
}

// Store inserts a new entry.
func (d *Datastore) Store(key []byte, data interface{}) error {
	d.cleanup()
	sKey := convertKey(key)
	d.items[sKey] = item{data, time.Now()}
	return nil
}

// Get returns an entry from the datastore. A stale entry can be returned as
// there is no guarantee that an item will be removed immediately after the
// specified amount of time passes.
func (d *Datastore) Get(key []byte) (interface{}, error) {
	sKey := convertKey(key)
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

// convertKey converts a key in bytes to a string required by maps.
func convertKey(key []byte) string {
	return fmt.Sprintf("%x", key)
}
