package protocol

import (
	"fmt"
	"strings"
)

// Message represents a single message exchanged using the IRC protocol.
type Message struct {
	Prefix  string
	Command string
	Params  []string
}

// Marhshal encodes a message as specified by the the IRC protocol.
func (msg *Message) Marshal() string {
	rv := []string{}
	// Prefix
	if msg.Prefix != "" {
		rv = append(rv, fmt.Sprintf(":%s", msg.Prefix))
	}
	// Command
	rv = append(rv, msg.Command)
	// Params
	for _, param := range msg.Params {
		if strings.Contains(param, " ") {
			rv = append(rv, fmt.Sprintf(":%s", param))
		} else {
			rv = append(rv, param)
		}
	}
	return strings.Join(rv, " ")
}

// Marhshal decodes a message encoded as specified by the the IRC protocol.
func UnmarshalMessage(data string) *Message {
	rv := &Message{}

	for i, part := range strings.Split(data, " ") {
		// Prefix
		if i == 0 && strings.Index(part, ":") == 0 {
			rv.Prefix = part[1:]
			continue
		}
		// Command
		if rv.Command == "" {
			rv.Command = part
			continue
		}
		// Parameter
		rv.Params = append(rv.Params, part)
	}

	// If a parameter starts with a colon is considered to be the last
	// parameter and may contain spaces
	for i, param := range rv.Params {
		if strings.Index(param, ":") == 0 {
			// Create a list with params before the one containing a colon
			params := rv.Params[:i]
			// Join the remaining parameters, remove the first character
			params = append(params, strings.Join(rv.Params[i:], " ")[1:])
			rv.Params = params
			break
		}
	}

	return rv
}
