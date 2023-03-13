package p2p

import "net"

// Peer is the interface that wraps the basic methods of a peer.
type Peer interface {
	Close() error
	RemoteAddr() net.Addr
	Send([]byte) error
}

// Transport is the interface that wraps the basic methods of a transport.
// between the nodes in network
// form (e.g. TCP, UDP, etc.).
type Transport interface {
	// ListenAndAccept starts listening for incoming connections.
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	Dial(addr string) error
}
