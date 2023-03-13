package main

import (
	"fmt"
	"github.com/fzft/decentralized-storage/p2p"
	"github.com/fzft/decentralized-storage/storage"
	"log"
	"net"
	"sync"
)

type FileServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc storage.PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	store  *storage.Store
	quitCh chan struct{}

	peerLock sync.RWMutex
	peers    map[net.Addr]p2p.Peer
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storageOpts := storage.StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          storage.NewStore(storageOpts),
		quitCh:         make(chan struct{}),
		peers:          make(map[net.Addr]p2p.Peer),
	}
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

// loop loops forever, waiting for RPCs.
func (s *FileServer) loop() {
	defer s.Transport.Close()
	for {
		select {
		case <-s.quitCh:
			return
		case rpc := <-s.Transport.Consume():
			fmt.Println(rpc)
		}
	}
}

// Stop stops the file server.
func (s *FileServer) Stop() error {
	close(s.quitCh)
	return nil
}
