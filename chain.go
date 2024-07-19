package main

import (
	"log"
	"sync"
	"time"
)

type Blockchain struct {
	Chain      []Block      `json:"chain"`
	Difficulty int          `json:"difficulty"`
	mu         sync.RWMutex `json:"-"`
}

// NewBlockchain creates a new blockchain with the genesis block.
func NewBlockchain(difficulty int) *Blockchain {
	genesisBlock := Block{
		Hash:      "0",
		Height:    0,
		Timestamp: time.Now().Unix(),
	}
	genesisBlock.Hash = genesisBlock.calculateHash()
	return &Blockchain{
		Chain:      []Block{genesisBlock},
		Difficulty: difficulty,
	}
}

// AppendBlock appends a new block to the blockchain.
func (b *Blockchain) AppendBlock(location string, waveHeight int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	blockData := BlockData{
		Location:   location,
		WaveHeight: waveHeight,
	}
	lastBlock := b.Chain[len(b.Chain)-1]
	newBlock := Block{
		Data:         blockData,
		PreviousHash: lastBlock.Hash,
		Timestamp:    time.Now().Unix(),
		Height:       lastBlock.Height + 1,
	}
	newBlock.mine(b.Difficulty)
	b.Chain = append(b.Chain, newBlock)
}

// IsValid checks if the blockchain is valid.
func (b *Blockchain) IsValid() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for i := 1; i < len(b.Chain); i++ {
		previousBlock := b.Chain[i-1]
		currentBlock := b.Chain[i]

		if currentBlock.Height != previousBlock.Height+1 {
			log.Println("Bad block height")
			return false
		}

		if currentBlock.Hash != currentBlock.calculateHash() {
			log.Println("Bad block hash")
			return false
		}

		if currentBlock.PreviousHash != previousBlock.Hash {
			log.Println("Bad block previous hash")
			return false
		}
	}

	return true
}
