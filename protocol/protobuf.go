package protocol

import (
	"errors"
	"github.com/boreq/lainnet/protocol/message"
	"github.com/golang/protobuf/proto"
	"reflect"
)

var cmdMap = map[reflect.Type]uint32{
	reflect.TypeOf(message.Init{}):             1,
	reflect.TypeOf(message.Handshake{}):        2,
	reflect.TypeOf(message.ConfirmHandshake{}): 3,
	reflect.TypeOf(message.Identity{}):         4,
	reflect.TypeOf(message.Ping{}):             5,
	reflect.TypeOf(message.Pong{}):             6,
	reflect.TypeOf(message.FindNode{}):         7,
	reflect.TypeOf(message.Nodes{}):            8,
	reflect.TypeOf(message.PrivateMessage{}):   9,
	reflect.TypeOf(message.StorePubKey{}):      10,
}

// cmdEncode returns a value used in the protocol to indicate the type of a
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
