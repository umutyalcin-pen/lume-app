package organizer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"lume-go/internal/logger"
	"lume-go/internal/metadata"
	"os"
	"path/filepath"
	"strings"
)

func SanitizeFolderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "Unknown"
	}

	invalidChars := `<>:"/\|?*.`
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, string(char), "_")
	}

	var cleaned strings.Builder
	for _, r := range name {
		if r < 0x20 {
			cleaned.WriteRune('_')
		} else {
			cleaned.WriteRune(r)
		}
	}
	name = cleaned.String()

	upper := strings.ToUpper(name)
	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
	}
	if reserved[upper] {
		return name + "_safe"
	}

	if (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && len(upper) == 4 {
		digit := upper[3]
		if digit >= '1' && digit <= '9' {
			return name + "_safe"
		}
	}

	if len(name) > 100 {
		return name[:100]
	}

	return name
}

type plannedFile struct {
	size int64
	hash string
}

type State struct {
	PlannedPaths map[string]bool
	PlannedMeta  map[string]plannedFile
	CreatedDirs  map[string]bool
}

func NewState() *State {
	return &State{
		PlannedPaths: make(map[string]bool),
		PlannedMeta:  make(map[string]plannedFile),
		CreatedDirs:  make(map[string]bool),
	}
}

func ArchiveFile(info metadata.FileInfo, targetBase string) error {
	_, _, err := ArchiveFileWithOptions(info, targetBase, false, false, nil)
	return err
}

func ArchiveFileWithOptions(info metadata.FileInfo, targetBase string, dryRun bool, rename bool, state *State) (finalPath string, skipped bool, err error) {
	if state == nil {
		state = NewState()
	}

	year := SanitizeFolderName(info.Year)
	month := SanitizeFolderName(info.Month)
	device := SanitizeFolderName(info.Device)

	if info.Source != "" && info.Source != "Other_Imports" {
		if info.Device == "Unknown" || info.Device == "" {
			device = SanitizeFolderName(info.Source)
		} else {
			device = SanitizeFolderName(info.Source + "_" + info.Device)
		}
	}
	if device == "Unknown" || device == "" {
		device = "Other_Sorted"
	}

	targetDir := filepath.Join(targetBase, year, month, device)

	absBase, err := filepath.Abs(targetBase)
	if err != nil {
		return "", false, fmt.Errorf("invalid target base path: %w", err)
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return "", false, fmt.Errorf("invalid target path: %w", err)
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false, fmt.Errorf("security violation: path traversal detected in destination folder structure")
	}

	if !dryRun && !state.CreatedDirs[targetDir] {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return "", false, fmt.Errorf("mkdir failed for %s: %w", targetDir, err)
		}
		state.CreatedDirs[targetDir] = true
	}

	filename := info.Filename
	if rename {
		ext := strings.ToLower(filepath.Ext(info.Filename))
		filename = fmt.Sprintf("%04d%02d%02d_%02d%02d%02d%s",
			info.Date.Year(), info.Date.Month(), info.Date.Day(),
			info.Date.Hour(), info.Date.Minute(), info.Date.Second(),
			ext)
	}

	finalPath = filepath.Join(targetDir, filename)
	
	isTargetConflict := false
	if _, err := os.Lstat(finalPath); !os.IsNotExist(err) {
		isTargetConflict = true
	} else if state.PlannedPaths[finalPath] {
		isTargetConflict = true
	}

	if isTargetConflict {
		isDup := false
		if dryRun {
			if planned, found := state.PlannedMeta[finalPath]; found {
				if planned.size == info.Size {
					srcHash, errHash := metadata.GetFileHash(info.Path)
					if errHash == nil {
						isDup = (planned.hash == srcHash)
					}
				}
			}
		} else {
			var err error
			isDup, err = IsDuplicate(info.Path, finalPath)
			if err != nil {
				logger.Error("Duplicate check fail for %s: %v", filename, err)
			}
		}

		if isDup {
			return finalPath, true, nil
		}

		finalPath, err = ResolveConflict(finalPath, state)
		if err != nil {
			return "", false, fmt.Errorf("conflict resolution failed for %s: %w", filename, err)
		}
	}
	state.PlannedPaths[finalPath] = true

	if !dryRun {
		if err := AtomicCopy(info.Path, finalPath); err != nil {
			delete(state.PlannedPaths, finalPath)
			return "", false, fmt.Errorf("archive copy error for %s: %w", filename, err)
		}
		logger.Info("Successfully archived (copied): %s -> %s", info.Filename, finalPath)
	} else {
		srcHash, errHash := metadata.GetFileHash(info.Path)
		if errHash == nil {
			state.PlannedMeta[finalPath] = plannedFile{size: info.Size, hash: srcHash}
		}
		logger.Info("Simulation: would archive %s -> %s", info.Filename, finalPath)
	}

	return finalPath, false, nil
}

func IsDuplicate(p1, p2 string) (bool, error) {
	s1, err := os.Stat(p1)
	if err != nil {
		return false, fmt.Errorf("stat src: %w", err)
	}
	s2, err := os.Stat(p2)
	if err != nil {
		return false, fmt.Errorf("stat dst: %w", err)
	}

	if !s1.Mode().IsRegular() || !s2.Mode().IsRegular() {
		return false, nil
	}

	if s1.Size() != s2.Size() {
		return false, nil
	}

	h1, err := metadata.GetFileHash(p1)
	if err != nil {
		return false, fmt.Errorf("hash src: %w", err)
	}
	h2, err := metadata.GetFileHash(p2)
	if err != nil {
		return false, fmt.Errorf("hash dst: %w", err)
	}
	return h1 == h2, nil
}

func ResolveConflict(path string, state *State) (string, error) {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i < 10000; i++ {
		newPath := fmt.Sprintf("%s_%d%s", base, i, ext)
		_, err := os.Lstat(newPath)
		if os.IsNotExist(err) && !state.PlannedPaths[newPath] {
			return newPath, nil
		}
	}
	return "", fmt.Errorf("10000 çakışma varyantı denendi, uygun isim bulunamadı: %s", path)
}

func AtomicCopy(src, dst string) error {
	sh, err := CopyFile(src, dst)
	if err != nil {
		os.Remove(dst)
		return fmt.Errorf("copy failed: %w", err)
	}

	th, err := metadata.GetFileHash(dst)
	if err != nil {
		os.Remove(dst)
		return fmt.Errorf("post-copy hash failed: %w", err)
	}

	if sh != th {
		os.Remove(dst)
		return fmt.Errorf("integrity failed: hash mismatch during copy")
	}

	return nil
}

func CopyFile(src, dst string) (string, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", err
	}

	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return "", err
	}
	
	var outClosed bool
	defer func() {
		if !outClosed {
			out.Close()
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, h), in); err != nil {
		out.Close()
		outClosed = true
		return "", err
	}

	if err := out.Sync(); err != nil {
		out.Close()
		outClosed = true
		return "", fmt.Errorf("dosya senkronize edilemedi: %w", err)
	}

	if err := out.Close(); err != nil {
		outClosed = true
		return "", fmt.Errorf("dosya kapatılamadı: %w", err)
	}
	outClosed = true

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		logger.Error("İzinler ayarlanamadı: %s: %v", filepath.Base(dst), err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
