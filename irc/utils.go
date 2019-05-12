package irc

import (
	"fmt"

	"github.com/boreq/starlight/irc/protocol"
	"golang.org/x/net/context"
)

// makeServerReply creates a message originating from the server containing
// a numeric protocol reply.
func (s *Server) makeServerReply(code protocol.Numeric, params []string) *protocol.Message {
	params = append([]string{s.nick}, params...)
	return s.makeServerMessage(fmt.Sprintf("%03d", code), params)
}

// makeServerMessage creates a message originating from the server.
func (s *Server) makeServerMessage(command string, params []string) *protocol.Message {
	return &protocol.Message{
		Prefix:  "starlight",
		Command: command,
		Params:  params,
	}
}

// makeErrorMessage creates a message used to send information about various
// errors not specified by the protocol to the user.
func (s *Server) makeErrorMessage(err error) *protocol.Message {
	return &protocol.Message{
		Prefix:  "starlight",
		Command: "NOTICE",
		Params:  []string{s.nick, fmt.Sprintf("Error: %s", err)},
	}
}

// makeUserMessage creates a command message originating from the local user.
func (s *Server) makeUserMessage(nick string, command string, params []string) *protocol.Message {
	return &protocol.Message{
		Prefix:  fmt.Sprintf("%s@%s.wired", nick, s.core.Identity().Id),
		Command: command,
		Params:  params,
	}
}

// sendToAll sends a message to all connected clients.
func (s *Server) sendToAll(ctx context.Context, msg *protocol.Message) {
	for e := s.users.Front(); e != nil; e = e.Next() {
		user := e.Value.(*User)
		user.Send(ctx, msg)
	}
}

// sendToAllExcept sends a message to all connected clients except a single
// client.
func (s *Server) sendToAllExcept(ctx context.Context, msg *protocol.Message, u *User) {
	for e := s.users.Front(); e != nil; e = e.Next() {
		user := e.Value.(*User)
		if user != u {
			user.Send(ctx, msg)
		}
	}
}
