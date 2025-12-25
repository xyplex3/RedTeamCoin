//go:build windows

// Package main implements Windows-specific self-deletion functionality using
// advanced NTFS stream manipulation and helper process fallback techniques.
//
// The Windows implementation uses two approaches for maximum compatibility:
//
//  1. Advanced technique: Renames the executable to an alternate data stream
//     (:Zone.Identifier) and marks it for deletion using Windows internal APIs.
//     This is stealthier but may fail on some Windows 11 24H2 systems.
//
//  2. Helper process fallback: Spawns a detached process that waits for the
//     parent to exit, then deletes the executable. More reliable but creates
//     a temporary child process.
//
// Implementation based on techniques from https://github.com/secur30nly/go-self-delete
package main

import (
	"log"
	"os"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// FileRenameInfo is the Windows FileInformationClass value for renaming
	// files via SetFileInformationByHandle. Used to rename the executable to
	// an alternate data stream before deletion.
	FileRenameInfo = 3

	// FileDispositionInfo is the Windows FileInformationClass value for
	// marking files for deletion. When set, the file is deleted when all
	// handles are closed.
	FileDispositionInfo = 4
)

// FILE_RENAME_INFO represents the Windows FILE_RENAME_INFO structure used
// with SetFileInformationByHandle to rename files. The FileName field is
// variable-length; this structure must be allocated with sufficient buffer
// space for the target filename.
type FILE_RENAME_INFO struct {
	ReplaceIfExists uint32
	RootDirectory   windows.Handle
	FileNameLength  uint32
	FileName        [1]uint16
}

// FILE_DISPOSITION_INFO represents the Windows FILE_DISPOSITION_INFO
// structure used with SetFileInformationByHandle to mark files for deletion.
// When DeleteFile is set to 1, the file is deleted when all handles close.
type FILE_DISPOSITION_INFO struct {
	DeleteFile uint32
}

var (
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procSetFileInformationByHandle = kernel32.NewProc("SetFileInformationByHandle")
	ntdll                          = windows.NewLazySystemDLL("ntdll.dll")
	procRtlCopyMemory              = ntdll.NewProc("RtlCopyMemory")
)

// deleteSelf removes the executable at path using Windows-specific deletion
// techniques. It first attempts the advanced NTFS stream method; if that fails,
// falls back to spawning a helper process. This two-tier approach ensures
// maximum compatibility across Windows versions.
func deleteSelf(path string) {
	if err := deleteSelfAdvanced(path); err != nil {
		log.Printf("Advanced deletion failed (%v), falling back to helper process", err)
		deleteSelfHelper(path)
	} else {
		log.Println("Executable marked for self-deletion (advanced technique)")
	}
}

// deleteSelfAdvanced implements the advanced self-deletion technique using
// NTFS alternate data streams. It opens the executable, renames it to an ADS
// (:Zone.Identifier), and marks it for deletion. Returns an error if any step
// fails, allowing fallback to helper process method. This technique is stealthy
// but may fail on Windows 11 24H2.
func deleteSelfAdvanced(path string) error {
	handle, err := openHandleForDeletion(path)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	if err := renameToADS(handle); err != nil {
		return err
	}

	if err := markForDeletion(handle); err != nil {
		return err
	}

	return nil
}

// deleteSelfHelper spawns a detached helper process that waits for the
// parent to exit and then deletes the executable. This is the fallback method
// when advanced deletion fails. The helper runs as a separate, hidden process
// group to avoid inheriting console/signals. More compatible with Windows 11
// 24H2 but less stealthy than the advanced technique.
func deleteSelfHelper(path string) {
	pid := windows.GetCurrentProcessId()

	exePath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		log.Printf("Failed to convert path: %v", err)
		return
	}

	cmdLine := path + " --delete-helper " + strconv.Itoa(int(pid)) + " " + path
	cmdLinePtr, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		log.Printf("Failed to build command line: %v", err)
		return
	}

	var si windows.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si)) // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	si.Flags = windows.STARTF_USESHOWWINDOW
	si.ShowWindow = windows.SW_HIDE

	var pi windows.ProcessInformation

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

	windows.CloseHandle(pi.Process)
	windows.CloseHandle(pi.Thread)

	log.Println("Executable scheduled for deletion via helper process (fallback)")
}

// openHandleForDeletion opens the file at path with DELETE access rights and
// sharing permissions that allow reading and deletion while the handle is open.
// This handle is used for subsequent rename and deletion operations. Returns
// the file handle or an error if opening fails.
func openHandleForDeletion(path string) (windows.Handle, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE,
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

// renameToADS renames the file handle to an NTFS alternate data stream
// (:Zone.Identifier), making it invisible in directory listings. Uses
// SetFileInformationByHandle with FileRenameInfo. The file content remains
// accessible via the original path, but the primary stream is renamed.
// Returns an error if the rename operation fails.
func renameToADS(handle windows.Handle) error {
	newName := ":Zone.Identifier"
	newNameU16 := windows.StringToUTF16(newName)

	fileNameBytes := len(newNameU16) * 2
	structSize := int(unsafe.Sizeof(FILE_RENAME_INFO{})) + fileNameBytes - 2 // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block

	buffer := make([]byte, structSize)
	renameInfo := (*FILE_RENAME_INFO)(unsafe.Pointer(&buffer[0])) // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	renameInfo.ReplaceIfExists = 0
	renameInfo.RootDirectory = 0
	renameInfo.FileNameLength = uint32(fileNameBytes)

	fileNamePtr := unsafe.Pointer(uintptr(unsafe.Pointer(renameInfo)) + unsafe.Offsetof(renameInfo.FileName)) // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	_, _, _ = procRtlCopyMemory.Call(
		uintptr(fileNamePtr),
		uintptr(unsafe.Pointer(&newNameU16[0])), // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
		uintptr(fileNameBytes),
	)

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

// markForDeletion sets the deletion disposition on the file handle using
// SetFileInformationByHandle with FileDispositionInfo. When DeleteFile is
// set to 1, Windows marks the file for deletion when all handles are closed.
// This is the final step in the advanced deletion technique. Returns an error
// if the operation fails.
func markForDeletion(handle windows.Handle) error {
	dispInfo := FILE_DISPOSITION_INFO{
		DeleteFile: 1,
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

// runDeletionHelper is the helper process entry point that waits for the
// parent process (identified by pid) to exit, then deletes the executable
// at path. It polls the parent process every 100ms until it's no longer
// accessible, waits an additional 500ms for handle release, then performs
// the deletion. This function is invoked when the program is launched with
// the --delete-helper flag.
func runDeletionHelper(pidStr, path string) {
	logFile, err := os.OpenFile(path+".deletion.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)
	logger.Printf("Deletion helper started for PID %s, path %s", pidStr, path)

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		logger.Printf("Failed to parse PID: %v", err)
		return
	}

	logger.Printf("Waiting for parent process %d to exit...", pid)
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		logger.Printf("Failed to open parent process: %v", err)
		return
	}
	defer windows.CloseHandle(handle)

	// Use WaitForSingleObject to efficiently wait for process termination
	const INFINITE = 0xFFFFFFFF
	logger.Printf("Waiting for process object to be signaled...")
	ret, err := windows.WaitForSingleObject(handle, INFINITE)
	if ret == windows.WAIT_OBJECT_0 {
		logger.Printf("Parent process exited (process object signaled)")
	} else {
		logger.Printf("Wait failed: ret=%d, err=%v", ret, err)
		return
	}

	logger.Printf("Parent exited, waiting for file handles to release...")
	time.Sleep(500 * time.Millisecond)

	logger.Printf("Attempting to delete %s using advanced technique", path)

	// Try advanced NTFS deletion first (rename to ADS + mark for deletion)
	fileHandle, err := openHandleForDeletion(path)
	if err != nil {
		logger.Printf("Failed to open file for deletion: %v", err)
		return
	}
	defer windows.CloseHandle(fileHandle)

	logger.Printf("Successfully opened file handle for deletion")

	err = renameToADS(fileHandle)
	if err != nil {
		logger.Printf("Failed to rename to ADS: %v, trying direct deletion", err)
		windows.CloseHandle(fileHandle)

		// Fallback to simple deletion with retry
		maxRetries := 10
		for i := 0; i < maxRetries; i++ {
			pathPtr, err := windows.UTF16PtrFromString(path)
			if err != nil {
				logger.Printf("Failed to convert path to UTF16: %v", err)
				return
			}

			err = windows.DeleteFile(pathPtr)
			if err == nil {
				logger.Printf("Successfully deleted %s using DeleteFile on attempt %d", path, i+1)
				return
			}

			if i < maxRetries-1 {
				waitTime := time.Duration(50*(1<<uint(i))) * time.Millisecond
				logger.Printf("DeleteFile attempt %d failed: %v, retrying in %v...", i+1, err, waitTime)
				time.Sleep(waitTime)
			} else {
				logger.Printf("DeleteFile failed after %d attempts: %v", maxRetries, err)
			}
		}

		// Final fallback: mark for deletion on reboot
		logger.Printf("All deletion attempts failed, marking for deletion on reboot...")
		pathPtr, err := windows.UTF16PtrFromString(path)
		if err != nil {
			logger.Printf("Failed to convert path for MoveFileEx: %v", err)
			return
		}

		const MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
		err = windows.MoveFileEx(pathPtr, nil, MOVEFILE_DELAY_UNTIL_REBOOT)
		if err != nil {
			logger.Printf("MoveFileEx DELAY_UNTIL_REBOOT failed: %v", err)
			return
		}
		logger.Printf("Successfully marked %s for deletion on next reboot", path)
		return
	}

	logger.Printf("Successfully renamed to ADS, marking for deletion")
	err = markForDeletion(fileHandle)
	if err != nil {
		logger.Printf("Failed to mark for deletion: %v", err)
		return
	}

	logger.Printf("Successfully marked %s for deletion", path)
}
