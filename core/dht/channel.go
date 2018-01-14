package dht

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/boreq/starlight/core/channel"
	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"time"
)

func (d *dht) PutChannel(ctx context.Context, id []byte) error {
	// Prepare a message.
	msg, err := CreateStoreChannelMessage(d.self.PrivKey, d.self.Id, id)
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
			peer, err := d.netDial(nodeInfo)
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

	findNodeDone := make(chan int)

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
						log.Debugf("getChannel %x new result", id)
						results = append(results, pMsg)
						t.Reset(2 * time.Second)
						if first {
							ctxTimeout, _ = context.WithTimeout(ctx, 5*time.Second)
							first = false
						}
					}
				}
			case <-findNodeDone:
				log.Debugf("getChannel %x findNodeDone", id)
				t.Reset(2 * time.Second)
			case <-t.C:
				log.Debugf("getChannel %x sendResults - t", id)
				sendResult(results)
				return
			case <-ctxTimeout.Done():
				log.Debugf("getChannel %x sendResults - ctxTimeout", id)
				sendResult(results)
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Run the lookup procedure.
	go func() {
		msgFactory := func(id node.ID) proto.Message {
			rv := &message.FindChannel{
				ChannelId: id,
			}
			return rv
		}
		d.findNodeCustom(ctx, id, msgFactory, false)
		select {
		case findNodeDone <- 0:
		case <-ctx.Done():
		}
	}()

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
	if node.CompareId(msg.GetNodeId(), d.self.Id) {
		log.Debugf("Ignoring my own store channel message")
		return errors.New("Received my own message")
	}

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

// CreateStoreChannelMessage creates a StoreChannel message which can be sent
// to other nodes.
func CreateStoreChannelMessage(key crypto.PrivateKey, nodeId node.ID, channelId []byte) (*message.StoreChannel, error) {
	timestamp := time.Now().UTC().Unix()
	msg := &message.StoreChannel{
		ChannelId: channelId,
		NodeId:    nodeId,
		Timestamp: &timestamp,
	}
	msgBytes, err := storeChannelMessageDataToSign(msg)
	if err != nil {
		return nil, err
	}
	msg.Signature, err = key.Sign(msgBytes, SigningHash)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// If a message is further in the future than maxStoreChannelMessageFutureAge
// then it is rejected.
const maxStoreChannelMessageFutureAge = 30 * time.Second

// Stored channel memberships will be removed/rejected after this time passes
// since they have been signed.
const maxStoreChannelMessageAge = 5 * time.Minute

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
	if t.After(time.Now().UTC().Add(maxStoreChannelMessageFutureAge)) {
		return errors.New("Timestamp is too far in the future")
	}
	if t.Before(time.Now().UTC().Add(-maxStoreChannelMessageAge)) {
		return errors.New("Message is too old")
	}

	// Confirm the signature.
	msgBytes, err := storeChannelMessageDataToSign(msg)
	if err != nil {
		return err
	}
	key, err := d.GetPubKey(ctx, msg.GetNodeId())
	if err != nil {
		return err
	}
	err = key.Validate(msgBytes, msg.Signature, SigningHash)
	if err != nil {
		return err
	}

	return nil
}

// storeChannelMessageDataToSign produces an output which is used to create
// a signature for a StoreChannel message.
func storeChannelMessageDataToSign(msg *message.StoreChannel) ([]byte, error) {
	b := &bytes.Buffer{}
	b.Write(msg.GetChannelId())
	b.Write(msg.GetNodeId())
	if err := binary.Write(b, binary.BigEndian, msg.GetTimestamp()); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
