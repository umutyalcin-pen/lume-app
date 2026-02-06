package organizer

import (
	"fmt"
	"io"
	"lume-go/internal/logger"
	"lume-go/internal/metadata"
	"os"
	"path/filepath"
	"strings"
)

// SanitizeFolderName cleans folder names for OS compatibility. (Audit Point 6 Tested)
func SanitizeFolderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "Unknown"
	}

	invalidChars := `<>:"/\|?*.`
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, string(char), "_")
	}

	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "LPT1": true,
	}
	if reserved[strings.ToUpper(name)] {
		return name + "_safe"
	}

	if len(name) > 100 {
		return name[:100]
	}

	return name
}

// MoveFile handles the movement of a file with detailed result reporting. (Elite Error Wrapping)
func MoveFile(info metadata.FileInfo, targetBase string) error {
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
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("mkdir failed for %s: %w", targetDir, err)
	}

	finalPath := filepath.Join(targetDir, info.Filename)
	if _, err := os.Stat(finalPath); err == nil {
		isDup, err := IsDuplicate(info.Path, finalPath)
		if err != nil {
			logger.Error("Duplicate check fail for %s: %v", info.Filename, err)
		} else if isDup {
			return nil
		}
		finalPath = ResolveConflict(finalPath)
	}

	if err := AtomicMove(info.Path, finalPath); err != nil {
		return fmt.Errorf("archive move error for %s: %w", info.Filename, err)
	}
	
	logger.Info("Successfully archived: %s -> %s", info.Filename, finalPath)
	return nil
}

func IsDuplicate(p1, p2 string) (bool, error) {
	s1, err := os.Stat(p1); if err != nil { return false, fmt.Errorf("stat src: %w", err) }
	s2, err := os.Stat(p2); if err != nil { return false, fmt.Errorf("stat dst: %w", err) }
	if s1.Size() != s2.Size() { return false, nil }

	h1, err := metadata.GetFileHash(p1); if err != nil { return false, fmt.Errorf("hash src: %w", err) }
	h2, err := metadata.GetFileHash(p2); if err != nil { return false, fmt.Errorf("hash dst: %w", err) }
	return h1 == h2, nil
}

func ResolveConflict(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i < 10000; i++ {
		newPath := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}
	return path
}

func AtomicMove(src, dst string) error {
	sh, err := metadata.GetFileHash(src); if err != nil { return fmt.Errorf("pre-move hash: %w", err) }
	if err := os.Rename(src, dst); err != nil {
		if err := CopyFile(src, dst); err != nil { return fmt.Errorf("copy failed: %w", err) }
		if err := os.Remove(src); err != nil { logger.Error("Cleanup error: %v", err) }
	}
	th, err := metadata.GetFileHash(dst); if err != nil { return fmt.Errorf("post-move hash: %w", err) }
	if sh != th { os.Remove(dst); return fmt.Errorf("integrity failed: hash mismatch") }
	return nil
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src); if err != nil { return err }; defer in.Close()
	out, err := os.Create(dst); if err != nil { return err }; defer out.Close()
	if _, err := io.Copy(out, in); err != nil { return err }
	return out.Sync()
}
