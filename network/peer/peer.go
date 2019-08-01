package peer

import (
	"context"
	"sync"

	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/network/stream"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type Peer interface {
	// Id returns the id of this peer.
	Id() node.ID

	// Returns the node's public key.
	PubKey() crypto.PublicKey

	// Sends a message to the node.
	Send(proto.Message) error

	// Sends a message to the node, returns an error if context is closed.
	SendWithContext(context.Context, proto.Message) error

	// Closed returns true if all streams have been closed.
	Closed() bool

	// AddStream adds a stream used for sending data to this peer.
	AddStream(stream stream.Stream) error
}

func New(s stream.Stream) Peer {
	rv := &peer{
		id:      s.Info().Id,
		pubKey:  s.PubKey(),
		streams: []stream.Stream{s},
	}
	return rv
}

type peer struct {
	id           node.ID
	pubKey       crypto.PublicKey
	streams      []stream.Stream
	streamsMutex sync.Mutex
}

func (p *peer) AddStream(s stream.Stream) error {
	p.streamsMutex.Lock()
	defer p.streamsMutex.Unlock()

	if !node.CompareId(s.Info().Id, p.id) {
		return errors.Errorf("expected stream with node id %s but got %s", p.id, s.Info().Id)
	}
	p.streams = append(p.streams, s)
	return nil
}

func (p *peer) Id() node.ID {
	return p.id
}

func (p *peer) PubKey() crypto.PublicKey {
	return p.pubKey
}

func (p *peer) Send(msg proto.Message) error {
	p.streamsMutex.Lock()
	defer p.streamsMutex.Unlock()

	for _, s := range p.streams {
		if !s.Closed() {
			return s.Send(msg)
		}
	}
	return errors.New("no open streams available")
}

func (p *peer) SendWithContext(ctx context.Context, msg proto.Message) error {
	p.streamsMutex.Lock()
	defer p.streamsMutex.Unlock()

	for _, s := range p.streams {
		if !s.Closed() {
			return s.SendWithContext(ctx, msg)
		}
	}
	return errors.New("no open streams available")
}

func (p *peer) Closed() bool {
	p.streamsMutex.Lock()
	defer p.streamsMutex.Unlock()

	for _, s := range p.streams {
		if !s.Closed() {
			return false
		}
	}
	return true
}
