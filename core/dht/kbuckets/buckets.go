package kbuckets

import (
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/utils"
)

// Entry is used to store information about a node in a bucket.
type Entry struct {
	Id      node.ID
	Address string
}

// Implements RoutingTable.
type Buckets struct {
	buckets []*bucket
	k       int
	self    node.ID
}

func (b *Buckets) Init() {
	b.buckets = []*bucket{&bucket{}}
}

func (b *Buckets) Update(id node.ID, address string) {
	i, err := bucketIndex(b.self, id)
	if err != nil {
		return
	}

	if i >= len(b.buckets) {
		i = len(b.buckets) - 1
	}

	// If not full, insert.
	if b.buckets[i].Len() < b.k {
		b.buckets[i].Update(b.k, id, address)
	} else {
		// TODO
		//b.buckets[]
	}

	if b.buckets[i].Contains(id) {
	}
}

func (b *Buckets) Get(id node.ID) string {
	return ""
}

// Returns an index of a bucket in which one of the given nodes should reside
// as long as the other value is the local node id.
func bucketIndex(a, b node.ID) (int, error) {
	dis, err := distance(a, b)
	if err != nil {
		return 0, err
	}
	return utils.ZerosLen(dis), nil
}

// Calculates the distance between two nodes.
func distance(a, b node.ID) ([]byte, error) {
	// XOR is the distance metric, to actually get a meaningful distance
	// from it we just count the preceeding zeros.
	return utils.XOR(a, b)
}
