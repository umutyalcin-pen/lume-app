package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dsoprea/go-exif/v3"
)

const maxExifScanSize = 250 * 1024 * 1024

func GetMediaDate(path string, entry fileEntry, useExif bool) (year, month, day string, t time.Time) {
	t = entry.modTime
	if useExif {
		ext := strings.ToLower(filepath.Ext(path))
		if isExifSupportedExt(ext) && entry.size <= maxExifScanSize {
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("[WARN] EXIF parsing panic for %s: %v\n", path, r)
					}
				}()
				if exifTime, err := ExtractExif(path); err == nil && exifTime != nil {
					currentYear := time.Now().Year()
					if exifTime.Year() >= 1990 && exifTime.Year() <= currentYear+1 {
						t = *exifTime
					}
				}
			}()
		}
	}
	year = fmt.Sprintf("%d", t.Year())
	month = fmt.Sprintf("%02d", t.Month())
	day = fmt.Sprintf("%02d", t.Day())
	return
}

func isExifSupportedExt(ext string) bool {
	mediaType, found := supportedFiles[ext]
	return found && (mediaType == TypeImage || mediaType == TypeRaw)
}

func ExtractExif(path string) (*time.Time, error) {
	rawExif, err := exif.SearchFileAndExtractExif(path)
	if err != nil {

		if errors.Is(err, exif.ErrNoExif) {
			return nil, nil
		}

		return nil, err
	}

	entries, _, err := exif.GetFlatExifData(rawExif, nil)
	if err != nil {

		return nil, err
	}

	var dateOriginal, dateDigitized, dateGeneric string
	for _, entry := range entries {
		switch entry.TagName {
		case "DateTimeOriginal":
			dateOriginal = entry.FormattedFirst
		case "DateTimeDigitized":
			dateDigitized = entry.FormattedFirst
		case "DateTime":
			dateGeneric = entry.FormattedFirst
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
					return &t, nil
				}
			}
		}
	}

	return nil, nil
}
