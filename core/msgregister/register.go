// Package msgregister provides a data structure used for keeping track of
// already forwarded channel messages.
package msgregister

import (
	"bytes"
	"container/list"
	"errors"
	"sync"
	"time"
)

func New() *Register {
	rv := &Register{
		entries: list.New(),
	}
	return rv
}

type entry struct {
	Id   []byte
	Time time.Time
}

// Register stores data until a specified point in time passes.
type Register struct {
	entries *list.List
	mutex   sync.Mutex
}

// Insert attempts to insert an id into the register. If the same id already
// exists in the register an error is returned. The id will be stored until
// the specified point in time.
func (r *Register) Insert(id []byte, t time.Time) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.cleanup()
	e := r.find(id)
	if e != nil {
		return errors.New("already in the register")
	}

	newEntry := entry{
		Id:   id,
		Time: t,
	}
	r.entries.PushBack(newEntry)
	return nil
}

// cleanup removes all old entires.
func (r *Register) cleanup() {
	now := time.Now().UTC()
	var next *list.Element
	for e := r.entries.Front(); e != nil; e = next {
		next = e.Next()
		entry := e.Value.(entry)
		if now.After(entry.Time) {
			r.entries.Remove(e)
		}
	}
}

// find returns a list element which is used to store the specified id.
func (r *Register) find(id []byte) *list.Element {
	for e := r.entries.Front(); e != nil; e = e.Next() {
		entry := e.Value.(entry)
		if bytes.Equal(entry.Id, id) {
			return e
		}
	}
	return nil
}
