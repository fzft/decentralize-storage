package main

import (
	"fmt"
	"github.com/fzft/decentralized-storage/p2p"
	"github.com/fzft/decentralized-storage/storage"
)

func main() {

	s1 := makeServer("3000", "")
	s2 := makeServer("3001", ":3000")
	go func() {
		s1.Start()
	}()

	s2.Start()

}

func makeServer(listenAddr string, nodes ...string) *FileServer {
	listenAddr = fmt.Sprintf(":%s", listenAddr)
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress: listenAddr,
		OnPeer:
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       fmt.Sprintf("%s_storage", listenAddr),
		PathTransformFunc: storage.CASPathTransformFunc,
		Transport:         tr,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOpts)
	return s
}
