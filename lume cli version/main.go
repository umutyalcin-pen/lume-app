package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const AppVersion = "2.1-LITE"

var supportedExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".heic": true, ".tiff": true, ".mp4": true, ".mov": true, ".avi": true,
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf(`
Lume LITE v%s - Ultra Hafif Fotoğraf Arşivleyici

Kullanım: lume-lite <kaynak> <hedef>
Örnek:   lume-lite "C:\Fotos" "C:\Arsiv"

Not: EXIF desteği yok, dosya tarihi kullanılır.
`, AppVersion)
		os.Exit(1)
	}

	src, dst := os.Args[1], os.Args[2]

	// Path validation
	absSrc, _ := filepath.Abs(src)
	absDst, _ := filepath.Abs(dst)
	if absSrc == absDst {
		fmt.Println("❌ Kaynak ve hedef aynı olamaz!")
		os.Exit(1)
	}
	if strings.HasPrefix(absDst, absSrc+string(filepath.Separator)) {
		fmt.Println("❌ Hedef klasör kaynak klasörün içinde olamaz!")
		os.Exit(1)
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		fmt.Printf("❌ Kaynak bulunamadı: %s\n", src)
		os.Exit(1)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		fmt.Printf("❌ Hedef klasör oluşturulamadı: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 Lume LITE v%s\n", AppVersion)
	fmt.Printf("📂 %s → %s\n", src, dst)
	fmt.Println(strings.Repeat("-", 40))

	success, errors := 0, 0

	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExt[ext] {
			return nil
		}

		// Use file modification time
		t := info.ModTime()
		year := fmt.Sprintf("%d", t.Year())
		month := fmt.Sprintf("%02d", t.Month())

		targetDir := filepath.Join(dst, year, month)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			fmt.Printf("❌ Klasör oluşturulamadı: %v\n", err)
			errors++
			return nil
		}

		targetPath := filepath.Join(targetDir, info.Name())

		// Check duplicate
		if _, err := os.Stat(targetPath); err == nil {
			if isDuplicate(path, targetPath) {
				fmt.Printf("⏭️  Kopya atlandı: %s\n", info.Name())
				return nil
			}
			targetPath = resolveConflict(targetPath)
		}

		// Move file
		if err := os.Rename(path, targetPath); err != nil {
			// Try copy if rename fails (cross-drive)
			if err := copyFile(path, targetPath); err != nil {
				fmt.Printf("❌ %s: %v\n", info.Name(), err)
				errors++
				return nil
			}

			// Verify copy integrity before deleting source
			srcHash, err1 := fileHash(path)
			dstHash, err2 := fileHash(targetPath)
			if err1 != nil || err2 != nil || srcHash != dstHash {
				fmt.Printf("❌ %s: Kopyalama doğrulama hatası, kaynak korundu\n", info.Name())
				if err := os.Remove(targetPath); err != nil {
					fmt.Printf("⚠️  Bozuk dosya silinemedi: %s\n", targetPath)
				}
				errors++
				return nil
			}

			// Safe to delete source now
			if err := os.Remove(path); err != nil {
				fmt.Printf("⚠️  %s → %s/%s (kaynak korundu)\n", info.Name(), year, month)
			} else {
				fmt.Printf("✅ %s → %s/%s\n", info.Name(), year, month)
			}
			success++
			return nil
		}

		fmt.Printf("✅ %s → %s/%s\n", info.Name(), year, month)
		success++
		return nil
	})

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("✨ %d başarılı, %d hata\n", success, errors)
}

func isDuplicate(p1, p2 string) bool {
	h1, e1 := fileHash(p1)
	h2, e2 := fileHash(p2)
	return e1 == nil && e2 == nil && h1 == h2
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func resolveConflict(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i < 10000; i++ {
		np := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(np); os.IsNotExist(err) {
			return np
		}
	}
	return path
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// Ensure data is flushed to disk
	return out.Sync()
}
