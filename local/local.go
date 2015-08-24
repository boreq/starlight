package local

import (
	"fmt"
	"github.com/boreq/netblog/core"
	"github.com/boreq/netblog/local/backend"
	"github.com/boreq/netblog/network/node"
	"net"
	"net/http"
	"net/rpc"
)

// RunServer runs a RPC server on a unix domain socket. The socket file will not
// be removed after the listener closes.
func RunServer(netblog core.Netblog, address string) error {
	bend := backend.NewBackend(netblog)
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
	return fmt.Sprintf("/tmp/netblog_%s.socket", localId)
}