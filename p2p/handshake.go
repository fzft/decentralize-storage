package p2p

// HandshakeFunc ...
type HandshakeFunc func(Peer) error

func NOPHandshake(Peer) error {
	return nil
}
