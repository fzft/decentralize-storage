package main

import (
	"bytes"
	"fmt"
	"github.com/fzft/decentralized-storage/p2p"
	"io/ioutil"
	"time"
)

func main() {

	s1 := makeServer("3000", "")
	s2 := makeServer("3001", ":3000")
	go func() {
		s1.Start()
	}()

	go s2.Start()
	time.Sleep(1 * time.Second)

	for i := 0; i < 10; i++ {
		payload := fmt.Sprintf("hello world %d", i)
		key := fmt.Sprintf("hello%d", i)
		data := bytes.NewReader([]byte(payload))
		err := s2.Store(key, data)
		if err != nil {
			panic(err)
		}
		time.Sleep(2 * time.Second)

		r, _ := s2.Get(key)
		b, err := ioutil.ReadAll(r)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
	}

	select {}
}

func makeServer(listenAddr string, nodes ...string) *FileServer {
	listenAddr = fmt.Sprintf(":%s", listenAddr)
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress: listenAddr,
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	fileServerOpts := FileServerOpts{
		EncryptKey:        newEncryptionKey(),
		StorageRoot:       fmt.Sprintf("%s_storage", listenAddr),
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tr,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOpts)

	tr.OnPeer = s.OnPeer

	return s
}
