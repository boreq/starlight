// Package local implements functionality related to communication beetween CLI
// clients and the local daemon.
package local

import (
	"fmt"
	"github.com/boreq/lainnet/core"
	"github.com/boreq/lainnet/local/backend"
	"github.com/boreq/lainnet/network/node"
	"net"
	"net/http"
	"net/rpc"
)

// RunServer runs a RPC server on a unix domain socket. The socket file will not
// be removed after the listener closes.
func RunServer(lainnet core.Lainnet, address string) error {
	bend := backend.NewBackend(lainnet)
	err := rpc.Register(bend)
	if err != nil {
		return err
	}

	rpc.HandleHTTP()
	listener, err := net.Listen("unix", address)
	if err != nil {
		return err
	}
	go http.Serve(listener, nil)
	return nil
}

// Dial connectes to a running RPC server on a unix domain socket.
func Dial(address string) (*rpc.Client, error) {
	return rpc.DialHTTP("unix", address)
}

// GetAddress returns an address of a RPC server for the provided node id.
func GetAddress(localId node.ID) string {
	return fmt.Sprintf("/tmp/lainnet_%s.socket", localId)
}
