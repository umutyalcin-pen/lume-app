package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

type plannedFile struct {
	size int64
	hash string
}

type Organizer struct {
	absDst       string
	totalFiles   int
	success      int
	errors       int
	duplicates   int
	processed    int
	createdDirs  map[string]bool
	sigChan      chan os.Signal
	cancelled    atomic.Bool
	useExif      bool
	dryRun       bool
	rename       bool
	plannedPaths map[string]bool
	plannedMeta  map[string]plannedFile
	ctx          context.Context
}

func (o *Organizer) isCancelled() bool {
	return o.cancelled.Load()
}

func (o *Organizer) Process(entry fileEntry) error {
	if o.isCancelled() {
		return errInterrupted
	}

	o.processed++
	progressPrefix := fmt.Sprintf("[%d/%d]", o.processed, o.totalFiles)

	path := entry.path

	if entry.mode&os.ModeSymlink != 0 {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	if _, found := supportedFiles[ext]; !found {
		return nil
	}

	year, month, day, mediaTime := GetMediaDate(path, entry, o.useExif)

	filename := filepath.Base(path)
	if o.rename {
		filename = fmt.Sprintf("%04d%02d%02d_%02d%02d%02d%s",
			mediaTime.Year(), mediaTime.Month(), mediaTime.Day(),
			mediaTime.Hour(), mediaTime.Minute(), mediaTime.Second(),
			ext)
	}

	targetDir := filepath.Join(o.absDst, year, month, day)
	if !o.dryRun && !o.createdDirs[targetDir] {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			fmt.Printf("[ERROR] Klasör oluşturulamadı: %v\n", err)
			o.errors++
			return nil
		}
		o.createdDirs[targetDir] = true
	}

	targetPath := filepath.Join(targetDir, filename)

	var srcHash string
	var errHash error
	getSrcHash := func() (string, error) {
		if srcHash != "" || errHash != nil {
			return srcHash, errHash
		}
		srcHash, errHash = fileHash(path)
		return srcHash, errHash
	}

	if o.dryRun {
		isTargetConflict := false
		if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
			isTargetConflict = true
		} else if o.plannedPaths[targetPath] {
			isTargetConflict = true
		}

		if isTargetConflict {
			sizeMatches := false
			if planned, found := o.plannedMeta[targetPath]; found {
				sizeMatches = (planned.size == entry.size)
			} else if dstInfo, err := os.Stat(targetPath); err == nil {
				sizeMatches = (dstInfo.Size() == entry.size)
			}

			if sizeMatches {
				sHash, errH := getSrcHash()
				if errH == nil {
					if o.isDuplicate(sHash, entry.size, targetPath) {
						fmt.Printf("%s [WOULD-SKIP] Duplicate ignored: %s\n", progressPrefix, filename)
						o.duplicates++
						return nil
					}
				}
			}
			resolved, errConflict := o.resolveConflict(targetPath)
			if errConflict == nil {
				targetPath = resolved
				filename = filepath.Base(targetPath)
			}
		}
		sHash, errH := getSrcHash()
		if errH == nil {
			o.plannedMeta[targetPath] = plannedFile{size: entry.size, hash: sHash}
		}
		o.plannedPaths[targetPath] = true
		fmt.Printf("%s [WOULD-OK]   %s -> %s/%s/%s\n", progressPrefix, filename, year, month, day)
		o.success++
		return nil
	}

	isTargetConflict := false
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		isTargetConflict = true
	} else if o.plannedPaths[targetPath] {
		isTargetConflict = true
	}

	if isTargetConflict {
		sizeMatches := false
		if dstInfo, err := os.Stat(targetPath); err == nil {
			sizeMatches = (dstInfo.Size() == entry.size)
		}

		if sizeMatches {
			sHash, errH := getSrcHash()
			if errH != nil {
				fmt.Printf("[ERROR] %s: Kaynak dosya hash hesaplanamadı: %v\n", filename, errH)
				o.errors++
				return nil
			}
			if o.isDuplicate(sHash, entry.size, targetPath) {
				fmt.Printf("%s [SKIP]  Duplicate ignored: %s\n", progressPrefix, filename)
				o.duplicates++
				return nil
			}
		}
		resolved, errConflict := o.resolveConflict(targetPath)
		if errConflict != nil {
			fmt.Printf("[ERROR] %s: %v\n", filename, errConflict)
			o.errors++
			return nil
		}
		targetPath = resolved
		filename = filepath.Base(targetPath)
	}
	o.plannedPaths[targetPath] = true

	copiedSrcHash, errCopy := copyAndHashFile(o.ctx, path, targetPath, entry.mode)
	if errCopy != nil {
		if o.isCancelled() {
			fmt.Printf("[WARN]  %s: İptal edildi, hedef temizlendi\n", filename)
			o.errors++
			return errInterrupted
		}
		fmt.Printf("[ERROR] %s: %v\n", filename, errCopy)
		_ = os.Remove(targetPath)
		delete(o.plannedPaths, targetPath)
		o.errors++
		return nil
	}

	dstHash, err2 := fileHash(targetPath)
	if err2 != nil || copiedSrcHash != dstHash {
		fmt.Printf("[ERROR] %s: Kopyalama doğrulama hatası, hedef temizlendi\n", filename)
		if errDel := os.Remove(targetPath); errDel != nil {
			fmt.Printf("[WARN]  Bozuk dosya silinemedi: %s: %v\n", targetPath, errDel)
		}
		delete(o.plannedPaths, targetPath)
		o.errors++
		return nil
	}

	if err := os.Chtimes(targetPath, entry.modTime, entry.modTime); err != nil {
		fmt.Printf("[WARN]  %s: Zaman damgası geri yüklenemedi: %v\n", filename, err)
	}

	fmt.Printf("%s [OK]   %s -> %s/%s/%s\n", progressPrefix, filename, year, month, day)
	o.success++
	return nil
}

func (o *Organizer) isDuplicate(srcHash string, srcSize int64, targetPath string) bool {
	if o.dryRun {
		if planned, found := o.plannedMeta[targetPath]; found {
			return planned.size == srcSize && planned.hash == srcHash
		}
	}
	dstInfo, err := os.Stat(targetPath)
	if err != nil || dstInfo.Size() != srcSize {
		return false
	}
	dstHash, errHash := fileHash(targetPath)
	return errHash == nil && srcHash == dstHash
}

func (o *Organizer) resolveConflict(path string) (string, error) {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i < 10000; i++ {
		np := fmt.Sprintf("%s_%d%s", base, i, ext)
		_, err := os.Lstat(np)
		if os.IsNotExist(err) && !o.plannedPaths[np] {
			return np, nil
		}
	}
	return "", fmt.Errorf("10000 çakışma varyantı denendi, uygun isim bulunamadı: %s", path)
}
