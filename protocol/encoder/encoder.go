package encoder

import (
	"github.com/golang/protobuf/proto"
)

// Encoder provides a way of transitioning between locally used protobuf structs
// and the raw messages sent over the wire.
type Encoder interface {
	Encode(proto.Message) ([]byte, error)
	Decode([]byte) (proto.Message, error)
}
