package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/fzft/decentralized-storage/p2p"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type FileServerOpts struct {
	EncryptKey        []byte
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	store  *Store
	quitCh chan struct{}

	peerLock sync.RWMutex
	peers    map[net.Addr]p2p.Peer
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storageOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storageOpts),
		quitCh:         make(chan struct{}),
		peers:          make(map[net.Addr]p2p.Peer),
	}
}

type MessageGetFile struct {
	Key string
}

// Get returns the file with the given key.
func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Exists(key) {
		return s.store.Read(key)
	}

	fmt.Printf("file %s not found locally, asking peers\n", key)

	msg := Message{
		Payload: MessageGetFile{
			Key: hashKey(key),
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	for _, peer := range s.peers {
		// First read the file size which is encoded as a uint64
		var sizeSize int64
		binary.Read(peer, binary.LittleEndian, &sizeSize)
		n, err := s.store.writeDecryptStream(s.EncryptKey, key, io.LimitReader(peer, sizeSize))
		if err != nil {
			return nil, err
		}
		fmt.Println("received data", n)
		peer.CloseStream()
	}

	return s.store.Read(key)
}

// Store ...
func (s *FileServer) Store(key string, r io.Reader) error {

	var (
		n   int64
		err error
	)

	fileBuf := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuf)

	if n, err = s.store.Write(key, tee); err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  hashKey(key),
			Size: n,
		},
	}

	if err = s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.InboundedStreamType})
	nn, err := copyEncrypt(fileBuf, mw, s.EncryptKey)
	if err != nil {
		return err
	}
	fmt.Printf("sent %d bytes to peers", nn)

	return nil
}

// Start starts the file server.
func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.bootstrapNetwork()

	s.loop()
	return nil
}

// OnPeer ...
func (s *FileServer) OnPeer(peer p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[peer.RemoteAddr()] = peer
	log.Printf("peer connected: %s", peer.RemoteAddr())
	return nil
}

// Stop stops the file server.
func (s *FileServer) Stop() error {
	close(s.quitCh)
	return nil
}

// loop loops forever, waiting for RPCs.
func (s *FileServer) loop() {
	defer s.Transport.Close()
	for {
		select {
		case <-s.quitCh:
			return
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				continue
			}

			if err := s.handleMessage(rpc.From, msg); err != nil {
				log.Println("handle message error: ", err)
				continue
			}
		}
	}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

// handleMessage ...
func (s *FileServer) handleMessage(from net.Addr, msg Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	default:
		return fmt.Errorf("unknown message type: %v", v)
	}
	return nil
}

// handleMessageStoreFile ...
func (s *FileServer) handleMessageStoreFile(from net.Addr, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not found: %s", from)
	}
	n, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}
	fmt.Println("received store file message", msg.Key, n)
	peer.CloseStream()
	return nil
}

// stream ...
func (s *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

// broadcast ...
func (s *FileServer) broadcast(msg *Message) error {
	msgBuf := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.InboundedMessageType})
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// bootstrapNetwork ...
func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			if err := s.Transport.Dial(addr); err != nil {
				fmt.Println(err)
			}

		}(addr)
	}
	return nil
}

func (s *FileServer) handleMessageGetFile(from net.Addr, msg MessageGetFile) error {
	if !s.store.Exists(msg.Key) {
		return fmt.Errorf("need to server file: (%s), but it does not exist on disk", msg.Key)
	}
	r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	rc, ok := r.(io.ReadCloser)
	if ok {
		defer rc.Close()
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not found: %s", from)
	}

	peer.Send([]byte{p2p.InboundedStreamType})
	fileSize, err := s.store.FileSize(msg.Key)
	if err != nil {
		return err
	}
	binary.Write(peer, binary.LittleEndian, fileSize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Println("sent file", n)

	return nil
}

func init() {
	gob.Register(Message{})
	gob.Register(MessageGetFile{})
	gob.Register(MessageStoreFile{})
}
