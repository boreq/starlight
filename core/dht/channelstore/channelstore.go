// Package channelstore provides a data structure used for storing StoreChannel
// messages received by the local DHT node. Those messages need to be grouped
// by the channel id so this data structure is used instead of the one provided
// by the package datastore.
package channelstore

import (
	"fmt"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol/message"
	"sync"
	"time"
)

// New creates a new channelstore. The stored messages are removed when they are
// older than the threshold parameter.
func New(threshold time.Duration) *Channelstore {
	rv := &Channelstore{
		items:     make(map[string][]*message.StoreChannel),
		threshold: threshold,
	}
	return rv
}

// Channelstore stores StoreChannel messages which have not passed a certain age
// relative to the signing time.
type Channelstore struct {
	items     map[string][]*message.StoreChannel
	threshold time.Duration
	mutex     sync.Mutex
}

// Store inserts a new entry. If the new entry has an older timestamp than an
// entry already present in the channelstore then the entry will not be inserted
// and the method will return no errors.
func (d *Channelstore) Store(msg *message.StoreChannel) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	sKey := idToKey(msg.GetChannelId())
	d.cleanup(sKey)
	_, ok := d.items[sKey]
	if !ok {
		d.items[sKey] = []*message.StoreChannel{msg}
	} else {
		newT := time.Unix(msg.GetTimestamp(), 0)
		for i, m := range d.items[sKey] {
			if node.CompareId(msg.GetNodeId(), m.GetNodeId()) {
				oldT := time.Unix(m.GetTimestamp(), 0)
				if newT.After(oldT) {
					d.items[sKey][i] = msg
				}
				return nil
			}
		}
		d.items[sKey] = append(d.items[sKey], msg)
	}
	return nil
}

// Get returns a list of entries related to the given channel. A stale data will
// never be returned.
func (d *Channelstore) Get(key []byte) []*message.StoreChannel {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	sKey := idToKey(key)
	d.cleanup(sKey)
	item, ok := d.items[sKey]
	if !ok {
		return []*message.StoreChannel{}
	} else {
		return item
	}
}

// cleanup removes the stale data.
func (d *Channelstore) cleanup(key string) {
	if _, ok := d.items[key]; ok {
		n := time.Now()
		for i, msg := range d.items[key] {
			t := time.Unix(msg.GetTimestamp(), 0)
			if t.Add(d.threshold).Before(n) {
				d.items[key] = append(d.items[key][:i], d.items[key][i+1:]...)
			}
		}
		if len(d.items[key]) == 0 {
			delete(d.items, key)
		}
	}
}

// idToKey converts a channel id to a string which can be used as a map key.
func idToKey(id []byte) string {
	return fmt.Sprintf("%x", id)
}
