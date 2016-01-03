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
	"github.com/boreq/lainnet/utils"
	"golang.org/x/net/context"
	"math"
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

func (n *lainnet) SendChannelMessage(ctx context.Context, channelName string, text string) error {
	// Are we even in the channel?
	channelId := channel.CreateId(channelName)
	ch := n.getChannel(channelId)
	if ch == nil {
		return NotInChannelError
	}

	// Create the message.
	msg, err := n.createChannelMessage(channelId, text)
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

func (n *lainnet) createChannelMessage(channelId []byte, text string) (*message.ChannelMessage, error) {
	// Create the message.
	var nonce uint64
	timestamp := time.Now().UTC().Unix()
	msg := &message.ChannelMessage{
		ChannelId: channelId,
		NodeId:    n.ident.Id,
		Timestamp: &timestamp,
		Text:      &text,
		Nonce:     &nonce,
	}

	// Solve the puzzle.
	data, err := channelMessageToBytes(msg)
	if err != nil {
		return nil, err
	}
	err = solveCryptoPuzzle(&nonce, data, channelMessageCryptoPuzzleDifficulty)
	if err != nil {
		return nil, err
	}

	// Sign the message.
	msg.Signature, err = n.ident.PrivKey.Sign(data, dht.SigningHash)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (n *lainnet) createPrivateMessage(target node.ID, text string) (*message.PrivateMessage, error) {
	// Create the message.
	var nonce uint64
	msg := &message.PrivateMessage{
		TargetId: target,
		NodeId:   n.ident.Id,
		Text:     &text,
		Nonce:    &nonce,
	}

	// Solve the puzzle.
	data, err := privateMessageToBytes(msg)
	if err != nil {
		return nil, err
	}
	err = solveCryptoPuzzle(&nonce, data, privateMessageCryptoPuzzleDifficulty)
	if err != nil {
		return nil, err
	}

	return msg, nil
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

func (n *lainnet) handleChannelMessageMsg(msg *message.ChannelMessage, sender node.NodeInfo) {
	log.Print(msg)

	// Abort if we are not in the channel.
	n.channelsMutex.Lock()
	ch := n.getChannel(msg.GetChannelId())
	n.channelsMutex.Unlock()
	if ch == nil {
		return
	}

	// Ignore my own channel messages.
	if node.CompareId(msg.GetNodeId(), n.ident.Id) {
		return
	}

	// Validate.
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	err := n.validateChannelMessage(ctx, msg)
	if err != nil {
		return
	}

	// Try to insert into the register, abort if failed - it has been
	// received already.
	t := time.Unix(msg.GetTimestamp(), 0).Add(2 * maxChannelMessageAge)
	data, err := channelMessageToBytes(msg)
	if err != nil {
		return
	}
	data = lcrypto.Digest(msgRegisterHash.New(), data)
	err = n.msgRegister.Insert(data, t)
	if err != nil {
		return
	}

	// Dispatch.
	dMsg := &message.ChannelMessage{
		ChannelId: []byte(ch.Name),
		NodeId:    msg.NodeId,
		Timestamp: msg.Timestamp,
		Text:      msg.Text,
		Signature: msg.Signature,
	}
	n.disp.Dispatch(sender, dMsg)

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
	ch.Users.Insert(sender.Id, t)
}

func (n *lainnet) handlePrivateMessageMsg(msg *message.PrivateMessage, sender node.NodeInfo) {
	// Validate.
	err := n.validatePrivateMessage(msg, sender.Id)
	if err != nil {
		return
	}

	// Dispatch.
	n.disp.Dispatch(sender, msg)
}

// msgRegisterHash is a hashing functions applied to the output of
// channelMessageToBytes in order to identify the messages stored in the
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

	// Nonce and signature.
	data, err := channelMessageToBytes(msg)
	if err != nil {
		return err
	}

	// Nonce.
	err = validateCryptoPuzzle(msg.GetNonce(), data, channelMessageCryptoPuzzleDifficulty)
	if err != nil {
		return err
	}

	// Signature.
	key, err := n.dht.GetPubKey(ctx, msg.GetNodeId())
	if err != nil {
		return err
	}

	return key.Validate(data, msg.GetSignature(), dht.SigningHash)
}

func (n *lainnet) validatePrivateMessage(msg *message.PrivateMessage, sender node.ID) error {
	// IDs.
	if !node.CompareId(n.ident.Id, msg.GetTargetId()) {
		return errors.New("Invalid target id")
	}

	if !node.CompareId(sender, msg.GetNodeId()) {
		return errors.New("Invalid node id")
	}

	// Nonce.
	data, err := privateMessageToBytes(msg)
	if err != nil {
		return err
	}

	err = validateCryptoPuzzle(msg.GetNonce(), data, privateMessageCryptoPuzzleDifficulty)
	if err != nil {
		return err
	}

	return nil
}

// channelMessageToBytes produces an output which is used to create
// a signature for a ChannelMessage. This output is also used to calculate
// a crypto puzzle hash after appending a nonce to it.
func channelMessageToBytes(msg *message.ChannelMessage) ([]byte, error) {
	b := &bytes.Buffer{}
	b.Write(msg.GetChannelId())
	b.Write(msg.GetNodeId())
	if err := binary.Write(b, binary.BigEndian, msg.GetTimestamp()); err != nil {
		return nil, err
	}
	b.WriteString(msg.GetText())
	return b.Bytes(), nil
}

// privateMessageToBytes produces an output which is used to create
// a signature for a ChannelMessage. This output is also used to calculate
// a crypto puzzle hash after appending a nonce to it.
func privateMessageToBytes(msg *message.PrivateMessage) ([]byte, error) {
	b := &bytes.Buffer{}
	b.Write(msg.GetTargetId())
	b.Write(msg.GetNodeId())
	b.WriteString(msg.GetText())
	return b.Bytes(), nil
}

const channelMessageCryptoPuzzleDifficulty = 5
const privateMessageCryptoPuzzleDifficulty = 5

// cryptoPuzzleHash is a hashing functions applied to the output of
// channelMessageToBytes when solving the crypto puzzle for channel messages
// and private messages.
const cryptoPuzzleHash = crypto.SHA256

func validateCryptoPuzzle(nonce uint64, data []byte, difficulty int) error {
	hash := cryptoPuzzleHash.New()
	var sum []byte = make([]byte, 0, hash.Size())
	var bs []byte = make([]byte, 8)
	hash.Write(data)
	binary.BigEndian.PutUint64(bs, nonce)
	hash.Write(bs)
	sum = hash.Sum(sum)
	numBits := utils.ZerosLen(sum)
	if numBits >= difficulty {
		return nil
	}
	return errors.New("Invalid puzzle")
}

func solveCryptoPuzzle(nonce *uint64, data []byte, difficulty int) error {
	var solved bool
	hash := cryptoPuzzleHash.New()
	var sum []byte = make([]byte, 0, hash.Size())
	var bs []byte = make([]byte, 8)

	// Try to solve the puzzle.
	for *nonce = 0; *nonce <= math.MaxUint64; *nonce++ {
		hash.Reset()
		hash.Write(data)
		binary.BigEndian.PutUint64(bs, *nonce)
		hash.Write(bs)
		sum = hash.Sum(sum[:0])
		numBits := utils.ZerosLen(sum)
		if numBits >= difficulty {
			solved = true
			break
		}
	}

	// Make sure that it was solved correctly and the loop didn't simply end.
	if !solved {
		return errors.New("The crypto puzzle could not be solved for this message")
	}
	return nil
}
