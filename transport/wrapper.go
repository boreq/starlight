package transport

import (
	"bytes"
	"errors"
	"io"
)

// NewWrapper creates a new wrapper which receives data from the reader and
// sends data to the writer.
func NewWrapper(reader io.Reader, writer io.Writer) Wrapper {
	rv := &wrapper{
		reader: reader,
		writer: writer,
	}
	return rv
}

type wrapper struct {
	layers []Layer
	reader io.Reader
	writer io.Writer
}

func (w *wrapper) AddLayer(layer Layer) {
	w.layers = append(w.layers, layer)
}

func (w *wrapper) Send(data []byte) error {
	if len(w.layers) == 0 {
		return errors.New("No layers")
	}

	var in io.Reader = bytes.NewBuffer(data)
	for i := len(w.layers) - 1; i >= 0; i-- {
		layer := w.layers[i]
		newIn, out := io.Pipe()
		go encodeWithLayer(layer, in, out)
		in = newIn
	}
	_, err := io.Copy(w.writer, in)
	return err
}

func (w *wrapper) Receive() ([]byte, error) {
	if len(w.layers) == 0 {
		return nil, errors.New("No layers")
	}

	var in io.Reader = w.reader
	for i := 0; i < len(w.layers); i++ {
		layer := w.layers[i]
		newIn, out := io.Pipe()
		go decodeWithLayer(layer, in, out)
		in = newIn
	}
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, in)
	return buf.Bytes(), err
}

func encodeWithLayer(layer Layer, r io.Reader, w *io.PipeWriter) {
	if err := layer.Encode(r, w); err != nil {
		w.CloseWithError(err)
	} else {
		w.Close()
	}
}

func decodeWithLayer(layer Layer, r io.Reader, w *io.PipeWriter) {
	if err := layer.Decode(r, w); err != nil {
		w.CloseWithError(err)
	} else {
		w.Close()
	}
}
