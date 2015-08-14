package encoder

import (
	"github.com/golang/protobuf/proto"
)

// Encoder provides a way of transitioning between  localy used protobuf structs
// and the values sent over the wire.
type Encoder interface {
	Encode(proto.Message) ([]byte, error)
	Decode([]byte) (proto.Message, error)
}
