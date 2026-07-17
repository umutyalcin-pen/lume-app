package metadata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetFileHash_Length(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.bin")
	os.WriteFile(f, []byte("lume test data"), 0644)

	hash, err := GetFileHash(f)
	if err != nil {
		t.Fatalf("GetFileHash hata döndü: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("SHA-256 hash uzunluğu = %d; beklenen 64", len(hash))
	}
}

func TestGetFileHash_Deterministic(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "same.bin")
	os.WriteFile(f, []byte("deterministic content"), 0644)

	h1, _ := GetFileHash(f)
	h2, _ := GetFileHash(f)
	if h1 != h2 {
		t.Errorf("Aynı dosya farklı hash üretti: %q vs %q", h1, h2)
	}
}

func TestGetFileHash_Different(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.bin")
	f2 := filepath.Join(tmpDir, "b.bin")
	os.WriteFile(f1, []byte("content A"), 0644)
	os.WriteFile(f2, []byte("content B"), 0644)

	h1, _ := GetFileHash(f1)
	h2, _ := GetFileHash(f2)
	if h1 == h2 {
		t.Error("Farklı içerikli dosyalar aynı hash üretmemeli")
	}
}

func TestGetFileHash_NonExistent(t *testing.T) {
	_, err := GetFileHash("/nonexistent/file.bin")
	if err == nil {
		t.Error("Var olmayan dosya için hata dönmeliydi")
	}
}

func TestDetectSource_WhatsApp(t *testing.T) {
	tests := []string{"IMG-20231020-WA0001.jpg", "whatsapp_photo.png"}
	for _, name := range tests {
		src := DetectSource(name)
		if src != "WhatsApp" {
			t.Errorf("DetectSource(%q) = %q; beklenen WhatsApp", name, src)
		}
	}
}

func TestDetectSource_Telegram(t *testing.T) {
	src := DetectSource("telegram_photo_2023.jpg")
	if src != "Telegram" {
		t.Errorf("DetectSource = %q; beklenen Telegram", src)
	}
}

func TestDetectSource_Screenshot(t *testing.T) {
	tests := map[string]string{
		"Screenshot_2023.png": "Screenshots",
		"ekran_goruntusu.jpg": "Screenshots",
	}
	for name, expected := range tests {
		src := DetectSource(name)
		if src != expected {
			t.Errorf("DetectSource(%q) = %q; beklenen %q", name, src, expected)
		}
	}
}

func TestDetectSource_Camera(t *testing.T) {
	patterns := []string{"IMG_20231020.jpg", "PXL_20231020.jpg", "VID_20231020.mp4", "DCIM_photo.jpg"}
	for _, name := range patterns {
		src := DetectSource(name)
		if src != "Camera" {
			t.Errorf("DetectSource(%q) = %q; beklenen Camera", name, src)
		}
	}
}

func TestDetectSource_Unknown(t *testing.T) {
	src := DetectSource("random_file_12345.jpg")
	if src != "Other_Imports" {
		t.Errorf("DetectSource bilinmeyen dosya için %q döndü; beklenen Other_Imports", src)
	}
}

func TestGetFileInfo_UnsupportedExt(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "document.pdf")
	os.WriteFile(f, []byte("pdf content"), 0644)

	_, err := GetFileInfo(f)
	if err == nil {
		t.Error("Desteklenmeyen uzantı (.pdf) için hata dönmeliydi")
	}
	if err != nil && !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("Hata mesajı 'unsupported' içermeli; alınan: %v", err)
	}
}
