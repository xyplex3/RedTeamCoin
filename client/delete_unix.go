//go:build !windows

// Package main implements a cryptocurrency mining client with
// self-deletion capabilities (no-op on Unix-like systems).
package main

import (
	"log"
	"os"
)

// deleteSelf attempts to delete the executable on Unix-like systems
// Uses a simple os.Remove which works because Unix allows deletion
// of running executables (unlike Windows).
func deleteSelf(path string) {
	if err := os.Remove(path); err != nil {
		log.Printf("Failed to delete executable: %v", err)
	} else {
		log.Println("Executable deleted successfully")
	}
}

// runDeletionHelper is not needed on Unix-like systems
// Included for compatibility with main.go argument parsing
func runDeletionHelper(pidStr, path string) {
	// No-op on Unix-like systems
	log.Println("Deletion helper not needed on Unix-like systems")
}
