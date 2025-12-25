//go:build !windows

// Package main implements Unix-specific self-deletion functionality using
// simple file removal. Unix-like systems (Linux, macOS, BSD) allow deletion
// of executables while they are running, unlike Windows. The executable
// continues running from its inode until the process terminates, at which
// point the disk space is reclaimed.
package main

import (
	"log"
	"os"
)

// deleteSelf removes the executable at path using os.Remove. On Unix-like
// systems, this operation succeeds even while the executable is running
// because the filesystem unlinks the directory entry but retains the inode
// and file content until all file descriptors are closed. The running process
// continues executing from memory until termination.
func deleteSelf(path string) {
	if err := os.Remove(path); err != nil {
		log.Printf("Failed to delete executable: %v", err)
	} else {
		log.Println("Executable deleted successfully")
	}
}

// runDeletionHelper is a no-op on Unix-like systems, included only for
// cross-platform compatibility with the Windows helper process method. On
// Unix, no helper process is needed because executables can be deleted
// while running. This function is called when launched with --delete-helper
// flag but performs no action.
func runDeletionHelper(pidStr, path string) {
	log.Println("Deletion helper not needed on Unix-like systems")
}
