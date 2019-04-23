package irc

import (
	"errors"
	"github.com/boreq/starlight/irc/protocol"
	"golang.org/x/net/context"
	"net"
)

// NewUser should be used to create users. If the context closes the user will
// be disconnected.
func NewUser(ctx context.Context, conn net.Conn) *User {
	ctx, cancel := context.WithCancel(ctx)
	in, out := wrap(ctx, conn)
	rv := &User{
		in:     in,
		out:    out,
		ctx:    ctx,
		cancel: cancel,
	}
	// Close the connection after the context is closed.
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	return rv
}

// User represents a single IRC client that has connected to the server.
type User struct {
	Initialized bool // set to true by the server after the user sends the NICK message
	in          <-chan *protocol.Message
	out         chan<- *protocol.Message
	ctx         context.Context
	cancel      context.CancelFunc
}

// Send sends an IRC protocol message to the client.
func (u *User) Send(ctx context.Context, msg *protocol.Message) error {
	log.Print("Sending ", msg.Marshal())
	select {
	case u.out <- msg:
		return nil
	case <-u.ctx.Done():
		return u.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Receive receives an IRC protocol message from the client.
func (u *User) Receive(ctx context.Context) (*protocol.Message, error) {
	select {
	case msg, ok := <-u.in:
		if !ok {
			return nil, errors.New("channel closed")
		}
		return msg, nil
	case <-u.ctx.Done():
		return nil, u.ctx.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close disconnects the user.
func (u *User) Close() error {
	u.cancel()
	return nil
}

// wrap wraps a connection and returns channels which can be used to receive
// and send IRC protocol messages.
func wrap(ctx context.Context, conn net.Conn) (<-chan *protocol.Message, chan<- *protocol.Message) {
	in := make(chan *protocol.Message)
	out := make(chan *protocol.Message)

	// Read
	go func() {
		defer close(in)
		decoder := protocol.NewDecoder(conn)
		for {
			msg, err := decoder.Decode()
			if err != nil {
				return
			}
			in <- msg
		}
	}()

	// Write
	go func() {
		encoder := protocol.NewEncoder(conn)
		for {
			select {
			case msg := <-out:
				encoder.Encode(msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	return in, out
}
