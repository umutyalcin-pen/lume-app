package metadata

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dsoprea/go-exif/v3"
)

// SupportedExtensions defines the formats Lume is willing to process.
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

// FileInfo carries the metadata extracted from a file.
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
	MD5      string
}

// GetFileHash calculates the MD5 hash of a file using streaming.
func GetFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GetFileInfo gathers basic file information and extracts EXIF metadata.
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
		Date:     stat.ModTime(), // Fallback
		Device:   "Unknown",
		Source:   DetectSource(filepath.Base(path)),
	}

	// Extract EXIF for images
	isImage := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".heic": true, ".tiff": true}
	if isImage[ext] {
		if exifDate, device, err := ExtractExif(path); err == nil {
			if exifDate != nil {
				info.Date = *exifDate
			}
			if device != "" {
				info.Device = device
			}
		}
	} else {
		// Video or other: Try creation time if available
		if createTime, err := GetCreationTime(path); err == nil {
			info.Date = createTime
		}
	}

	info.Year = fmt.Sprintf("%d", info.Date.Year())
	info.Month = fmt.Sprintf("%02d", info.Date.Month())

	return info, nil
}

// ExtractExif uses go-exif to extract the date and device model.
func ExtractExif(path string) (*time.Time, string, error) {
	rawExif, err := exif.SearchFileAndExtractExif(path)
	if err != nil {
		return nil, "", err
	}

	entries, _, err := exif.GetFlatExifData(rawExif, nil)
	if err != nil {
		return nil, "", err
	}

	var date *time.Time
	var device string

	for _, entry := range entries {
		if entry.TagName == "DateTimeOriginal" {
			// Format is usually "2023:10:20 15:04:05"
			t, err := time.Parse("2006:01:02 15:04:05", entry.FormattedFirst)
			if err == nil {
				date = &t
			}
		} else if entry.TagName == "Model" {
			device = strings.TrimSpace(entry.FormattedFirst)
		}
	}

	return date, device, nil
}

// DetectSource identifies the source based on professional patterns.
func DetectSource(filename string) string {
	lower := strings.ToLower(strings.TrimSpace(filename))
	
	patterns := map[string]string{
		"whatsapp":  "WhatsApp",
		"-wa":       "WhatsApp",
		"telegram":  "Telegram",
		"screenshot": "Screenshots",
		"ekran":      "Screenshots",
		"instagram":  "Instagram",
		"ig_":        "Instagram",
		"camera":     "Camera",
		"dcim":       "Camera",
		"pxl_":       "Camera",
		"img_":       "Camera",
		"vid_":       "Camera",
	}

	for pattern, source := range patterns {
		if strings.Contains(lower, pattern) {
			return source
		}
	}
	
	// Better fallback: avoid anemic folder names
	return "Other_Imports"
}

// GetCreationTime attempts to get the OS-level creation time (Windows specific)
func GetCreationTime(path string) (time.Time, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	// On Windows, sys is *syscall.Win32FileAttributeData
	if winAttr, ok := fileInfo.Sys().(*syscall.Win32FileAttributeData); ok {
		t := time.Unix(0, winAttr.CreationTime.Nanoseconds())
		return t, nil
	}
	return fileInfo.ModTime(), nil
}
