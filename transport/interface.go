// Package transport defines encoder and decoder interfaces used to send
// bariable length data chunks via a stream connection.
package transport

// Encoder sends data packets to the underlying writer.
type Encoder interface {
	Encode([]byte) error
}

// Decoder receives data packets from the underlying reader.
type Decoder interface {
	Decode() ([]byte, error)
}
