package irc

import (
	"fmt"

	"github.com/boreq/starlight/irc/protocol"
	"github.com/boreq/starlight/network/node"
	"github.com/pkg/errors"
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

// makeGlobalErrorMessage creates a NOTICE sent to a user globally.
func (s *Server) makeGlobalErrorMessage(err error) *protocol.Message {
	return &protocol.Message{
		Prefix:  "starlight",
		Command: "NOTICE",
		Params:  []string{s.nick, fmt.Sprintf("Error: %s", err)},
	}
}

// makeUserMessage creates a command message originating from the local user.
func (s *Server) makeUserMessage(nick string, command string, params []string) (*protocol.Message, error) {
	prefix, err := s.getPrefix(s.core.Identity().Id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get the prefix")
	}

	return &protocol.Message{
		Prefix:  prefix,
		Command: command,
		Params:  params,
	}, nil
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

func (s *Server) getPrefix(id node.ID) (string, error) {
	host, err := s.humanizer.HumanizeHost(id)
	if err != nil {
		return "", err
	}

	nick, err := s.humanizer.HumanizeNick(id)
	if err != nil {
		return "", err
	}

	return createPrefix(nick, "user", host), nil
}

func createPrefix(nick, user, host string) string {
	return fmt.Sprintf("%s!~%s@%s", nick, user, host)
}
