package irc

import (
	"fmt"
	"github.com/boreq/lainnet/irc/protocol"
	"github.com/boreq/lainnet/network/dispatcher"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol/message"
	"golang.org/x/net/context"
)

// receiveMessages subscribes to incoming messages and runs a loop which
// processes them until the context is closed.
func (s *Server) receiveMessages(ctx context.Context) {
	c, cancel := s.lainnet.Subscribe()
	defer cancel()
	for {
		select {
		case msg := <-c:
			go s.handleMessage(ctx, msg)
		case <-ctx.Done():
			return
		}
	}
}

// handleMessage is responsible for dispatching incoming messages to the
// appropriate handlers.
func (s *Server) handleMessage(ctx context.Context, msg dispatcher.IncomingMessage) {
	switch pMsg := msg.Message.(type) {

	case *message.PrivateMessage:
		s.handlePrivateMessage(ctx, msg.Sender.Id, pMsg)

	case *message.ChannelMessage:
		s.handleChannelMessage(ctx, msg.Sender.Id, pMsg)

	}
}

// handlePrivateMessage handles incoming private messages which are received
// directly from the other nodes.
func (s *Server) handlePrivateMessage(ctx context.Context, sender node.ID, msg *message.PrivateMessage) {
	s.usersMutex.Lock()
	defer s.usersMutex.Unlock()

	pMsg := &protocol.Message{
		Prefix:  fmt.Sprintf("%s!~%s@%s.wired", sender, "todo", sender),
		Command: "PRIVMSG",
		Params:  []string{s.nick, *msg.Text},
	}
	s.sendToAll(ctx, pMsg)
}

// handleChannelMessage handles incoming channel messages which are received
// by all nodes subscribed to a certain channel.
func (s *Server) handleChannelMessage(ctx context.Context, sender node.ID, msg *message.ChannelMessage) {
	s.usersMutex.Lock()
	defer s.usersMutex.Unlock()

	cMsg := &protocol.Message{
		Prefix:  fmt.Sprintf("%x!~%s@%x.wired", msg.GetNodeId(), "todo", msg.GetNodeId()),
		Command: "PRIVMSG",
		Params:  []string{string(msg.ChannelId), *msg.Text},
	}
	s.sendToAll(ctx, cMsg)
}
