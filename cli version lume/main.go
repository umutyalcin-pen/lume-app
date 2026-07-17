package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

const AppVersion = "1.1-CLI"

type MediaType int

const (
	TypeImage MediaType = iota
	TypeRaw
	TypeVideo
)

var supportedFiles = map[string]MediaType{
	".jpg": TypeImage, ".jpeg": TypeImage, ".png": TypeImage, ".webp": TypeImage,
	".heic": TypeImage, ".tiff": TypeImage, ".gif": TypeImage, ".bmp": TypeImage,
	".dng": TypeRaw, ".cr2": TypeRaw, ".nef": TypeRaw, ".arw": TypeRaw, ".orf": TypeRaw,
	".mp4": TypeVideo, ".mov": TypeVideo, ".avi": TypeVideo, ".mkv": TypeVideo,
	".m4v": TypeVideo, ".flv": TypeVideo, ".wmv": TypeVideo, ".mpg": TypeVideo, ".mpeg": TypeVideo,
	".3gp": TypeVideo,
}

var errInterrupted = fmt.Errorf("interrupted")

func printHelp() {
	fmt.Printf(`
Lume CLI v%s - Fotoğraf ve Video Arşivleyici

Kullanım: Lume_LITE.exe <kaynak> <hedef> [seçenekler]
Örnek:   Lume_LITE.exe "C:\Fotos" "C:\Arsiv" --exif --dry-run --rename

Seçenekler:
  --exif       Görsellerde ve RAW dosyalarında EXIF çekim tarihini (DateTimeOriginal) okur.
               Varsayılan olarak EXIF okunmaz, doğrudan dosya değiştirme tarihi (ModTime) kullanılır.
  --dry-run    Simülasyon modunda çalışır. Diske hiçbir klasör oluşturulmaz veya kopyalama yapılmaz.
  --rename     Dosyaları hedef dizine kopyalarken YYYYMMDD_HHMMSS formatında yeniden adlandırır.
  --help, -h   Bu yardım metnini gösterir.

Lume CLI v%s - Photo & Video Archiver

Usage:   Lume_LITE.exe <source> <target> [options]
Example: Lume_LITE.exe "C:\Photos" "C:\Archive" --exif --dry-run --rename

Options:
  --exif       Reads EXIF shooting date (DateTimeOriginal) from images and RAW files.
               By default EXIF is not read; file modification date (ModTime) is used directly.
  --dry-run    Runs in simulation mode. No folders are created on disk.
  --rename     Renames files to YYYYMMDD_HHMMSS format when copying to target.
  --help, -h   Shows this help text.
`, AppVersion, AppVersion)
}

func main() {
	var useExif bool
	var dryRun bool
	var rename bool
	var paths []string

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h", "/?":
			printHelp()
			os.Exit(0)
		case "--exif", "-exif":
			useExif = true
		case "--dry-run", "-dry-run":
			dryRun = true
		case "--rename", "-rename":
			rename = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Printf("[WARN] Bilinmeyen seçenek: %s (yardım için --help)\n", arg)
				continue
			}
			paths = append(paths, arg)
		}
	}

	if len(paths) < 2 {
		printHelp()
		os.Exit(1)
	}

	src, dst := paths[0], paths[1]

	absSrc, errAbsSrc := filepath.Abs(src)
	if errAbsSrc != nil {
		fmt.Printf("[FATAL] Kaynak yol çözümlenemedi: %v\n", errAbsSrc)
		os.Exit(1)
	}
	absDst, errAbsDst := filepath.Abs(dst)
	if errAbsDst != nil {
		fmt.Printf("[FATAL] Hedef yol çözümlenemedi: %v\n", errAbsDst)
		os.Exit(1)
	}
	if absSrc == absDst {
		fmt.Println("[FATAL] Kaynak ve hedef aynı olamaz!")
		os.Exit(1)
	}

	rel, errRel := filepath.Rel(absSrc, absDst)
	if errRel == nil && !strings.HasPrefix(rel, "..") && rel != "." {
		fmt.Println("[FATAL] Hedef klasör kaynak klasörün içinde olamaz!")
		os.Exit(1)
	}

	relSrc, errRelSrc := filepath.Rel(absDst, absSrc)
	if errRelSrc == nil && !strings.HasPrefix(relSrc, "..") && relSrc != "." {
		fmt.Println("[FATAL] Kaynak klasör hedef klasörün içinde olamaz!")
		os.Exit(1)
	}

	if isSystemDir(absSrc) {
		fmt.Println("[FATAL] Kaynak klasör korumalı bir sistem dizini olamaz!")
		os.Exit(1)
	}
	if isSystemDir(absDst) {
		fmt.Println("[FATAL] Hedef klasör korumalı bir sistem dizini olamaz!")
		os.Exit(1)
	}

	if _, err := os.Stat(absSrc); os.IsNotExist(err) {
		fmt.Printf("[FATAL] Kaynak bulunamadı: %s\n", absSrc)
		os.Exit(1)
	}
	if !dryRun {
		if err := os.MkdirAll(absDst, 0755); err != nil {
			fmt.Printf("[FATAL] Hedef klasör oluşturulamadı: %v\n", err)
			os.Exit(1)
		}
	}

	files, totalSize, errCollect := collectFiles(absSrc)
	if errCollect != nil {
		fmt.Printf("[FATAL] Dosyalar taranırken hata oluştu: %v\n", errCollect)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("[INFO] Kaynak klasörde arşivlenecek desteklenen dosya bulunamadı.")
		os.Exit(0)
	}

	header := fmt.Sprintf("Lume CLI v%s - Active Organizer", AppVersion)
	if dryRun {
		header += " [SIMULATION MODE]"
	}
	fmt.Println(header)
	fmt.Printf("Source: %s\n", absSrc)
	fmt.Printf("Target: %s\n", absDst)
	fmt.Println(strings.Repeat("-", 40))

	if !dryRun {
		if err := checkDiskSpace(absDst, totalSize); err != nil {
			fmt.Printf("[FATAL] Yetersiz disk alanı: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println(strings.Repeat("-", 40))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	org := &Organizer{
		absDst:       absDst,
		totalFiles:   len(files),
		createdDirs:  make(map[string]bool),
		sigChan:      make(chan os.Signal, 1),
		useExif:      useExif,
		dryRun:       dryRun,
		rename:       rename,
		plannedPaths: make(map[string]bool),
		plannedMeta:  make(map[string]plannedFile),
		ctx:          ctx,
	}

	signal.Notify(org.sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		first := true
		for range org.sigChan {
			if first {
				fmt.Println("\n[WARN] İptal sinyali alındı! Güvenle durduruluyor...")
				cancel()
				org.cancelled.Store(true)
				first = false
			} else {
				fmt.Println("\n[WARN] Zaten durduruluyor, lütfen bekleyin...")
			}
		}
	}()

	for _, entry := range files {
		err := org.Process(entry)
		if err == errInterrupted {
			break
		}
	}
	signal.Stop(org.sigChan)
	close(org.sigChan)

	fmt.Println(strings.Repeat("-", 40))
	if org.isCancelled() {
		fmt.Println("[WARN] İşlem kullanıcı tarafından iptal edildi.")
	}
	suffix := ""
	if org.dryRun {
		suffix = " (Simulation)"
	}
	fmt.Printf("[DONE] %d success, %d skipped, %d errors%s\n", org.success, org.duplicates, org.errors, suffix)
}
