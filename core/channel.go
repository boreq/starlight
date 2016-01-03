package core

import (
	"bytes"
	"errors"
	"github.com/boreq/lainnet/core/channel"
	"github.com/boreq/lainnet/core/dht"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol/message"
	"golang.org/x/net/context"
	"time"
)

var AlreadyInChannelError = errors.New("Already joined this channel")
var NotInChannelError = errors.New("Not in the channel")

// channelBootstrapInterval specifies how often bootstrapChannel is run.
const channelBootstrapInterval = 5 * time.Minute

// channelA is a concurency parameter used when resending channel related
// messages to other nodes.
const channelA = 3

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

func (n *lainnet) ListChannels() []string {
	n.channelsMutex.Lock()
	defer n.channelsMutex.Unlock()

	var rv []string
	for _, ch := range n.channels {
		rv = append(rv, ch.Name)
	}
	return rv
}

// inChannel returns true if the local node is already in the channel.
func (n *lainnet) inChannel(name string) bool {
	id := channel.CreateId(name)
	return n.getChannel(id) != nil
}

// getChannel returns a pointer to a channel.Channel or nil if the channel has
// not been joined.
func (n *lainnet) getChannel(id []byte) *channel.Channel {
	for _, ch := range n.channels {
		if bytes.Equal(ch.Id, id) {
			return ch
		}
	}
	return nil
}

// runBootstrapChannel runs bootstrapChannel immediately after it is called and
// then continues to run it periodically until the context is closed.
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

// userTimeout specifies the time after which a user is removed from the channel
// buckets.
var userTimeout = 5 * time.Minute

// bootstrapChannel performs required housekeeping procedures related to being
// in a channel such as republishing the information about that in the DHT.
func (n *lainnet) bootstrapChannel(ctx context.Context, ch *channel.Channel) {
	log.Debugf("bootstrapChannel %s", ch.Name)

	// Republish.
	err := n.dht.PutChannel(ctx, ch.Id)
	if err != nil {
		log.Debugf("bootstrapChannel %s put, error %s", ch.Name, err)
	}

	// Get new.
	ids, err := n.dht.GetChannel(ctx, ch.Id)
	if err != nil {
		log.Debugf("bootstrapChannel %s get, error %s", ch.Name, err)
	}

	// Insert into buckets.
	t := time.Now().UTC().Add(userTimeout)
	for _, id := range ids {
		if !node.CompareId(id, n.ident.Id) {
			ch.Users.Insert(id, t)
		}
	}
}

// handleFindChannelMsg is only responsible for additionally sending a store
// channel message of the local node. Stored messages are returned on the DHT
// level.
func (n *lainnet) handleFindChannelMsg(msg *message.FindChannel, sender node.ID) {
	n.channelsMutex.Lock()
	defer n.channelsMutex.Unlock()

	if ch := n.getChannel(msg.GetChannelId()); ch != nil {
		// Create my own store channel message.
		msg, err := dht.CreateStoreChannelMessage(n.ident.PrivKey, n.ident.Id, ch.Id)
		if err != nil {
			return
		}

		// Send my own store channel message.
		ctx, cancel := context.WithTimeout(n.ctx, 60*time.Second)
		defer cancel()
		peer, err := n.dht.Dial(ctx, sender)
		if err == nil {
			peer.SendWithContext(ctx, msg)
		}
	}
}

// handleStoreChannelMsg is only responsible for forwarding of the message to
// other channel members as the message itself is stored on the DHT level.
func (n *lainnet) handleStoreChannelMsg(msg *message.StoreChannel, sender node.ID) {
	n.channelsMutex.Lock()
	defer n.channelsMutex.Unlock()

	if ch := n.getChannel(msg.GetChannelId()); ch != nil {
		// Forward the message to other channel members.
		go func() {
			for _, id := range ch.Users.Get(channelA) {
				peer, err := n.dht.Dial(n.ctx, id)
				if err == nil {
					peer.Send(msg)
				}
			}
		}()

		// Insert the original sender into the channel buckets.
		t := time.Now().UTC().Add(userTimeout)
		ch.Users.Insert(msg.GetNodeId(), t)
	}
}
