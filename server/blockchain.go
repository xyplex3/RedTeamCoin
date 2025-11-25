package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// Block represents a block in the blockchain
type Block struct {
	Index        int64
	Timestamp    int64
	Data         string
	PreviousHash string
	Hash         string
	Nonce        int64
	MinedBy      string
}

// Blockchain represents the chain of blocks
type Blockchain struct {
	Blocks     []*Block
	Difficulty int
	mu         sync.RWMutex
}

// NewBlockchain creates a new blockchain with genesis block
func NewBlockchain(difficulty int) *Blockchain {
	bc := &Blockchain{
		Blocks:     make([]*Block, 0),
		Difficulty: difficulty,
	}

	// Create genesis block
	genesis := &Block{
		Index:        0,
		Timestamp:    time.Now().Unix(),
		Data:         "Genesis Block",
		PreviousHash: "0",
		Nonce:        0,
	}
	genesis.Hash = calculateHash(genesis)
	bc.Blocks = append(bc.Blocks, genesis)

	return bc
}

// calculateHash calculates the hash of a block
func calculateHash(block *Block) string {
	record := strconv.FormatInt(block.Index, 10) +
		strconv.FormatInt(block.Timestamp, 10) +
		block.Data +
		block.PreviousHash +
		strconv.FormatInt(block.Nonce, 10)

	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// GetLatestBlock returns the last block in the chain
func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

// AddBlock adds a new block to the blockchain
func (bc *Blockchain) AddBlock(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Verify block
	if !bc.isValidNewBlock(block, bc.Blocks[len(bc.Blocks)-1]) {
		return fmt.Errorf("invalid block")
	}

	bc.Blocks = append(bc.Blocks, block)
	return nil
}

// isValidNewBlock checks if a new block is valid
func (bc *Blockchain) isValidNewBlock(newBlock, previousBlock *Block) bool {
	if previousBlock.Index+1 != newBlock.Index {
		return false
	}

	if previousBlock.Hash != newBlock.PreviousHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	// Check if hash meets difficulty requirement
	prefix := ""
	for i := 0; i < bc.Difficulty; i++ {
		prefix += "0"
	}

	if len(newBlock.Hash) < bc.Difficulty || newBlock.Hash[:bc.Difficulty] != prefix {
		return false
	}

	return true
}

// GetBlockchain returns a copy of all blocks
func (bc *Blockchain) GetBlockchain() []*Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	blocks := make([]*Block, len(bc.Blocks))
	copy(blocks, bc.Blocks)
	return blocks
}

// GetBlockCount returns the number of blocks
func (bc *Blockchain) GetBlockCount() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.Blocks)
}

// ValidateChain validates the entire blockchain
func (bc *Blockchain) ValidateChain() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		previousBlock := bc.Blocks[i-1]

		if !bc.isValidNewBlock(currentBlock, previousBlock) {
			return false
		}
	}
	return true
}
