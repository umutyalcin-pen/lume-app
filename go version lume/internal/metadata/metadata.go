package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"lume-go/internal/logger"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dsoprea/go-exif/v3"
)

var SupportedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".heic": true,
	".tiff": true,
	".mp4":  true,
	".mov":  true,
	".avi":  true,
}

type FileInfo struct {
	Path     string
	Filename string
	Size     int64
	ModTime  time.Time
	Date     time.Time
	Year     string
	Month    string
	Device   string
	Source   string
	Hash     string
}

func GetFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func GetFileInfo(path string) (FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	if !SupportedExtensions[ext] {
		return FileInfo{}, fmt.Errorf("unsupported extension: %s", ext)
	}

	info := FileInfo{
		Path:     path,
		Filename: filepath.Base(path),
		Size:     stat.Size(),
		ModTime:  stat.ModTime(),
		Date:     stat.ModTime(),
		Device:   "Unknown",
		Source:   DetectSource(filepath.Base(path)),
	}

	isImage := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".heic": true, ".tiff": true}
	if isImage[ext] && stat.Size() <= 250*1024*1024 {

		func() {
			defer func() {
				if r := recover(); r != nil {

					logger.Error("EXIF parsing panic for %s: %v", path, r)
				}
			}()
			if exifDate, device, err := ExtractExif(path); err == nil {
				if exifDate != nil {
					info.Date = *exifDate
				}
				if device != "" {
					info.Device = device
				}
			}
		}()
	} else {

		if createTime, err := GetCreationTime(path); err == nil {
			info.Date = createTime
		}
	}

	info.Year = fmt.Sprintf("%d", info.Date.Year())
	info.Month = fmt.Sprintf("%02d", info.Date.Month())

	return info, nil
}

func ExtractExif(path string) (*time.Time, string, error) {
	rawExif, err := exif.SearchFileAndExtractExif(path)
	if err != nil {

		if errors.Is(err, exif.ErrNoExif) {
			return nil, "", nil
		}

		return nil, "", err
	}

	entries, _, err := exif.GetFlatExifData(rawExif, nil)
	if err != nil {

		return nil, "", err
	}

	var dateOriginal, dateDigitized, dateGeneric string
	var device string

	for _, entry := range entries {
		switch entry.TagName {
		case "DateTimeOriginal":
			dateOriginal = entry.FormattedFirst
		case "DateTimeDigitized":
			dateDigitized = entry.FormattedFirst
		case "DateTime":
			dateGeneric = entry.FormattedFirst
		case "Model":
			device = strings.TrimSpace(entry.FormattedFirst)
		}
	}

	candidates := []string{dateOriginal, dateDigitized, dateGeneric}
	layouts := []string{
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02T15:04:05",
		"2006:01:02",
		"2006-01-02",
		"2006/01/02",
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(strings.Trim(candidate, "\x00"))
		if candidate != "" {
			for _, layout := range layouts {
				t, err := time.ParseInLocation(layout, candidate, time.Local)
				if err == nil {
					return &t, device, nil
				}
			}
		}
	}

	return nil, device, nil
}

func DetectSource(filename string) string {
	lower := strings.ToLower(strings.TrimSpace(filename))

	type PatternMatch struct {
		Pattern string
		Source  string
	}

	patterns := []PatternMatch{
		{"whatsapp", "WhatsApp"},
		{"-wa", "WhatsApp"},
		{"telegram", "Telegram"},
		{"screenshot", "Screenshots"},
		{"ekran", "Screenshots"},
		{"instagram", "Instagram"},
		{"ig_", "Instagram"},
		{"camera", "Camera"},
		{"dcim", "Camera"},
		{"pxl_", "Camera"},
		{"img_", "Camera"},
		{"vid_", "Camera"},
	}

	for _, pm := range patterns {
		if strings.Contains(lower, pm.Pattern) {
			return pm.Source
		}
	}

	return "Other_Imports"
}

func GetCreationTime(path string) (time.Time, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}

	if winAttr, ok := fileInfo.Sys().(*syscall.Win32FileAttributeData); ok {
		t := time.Unix(0, winAttr.CreationTime.Nanoseconds())
		return t, nil
	}
	return fileInfo.ModTime(), nil
}
