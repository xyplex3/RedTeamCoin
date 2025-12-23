// Package main implements the RedTeamCoin mining pool server components.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// Block represents a single block in the blockchain containing transaction
// data, proof-of-work hash, and linkage to the previous block.
//
// Each block is immutable once added to the chain and contains a nonce value
// that was found through the mining process to satisfy the difficulty
// requirement.
type Block struct {
	Index        int64  // Position of the block in the chain
	Timestamp    int64  // Unix timestamp when block was created
	Data         string // Arbitrary data stored in the block
	PreviousHash string // Hash of the previous block in the chain
	Hash         string // SHA-256 hash of this block's contents
	Nonce        int64  // Proof-of-work nonce that satisfies difficulty
	MinedBy      string // ID of the miner who found this block
}

// Blockchain represents an immutable chain of blocks using proof-of-work
// consensus. All operations are thread-safe for concurrent access by
// multiple goroutines.
//
// The difficulty parameter controls mining complexity by requiring block
// hashes to have a certain number of leading zeros.
type Blockchain struct {
	Blocks     []*Block     // Ordered list of blocks in the chain
	Difficulty int          // Number of leading zeros required in block hashes
	mu         sync.RWMutex // Protects concurrent access to Blocks
}

// NewBlockchain creates a new blockchain initialized with a genesis block.
// The difficulty parameter specifies how many leading zeros are required
// in valid block hashes. Higher difficulty values require more computational
// work to mine blocks.
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

// calculateHash calculates the SHA-256 hash of a block by concatenating its
// index, timestamp, data, previous hash, and nonce. The result is returned
// as a hexadecimal string.
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

// GetLatestBlock returns the most recently added block in the blockchain,
// or nil if the chain is empty. This method is safe for concurrent use.
func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

// AddBlock adds a new block to the blockchain after validating it against
// the current chain state. It returns an error if the block is invalid
// according to consensus rules. This method is safe for concurrent use.
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

// isValidNewBlock reports whether a new block is valid for addition to the
// chain. It verifies the block index, previous hash linkage, hash correctness,
// and difficulty requirements.
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

// GetBlockchain returns a copy of all blocks in the chain.
// This method is safe for concurrent use and does not affect the original
// blockchain.
func (bc *Blockchain) GetBlockchain() []*Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	blocks := make([]*Block, len(bc.Blocks))
	copy(blocks, bc.Blocks)
	return blocks
}

// GetBlockCount returns the total number of blocks in the blockchain,
// including the genesis block. This method is safe for concurrent use.
func (bc *Blockchain) GetBlockCount() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.Blocks)
}

// ValidateChain reports whether the entire blockchain is valid by checking
// each block's linkage and proof-of-work. It returns false if any block in
// the chain violates consensus rules. This method is safe for concurrent use.
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
