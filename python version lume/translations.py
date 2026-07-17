TRANSLATIONS = {
    "en": {
        "file_list": "📋 File List",
        "target_folder_tag": "Target Folder",
        "not_selected": "Not Selected",
        "select": "Select",
        "supports": "Supports: JPG, PNG, WEBP, HEIC, TIFF, MP4, MOV",
        "start": "▶️ Start Organizing",
        "files": "files",
        "drop_main": "📁\nDrag & Drop Files Here",
        "drop_sub": "Images & Videos supported",
        "err_unsupported": "❌ File format not supported.",
        "err_duplicates": "⚠️ Files already in list.",
        "err_none": "⚠️ No new files found.",
        "err_security": "🔒 Some files blocked for security.",
        "err_invalid_folder": "❌ Invalid or unsafe folder selected.",
        "success_added": "✅ {count} files added (Some skipped)",
        "warn_select_folder": "Please select a target folder first.",
        "warn_no_files": "No files to organize.",
        "warn_file_limit": f"⚠️ Maximum 10,000 files allowed.",
        "status_organizing": "Organizing...",
        "status_archived": "✅ {count} files archived successfully!",
        "ready": "Ready",
        "processing": "{percentage}% - {current}/{total} files processed",
        "header_file": "File",
        "header_date": "Date",
        "header_device": "Device",
        "header_path": "New Path",
        "info_complete": "{count} files archived successfully.",
        "lang_name": "EN"
    },
    "tr": {
        "file_list": "📋 Dosya Listesi",
        "target_folder_tag": "Hedef Klasör",
        "not_selected": "Seçilmedi",
        "select": "Seç",
        "supports": "Desteklenen: JPG, PNG, WEBP, HEIC, TIFF, MP4, MOV",
        "start": "▶️ Düzenlemeyi Başlat",
        "files": "dosya",
        "drop_main": "📁\nDosyaları Buraya Sürükleyin",
        "drop_sub": "Görsel & Videolar desteklenir",
        "err_unsupported": "❌ Dosya formatı desteklenmiyor.",
        "err_duplicates": "⚠️ Dosyalar zaten listede.",
        "err_none": "⚠️ Yeni dosya bulunamadı.",
        "err_security": "🔒 Bazı dosyalar güvenlik nedeniyle engellendi.",
        "err_invalid_folder": "❌ Geçersiz veya güvenli olmayan klasör.",
        "success_added": "✅ {count} dosya eklendi (Bazıları atlandı)",
        "warn_select_folder": "Lütfen önce bir hedef klasör seçin.",
        "warn_no_files": "Düzenlenecek dosya yok.",
        "warn_file_limit": "⚠️ Maksimum 10.000 dosya eklenebilir.",
        "status_organizing": "Düzenleniyor...",
        "status_archived": "✅ {count} dosya başarıyla arşivlendi!",
        "ready": "Hazır",
        "processing": "%{percentage} - {current}/{total} dosya işlendi",
        "header_file": "Dosya",
        "header_date": "Tarih",
        "header_device": "Cihaz",
        "header_path": "Yeni Yol",
        "info_complete": "{count} dosya başarıyla arşivlendi.",
        "lang_name": "TR"
    }
}

def get_text(lang, key, **kwargs):

    lang_batch = TRANSLATIONS.get(lang, TRANSLATIONS["en"])
    text = lang_batch.get(key, key)

    if kwargs:
        try:
            return text.format(**kwargs)
        except Exception:
            return text
    return text
