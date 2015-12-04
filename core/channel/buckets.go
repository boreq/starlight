package channel

import (
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/utils"
	"sync"
)

// NewBuckets creates new buckets initialized with the local node's id.
func NewBuckets(id node.ID, k int) *buckets {
	rv := &buckets{
		self:    id,
		buckets: make([]bucket, len(id)*8),
		k:       k,
	}
	return rv
}

type buckets struct {
	self    node.ID
	buckets []bucket
	k       int
	lock    sync.Mutex
}

// Inserts inserts an id into the buckets.
func (b *buckets) Insert(id node.ID) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	i, err := b.bucketIndex(id)
	if err != nil {
		return err
	}

	// If not full and not in the bucket insert, otherwise don't do anything.
	if b.buckets[i].Len() < b.k || b.buckets[i].Contains(id) {
		b.buckets[i].Insert(id)
	}
	return nil
}

// Remove removes an id from the buckets.
func (b *buckets) Remove(id node.ID) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	i, err := b.bucketIndex(id)
	if err != nil {
		return err
	}
	b.buckets[i].Remove(id)
	return nil
}

func (b *buckets) bucketIndex(id node.ID) (int, error) {
	dis, err := node.Distance(b.self, id)
	if err != nil {
		return 0, err
	}
	i := utils.ZerosLen(dis)
	return i, nil
}
