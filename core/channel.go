package core

import (
	"errors"
	"github.com/boreq/lainnet/core/channel"
	"golang.org/x/net/context"
	"time"
)

var AlreadyInChannelError = errors.New("Already joined this channel")
var NotInChannelError = errors.New("Not in the channel")

// channelBootstrapInterval specifies how often bootstrapChannel is run.
var channelBootstrapInterval = 5 * time.Minute

func (n *lainnet) JoinChannel(name string) error {
	n.channelsMutex.Lock()
	defer n.channelsMutex.Unlock()

	if n.inChannel(name) {
		return AlreadyInChannelError
	} else {
		ch := channel.NewChannel(n.ctx, n.ident.Id, name)
		go n.runBootstrapChannel(ch.Ctx, channelBootstrapInterval, ch)
		n.channels = append(n.channels, ch)
		return nil
	}
}

func (n *lainnet) PartChannel(name string) error {
	n.channelsMutex.Lock()
	defer n.channelsMutex.Unlock()

	for i, ch := range n.channels {
		if ch.Name == name {
			err := ch.Cancel()
			if err == nil {
				n.channels = append(n.channels[:i], n.channels[i+1:]...)
			}
			return err
		}
	}
	return NotInChannelError
}

// inChannel returns true if a channel has already been joined.
func (n *lainnet) inChannel(name string) bool {
	for _, ch := range n.channels {
		if ch.Name == name {
			return true
		}
	}
	return false
}

// runBootstrapChannel runs bootstrapChannel immediately after it is called and
// then continues to run it in the specified interval until the context is
// closed.
func (n *lainnet) runBootstrapChannel(ctx context.Context, interval time.Duration, ch *channel.Channel) {
	n.bootstrapChannel(ctx, ch)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			n.bootstrapChannel(ctx, ch)
		case <-ctx.Done():
			return
		}
	}
}

// bootstrapChannel performs required housekeeping procedures related to being
// in a channel such as republishing the information about that in the DHT.
func (n *lainnet) bootstrapChannel(ctx context.Context, ch *channel.Channel) {
	log.Debugf("bootstrapChannel %s", ch.Name)
	n.dht.PutChannel(ctx, ch.Id)
}
