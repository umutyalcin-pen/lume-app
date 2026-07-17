# Lume CLI (Lume_LITE.exe) PowerShell Test Guide

This guide helps you test all Lume CLI (`Lume_LITE.exe`) features, parameters, and error-protection rules by using the `peashot` and `sunflow` folders on your Desktop.

---

## Preparation

- **EXE Location:** `C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe`
- **Test Source Folder (Desktop):** `C:\Users\Umut\Desktop\peashot`
- **Test Target Folder (Desktop):** `C:\Users\Umut\Desktop\sunflow`

---

## Test Commands

### 1. Help Menu Test

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" --help
```

### 2. Unknown / Invalid Parameter Warning Test

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" --hata-parametre
```

### 3. Standard Simulation Test (ModTime Dry-Run)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --dry-run
```

### 4. EXIF-Based Archive Simulation (EXIF Dry-Run)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --exif --dry-run
```

### 5. Rename Simulation (Rename Dry-Run)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --rename --dry-run
```

### 6. Full Simulation (EXIF + Rename + Dry-Run)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --exif --rename --dry-run
```

---

## Error and Safety Protection Tests

### 7. Same Folder Protection Test (Infinite Loop Prevention)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\peashot" --dry-run
```

### 8. Nested Folder Protection Test

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\peashot\hedef_alt_klasor" --dry-run
```

### 9. Missing Source Directory Protection Test

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\olmayan_klasor_yolu" "C:\Users\Umut\Desktop\sunflow" --dry-run
```

---

## Real Transfer Test (Writing to Disk)

### 10. Live Copying and Renaming (Real Archiving)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --exif --rename
```

---

## Cleanup

If you want to delete the generated archive files and reset the target folder, your source `peashot` folder will remain safely preserved.

```powershell
Remove-Item -Recurse -Force "C:\Users\Umut\Desktop\sunflow\*"
```
