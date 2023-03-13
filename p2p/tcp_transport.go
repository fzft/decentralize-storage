package p2p

import (
	"fmt"
	"io"
	"net"
)

type TCPPeer struct {
	conn net.Conn

	// if we dial a peer, we are the outbound peer
	// if we accept a peer, we are the inbound peer
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

// Close closes the peer.
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

// RemoteAddr returns the remote network address.
func (p *TCPPeer) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}

// Send sends a payload to the peer.
func (p *TCPPeer) Send(payload []byte) error {
	_, err := p.conn.Write(payload)
	return err
}

type TCPTransportOpts struct {
	ListenAddress string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	Listener net.Listener
	rpcCh    chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {

	if opts.HandshakeFunc == nil {
		opts.HandshakeFunc = NOPHandshake
	}

	if opts.Decoder == nil {
		opts.Decoder = DefaultDecoder{}
	}

	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcCh:            make(chan RPC),
	}
}

// ListenAndAccept starts listening for incoming connections.
func (t *TCPTransport) ListenAndAccept() error {
	listener, err := net.Listen("tcp", t.ListenAddress)
	if err != nil {
		return err
	}
	t.Listener = listener

	go t.acceptLoop()
	return nil
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcCh
}

// acceptLoop accepts incoming connections and adds them to the transport.
func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.Listener.Accept()
		if err != nil {
			return
		}
		go t.handleConn(conn, false)
	}
}

// handleConn handles an incoming connection.
func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	peer := NewTCPPeer(conn, outbound)

	defer func() {
		if err := peer.Close(); err != nil {
			fmt.Println("handleConn", err)
		}
	}()

	if t.OnPeer != nil {
		if err := t.OnPeer(peer); err != nil {
			return
		}
	}

	if err := t.HandshakeFunc(peer); err != nil {
		return
	}

	rpc := RPC{}
	for {
		err := t.Decoder.Decode(conn, &rpc)

		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			fmt.Println("handleConn", err)
			return
		}

		rpc.From = conn.RemoteAddr()
		t.rpcCh <- rpc
	}
}

// Close closes the transport.
func (t *TCPTransport) Close() error {
	return t.Listener.Close()
}

// Dial ...
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)
	return nil
}
