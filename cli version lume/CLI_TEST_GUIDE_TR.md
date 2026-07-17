# Lume CLI (Lume_LITE.exe) PowerShell Test Kılavuzu

Bu kılavuz, Lume CLI (`Lume_LITE.exe`) uygulamasının tüm özelliklerini, parametrelerini ve hata koruma kurallarını masaüstünüzdeki `peashot` ve `sunflow` klasörleri üzerinden test edebilmeniz için hazırlanmıştır.

---

## Hazırlık

- **EXE Konumu:** `C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe`
- **Test Kaynak Klasörü (Masaüstü):** `C:\Users\Umut\Desktop\peashot`
- **Test Hedef Klasörü (Masaüstü):** `C:\Users\Umut\Desktop\sunflow`

---

## Test Komutları

### 1. Yardım Menüsü Testi

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" --help
```

### 2. Bilinmeyen / Geçersiz Parametre Uyarısı Testi

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" --hata-parametre
```

### 3. Standart Simülasyon Testi (ModTime Dry-Run)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --dry-run
```

### 4. EXIF Tarihine Göre Arşivleme Simülasyonu (EXIF Dry-Run)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --exif --dry-run
```

### 5. Yeniden Adlandırma Simülasyonu (Rename Dry-Run)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --rename --dry-run
```

### 6. Tam Kapsamlı Simülasyon (EXIF + Rename + Dry-Run)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --exif --rename --dry-run
```

---

## Hata ve Güvenlik Korumaları Testleri

### 7. Aynı Klasör Koruması Testi (Sonsuz Döngü Engeli)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\peashot" --dry-run
```

### 8. İç İçe Klasör Koruması Testi (Nested Folder Prevention)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\peashot\hedef_alt_klasor" --dry-run
```

### 9. Olmayan Kaynak Dizin Koruması Testi

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\olmayan_klasor_yolu" "C:\Users\Umut\Desktop\sunflow" --dry-run
```

---

## Gerçek Aktarım Testi (Diske Yazma)

### 10. Canlı Kopyalama ve Yeniden Adlandırma (Gerçek Arşivleme)

```powershell
& "C:\Users\Umut\Downloads\lume-app-main\cli version lume\Lume_LITE.exe" "C:\Users\Umut\Desktop\peashot" "C:\Users\Umut\Desktop\sunflow" --exif --rename
```

---

## Temizlik

Oluşturulan gerçek arşiv dosyalarını silip hedef klasörü sıfırlamak isterseniz kaynak `peashot` klasörünüz güvenle korunacaktır.

```powershell
Remove-Item -Recurse -Force "C:\Users\Umut\Desktop\sunflow\*"
```
