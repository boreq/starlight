// Package irc implements an IRC server. That server supports only a certain
// subset of the functionality of the IRC protocol and allows the IRC clients
// to be used to communicate via the new network.
package irc

import (
	"bufio"
	"container/list"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/boreq/starlight/core"
	"github.com/boreq/starlight/irc/humanizer"
	"github.com/boreq/starlight/irc/protocol"
	"github.com/boreq/starlight/utils"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/rakyll/statik/fs"

	_ "github.com/boreq/starlight/irc/humanizer/statik"
)

var log = utils.GetLogger("irc")

// NewServer creates a new IRC server which interfaces with the provided core
// instance.
func NewServer(ctx context.Context, core core.Core, nickServerAddress string) (*Server, error) {
	dictionary, err := loadDictionary()
	if err != nil {
		return nil, errors.Wrap(err, "could not load the dictionary")
	}

	humanizer, err := humanizer.New(ctx, core.Identity(), nickServerAddress, dictionary)
	if err != nil {
		return nil, errors.Wrap(err, "could not create the humanizer")
	}

	rv := &Server{
		core:      core,
		humanizer: humanizer,
	}
	return rv, nil
}

// loadDictionary loads the list of words used by the humanizer for encoding
// hashes.
func loadDictionary() ([]string, error) {
	statikFS, err := fs.New()
	if err != nil {
		return nil, errors.Wrap(err, "could not create statikFS")
	}

	f, err := statikFS.Open("/words.txt")
	if err != nil {
		return nil, errors.Wrap(err, "could not open a file")
	}

	var dictionary []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		dictionary = append(dictionary, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "scanner error")
	}

	log.Debugf("loaded %d dictionary words", len(dictionary))

	return dictionary, nil
}

// Server interfaces Core with the standard IRC clients by creating a local
// IRC server. Use the NewServer function to create this structure.
type Server struct {
	// Since the nick is the same for all connected clients as they are
	// represented by a single ID the common nick is stored in this
	// variable.
	nick       string
	humanizer  *humanizer.Humanizer
	users      list.List
	core       core.Core
	usersMutex sync.Mutex
}

// Start starts listening to connections from IRC clients on addr and receiving
// messages from the underlying Core instance. The function will return
// if the server will be started correctly and will not block.
func (s *Server) Start(ctx context.Context, addr string) error {
	log.Printf("Listening on %s", addr)

	// Listen.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "listen error")
	}

	// Subscribe to messages incoming via Core.
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
		case "PART":
			s.handlerPart(ctx, user, msg)
		case "WHOIS":
			s.handlerPart(ctx, user, msg)
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

	if err := s.humanizer.SetNick(s.nick); err != nil {
		user.Send(ctx, s.makeGlobalErrorMessage(err))
	}

	// If the nick has changed inform all connected users.
	if s.nick != prevNick {
		for e := s.users.Front(); e != nil; e = e.Next() {
			u := e.Value.(*User)
			if u.Initialized {
				join, err := s.makeUserMessage(prevNick, "NICK", []string{s.nick})
				if err != nil {
					log.Debugf("error making user message: %s", err)
					return
				}
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
		for _, chName := range s.core.ListChannels() {
			join, err := s.makeUserMessage(s.nick, "JOIN", []string{chName})
			if err != nil {
				log.Debugf("error making user message: %s", err)
				return
			}
			user.Send(ctx, join)
		}
	}
}

func (s *Server) handlerPrivmsg(ctx context.Context, user *User, msg *protocol.Message) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	target := msg.Params[0]
	text := msg.Params[1]

	if protocol.IsChannelName(msg.Params[0]) {
		if err := s.core.SendChannelMessage(ctx, target, text); err != nil {
			// Inform the client about an error.
			eMsg := s.makeServerMessage("NOTICE", []string{target, "Error: " + err.Error()})
			user.Send(ctx, eMsg)
		} else {
			// Inform other clients about the message.
			s.usersMutex.Lock()
			defer s.usersMutex.Unlock()
			cMsg, err := s.makeUserMessage(s.nick, msg.Command, msg.Params)
			if err != nil {
				log.Debugf("error making user message: %s", err)
				return
			}
			s.sendToAllExcept(ctx, cMsg, user)
		}
	} else {
		nodeId, err := s.humanizer.DehumanizeNick(target)
		if err != nil {
			// Inform the client about an error.
			eMsg := &protocol.Message{
				Prefix:  fmt.Sprintf("%s!~%s@%s.wired", msg.Params[0], "todo", msg.Params[0]),
				Command: "NOTICE",
				Params:  []string{s.nick, "Error: " + err.Error()},
			}
			user.Send(ctx, eMsg)
		} else {
			err := s.core.SendMessage(ctx, nodeId, text)
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
	if err := s.core.JoinChannel(channelName); err != nil {
		if err != core.ErrAlreadyInChannel {
			user.Send(ctx, s.makeGlobalErrorMessage(err))
		}
	} else {
		s.usersMutex.Lock()
		defer s.usersMutex.Unlock()
		join, err := s.makeUserMessage(s.nick, "JOIN", []string{channelName})
		if err != nil {
			log.Debugf("error making user message: %s", err)
			return
		}
		s.sendToAll(ctx, join)
	}
}

func (s *Server) handlerPart(ctx context.Context, user *User, msg *protocol.Message) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if len(msg.Params) < 1 {
		err := s.makeServerReply(protocol.ERR_NEEDMOREPARAMS,
			[]string{msg.Params[0], "Not enough parameters"},
		)
		user.Send(ctx, err)
		return
	}

	channelName := msg.Params[0]
	if err := s.core.PartChannel(channelName); err != nil {
		if err == core.ErrNotInChannel {
			reply := s.makeServerReply(protocol.ERR_NOTONCHANNEL,
				[]string{msg.Params[0], "You're not on that channel"},
			)
			user.Send(ctx, reply)
		} else {
			user.Send(ctx, s.makeGlobalErrorMessage(err))
		}
	} else {
		s.usersMutex.Lock()
		defer s.usersMutex.Unlock()
		join, err := s.makeUserMessage(s.nick, "PART", []string{channelName})
		if err != nil {
			log.Debugf("error making user message: %s", err)
			return
		}
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
	motd := `This IRC server is merely an interface to the Starlight network and implements
only basic functionality of the IRC protocol. All clients connected to this
server share the same nickname, receive the same messages and are in the same
channels - they are all considered to be the same Starlight user interfacing with
the underlying Starlight node.

Supported functionality:
    1. JOIN
        The JOIN command can be used to subscribe to Starlight channels.
    2. PART
        The PART command can be used to unsubscribe from Starlight channels.
    3. PRIVMSG
        The PRIVMSG can be used to send messages directly to other Starlight
        nodes as well as to send messages in the Starlight channels which were
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
