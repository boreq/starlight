// Package irc implements an IRC server. That server supports only a certain
// subset of the functionality of the IRC protocol and allows the IRC clients
// to be used to communicate via the lainnet network.
package irc

import (
	"container/list"
	"fmt"
	"github.com/boreq/lainnet/core"
	"github.com/boreq/lainnet/irc/protocol"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/utils"
	"golang.org/x/net/context"
	"net"
	"strings"
	"sync"
	"time"
)

var log = utils.GetLogger("irc")

// NewServer creates a new IRC server which interfaces with the provided lainnet
// instance.
func NewServer(lainnet core.Lainnet) *Server {
	rv := &Server{
		lainnet: lainnet,
	}
	return rv
}

// Server interfaces Lainnet with the standard IRC clients by creating a local
// IRC server. Use the NewServer function to create this structure.
type Server struct {
	// Since the nick is the same for all connected clients as they are
	// represented by a single lainnet ID the common nick is stored in this
	// variable.
	nick       string
	users      list.List
	lainnet    core.Lainnet
	usersMutex sync.Mutex
}

// Start starts listening to connections from IRC clients on addr and receiving
// messages from the underlying Lainnet instance. The function will return
// if the server will be started correctly and will not block.
func (s *Server) Start(ctx context.Context, addr string) error {
	log.Printf("Listening on %s", addr)

	// Listen.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// Subscribe to messages incoming via lainnet.
	go s.receiveMessages(ctx)

	// Accept the connections to the IRC server.
	go s.acceptLoop(ctx, listener)

	return nil
}

// acceptLoop continously accepts all incoming connections from IRC clients.
func (s *Server) acceptLoop(ctx context.Context, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go s.accept(ctx, conn)
	}
}

// accept creates a new client and starts a listening loop on his connection.
func (s *Server) accept(ctx context.Context, conn net.Conn) {
	log.Printf("Accepting a new connection")
	user := NewUser(ctx, conn)
	s.users.PushBack(user)
	go s.runLoop(ctx, user)
}

// runLoop runs an infinite loop receiving messages from a client and handling
// them.
func (s *Server) runLoop(ctx context.Context, user *User) {
	for {
		msg, err := user.Receive(ctx)
		if err != nil {
			s.drop(user)
			return
		}
		log.Printf("Received: %s", msg.Marshal())

		switch msg.Command {
		case "QUIT":
			user.Close()
		case "MOTD":
			s.sendMotd(ctx, user)
		case "PING":
			pong := s.makeServerMessage("PONG", msg.Params)
			user.Send(ctx, pong)
		case "NICK":
			s.handlerNick(ctx, user, msg)
		case "PRIVMSG":
			s.handlerPrivmsg(ctx, user, msg)
		case "JOIN":
			s.handlerJoin(ctx, user, msg)
		}
	}
}

func (s *Server) handlerNick(ctx context.Context, user *User, msg *protocol.Message) {
	// Confirm that the required parameters are present.
	if len(msg.Params) < 1 {
		err := s.makeServerReply(protocol.ERR_NONICKNAMEGIVEN,
			[]string{"No nickname given"},
		)
		user.Send(ctx, err)
		return
	}

	s.usersMutex.Lock()
	defer s.usersMutex.Unlock()

	// Change the nick.
	prevNick := s.nick
	s.nick = msg.Params[0]

	// If the nick has changed inform all connected users.
	if s.nick != prevNick {
		for e := s.users.Front(); e != nil; e = e.Next() {
			u := e.Value.(*User)
			if u.Initialized {
				join := s.makeUserMessage(prevNick, "NICK", []string{s.nick})
				u.Send(ctx, join)
			}
		}
	}

	// If this is the first time this user sent a nick message send special
	// messages.
	if !user.Initialized {
		user.Initialized = true
		// Send the welcome messages.
		s.sendWelcome(ctx, user)
		s.sendMotd(ctx, user)
		// Inform the user about the joined channels.
		for _, chName := range s.lainnet.ListChannels() {
			join := s.makeUserMessage(s.nick, "JOIN", []string{chName})
			user.Send(ctx, join)
		}
	}
}

func (s *Server) handlerPrivmsg(ctx context.Context, user *User, msg *protocol.Message) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if protocol.IsChannelName(msg.Params[0]) {
		// Send a channel message.
		err := s.lainnet.SendChannelMessage(ctx, msg.Params[0], msg.Params[1])
		if err != nil {
			// Inform the client about an error.
			eMsg := s.makeServerMessage("NOTICE", []string{msg.Params[0], "Error: " + err.Error()})
			user.Send(ctx, eMsg)
		} else {
			// Inform other clients about the message.
			s.usersMutex.Lock()
			defer s.usersMutex.Unlock()
			cMsg := s.makeUserMessage(s.nick, msg.Command, msg.Params)
			s.sendToAllExcept(ctx, cMsg, user)
		}
	} else {
		// Send a private message to a user.
		target, err := node.NewId(msg.Params[0])
		if err != nil {
			// Inform the client about an error.
			eMsg := &protocol.Message{
				Prefix:  fmt.Sprintf("%s!~%s@%s.wired", msg.Params[0], "todo", msg.Params[0]),
				Command: "NOTICE",
				Params:  []string{s.nick, "Error: " + err.Error()},
			}
			user.Send(ctx, eMsg)
		} else {
			err := s.lainnet.SendMessage(ctx, target, msg.Params[1])
			if err != nil {
				// Inform the client about the error.
				eMsg := &protocol.Message{
					Prefix:  fmt.Sprintf("%s!~%s@%s.wired", msg.Params[0], "todo", msg.Params[0]),
					Command: "NOTICE",
					Params:  []string{s.nick, "Error: " + err.Error()},
				}
				user.Send(ctx, eMsg)
			}
		}
	}
}

func (s *Server) handlerJoin(ctx context.Context, user *User, msg *protocol.Message) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if len(msg.Params) < 1 {
		welcome := s.makeServerReply(protocol.ERR_NEEDMOREPARAMS,
			[]string{msg.Params[0], "Not enough parameters"},
		)
		user.Send(ctx, welcome)
		return
	}

	channelName := msg.Params[0]
	err := s.lainnet.JoinChannel(channelName)
	if err == nil {
		s.usersMutex.Lock()
		defer s.usersMutex.Unlock()
		join := s.makeUserMessage(s.nick, "JOIN", []string{channelName})
		s.sendToAll(ctx, join)
	}
}

// sendWelcome sends the RPL_WELCOME message to the specified user.
func (s *Server) sendWelcome(ctx context.Context, user *User) {
	welcome := s.makeServerReply(protocol.RPL_WELCOME,
		[]string{fmt.Sprintf("Welcome to the wired %s", s.nick)},
	)
	user.Send(ctx, welcome)
}

// sendMotd sends the message of the day to the specified user.
func (s *Server) sendMotd(ctx context.Context, user *User) {
	motd := `This IRC server is merely an interface to the Lainnet network and implements
only basic functionality of the IRC protocol. All clients connected to this
server share the same nickname, receive the same messages and are in the same
channels - they are all considered to be the same Lainnet user interfacing with
the underlying Lainnet node.

Supported functionality:
    1. JOIN
        The JOIN command can be used to subscribe to Lainnet channels.
    2. PART
        The PART command can be used to unsubscribe from Lainnet channels.
    3. PRIVMSG
        The PRIVMSG can be used to send messages directly to other Lainnet
        nodes as well as to send messages in the Lainnet channels which were
        previously joined using the JOIN command.`

	motdStart := s.makeServerReply(protocol.RPL_MOTDSTART,
		[]string{"- Message of the day -"},
	)
	user.Send(ctx, motdStart)

	for _, line := range strings.Split(motd, "\n") {
		motdMsg := s.makeServerReply(protocol.RPL_MOTD,
			[]string{fmt.Sprintf("- %s", line)},
		)
		user.Send(ctx, motdMsg)
	}

	motdEnd := s.makeServerReply(protocol.RPL_ENDOFMOTD,
		[]string{"End of MOTD command"},
	)
	user.Send(ctx, motdEnd)
}

// drop removes the user from the user list.
func (s *Server) drop(user *User) {
	for e := s.users.Front(); e != nil; e = e.Next() {
		u := e.Value.(*User)
		if u == user {
			log.Print("Droping user")
			s.users.Remove(e)
			return
		}
	}
}
