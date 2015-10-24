// Package protocol implements IRC protocol message serialization and
// deserialization.
package protocol

// Encoder wraps an io.Writer and can be used to write messages to it.
type Encoder interface {
	// Encode encodes a message and writes it to the underlying writer.
	Encode(*Message) error
}

// Decoder wraps an io.Reader and can be used to receive messages from it.
type Decoder interface {
	// Decode receives a single message from the underlying reader and
	// decodes it into a message struct.
	Decode() (*Message, error)
}
