package dht

import (
	"errors"
	"github.com/boreq/lainnet/core/channel"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol/message"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"time"
)

func (d *dht) PutChannel(ctx context.Context, id []byte) error {
	// Prepare a message.
	msg, err := createStoreChannelMessage(d.self.PrivKey, d.self.Id, id)

	// Locate the closest nodes.
	nodes, err := d.findNode(ctx, id, false)
	if err != nil {
		return err
	}

	// Send 'k' store RPCs. We don't have to wait for this to finish so
	// a goroutine with the DHT's context is used instead of blocking.
	go func() {
		counter := 0
		for _, nodeInfo := range nodes {
			peer, err := d.net.Dial(nodeInfo)
			if err == nil {
				err := peer.SendWithContext(d.ctx, msg)
				if err == nil {
					counter++
					if counter > k {
						return
					}
				}
			}
		}
	}()
	return nil
}

// handlePutChannelMsg processes an incoming StoreChannel message.
func (d *dht) handleStoreChannelMsg(ctx context.Context, msg *message.StoreChannel) error {
	err := d.validateStoreChannelMessage(ctx, msg)
	if err != nil {
		return err
	}
	// TODO store
	return nil
}

// createStoreChannelMessage creates a StoreChannel message which can be sent
// to other nodes.
func createStoreChannelMessage(key crypto.PrivateKey, nodeId node.ID, channelId []byte) (*message.StoreChannel, error) {
	timestamp := time.Now().UTC().Unix()
	msg := &message.StoreChannel{
		ChannelId: channelId,
		NodeId:    nodeId,
		Timestamp: &timestamp,
		Signature: nil,
	}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	signature, err := key.Sign(msgBytes, signingHash)
	msg.Signature = signature
	return msg, nil
}

// If a message is further in the future than putChannelFutureThreshold then
// it is rejected.
var putChannelFutureThreshold = 30 * time.Second

// validateStoreChannelMessage returns an error if a StoreChannel message is
// invalid and should not be processed (stored).
func (d *dht) validateStoreChannelMessage(ctx context.Context, msg *message.StoreChannel) error {
	if !node.ValidateId(msg.GetNodeId()) {
		return errors.New("Invalid node id")
	}

	if !channel.ValidateId(msg.GetChannelId()) {
		return errors.New("Invalid channel id")
	}

	t := time.Unix(msg.GetTimestamp(), 0)
	if t.After(time.Now().UTC().Add(putChannelFutureThreshold)) {
		return errors.New("Timestamp is too far in the future")
	}

	// Recreate bytes which were signed - encoded message without the signature.
	var tmp *message.StoreChannel
	*tmp = *msg
	tmp.Signature = nil
	msgBytes, err := proto.Marshal(tmp)
	if err != nil {
		return err
	}

	// Confirm the signature.
	key, err := d.GetPubKey(ctx, msg.GetNodeId())
	if err != nil {
		return err
	}
	err = key.Validate(msgBytes, msg.Signature, signingHash)
	if err != nil {
		return err
	}

	return nil
}
