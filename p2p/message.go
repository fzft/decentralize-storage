package p2p

import (
	"net"
)

const (
	InboundedMessageType = 0x01
	InboundedStreamType  = 0x02
)

type RPC struct {
	Payload []byte
	From    net.Addr
	Stream  bool
}
