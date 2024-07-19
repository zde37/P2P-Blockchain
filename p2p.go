package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type UserMessage struct {
	Location   string `json:"location"`
	WaveHeight int    `json:"wave_height"`
}

var (
	mut     sync.Mutex
	peerList   []network.Stream
	peerListMu sync.RWMutex
)

func handleStream(s network.Stream) {
	log.Println("New stream detected")

	peerListMu.Lock()
	peerList = append(peerList, s)
	peerListMu.Unlock()

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
	go writeData(rw)
}

func broadcastData(data []byte) {
	peerListMu.RLock()
	defer peerListMu.RUnlock()

	for _, peer := range peerList {
		rw := bufio.NewReadWriter(bufio.NewReader(peer), bufio.NewWriter(peer))
		rw.WriteString(fmt.Sprintf("%s\n", string(data)))
		rw.Flush()
	}
}

// makeHost creates a new libp2p host with the given port.
func makeHost(port int, randomness io.Reader) (host.Host, error) {
	privateKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, randomness)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to create multiaddr: %w", err)
	}

	return libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(privateKey),
	)
}

// readData reads data from the stream.
func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Error while reading data: %v", err)
			}
			return
		}

		if str == "" || str == "\n" {
			continue
		}

		var chain Blockchain
		if err := json.Unmarshal([]byte(str), &chain); err != nil {
			log.Printf("Failed to unmarshal str data: %v", err)
			continue
		}

		mut.Lock()
		if len(chain.Chain) > len(myChain.Chain) {
			myChain = &chain
			spew.Dump(myChain)
			// Broadcast the new chain to all peers
			chainData, _ := json.Marshal(myChain)
			go broadcastData(chainData)
		}
		mut.Unlock()
	}
}

// writeData writes data to the stream.
func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read data: %v", err)
		}

		sendData = strings.TrimSpace(sendData)
		if sendData == "print" {
			spew.Dump(myChain)
			continue
		}	
		if sendData == "exit" {
			return
		}

		var userMsg UserMessage
		if err := json.Unmarshal([]byte(sendData), &userMsg); err != nil {
			log.Printf("Failed to unmarshal send data: %v", err)
			continue
		}

		spew.Dump(userMsg)

		mut.Lock()
		myChain.AppendBlock(userMsg.Location, userMsg.WaveHeight)
		if !myChain.IsValid() {
			spew.Dump(myChain)
			log.Println("Chain is not valid anymore")
			mut.Unlock()
			return
		}
		mut.Unlock()

		bytes, err := json.Marshal(myChain)
		if err != nil {
			log.Printf("Failed to marshal chain: %v", err)
			continue
		}

		spew.Dump(myChain)

		go broadcastData(bytes)
	}
}

// startPeer initializes and starts the peer node.
func startPeer(ctx context.Context, h host.Host, streamHandler network.StreamHandler) {
	// set a function as stream handler
	// this function is called when a peer connects, and starts a stream with this protocol
	// only applies on the receiving side
	h.SetStreamHandler("/chat/1.0.0", streamHandler)

	// let's get the actual tcp port from our listen multiaddr, in case we're using 0 (default: random available port)
	var port string
	for _, la := range h.Network().ListenAddresses() {
		if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
			port = p
			break
		}
	}

	if port == "" {
		log.Println("failed to find actual local port")
		return
	}

	log.Printf("RUN 'go run *.go -d /ip4/127.0.0.1/tcp/%v/p2p/%s' on another console. \n", port, h.ID())
	log.Println("You can replace 127.0.0.1 with public IP as well")
	log.Println("Waiting for incoming connection")
	log.Println()
}

// startPeerAndConnect connects to another peer node.
func startPeerAndConnect(ctx context.Context, h host.Host, destinations []string) error {
	for _, dest := range destinations {
		maddr, err := multiaddr.NewMultiaddr(dest)
		if err != nil {
			return fmt.Errorf("failed to parse multiaddr %s: %w", dest, err)
		}

		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return fmt.Errorf("failed to get peer info: %w", err)
		}

		if err := h.Connect(ctx, *info); err != nil {
			return fmt.Errorf("failed to connect to peer %s: %w", dest, err)
		}

		s, err := h.NewStream(ctx, info.ID, "/chat/1.0.0")
		if err != nil {
			return fmt.Errorf("failed to create new stream: %w", err)
		}

		go handleStream(s)
	}

	return nil
}
