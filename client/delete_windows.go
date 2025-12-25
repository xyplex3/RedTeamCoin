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
	// handles are closed. (Legacy API, pre-Windows 11 24H2)
	FileDispositionInfo = 4

	// FileDispositionInfoEx is the Windows FileInformationClass value for
	// marking files for deletion using extended disposition flags. This is
	// required for Windows 11 24H2 and later to enable POSIX deletion semantics.
	FileDispositionInfoEx = 21

	// FILE_DISPOSITION_FLAG_DELETE indicates the file should be deleted when
	// all handles are closed. Used with FileDispositionInfoEx.
	FILE_DISPOSITION_FLAG_DELETE = 0x1

	// FILE_DISPOSITION_FLAG_POSIX_SEMANTICS enables POSIX-style deletion where
	// the file is unlinked immediately when the handle closes, even if the file
	// is memory-mapped or has open handles. Required for Windows 11 24H2.
	FILE_DISPOSITION_FLAG_POSIX_SEMANTICS = 0x2

	// INFINITE represents an infinite timeout for Windows wait operations.
	// Used with WaitForSingleObject to wait indefinitely for process termination.
	INFINITE = 0xFFFFFFFF

	// maxDeletionRetries is the maximum number of retry attempts for file deletion.
	maxDeletionRetries = 10

	// baseDeletionBackoff is the base delay for exponential backoff during deletion retries.
	// Actual delays will be: 50ms, 100ms, 200ms, 400ms, ...
	baseDeletionBackoff = 50 * time.Millisecond

	// MOVEFILE_DELAY_UNTIL_REBOOT is the Windows flag for MoveFileEx to schedule
	// file deletion on next system reboot.
	MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
)

var (
	// debugMode controls whether debug logging is enabled for self-deletion operations.
	// Set via RTCOIN_DEBUG environment variable. Useful during development and testing
	// but should remain disabled in production to avoid forensic evidence.
	debugMode = os.Getenv("RTCOIN_DEBUG") == "1"
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
// This is the legacy structure for Windows versions prior to 11 24H2.
type FILE_DISPOSITION_INFO struct {
	DeleteFile uint32
}

// FILE_DISPOSITION_INFO_EX represents the Windows FILE_DISPOSITION_INFORMATION_EX
// structure used with SetFileInformationByHandle for extended deletion control.
// The Flags field accepts FILE_DISPOSITION_FLAG_DELETE and
// FILE_DISPOSITION_FLAG_POSIX_SEMANTICS for Windows 11 24H2+ compatibility.
type FILE_DISPOSITION_INFO_EX struct {
	Flags uint32
}

var (
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procSetFileInformationByHandle = kernel32.NewProc("SetFileInformationByHandle")
	ntdll                          = windows.NewLazySystemDLL("ntdll.dll")
	procRtlCopyMemory              = ntdll.NewProc("RtlCopyMemory")
)

// debugLog logs a message only when debug mode is enabled via RTCOIN_DEBUG=1.
// This allows detailed logging during development and testing without leaving
// forensic evidence in production deployments.
func debugLog(format string, v ...interface{}) {
	if debugMode {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// retryWithExponentialBackoff executes the given operation with exponential backoff
// retry logic. It will retry up to maxRetries times, waiting baseDelay * 2^attempt
// between each retry. Returns nil if the operation succeeds, or the last error if
// all retries are exhausted.
func retryWithExponentialBackoff(operation func() error, maxRetries int, baseDelay time.Duration) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
			debugLog("retry attempt %d/%d failed: %v", i+1, maxRetries, err)
		}

		if i < maxRetries-1 {
			waitTime := baseDelay * time.Duration(1<<uint(i))
			debugLog("waiting %v before retry %d", waitTime, i+2)
			time.Sleep(waitTime)
		}
	}
	return lastErr
}

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
// SetFileInformationByHandle. First attempts the Windows 11 24H2+ method using
// FileDispositionInfoEx with POSIX semantics, which enables immediate file
// unlinking even for memory-mapped files. Falls back to the legacy
// FileDispositionInfo API for older Windows versions. Returns an error if both
// methods fail.
func markForDeletion(handle windows.Handle) error {
	// Try Windows 11 24H2+ method first: FileDispositionInfoEx with POSIX semantics
	// This works around the 24H2 NTFS driver changes that broke the old technique
	dispInfoEx := FILE_DISPOSITION_INFO_EX{
		Flags: FILE_DISPOSITION_FLAG_DELETE | FILE_DISPOSITION_FLAG_POSIX_SEMANTICS,
	}

	debugLog("trying FileDispositionInfoEx with POSIX semantics")
	ret, _, err := procSetFileInformationByHandle.Call(
		uintptr(handle),
		uintptr(FileDispositionInfoEx),
		uintptr(unsafe.Pointer(&dispInfoEx)), // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
		uintptr(unsafe.Sizeof(dispInfoEx)),   // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	)

	if ret != 0 {
		debugLog("FileDispositionInfoEx succeeded")
		return nil
	}

	debugLog("FileDispositionInfoEx failed: %v, falling back to legacy API", err)

	// Fallback to legacy method for older Windows versions (pre-24H2)
	dispInfo := FILE_DISPOSITION_INFO{
		DeleteFile: 1,
	}

	debugLog("trying FileDispositionInfo (legacy)")
	ret, _, err = procSetFileInformationByHandle.Call(
		uintptr(handle),
		uintptr(FileDispositionInfo),
		uintptr(unsafe.Pointer(&dispInfo)), // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
		uintptr(unsafe.Sizeof(dispInfo)),   // nosemgrep: go.lang.security.audit.unsafe.use-of-unsafe-block
	)

	if ret == 0 {
		debugLog("FileDispositionInfo failed: %v", err)
		return err
	}

	debugLog("FileDispositionInfo succeeded")
	return nil
}

// runDeletionHelper is the helper process entry point that waits for the
// parent process (identified by pid) to exit, then deletes the executable
// at path. It waits for process termination, pauses for handle release, then
// performs the deletion. This function is invoked when the program is launched
// with the --delete-helper flag. Operations are silent by default but can be
// logged via RTCOIN_DEBUG=1 environment variable.
func runDeletionHelper(pidStr, path string) {
	debugLog("helper: starting deletion helper for PID %s, path %s", pidStr, path)

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		debugLog("helper: invalid PID %q: %v", pidStr, err)
		return
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		debugLog("helper: failed to open process %d: %v", pid, err)
		return
	}
	defer windows.CloseHandle(handle)

	// Use WaitForSingleObject to efficiently wait for process termination
	debugLog("helper: waiting for process %d to exit", pid)
	ret, _ := windows.WaitForSingleObject(handle, INFINITE)
	if ret != windows.WAIT_OBJECT_0 {
		debugLog("helper: wait failed with return code %d", ret)
		return
	}

	debugLog("helper: process exited, waiting for handle release")
	time.Sleep(500 * time.Millisecond)

	// Try advanced NTFS deletion first (rename to ADS + mark for deletion)
	debugLog("helper: attempting advanced deletion for %s", path)
	fileHandle, err := openHandleForDeletion(path)
	if err != nil {
		debugLog("helper: failed to open handle for deletion: %v", err)
		attemptFallbackDeletion(path)
		return
	}
	defer windows.CloseHandle(fileHandle)

	err = renameToADS(fileHandle)
	if err != nil {
		debugLog("helper: failed to rename to ADS: %v", err)
		windows.CloseHandle(fileHandle)
		attemptFallbackDeletion(path)
		return
	}

	err = markForDeletion(fileHandle)
	if err != nil {
		debugLog("helper: failed to mark for deletion: %v", err)
		windows.CloseHandle(fileHandle)
		attemptFallbackDeletion(path)
		return
	}

	debugLog("helper: successfully marked file for deletion")
}

// attemptFallbackDeletion tries to delete the file using simpler methods when
// advanced techniques fail. First attempts DeleteFile with exponential backoff
// retry, then marks for deletion on reboot as a last resort.
func attemptFallbackDeletion(path string) {
	debugLog("attempting fallback deletion for %s", path)

	// Attempt simple deletion with retry and exponential backoff
	err := retryWithExponentialBackoff(func() error {
		pathPtr, err := windows.UTF16PtrFromString(path)
		if err != nil {
			return err
		}
		return windows.DeleteFile(pathPtr)
	}, maxDeletionRetries, baseDeletionBackoff)

	if err == nil {
		debugLog("fallback deletion succeeded for %s", path)
		return
	}

	debugLog("fallback deletion failed after retries: %v, scheduling deletion on reboot", err)

	// Final fallback: mark for deletion on reboot
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		debugLog("failed to convert path for reboot deletion: %v", err)
		return
	}

	err = windows.MoveFileEx(pathPtr, nil, MOVEFILE_DELAY_UNTIL_REBOOT)
	if err != nil {
		debugLog("failed to schedule deletion on reboot: %v", err)
		return
	}

	debugLog("file scheduled for deletion on next reboot")
}
