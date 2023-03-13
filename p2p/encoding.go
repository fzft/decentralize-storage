package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GobDecoder struct{}

func (d GobDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (d DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	msg.Payload = buf[:n]
	return nil
}
