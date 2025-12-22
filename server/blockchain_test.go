package main

import (
	"testing"
	"time"
)

func TestNewBlockchain(t *testing.T) {
	difficulty := 4
	bc := NewBlockchain(difficulty)

	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	if bc.Difficulty != difficulty {
		t.Errorf("Expected difficulty %d, got %d", difficulty, bc.Difficulty)
	}

	if len(bc.Blocks) != 1 {
		t.Errorf("Expected 1 genesis block, got %d blocks", len(bc.Blocks))
	}

	genesis := bc.Blocks[0]
	if genesis.Index != 0 {
		t.Errorf("Expected genesis block index 0, got %d", genesis.Index)
	}

	if genesis.Data != "Genesis Block" {
		t.Errorf("Expected genesis block data 'Genesis Block', got '%s'", genesis.Data)
	}

	if genesis.PreviousHash != "0" {
		t.Errorf("Expected genesis block previous hash '0', got '%s'", genesis.PreviousHash)
	}
}

func TestCalculateHash(t *testing.T) {
	block := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Data:         "Test Block",
		PreviousHash: "abc123",
		Nonce:        42,
	}

	hash1 := calculateHash(block)
	hash2 := calculateHash(block)

	if hash1 != hash2 {
		t.Error("calculateHash should return consistent results for the same block")
	}

	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64 (SHA256), got %d", len(hash1))
	}

	// Change block data and verify hash changes
	block.Nonce = 43
	hash3 := calculateHash(block)
	if hash1 == hash3 {
		t.Error("Hash should change when block data changes")
	}
}

func TestGetLatestBlock(t *testing.T) {
	bc := NewBlockchain(4)

	latest := bc.GetLatestBlock()
	if latest == nil {
		t.Fatal("GetLatestBlock returned nil")
	}

	if latest.Index != 0 {
		t.Errorf("Expected latest block index 0, got %d", latest.Index)
	}

	// Add a new block
	newBlock := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Data:         "Block 1",
		PreviousHash: latest.Hash,
		Nonce:        0,
	}
	// Generate valid hash with correct difficulty
	for {
		newBlock.Hash = calculateHash(newBlock)
		if len(newBlock.Hash) >= bc.Difficulty && newBlock.Hash[:bc.Difficulty] == "0000" {
			break
		}
		newBlock.Nonce++
	}

	bc.AddBlock(newBlock)

	latest = bc.GetLatestBlock()
	if latest.Index != 1 {
		t.Errorf("Expected latest block index 1, got %d", latest.Index)
	}
}

func TestAddBlockValid(t *testing.T) {
	bc := NewBlockchain(4)
	latest := bc.GetLatestBlock()

	newBlock := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Data:         "Test Block",
		PreviousHash: latest.Hash,
		Nonce:        0,
	}

	// Mine a valid block
	for {
		newBlock.Hash = calculateHash(newBlock)
		if len(newBlock.Hash) >= bc.Difficulty && newBlock.Hash[:bc.Difficulty] == "0000" {
			break
		}
		newBlock.Nonce++
	}

	err := bc.AddBlock(newBlock)
	if err != nil {
		t.Errorf("AddBlock failed for valid block: %v", err)
	}

	if len(bc.Blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(bc.Blocks))
	}
}

func TestAddBlockInvalidIndex(t *testing.T) {
	bc := NewBlockchain(4)
	latest := bc.GetLatestBlock()

	newBlock := &Block{
		Index:        999, // Invalid index
		Timestamp:    time.Now().Unix(),
		Data:         "Test Block",
		PreviousHash: latest.Hash,
		Nonce:        0,
	}

	newBlock.Hash = calculateHash(newBlock)

	err := bc.AddBlock(newBlock)
	if err == nil {
		t.Error("AddBlock should fail for invalid index")
	}
}

func TestAddBlockInvalidPreviousHash(t *testing.T) {
	bc := NewBlockchain(4)

	newBlock := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Data:         "Test Block",
		PreviousHash: "invalid_hash", // Invalid previous hash
		Nonce:        0,
	}

	// Mine a valid hash
	for {
		newBlock.Hash = calculateHash(newBlock)
		if len(newBlock.Hash) >= bc.Difficulty && newBlock.Hash[:bc.Difficulty] == "0000" {
			break
		}
		newBlock.Nonce++
	}

	err := bc.AddBlock(newBlock)
	if err == nil {
		t.Error("AddBlock should fail for invalid previous hash")
	}
}

func TestAddBlockInvalidDifficulty(t *testing.T) {
	bc := NewBlockchain(4)
	latest := bc.GetLatestBlock()

	newBlock := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Data:         "Test Block",
		PreviousHash: latest.Hash,
		Nonce:        0,
	}

	// Set hash without meeting difficulty requirement
	newBlock.Hash = calculateHash(newBlock)

	err := bc.AddBlock(newBlock)
	if err == nil {
		t.Error("AddBlock should fail for block not meeting difficulty")
	}
}

func TestGetBlockchain(t *testing.T) {
	bc := NewBlockchain(4)

	blocks := bc.GetBlockchain()
	if len(blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(blocks))
	}

	// Verify the slice is a copy (different underlying array)
	// Note: Block pointers are shared, so modifying block content will affect both
	originalLen := len(bc.Blocks)
	blocks = append(blocks, &Block{Index: 999})

	if len(bc.Blocks) != originalLen {
		t.Error("Modifying returned slice should not affect original blockchain slice")
	}
}

func TestGetBlockCount(t *testing.T) {
	bc := NewBlockchain(4)

	count := bc.GetBlockCount()
	if count != 1 {
		t.Errorf("Expected block count 1, got %d", count)
	}

	// Add a block
	latest := bc.GetLatestBlock()
	newBlock := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Data:         "Block 1",
		PreviousHash: latest.Hash,
		Nonce:        0,
	}

	// Mine valid block
	for {
		newBlock.Hash = calculateHash(newBlock)
		if len(newBlock.Hash) >= bc.Difficulty && newBlock.Hash[:bc.Difficulty] == "0000" {
			break
		}
		newBlock.Nonce++
	}

	bc.AddBlock(newBlock)

	count = bc.GetBlockCount()
	if count != 2 {
		t.Errorf("Expected block count 2, got %d", count)
	}
}

func TestValidateChain(t *testing.T) {
	bc := NewBlockchain(4)

	// Initially valid
	if !bc.ValidateChain() {
		t.Error("New blockchain should be valid")
	}

	// Add valid blocks
	for i := 1; i <= 3; i++ {
		latest := bc.GetLatestBlock()
		newBlock := &Block{
			Index:        latest.Index + 1,
			Timestamp:    time.Now().Unix(),
			Data:         "Block",
			PreviousHash: latest.Hash,
			Nonce:        0,
		}

		// Mine valid block
		for {
			newBlock.Hash = calculateHash(newBlock)
			if len(newBlock.Hash) >= bc.Difficulty && newBlock.Hash[:bc.Difficulty] == "0000" {
				break
			}
			newBlock.Nonce++
		}

		bc.AddBlock(newBlock)
	}

	// Chain should still be valid
	if !bc.ValidateChain() {
		t.Error("Blockchain with valid blocks should be valid")
	}

	// Corrupt a block
	bc.Blocks[2].Data = "Corrupted"

	// Chain should now be invalid
	if bc.ValidateChain() {
		t.Error("Blockchain with corrupted block should be invalid")
	}
}

func TestIsValidNewBlock(t *testing.T) {
	bc := NewBlockchain(4)
	latest := bc.GetLatestBlock()

	// Create valid block
	validBlock := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Data:         "Valid Block",
		PreviousHash: latest.Hash,
		Nonce:        0,
	}

	// Mine valid block
	for {
		validBlock.Hash = calculateHash(validBlock)
		if len(validBlock.Hash) >= bc.Difficulty && validBlock.Hash[:bc.Difficulty] == "0000" {
			break
		}
		validBlock.Nonce++
	}

	if !bc.isValidNewBlock(validBlock, latest) {
		t.Error("Valid block should pass validation")
	}

	// Test wrong index
	wrongIndexBlock := &Block{
		Index:        999,
		Timestamp:    time.Now().Unix(),
		Data:         "Block",
		PreviousHash: latest.Hash,
		Nonce:        validBlock.Nonce,
		Hash:         validBlock.Hash,
	}

	if bc.isValidNewBlock(wrongIndexBlock, latest) {
		t.Error("Block with wrong index should fail validation")
	}

	// Test wrong previous hash
	wrongPrevBlock := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Data:         "Block",
		PreviousHash: "wrong_hash",
		Nonce:        0,
	}
	wrongPrevBlock.Hash = calculateHash(wrongPrevBlock)

	if bc.isValidNewBlock(wrongPrevBlock, latest) {
		t.Error("Block with wrong previous hash should fail validation")
	}
}

func TestConcurrentAccess(t *testing.T) {
	bc := NewBlockchain(4)

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = bc.GetLatestBlock()
			_ = bc.GetBlockCount()
			_ = bc.GetBlockchain()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify blockchain is still valid
	if !bc.ValidateChain() {
		t.Error("Blockchain should remain valid after concurrent reads")
	}
}
