package core

import (
	"bytes"
	"crypto"
	"encoding/binary"
	"errors"
	"github.com/boreq/lainnet/core/channel"
	"github.com/boreq/lainnet/core/dht"
	lcrypto "github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol/message"
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

func (n *lainnet) SendChannelMessage(ctx context.Context, channelName string, text string) error {
	// Are we even in the channel?
	channelId := channel.CreateId(channelName)
	ch := n.getChannel(channelId)
	if ch == nil {
		return NotInChannelError
	}

	// Create the message.
	timestamp := time.Now().UTC().Unix()
	msg := &message.ChannelMessage{
		ChannelId: channelId,
		NodeId:    n.ident.Id,
		Timestamp: &timestamp,
		Text:      &text,
	}

	// Sign the message.
	data, err := channelMessageDataToSign(msg)
	if err != nil {
		return err
	}
	msg.Signature, err = n.ident.PrivKey.Sign(data, dht.SigningHash)
	if err != nil {
		return err
	}

	// Send the message to other channel members.
	for _, id := range ch.Users.Get(channelA) {
		peer, err := n.dht.Dial(n.ctx, id)
		if err == nil {
			peer.SendWithContext(ctx, msg)
		}
	}

	return nil
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

func (n *lainnet) handleFindChannelMsg(msg *message.FindChannel, sender node.ID) {
	n.channelsMutex.Lock()
	defer n.channelsMutex.Unlock()

	if ch := n.getChannel(msg.GetChannelId()); ch != nil {
		msg, err := dht.CreateStoreChannelMessage(n.ident.PrivKey, n.ident.Id, ch.Id)
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(n.ctx, 60*time.Second)
		defer cancel()

		peer, err := n.dht.Dial(ctx, sender)
		if err == nil {
			peer.SendWithContext(ctx, msg)
		}
	}
}

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

		// Insert into buckets.
		t := time.Now().UTC().Add(userTimeout)
		ch.Users.Insert(msg.GetNodeId(), t)
	}
}

func (n *lainnet) handleChannelMessageMsg(msg *message.ChannelMessage, sender node.ID) {
	log.Print(msg)

	// Abort if we are not in the channel.
	n.channelsMutex.Lock()
	ch := n.getChannel(msg.GetChannelId())
	n.channelsMutex.Unlock()
	if ch == nil {
		return
	}

	// Validate.
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	err := n.validateChannelMessage(ctx, msg)
	if err != nil {
		return
	}

	// Try to insert into the register, abort if failed.
	t := time.Unix(msg.GetTimestamp(), 0).Add(2 * maxChannelMessageAge)
	data, err := channelMessageDataToSign(msg)
	if err != nil {
		return
	}
	data = lcrypto.Digest(msgRegisterHash.New(), data)
	err = n.msgRegister.Insert(data, t)
	if err != nil {
		return
	}

	// Forward the message to other channel members.
	for _, id := range ch.Users.Get(channelA) {
		go func() {
			peer, err := n.dht.Dial(n.ctx, id)
			if err == nil {
				peer.Send(msg)
			}
		}()
	}

	// Insert the sender into buckets.
	t = time.Now().UTC().Add(userTimeout)
	ch.Users.Insert(sender, t)
}

// msgRegisterHash is a hashing functions applied to the output of
// channelMessageDataToSign in order to identify the messages stored in the
// msgregister.
const msgRegisterHash = crypto.SHA256

// If a channel message has a timestamp further in the future than it will be
// rejected.
var maxChannelMessageFutureAge = 30 * time.Second

// If a channel message has a timestamp older than maxChannelMessageFutureAge
// then it will be rejected.
var maxChannelMessageAge = 5 * time.Minute

// maxChannelMessageAge is the max total length of a message sent in a channel.
var maxChannelMessageLength = 500

func (n *lainnet) validateChannelMessage(ctx context.Context, msg *message.ChannelMessage) error {
	// IDs.
	if !channel.ValidateId(msg.GetChannelId()) {
		return errors.New("Invalid channel id")
	}

	if !node.ValidateId(msg.GetNodeId()) {
		return errors.New("Invalid node id")
	}

	// Timestamp.
	t := time.Unix(msg.GetTimestamp(), 0)

	if t.After(time.Now().UTC().Add(maxChannelMessageFutureAge)) {
		return errors.New("Timestamp is too far in the future")
	}

	if t.Before(time.Now().UTC().Add(-maxChannelMessageAge)) {
		return errors.New("Message is too old")
	}

	// Text.
	if len(msg.GetText()) > maxChannelMessageLength {
		return errors.New("Message is too long")
	}

	// Signature.
	data, err := channelMessageDataToSign(msg)
	if err != nil {
		return err
	}

	key, err := n.dht.GetPubKey(ctx, msg.GetNodeId())
	if err != nil {
		return err
	}

	return key.Validate(data, msg.GetSignature(), dht.SigningHash)
}

// channelMessageDataToSign produces an output which is used to create
// a signature for a ChannelMessage.
func channelMessageDataToSign(msg *message.ChannelMessage) ([]byte, error) {
	b := &bytes.Buffer{}
	b.Write(msg.GetChannelId())
	b.Write(msg.GetNodeId())
	if err := binary.Write(b, binary.BigEndian, msg.GetTimestamp()); err != nil {
		return nil, err
	}
	b.WriteString(msg.GetText())
	return b.Bytes(), nil
}
