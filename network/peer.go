package network

import (
	//	"bytes"
	"errors"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/protocol"
	"golang.org/x/net/context"
	"io"
	"log"
	"net"
)

// Use this instead of creating peer structs directly. Initiates communication
// channels and context.
func newPeer(ctx context.Context, conn net.Conn) *peer {
	ctx, cancel := context.WithCancel(ctx)
	in, out := handleConnection(ctx, conn)
	p := &peer{
		In:     in,
		Out:    out,
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}
	return p
}

type peer struct {
	Id     node.ID
	In     <-chan protocol.Message
	Out    chan<- protocol.Message
	ctx    context.Context
	cancel context.CancelFunc
	conn   net.Conn
}

func (p *peer) Info() node.NodeInfo {
	return node.NodeInfo{
		Id:      p.Id,
		Address: p.conn.RemoteAddr().String(),
	}
}

func (p *peer) Close() {
	p.cancel()
	p.conn.Close()
}

func (p *peer) Send(msg protocol.Message) error {
	select {
	case p.Out <- msg:
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
	buf := make([]byte, 0)
	defer close(in)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := reader.Read(buf)
			log.Printf("Received %d bytes", n)
			if err != nil {
				return
			}
			unmarshaler.Write(buf)
		}
	}
}

// Reads messages from a channel and sends them to a peer.
func sendToPeer(ctx context.Context, out <-chan protocol.Message, writer io.Writer) {
	// TODO
	//buf := bytes.Buffer{}
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			log.Print(msg)
			// TODO: send the data instead of trashing it.
		}
	}
}

// Returns two channels, first allows to receive messages from a peer and the
// second allows to send messages to the peer.
func handleConnection(ctx context.Context, conn net.Conn) (<-chan protocol.Message, chan<- protocol.Message) {
	in := make(chan protocol.Message)
	out := make(chan protocol.Message)
	go receiveFromPeer(ctx, in, conn)
	go sendToPeer(ctx, out, conn)
	return in, out
}
