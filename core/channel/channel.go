// Package channel provides a structure used to keep track of other channel
// members.
package channel

import (
	"github.com/boreq/lainnet/network/node"
	"golang.org/x/net/context"
)

// The amount of entries in each bucket.
const k = 10

// Creates a new channel and uses the provided context to create a child context
// used to stop async taks related to a channel. Id should be set to the id of
// the local node - it is used to initialize the buckets, name is the name of
// the channel.
func NewChannel(ctx context.Context, id node.ID, name string) *Channel {
	ctx, cancel := context.WithCancel(ctx)
	rv := &Channel{
		Name:   name,
		Id:     CreateId(name),
		Users:  newBuckets(id, k),
		Ctx:    ctx,
		cancel: cancel,
	}
	return rv
}

// Channel keeps track of other channel members and stores a context which is
// used for certain channel related activities such as the bootstrap method.
type Channel struct {
	Name   string
	Id     []byte
	Users  *Buckets
	Ctx    context.Context
	cancel context.CancelFunc
}

// Cancel closes the channel context.
func (c *Channel) Cancel() error {
	c.cancel()
	return nil
}
