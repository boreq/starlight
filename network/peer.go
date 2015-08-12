package network

import (
	"bytes"
	"errors"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/protocol"
	"github.com/boreq/netblog/protocol/message"
	"golang.org/x/net/context"
	"io"
	"net"
)

// Use this instead of creating peer structs directly. Initiates communication
// channels and context.
func newPeer(ctx context.Context, iden node.Identity, conn net.Conn) (*peer, error) {
	ctx, cancel := context.WithCancel(ctx)
	p := &peer{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}

	p.initChannels()
	err := handshake(iden, p)
	if err != nil {
		log.Print("Hanshake error")
		p.Close()
		return nil, err
	}

	return p, nil
}

type peer struct {
	Id     node.ID
	ctx    context.Context
	in     chan protocol.Message
	out    chan protocol.Message
	cancel context.CancelFunc
	conn   net.Conn
}

func (p *peer) initChannels() {
	// In
	interIn := make(chan protocol.Message)
	in := make(chan protocol.Message)
	go receiveFromPeer(p.ctx, interIn, p.conn)
	go func() {
		for {
			select {
			case msg, ok := <-interIn:
				if !ok {
					p.Close()
					return
				}
				select {
				case in <- msg:

				case <-p.ctx.Done():
					return
				}
			case <-p.ctx.Done():
				return
			}
		}
	}()
	p.in = in

	// Out
	out := make(chan protocol.Message)
	go sendToPeer(p.ctx, out, p.conn)
	p.out = out
}

func (p *peer) Info() node.NodeInfo {
	return node.NodeInfo{
		Id:      p.Id,
		Address: p.conn.RemoteAddr().String(),
	}
}

func (p *peer) In() <-chan protocol.Message {
	return p.in
}

func (p *peer) Close() {
	log.Printf("peer close %x", p.Id)
	p.cancel()
	close(p.in)
	p.conn.Close()
}

func (p *peer) Send(msg protocol.Message) error {
	select {
	case p.out <- msg:
		return nil
	case <-p.ctx.Done():
		return errors.New("Context closed, can not send")
	}
	return nil
}

// Receives messages from a peer and sends them into the channel. In case of
// error the channel is closed.
func receiveFromPeer(ctx context.Context, in chan<- protocol.Message, reader io.Reader) {
	unmarshaler := protocol.NewUnmarshaler(in)
	buf := make([]byte, 1024)
	defer close(in)
	for {
		select {
		case <-ctx.Done():
			log.Print("receiveFromPeer context done")
			return
		default:
			n, err := reader.Read(buf)
			log.Printf("receiveFromPeer %d bytes", n)
			if err != nil {
				log.Printf("receiveFromPeer error %s", err)
				return
			}
			unmarshaler.Write(buf[:n])
		}
	}
}

// Reads messages from a channel and sends them to a peer.
func sendToPeer(ctx context.Context, out <-chan protocol.Message, writer io.Writer) {
	buf := bytes.Buffer{}
	for {
		select {
		case <-ctx.Done():
			log.Print("sendToPeer context done")
			return
		case msg := <-out:
			log.Print("sendToPeer sending message")
			buf.Reset()
			data, err := protocol.Marshal(msg)
			if err != nil {
				log.Print("sendToPeer error", err)
				continue
			}
			buf.Write(data)
			buf.WriteTo(writer)
			log.Print("sendToPeer sent message")
		}
	}
}

func handshake(iden node.Identity, p *peer) error {
	log.Print("handshake")

	pubKeyBytes, err := iden.PubKey.Bytes()
	if err != nil {
		return err
	}

	init := &message.Init{
		PubKey: pubKeyBytes,
	}
	msg, err := protocol.NewMessage(protocol.Init, init)
	if err != nil {
		return err
	}

	p.Send(*msg)
	return nil
}
