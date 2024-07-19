package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type BlockData struct {
	Location   string `json:"location"`
	WaveHeight int    `json:"wave_height"`
}

type Block struct {
	Data         BlockData `json:"data"`
	Hash         string    `json:"hash"`
	PreviousHash string    `json:"previous_hash"`
	Timestamp    int64     `json:"timestamp"`
	Height       int       `json:"height"`
	Pow          int       `json:"pow"`
}

// calculateHash computes the hash of the block's data.
func (b *Block) calculateHash() string {
	data, err := json.Marshal(b.Data)
	if err != nil {
		log.Printf("Failed to marshal block data: %v", err)
		return ""
	}
	blockData := fmt.Sprintf("%s%s%d%d%d", b.PreviousHash, data, b.Timestamp, b.Height, b.Pow)
	blockHash := sha256.Sum256([]byte(blockData))
	return fmt.Sprintf("%x", blockHash)
}

// mine mines the block by finding a hash with the required difficulty.
func (b *Block) mine(difficulty int) {
	target := strings.Repeat("0", difficulty)
	for !strings.HasPrefix(b.Hash, target) {
		b.Pow++
		b.Hash = b.calculateHash()
	}
}
