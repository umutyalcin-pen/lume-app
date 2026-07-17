package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type fileEntry struct {
	path    string
	size    int64
	modTime time.Time
	mode    os.FileMode
}

func collectFiles(src string) ([]fileEntry, int64, error) {
	var files []fileEntry
	var totalSize int64
	var scanned atomic.Int64
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Printf("\r[INFO] Dosyalar taranıyor... %d bulundu", scanned.Load())
			case <-done:
				return
			}
		}
	}()

	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if path == src {
				return err
			}
			return nil
		}
		if d.IsDir() {
			if isSystemDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if _, found := supportedFiles[ext]; found {
			info, errInfo := d.Info()
			if errInfo != nil {
				return nil
			}
			files = append(files, fileEntry{
				path:    path,
				size:    info.Size(),
				modTime: info.ModTime(),
				mode:    info.Mode(),
			})
			totalSize += info.Size()
			scanned.Add(1)
		}
		return nil
	})

	close(done)

	fmt.Printf("\r[INFO] Toplam bulunan dosya: %-10d\n", len(files))

	return files, totalSize, err
}

func checkDiskSpace(path string, requiredBytes int64) error {
	volName := filepath.VolumeName(path)
	if volName == "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("mutlak yol çözümlenemedi: %v", err)
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
		return fmt.Errorf("disk alanı bilgisi alınamadı: %v", err)
	}

	if freeBytes < requiredBytes {
		return fmt.Errorf("yetersiz disk alanı: gereken %d byte, mevcut %d", requiredBytes, freeBytes)
	}

	return nil
}

func isSystemDir(path string) bool {
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
