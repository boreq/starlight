package channel

import (
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/utils"
	"sync"
	"time"
)

// newBuckets creates new buckets initialized with the local node's id. Each
// bucket holds up to k nodes.
func newBuckets(id node.ID, k int) *Buckets {
	rv := &Buckets{
		self:    id,
		buckets: make([]bucket, len(id)*8),
		k:       k,
	}
	return rv
}

// Buckets are used to store members present in a specified channel. They are
// used to propagate channel related messages.
type Buckets struct {
	self    node.ID
	buckets []bucket
	k       int
	lock    sync.Mutex
}

// Inserts inserts an id into the buckets. The entry is removed after the
// specified point in time passes.
func (b *Buckets) Insert(id node.ID, t time.Time) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	i, err := b.bucketIndex(id)
	if err != nil {
		return err
	}

	// Remove old entries from the bucket.
	b.buckets[i].Cleanup()

	// If the buckets is not full insert, otherwise don't do anything.
	if b.buckets[i].Len() < b.k || b.buckets[i].Contains(id) {
		b.buckets[i].Insert(id, t)
	}
	return nil
}

// Get picks i random nodes from each bucket and returns all of them. That means
// that the total number of the returned nodes will be <= len(b.self) * 8 * i.
func (b *Buckets) Get(i int) []node.ID {
	var rv []node.ID
	for j := 0; j < len(b.buckets); j++ {
		b.buckets[j].Cleanup()
		nodes := b.buckets[j].Get(i)
		rv = append(rv, nodes...)
	}
	return rv
}

func (b *Buckets) bucketIndex(id node.ID) (int, error) {
	dis, err := node.Distance(b.self, id)
	if err != nil {
		return 0, err
	}
	i := utils.ZerosLen(dis)
	return i, nil
}
