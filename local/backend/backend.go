package backend

import (
	"github.com/boreq/lainnet/core"
)

func NewBackend(netblog core.Netblog) *Backend {
	rw := &Backend{
		netblog: netblog,
	}
	return rw
}

// Backend is an object which is registered on an RPC object and is used to
// execute commands on a running daemon.
type Backend struct {
	netblog core.Netblog
}
