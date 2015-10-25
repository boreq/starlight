package backend

import (
	"github.com/boreq/lainnet/network/node"
)

type PingArgs struct {
	NodeId string
}

// Ping is an RPC call used by the ping CLI command.
func (b *Backend) Ping(args *PingArgs, latency *float64) error {
	id, err := node.NewId(args.NodeId)
	if err != nil {
		return err
	}

	duration, err := b.lainnet.Dht().Ping(id)
	if err != nil {
		return err
	}

	*latency = duration.Seconds()
	return nil
}
