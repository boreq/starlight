package kbuckets

import (
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/utils"
	"sort"
	"sync"
	"time"
)

type RoutingTable interface {
	Update(id node.ID, address string)
	Unresponsive(id node.ID, address string)
	GetClosest(id node.ID, a int) []node.NodeInfo
	PerformedLookup(id node.ID)
	GetForRefresh() []node.ID
	GetForInitialRefresh() []node.ID
}

func New(self node.ID, k int, refreshAfter time.Duration) RoutingTable {
	rw := &buckets{
		buckets:      []*bucket{&bucket{}},
		cache:        []*bucket{&bucket{}},
		k:            k,
		refreshAfter: refreshAfter,
		self:         self,
	}
	return rw
}

var log = utils.GetLogger("kbuckets")

type buckets struct {
	buckets      []*bucket
	cache        []*bucket
	k            int
	refreshAfter time.Duration
	self         node.ID
	lock         sync.Mutex
}

func (b *buckets) PerformedLookup(id node.ID) {
	b.lock.Lock()
	defer b.lock.Unlock()

	i, err := b.bucketIndex(id)
	if err != nil {
		return
	}

	now := time.Now()
	b.buckets[i].LastLookup = &now
}

func (b *buckets) GetForRefresh() []node.ID {
	b.lock.Lock()
	defer b.lock.Unlock()

	var rv []node.ID
	now := time.Now()
	for i, bu := range b.buckets {
		if bu.LastLookup == nil || (*bu.LastLookup).Add(b.refreshAfter).Before(now) {
			r := randomId(b.self, i)
			rv = append(rv, r)
		}
	}
	return rv
}

func (b *buckets) GetForInitialRefresh() []node.ID {
	var rv []node.ID
	return rv
}

func (b *buckets) Update(id node.ID, address string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.update(id, address)
}

func (b *buckets) Unresponsive(id node.ID, address string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	i, err := b.bucketIndex(id)
	if err != nil {
		return
	}

	b.buckets[i].Unresponsive(id, address)
	b.buckets[i].TryReplaceLast(b.cache[i])
}

func (b *buckets) update(id node.ID, address string) {
	i, err := b.bucketIndex(id)
	if err != nil {
		return
	}

	// If not full just insert.
	if b.buckets[i].Len() < b.k || b.buckets[i].Contains(id) {
		b.buckets[i].Update(id, address)
	} else {
		// Only the last bucket can be split and can we add more buckets?
		if i == len(b.buckets)-1 && len(b.buckets) < len(b.self)*8 {
			// Cache will always be empty at this point.
			if b.cache[i].Len() != 0 {
				log.Debug("Warning, replacement cache was not empty!")
				b.cache[i].Clear()
			}
			b.cache = append(b.cache, &bucket{})
			// Split buckets.
			tmp := b.buckets[i]
			b.buckets[i] = &bucket{}
			b.buckets = append(b.buckets, &bucket{})
			// Reinsert.
			for {
				entry, err := tmp.DropLast()
				if err != nil {
					break
				}
				b.update(entry.Id, entry.Address)
			}
			// Try insert new.
			b.update(id, address)
		} else {
			// We can't split, drop last and insert.
			b.cache[i].Update(id, address)
			if b.cache[i].Len() > b.k {
				b.cache[i].DropLast()
			}
			b.buckets[i].TryReplaceLast(b.cache[i])
		}
	}
}

// Returns an index of a bucket in which the given node should reside.
func (b *buckets) bucketIndex(id node.ID) (int, error) {
	dis, err := node.Distance(b.self, id)
	if err != nil {
		return 0, err
	}
	i := utils.ZerosLen(dis)
	if i >= len(b.buckets) {
		i = len(b.buckets) - 1
	}
	return i, nil
}

func (b *buckets) GetClosest(id node.ID, a int) []node.NodeInfo {
	b.lock.Lock()
	defer b.lock.Unlock()

	i, err := b.bucketIndex(id)
	if err != nil {
		return nil
	}

	if b.buckets[i].Len() >= a {
		return b.buckets[i].Entries()[:a]
	} else {
		se := &sortEntries{nil, id}
		for _, bucket := range b.buckets {
			se.e = append(se.e, bucket.Entries()...)
		}
		sort.Sort(se)

		var n int
		if a > len(se.e) {
			n = len(se.e)
		} else {
			n = a
		}
		return se.e[:n]
	}
}
