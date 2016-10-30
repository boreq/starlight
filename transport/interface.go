// Package transport uses layers to encode the data for transport across the
// network. Multiple layers are added to a wrapper which uses them one by one
// to perform various tasks such as compression or encryption on the data.
package transport

import (
	"io"
)

type Layer interface {
	Encode(io.Reader, io.Writer) error
	Decode(io.Reader, io.Writer) error
}

type Wrapper interface {
	AddLayer(Layer)
	Send([]byte) error
	Receive() ([]byte, error)
}
