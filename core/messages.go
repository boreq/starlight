package core

import (
	"bytes"
	"crypto"
	"encoding/binary"
	"errors"
	"github.com/boreq/starlight/core/channel"
	"github.com/boreq/starlight/core/dht"
	lcrypto "github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/boreq/starlight/utils"
	"golang.org/x/net/context"
	"math"
	"time"
)

// msgRegisterHash is a hashing functions applied to the output of
// channelMessageToBytes in order to identify the messages stored in the
// msgregister.
const msgRegisterHash = crypto.SHA256

func (n *core) handleChannelMessageMsg(msg *message.ChannelMessage, sender node.NodeInfo) {
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
		go func(id node.ID) {
			peer, err := n.dht.Dial(n.ctx, id)
			if err == nil {
				peer.Send(msg)
			}
		}(id)
	}

	// Insert the sender into buckets.
	t = time.Now().UTC().Add(userTimeout)
	ch.Users.Insert(sender.Id, t)
}

func (n *core) handlePrivateMessageMsg(msg *message.PrivateMessage, sender node.NodeInfo) {
	// Validate.
	err := n.validatePrivateMessage(msg, sender.Id)
	if err != nil {
		return
	}

	// Dispatch.
	n.disp.Dispatch(sender, msg)
}

func (n *core) SendChannelMessage(ctx context.Context, channelName string, text string) error {
	if len(text) > maxChannelMessageLength {
		return errors.New("message is too long")
	}

	// Are we even in the channel?
	channelId := channel.CreateId(channelName)
	ch := n.getChannel(channelId)
	if ch == nil {
		return ErrNotInChannel
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

func (n *core) SendMessage(ctx context.Context, id node.ID, text string) error {
	if len(text) > maxPrivateMessageLength {
		return errors.New("message is too long")
	}

	p, err := n.dht.Dial(ctx, id)
	if err != nil {
		return err
	}
	msg, err := n.createPrivateMessage(id, text)
	if err != nil {
		return err
	}
	return p.SendWithContext(ctx, msg)
}

// createChannelMessage creates a message containing text sent in the specified
// channel. This function solves the crypto puzzle and signs the message.
func (n *core) createChannelMessage(channelId []byte, text string) (*message.ChannelMessage, error) {
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

// createChannelMessage creates a private message. This function solves the
// crypto puzzle.
func (n *core) createPrivateMessage(target node.ID, text string) (*message.PrivateMessage, error) {
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

// If a channel message has a timestamp further in the future than it will be
// rejected.
const maxChannelMessageFutureAge = 30 * time.Second

// If a channel message has a timestamp older than maxChannelMessageFutureAge
// then it will be rejected.
const maxChannelMessageAge = 5 * time.Minute

// maxChannelMessageLength is the max length of a message sent in a channel.
const maxChannelMessageLength = 500

// maxPrivateMessageLength is the max length of a private message.
const maxPrivateMessageLength = 500

func (n *core) validateChannelMessage(ctx context.Context, msg *message.ChannelMessage) error {
	// IDs.
	if !channel.ValidateId(msg.GetChannelId()) {
		return errors.New("invalid channel id")
	}

	if !node.ValidateId(msg.GetNodeId()) {
		return errors.New("invalid node id")
	}

	// Timestamp.
	t := time.Unix(msg.GetTimestamp(), 0)

	if t.After(time.Now().UTC().Add(maxChannelMessageFutureAge)) {
		return errors.New("timestamp is too far in the future")
	}

	if t.Before(time.Now().UTC().Add(-maxChannelMessageAge)) {
		return errors.New("message is too old")
	}

	// Text.
	if len(msg.GetText()) > maxChannelMessageLength {
		return errors.New("message is too long")
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

func (n *core) validatePrivateMessage(msg *message.PrivateMessage, sender node.ID) error {
	// IDs.
	if !node.CompareId(n.ident.Id, msg.GetTargetId()) {
		return errors.New("invalid target id")
	}

	if !node.CompareId(sender, msg.GetNodeId()) {
		return errors.New("invalid node id")
	}

	// Text.
	if len(msg.GetText()) > maxPrivateMessageLength {
		return errors.New("message is too long")
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
	return errors.New("invalid puzzle")
}

// solveCryptoPuzzle attmpts to find a nonce which when concatenated with
// the data will create a hash with the right difficulty.
func solveCryptoPuzzle(nonce *uint64, data []byte, difficulty int) error {
	var solved bool
	hash := cryptoPuzzleHash.New()
	var sum []byte = make([]byte, 0, hash.Size())
	var bs []byte = make([]byte, 8)

	// Try to solve the puzzle.
	for *nonce = 0; *nonce < math.MaxUint64; *nonce++ {
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
		return errors.New("the crypto puzzle could not be solved for this message")
	}
	return nil
}
