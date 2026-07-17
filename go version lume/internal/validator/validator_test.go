package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSystemDir_WindowsSystem(t *testing.T) {
	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		t.Skip("SystemRoot ortam değişkeni bulunamadı (Windows dışı ortam)")
	}
	if !IsSystemDir(sysRoot) {
		t.Errorf("IsSystemDir(%q) = false; Windows sistem dizini korumalı olmalıydı", sysRoot)
	}
}

func TestIsSystemDir_WindowsSubDir(t *testing.T) {
	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		t.Skip("SystemRoot ortam değişkeni bulunamadı")
	}
	subDir := filepath.Join(sysRoot, "System32")
	if !IsSystemDir(subDir) {
		t.Errorf("IsSystemDir(%q) = false; sistem alt dizini korumalı olmalıydı", subDir)
	}
}

func TestIsSystemDir_UserDir(t *testing.T) {
	userDir := os.Getenv("USERPROFILE")
	if userDir == "" {
		t.Skip("USERPROFILE ortam değişkeni bulunamadı")
	}
	testDir := filepath.Join(userDir, "Desktop")
	if IsSystemDir(testDir) {
		t.Errorf("IsSystemDir(%q) = true; kullanıcı dizini korumalı olmamalıydı", testDir)
	}
}

func TestIsSystemDir_ProgramFiles(t *testing.T) {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		t.Skip("ProgramFiles ortam değişkeni bulunamadı")
	}
	if !IsSystemDir(pf) {
		t.Errorf("IsSystemDir(%q) = false; Program Files korumalı olmalıydı", pf)
	}
}

func TestIsNested_TargetInsideSource(t *testing.T) {
	src := `C:\Users\Test\Photos`
	dst := `C:\Users\Test\Photos\Archive`
	if !IsNested(src, dst) {
		t.Errorf("IsNested(%q, %q) = false; hedef kaynak içinde, iç içe olmalıydı", src, dst)
	}
}

func TestIsNested_SourceInsideTarget(t *testing.T) {
	src := `C:\Users\Test\Archive\SubFolder`
	dst := `C:\Users\Test\Archive`
	if !IsNested(src, dst) {
		t.Errorf("IsNested(%q, %q) = false; kaynak hedef içinde, iç içe olmalıydı", src, dst)
	}
}

func TestIsNested_SeparateDirs(t *testing.T) {
	src := `C:\Users\Test\Photos`
	dst := `C:\Users\Test\Archive`
	if IsNested(src, dst) {
		t.Errorf("IsNested(%q, %q) = true; bağımsız dizinler iç içe olmamalıydı", src, dst)
	}
}

func TestIsPathSafe_Traversal(t *testing.T) {
	if IsPathSafe(`C:\Users\..\Windows\System32\evil.exe`) {
		t.Error("IsPathSafe path traversal içeren yolu güvenli olarak kabul etmemeli")
	}
}

func TestIsPathSafe_ReservedName(t *testing.T) {
	reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "LPT1"}
	for _, name := range reserved {
		path := `C:\Users\Test\` + name + ".jpg"
		if IsPathSafe(path) {
			t.Errorf("IsPathSafe(%q) = true; ayrılmış isim reddedilmeliydi", path)
		}
	}
}

func TestCheckWritability_TempDir(t *testing.T) {
	tmpDir := t.TempDir()
	if err := CheckWritability(tmpDir); err != nil {
		t.Errorf("CheckWritability(%q) = %v; geçici dizin yazılabilir olmalıydı", tmpDir, err)
	}
}
