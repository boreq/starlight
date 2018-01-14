package kbuckets

import (
	"github.com/boreq/starlight/network/node"
)

type RoutingTable interface {
	Update(id node.ID, address string)
	Unresponsive(id node.ID, address string)
	GetClosest(id node.ID, a int) []node.NodeInfo
	PerformedLookup(id node.ID)
	GetForRefresh() []node.ID
	GetForInitialRefresh() []node.ID
}
