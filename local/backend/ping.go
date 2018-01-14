package backend

import (
	"github.com/boreq/starlight/network/node"
	"golang.org/x/net/context"
)

type PingArgs struct {
	NodeId string
}

// Ping is a RPC used by the ping CLI command.
func (b *Backend) Ping(args *PingArgs, latency *float64) error {
	id, err := node.NewId(args.NodeId)
	if err != nil {
		return err
	}

	duration, err := b.core.Dht().Ping(context.TODO(), id)
	if err != nil {
		return err
	}

	*latency = duration.Seconds()
	return nil
}
