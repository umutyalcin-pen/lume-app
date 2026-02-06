package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// CheckDiskSpace checks if there is enough space on the destination drive
func CheckDiskSpace(path string, requiredBytes int64) error {
	// Robust volume name detection for UNC or relative paths
	volName := filepath.VolumeName(path)
	if volName == "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("could not resolve absolute path: %v", err)
		}
		volName = filepath.VolumeName(absPath)
	}
	
	pathPtr, err := syscall.UTF16PtrFromString(volName + "\\")
	if err != nil {
		return err
	}

	var freeBytes int64
	var totalBytes int64
	var totalFreeBytes int64

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	ret, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if ret == 0 {
		return fmt.Errorf("failed to get disk space: %v", err)
	}

	if freeBytes < requiredBytes {
		return fmt.Errorf("insufficient disk space: need %d bytes, have %d", requiredBytes, freeBytes)
	}

	return nil
}

// CheckWritability verifies if the application has write permissions for the folder
func CheckWritability(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("target directory does not exist: %s", path)
	}

	tempFile := filepath.Join(path, ".lume_write_test")
	err := os.WriteFile(tempFile, []byte("test"), 0644)
	if err != nil {
		return fmt.Errorf("folder is not writable: %v", err)
	}
	os.Remove(tempFile)
	return nil
}

// IsPathSafe checks for reserved Windows names and traversal
func IsPathSafe(path string) bool {
	base := filepath.Base(path)
	reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "LPT1", "LPT2", "LPT3"}
	upperBase := strings.ToUpper(base)
	for _, r := range reserved {
		if upperBase == r {
			return false
		}
	}
	if strings.Contains(path, "..") {
		return false
	}
	return true
}
