package organizer

import (
	"lume-go/internal/metadata"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CON", "CON_safe"},
		{"PRN", "PRN_safe"},
		{"AUX", "AUX_safe"},
		{"NUL", "NUL_safe"},
		{"COM1", "COM1_safe"},
		{"COM2", "COM2_safe"},
		{"COM9", "COM9_safe"},
		{"LPT1", "LPT1_safe"},
		{"LPT5", "LPT5_safe"},
		{"LPT9", "LPT9_safe"},
		{"COM10", "COM10"},
		{"my<file>", "my_file_"},
		{"folder/path", "folder_path"},
		{"file:name", "file_name"},
		{"  trim  ", "trim"},
		{"", "Unknown"},
		{".", "Unknown"},
		{"..", "Unknown"},
		{"a" + strings.Repeat("b", 150), "a" + strings.Repeat("b", 99)},
	}
	for _, tt := range tests {
		got := SanitizeFolderName(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeFolderName(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestCopyFile_ContentPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.dat")
	dst := filepath.Join(tmpDir, "dest.dat")
	content := []byte("Lume arşiv bütünlük testi - içerik korunmalı")
	os.WriteFile(src, content, 0644)

	if _, err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile hata döndü: %v", err)
	}

	dstContent, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("Hedef dosya okunamadı: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Error("Kopyalanan dosyanın içeriği orijinal ile eşleşmiyor")
	}
}

func TestCopyFile_HashIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "hash_src.bin")
	dst := filepath.Join(tmpDir, "hash_dst.bin")
	os.WriteFile(src, []byte("SHA-256 bütünlük testi verileri 1234567890"), 0644)

	if _, err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile hata döndü: %v", err)
	}

	srcHash, _ := metadata.GetFileHash(src)
	dstHash, _ := metadata.GetFileHash(dst)
	if srcHash != dstHash {
		t.Errorf("Kaynak hash (%s) != Hedef hash (%s); bütünlük bozulmuş", srcHash, dstHash)
	}
}

func TestCopyFile_PermissionsPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "perm_src.dat")
	dst := filepath.Join(tmpDir, "perm_dst.dat")
	os.WriteFile(src, []byte("permission test"), 0644)

	if _, err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile hata döndü: %v", err)
	}

	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("Dosya izinleri korunmadı: kaynak=%v, hedef=%v", srcInfo.Mode(), dstInfo.Mode())
	}
}

func TestIsDuplicate_IdenticalFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "dup1.bin")
	f2 := filepath.Join(tmpDir, "dup2.bin")
	content := []byte("kopya algılama test verisi")
	os.WriteFile(f1, content, 0644)
	os.WriteFile(f2, content, 0644)

	isDup, err := IsDuplicate(f1, f2)
	if err != nil {
		t.Fatalf("IsDuplicate hata döndü: %v", err)
	}
	if !isDup {
		t.Error("Aynı içerikli dosyalar kopya olarak tespit edilmeliydi")
	}
}

func TestIsDuplicate_DifferentFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "diff1.bin")
	f2 := filepath.Join(tmpDir, "diff2.bin")
	os.WriteFile(f1, []byte("içerik A"), 0644)
	os.WriteFile(f2, []byte("içerik B farklı uzunluk"), 0644)

	isDup, err := IsDuplicate(f1, f2)
	if err != nil {
		t.Fatalf("IsDuplicate hata döndü: %v", err)
	}
	if isDup {
		t.Error("Farklı içerikli dosyalar kopya olarak tespit edilmemeli")
	}
}

func TestResolveConflict_FirstConflict(t *testing.T) {
	tmpDir := t.TempDir()
	original := filepath.Join(tmpDir, "photo.jpg")
	os.WriteFile(original, []byte("existing"), 0644)

	resolved, err := ResolveConflict(original, NewState())
	if err != nil {
		t.Fatalf("ResolveConflict hata döndü: %v", err)
	}
	expected := filepath.Join(tmpDir, "photo_1.jpg")
	if resolved != expected {
		t.Errorf("ResolveConflict = %q; beklenen %q", resolved, expected)
	}
}

func TestResolveConflict_MultipleConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	base := filepath.Join(tmpDir, "photo.jpg")
	os.WriteFile(base, []byte("existing"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "photo_1.jpg"), []byte("conflict1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "photo_2.jpg"), []byte("conflict2"), 0644)

	resolved, err := ResolveConflict(base, NewState())
	if err != nil {
		t.Fatalf("ResolveConflict hata döndü: %v", err)
	}
	expected := filepath.Join(tmpDir, "photo_3.jpg")
	if resolved != expected {
		t.Errorf("ResolveConflict = %q; beklenen %q", resolved, expected)
	}
}

func TestAtomicCopy_Integrity(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "copy_src.bin")
	dst := filepath.Join(tmpDir, "subdir", "copy_dst.bin")
	content := []byte("atomik kopyalama bütünlük test verisi")
	os.WriteFile(src, content, 0644)
	os.MkdirAll(filepath.Dir(dst), 0755)

	srcHash, _ := metadata.GetFileHash(src)

	if err := AtomicCopy(src, dst); err != nil {
		t.Fatalf("AtomicCopy hata döndü: %v", err)
	}

	dstHash, err := metadata.GetFileHash(dst)
	if err != nil {
		t.Fatalf("Hedef hash hesaplanamadı: %v", err)
	}
	if srcHash != dstHash {
		t.Error("AtomicCopy sonrası hash uyuşmazlığı — bütünlük bozulmuş")
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Error("Kaynak dosya kopyalamadan sonra silinmiş!")
	}
}

func TestArchiveFile_FullArchiveFlow(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	dstDir := filepath.Join(tmpDir, "archive")
	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(dstDir, 0755)

	srcFile := filepath.Join(srcDir, "IMG_20231020.jpg")
	os.WriteFile(srcFile, []byte("fake jpg content for archive test"), 0644)

	info, err := metadata.GetFileInfo(srcFile)
	if err != nil {
		t.Fatalf("GetFileInfo hata döndü: %v", err)
	}

	if err := ArchiveFile(info, dstDir); err != nil {
		t.Fatalf("ArchiveFile hata döndü: %v", err)
	}

	archivePath := filepath.Join(dstDir, info.Year, info.Month)
	entries, err := os.ReadDir(archivePath)
	if err != nil {

		found := false
		subDirs, _ := os.ReadDir(filepath.Join(dstDir, info.Year, info.Month))
		for _, sd := range subDirs {
			if sd.IsDir() {
				subEntries, _ := os.ReadDir(filepath.Join(dstDir, info.Year, info.Month, sd.Name()))
				for _, se := range subEntries {
					if se.Name() == "IMG_20231020.jpg" {
						found = true
					}
				}
			}
		}
		if !found {
			t.Error("Arşivlenen dosya hedef yapısında bulunamadı")
		}
	} else {
		found := false
		for _, e := range entries {
			if e.Name() == "IMG_20231020.jpg" || e.IsDir() {
				found = true
			}
		}
		if !found {
			t.Error("Arşivlenen dosya hedef yapısında bulunamadı")
		}
	}
}

func TestArchiveFile_DuplicateSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	dstDir := filepath.Join(tmpDir, "archive")
	os.MkdirAll(srcDir, 0755)

	content := []byte("duplicate detection test content")
	srcFile := filepath.Join(srcDir, "photo.jpg")
	os.WriteFile(srcFile, content, 0644)

	info, err := metadata.GetFileInfo(srcFile)
	if err != nil {
		t.Fatalf("GetFileInfo hata döndü: %v", err)
	}

	if err := ArchiveFile(info, dstDir); err != nil {
		t.Fatalf("İlk ArchiveFile hata döndü: %v", err)
	}

	os.WriteFile(srcFile, content, 0644)
	info2, _ := metadata.GetFileInfo(srcFile)

	if err := ArchiveFile(info2, dstDir); err != nil {
		t.Fatalf("Kopya ArchiveFile hata döndü: %v", err)
	}
}
