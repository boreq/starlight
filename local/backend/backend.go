package backend

import (
	"github.com/boreq/lainnet/core"
)

func NewBackend(lainnet core.Lainnet) *Backend {
	rw := &Backend{
		lainnet: lainnet,
	}
	return rw
}

// Backend is an object which is registered on an RPC object and is used to
// execute commands on a running daemon.
type Backend struct {
	lainnet core.Lainnet
}
