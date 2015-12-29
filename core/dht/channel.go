package dht

import (
	"bytes"
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
	if err != nil {
		return err
	}

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

func (d *dht) GetChannel(ctx context.Context, id []byte) ([]node.ID, error) {
	var rv []node.ID

	// Run the lookup procedure.
	msgs, err := d.getChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, msg := range msgs {
		// TODO validate message
		rv = append(rv, msg.GetNodeId())
	}

	// Include locally stored data.
	localMsgs := d.channelStore.Get(id)
	for _, msg := range localMsgs {
		rv = append(rv, msg.GetNodeId())
	}

	return rv, nil
}

func (d *dht) getChannel(ctx context.Context, id []byte) ([]*message.StoreChannel, error) {
	log.Debugf("getChannel %x", id)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	result := make(chan []*message.StoreChannel)
	sendResult := func(r []*message.StoreChannel) {
		select {
		case result <- r:
			return
		case <-ctx.Done():
			return
		}
	}

	// Process incoming messages.
	go func() {
		c, cancel := d.net.Subscribe()
		defer cancel()

		var results []*message.StoreChannel

		// Used to timeout when no relevant new messages received for 2
		// seconds.
		var first bool = true
		t := time.NewTimer(time.Second)
		t.Stop()

		// Used to timeout when 5 seconds passed from first relevant
		// message.
		ctxTimeout := context.Background()

		for {

			select {
			case msg := <-c:
				switch pMsg := msg.Message.(type) {
				case *message.StoreChannel:
					if bytes.Equal(pMsg.GetChannelId(), id) {
						results = append(results, pMsg)
						t.Reset(2 * time.Second)
						if first {
							ctxTimeout, _ = context.WithTimeout(ctx, 5*time.Second)
							first = false
						}
					}
				}
			case <-t.C:
				sendResult(results)
				return
			case <-ctxTimeout.Done():
				sendResult(results)
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Run the lookup procedure.
	msgFactory := func(id node.ID) proto.Message {
		rv := &message.FindPubKey{
			Id: id,
		}
		return rv
	}
	go d.findNodeCustom(ctx, id, msgFactory, false)

	// Await results.
	select {
	case results := <-result:
		return results, nil
	case <-ctx.Done():
		return nil, errors.New("Key not found")
	}
}

// handlePutChannelMsg processes an incoming StoreChannel message.
func (d *dht) handleStoreChannelMsg(ctx context.Context, sender node.NodeInfo, msg *message.StoreChannel) error {
	err := d.validateStoreChannelMessage(ctx, msg)
	if err != nil {
		log.Debugf("INVALID channel %x info for %x: %s", msg.GetChannelId(), msg.GetNodeId(), err)
		return err
	}

	go d.disp.Dispatch(sender, msg)

	log.Debugf("Storing channel %x info for %x", msg.GetChannelId(), msg.GetNodeId())
	err = d.channelStore.Store(msg)
	if err != nil {
		return err
	}

	return nil
}

// handlePutChannelMsg processes an incoming StoreChannel message.
func (d *dht) handleFindChannelMsg(ctx context.Context, sender node.NodeInfo, msg *message.FindChannel) error {
	id := msg.GetChannelId()
	if !channel.ValidateId(id) {
		return errors.New("Invalid id")
	}

	go d.disp.Dispatch(sender, msg)

	peer, err := d.Dial(ctx, sender.Id)
	if err != nil {
		return err
	}

	// Send known channel members.
	msgs := d.channelStore.Get(id)
	for _, storeMsg := range msgs {
		peer.SendWithContext(d.ctx, storeMsg)
	}

	// Send closer nodes.
	response := d.createNodesMessage(id)
	peer.SendWithContext(d.ctx, response)
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
		Signature: []byte{},
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
	var tmp *message.StoreChannel = &message.StoreChannel{}
	*tmp = *msg
	tmp.Signature = []byte{}
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
