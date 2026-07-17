package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func CheckDiskSpace(path string, requiredBytes int64) error {

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

func IsPathSafe(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	upperName := strings.ToUpper(nameWithoutExt)

	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
	}
	if reserved[upperName] {
		return false
	}

	if (strings.HasPrefix(upperName, "COM") || strings.HasPrefix(upperName, "LPT")) && len(upperName) == 4 {
		digit := upperName[3]
		if digit >= '1' && digit <= '9' {
			return false
		}
	}

	if strings.Contains(path, "..") {
		return false
	}
	return true
}

func IsSystemDir(path string) bool {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		realPath, err = filepath.Abs(path)
		if err != nil {
			return false
		}
	}
	realPath = strings.ToLower(realPath)

	var systemDirs []string
	if runtime.GOOS == "windows" {
		sysRoot := os.Getenv("SystemRoot")
		if sysRoot == "" {
			sysRoot = "C:\\Windows"
		}
		pf := os.Getenv("ProgramFiles")
		if pf == "" {
			pf = "C:\\Program Files"
		}
		pfx86 := os.Getenv("ProgramFiles(x86)")
		if pfx86 == "" {
			pfx86 = "C:\\Program Files (x86)"
		}
		pd := os.Getenv("ProgramData")
		if pd == "" {
			pd = "C:\\ProgramData"
		}
		systemDirs = []string{
			strings.ToLower(sysRoot),
			strings.ToLower(pf),
			strings.ToLower(pfx86),
			strings.ToLower(pd),
		}
	} else {
		systemDirs = []string{
			"/etc", "/usr", "/var", "/bin", "/sbin", "/lib", "/sys", "/proc", "/dev", "/boot", "/root",
		}
	}

	for _, dir := range systemDirs {
		if dir == "" {
			continue
		}
		dirLower := strings.ToLower(dir)
		realSysDir, errSys := filepath.EvalSymlinks(dirLower)
		if errSys == nil {
			dirLower = strings.ToLower(realSysDir)
		}
		rel, err := filepath.Rel(dirLower, realPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}

func IsNested(src, dst string) bool {
	absSrc, errSrc := filepath.Abs(src)
	absDst, errDst := filepath.Abs(dst)
	if errSrc != nil || errDst != nil {
		return false
	}

	rel, err := filepath.Rel(absSrc, absDst)
	if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
		return true
	}

	relSrc, errRel := filepath.Rel(absDst, absSrc)
	if errRel == nil && !strings.HasPrefix(relSrc, "..") && relSrc != "." {
		return true
	}

	return false
}
