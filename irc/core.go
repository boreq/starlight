package irc

import (
	"github.com/boreq/starlight/irc/protocol"
	"github.com/boreq/starlight/network/dispatcher"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"golang.org/x/net/context"
)

// receiveMessages subscribes to incoming messages and runs a loop which
// processes them until the context is closed.
func (s *Server) receiveMessages(ctx context.Context) {
	c, cancel := s.core.Subscribe()
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

	prefix, err := s.getPrefix(sender)
	if err != nil {
		log.Debugf("error getting a prefix: %s", err)
		return
	}

	pMsg := &protocol.Message{
		Prefix:  prefix,
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

	prefix, err := s.getPrefix(msg.GetNodeId())
	if err != nil {
		log.Debugf("error getting a prefix: %s", err)
		return
	}

	cMsg := &protocol.Message{
		Prefix:  prefix,
		Command: "PRIVMSG",
		Params:  []string{string(msg.ChannelId), *msg.Text},
	}
	s.sendToAll(ctx, cMsg)
}
