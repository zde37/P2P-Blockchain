package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"os"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
)

var (
	mutex   = &sync.Mutex{}
	myChain = NewBlockchain(2)
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sourcePort := flag.Int("sp", 0, "Source port number")
	destList := flag.String("d", "", "Comma-separated list of destination multiaddr strings")
	help := flag.Bool("help", false, "Display help")
	debug := flag.Bool("debug", false, "Debug generates the same node ID on every execution")
	flag.Parse()

	if *help {
		fmt.Println("This program demonstrates a simple p2p blockchain application")
		fmt.Println("Usage: Run 'go run *.go -sp <SOURCE_PORT>' where <SOURCE_PORT> can be any port number.")
		fmt.Println("Now run 'go run *.go -d <MULTIADDR1>,<MULTIADDR2>,...' where <MULTIADDRx> are the multiaddresses of previous listener hosts.")
		os.Exit(0)
	}

	var r io.Reader
	if *debug {
		r = mrand.New(mrand.NewSource(int64(*sourcePort)))
	} else {
		r = rand.Reader
	}

	h, err := makeHost(*sourcePort, r)
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(h.Addrs())

	if *destList == "" {
		startPeer(ctx, h, handleStream) 
		select {}
	} else {
		destinations := strings.Split(*destList, ",")
		log.Println(destinations)
		err := startPeerAndConnect(ctx, h, destinations)
		if err != nil {
			log.Fatal(err)
		}

		stdReader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("> ")
			sendData, err := stdReader.ReadString('\n')
			if err != nil {
				log.Fatalf("Error reading from stdin: %v", err)
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

			mutex.Lock()
			myChain.AppendBlock(userMsg.Location, userMsg.WaveHeight)
			if !myChain.IsValid() {
				spew.Dump(myChain)
				log.Println("Chain is not valid anymore")
				mut.Unlock()
				return
			}
			mutex.Unlock()

			bytes, err := json.Marshal(myChain)
			if err != nil {
				log.Printf("Failed to marshal chain: %v", err)
				continue
			}

			go broadcastData(bytes)
		}
	}
}
