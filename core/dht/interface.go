package dht

import (
	"github.com/boreq/netblog/core/dht/kbuckets"
	"github.com/boreq/netblog/network/node"
)

type DHT interface {
	Init([]node.NodeInfo) error
	Ping(node.ID) error
	FindNode(node.ID) (node.NodeInfo, error)
}

type RoutingTable interface {
	Update(id node.ID, address string)
	Get(id node.ID) string
	//GetClosest(id node.ID, a int) []node.NodeInfo
}

func NewRoutingTable() RoutingTable {
	rw := &kbuckets.Buckets{}
	return rw
}
