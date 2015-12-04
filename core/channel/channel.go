package channel

import (
	"github.com/boreq/lainnet/network/node"
	"golang.org/x/net/context"
)

// The amount of entries in each bucket.
const k = 10

// Creates a new channel and uses the provided context to create a child context
// used to stop async taks related to a channel.
func NewChannel(ctx context.Context, id node.ID, name string) *Channel {
	ctx, cancel := context.WithCancel(ctx)
	rv := &Channel{
		Name:   name,
		Id:     CreateId(name),
		users:  NewBuckets(id, k),
		Ctx:    ctx,
		cancel: cancel,
	}
	return rv
}

type Channel struct {
	Name   string
	Id     []byte
	users  *buckets
	Ctx    context.Context
	cancel context.CancelFunc
}

func (c *Channel) Cancel() error {
	c.cancel()
	return nil
}
