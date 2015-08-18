package encoder

import (
	"errors"
	"github.com/boreq/netblog/protocol/message"
	"github.com/golang/protobuf/proto"
	"reflect"
)

var cmdMap = map[reflect.Type]uint32{
	reflect.TypeOf(message.Init{}):             1,
	reflect.TypeOf(message.Handshake{}):        2,
	reflect.TypeOf(message.ConfirmHandshake{}): 3,
	reflect.TypeOf(message.Identity{}):         4,
}

// CmdEncode returns a value used in the protocol to indicate the type of a
// message.
func cmdEncode(msg proto.Message) (uint32, error) {
	typ := reflect.TypeOf(msg)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	rw, ok := cmdMap[typ]
	if !ok {
		return 0, errors.New("Unknown message type")
	}
	return rw, nil
}

// CmdDecode returns a type which matches the value used in the protocol.
func cmdDecode(cmd uint32) (reflect.Type, error) {
	for typ, value := range cmdMap {
		if value == cmd {
			return typ, nil
		}
	}
	return nil, errors.New("Unknown message type")
}
