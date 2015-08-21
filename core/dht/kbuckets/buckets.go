package kbuckets

import (
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/utils"
	"sort"
)

type RoutingTable interface {
	Update(id node.ID, address string)
	GetClosest(id node.ID, a int) []node.NodeInfo
}

func New(self node.ID, k int) RoutingTable {
	rw := &buckets{
		buckets: []*bucket{&bucket{}},
		k:       k,
		self:    self,
	}
	return rw
}

type buckets struct {
	buckets []*bucket
	k       int
	self    node.ID
}

func (b *buckets) Update(id node.ID, address string) {
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
			// Split.
			tmp := b.buckets[i]
			b.buckets[i] = &bucket{}
			b.buckets = append(b.buckets, &bucket{})
			// Reinsert.
			for {
				entry, err := tmp.DropLast()
				if err != nil {
					break
				}
				b.Update(entry.Id, entry.Address)
			}
			// Try insert new.
			b.Update(id, address)
		} else {
			// We can't split, drop last and insert.
			b.buckets[i].DropLast()
			b.buckets[i].Update(id, address)
		}
	}
}

// Returns an index of a bucket in which the given node should reside.
func (b *buckets) bucketIndex(id node.ID) (int, error) {
	dis, err := distance(b.self, id)
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
		return se.e[:a]
	}
}

// Calculates the distance between two nodes.
func distance(a, b node.ID) ([]byte, error) {
	// XOR is the distance metric, to actually get a meaningful distance
	// from it we just count the preceeding zeros.
	return utils.XOR(a, b)
}
