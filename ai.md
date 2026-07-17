# AI Context & Developer Onboarding Guide: lume-app

Bu doküman, **lume-app** deposu üzerinde çalışacak AI modelleri (Claude, GPT, Copilot vb.) ve geliştiriciler için projenin gerçek mimarisini, dizin yapısını ve kurallarını açıklayan kılavuzdur. AI asistanlarının kod üretirken, hata ayıklarken veya öneri sunarken bu bağlamı temel alması gerekmektedir.

> **Not:** Bu dosya, projenin gerçek kod tabanı (Python + Go, masaüstü dosya organizatörü) baz alınarak hazırlanmıştır. Eski taslakta yer alan React/TypeScript/JWT mimarisi bu projeye ait değildir ve kullanılmamalıdır.

---

## 1. Proje Genel Bakışı (Project Overview)

* **Proje Adı:** Lume 🍃 (`lume-app`)
* **Geliştirici:** Umut Yalçın (@umutyalcin-pen)
* **İletişim:** artabqos251@gmail.com
* **Lisans:** MIT
* **Ana Amaç:** Lume, dağınık dosyaları (özellikle fotoğrafları) **EXIF metadata'sına** göre otomatik olarak Yıl/Ay/Gün klasör hiyerarşisine düzenleyen, tamamen yerel çalışan bir masaüstü aracıdır.
* **Temel Özellikler:**
  * EXIF verisiyle otomatik Yıl/Ay/Gün klasörleme
  * Sürükle-bırak (drag & drop) desteği
  * SHA-256 / MD5 hash tabanlı kopya (duplicate) dosya algılama
  * Karanlık / aydınlık tema desteği
  * Format odaklı filtreleme (yalnızca istenen uzantılar: .jpg, .pdf, .mp4 vb.)
* **AI İçin Temel Talimat:** Kod tabanında değişiklik yaparken güvenli kodlama ilkelerine riayet et, dosya sistemi işlemlerinde (taşıma/kopyalama/silme) veri kaybı riskini en aza indir, ve projenin "%100 yerel, ağ bağlantısı yok" ilkesini asla ihlal etme (dış servislere istek atan kod ekleme).

---

## 2. Teknoloji Yığını (Tech Stack)

Proje üç ayrı versiyon olarak geliştirilmiştir; AI bu ayrımı mutlaka gözetmelidir:

| Versiyon | Dil | Klasör | Açıklama |
|---|---|---|---|
| **Python versiyonu** | Python | `python version lume/` | ~10 MB'lık exe, tam GUI |
| **Go versiyonu** | Go | `go version lume/` | Ultra hafif, düşük kaynak tüketimi |
| **CLI versiyonu** | Python/Go tabanlı (`Lume_LITE.exe`) | `cli version lume/` | Komut satırından `KaynakKlasör` ve `HedefKlasör` argümanlarıyla çalışır |

* **Dil Dağılımı (repo geneli):** Python %66.5, Go %33.5
* **TypeScript, React, JWT, Node.js gibi teknolojiler bu projede YOKTUR.**
* **Mimari:** Masaüstü uygulaması / CLI aracı — web veya mobil bileşen mimarisi değildir.

---

## 3. Depo Dizin Haritası (Repository Directory Map)

```text
lume-app/
├── cli version lume/              # CLI (Lume_LITE.exe) kaynak/dağıtım dosyaları
├── cli version lume schreenshots/ # CLI versiyonuna ait ekran görüntüleri
├── go version lume/                # Go ile yazılmış hafif GUI versiyonu
├── go version screenshots/         # Go versiyonuna ait ekran görüntüleri
├── python version lume/            # Python ile yazılmış ana GUI versiyonu
├── python version screenshots/     # Python versiyonuna ait ekran görüntüleri
├── screenshots/                    # Genel/ortak ekran görüntüleri
├── LICENSE                         # MIT Lisansı
├── README.md                       # Proje açıklaması (TR/EN)
└── ai.md                           # Bu doküman (AI bağlam dosyası)
```

> **Önemli:** `src/`, `services/`, `store/`, `hooks/`, `navigation/` gibi klasörler bu projede **mevcut değildir**. AI, kod önerirken veya dosya oluştururken yukarıdaki gerçek klasör yapısını referans almalıdır.

---

## 4. Veri Depolama ve Çalışma Zamanı Davranışı

* `lume_config.json` — Yerel ayarlar (tema, dil, hedef klasör) burada tutulur.
* `lume_app.log` — Yerel işlem günlüğü (log) burada tutulur.
* Hiçbir kişisel veri saklanmaz veya dışarıya iletilmez.

---

## 5. Gizlilik & Güvenlik İlkeleri (AI için bağlayıcı kurallar)

AI, bu projeye kod eklerken/düzenlerken aşağıdaki ilkeleri **kesinlikle ihlal etmemelidir**:

* **%100 Yerel Çalışma:** Uygulama internet bağlantısı kullanmaz. Herhangi bir HTTP isteği, telemetri, analytics SDK'sı veya bulut entegrasyonu **eklenmemelidir**.
* **Telemetri Yok:** Kullanıcı davranışı veya dosya içerikleri hiçbir şekilde toplanmamalı, loglanmamalı (log dosyası yalnızca yerel işlem takibi içindir) veya iletilmemelidir.
* **Dosya Sistemi Güvenliği:** Taşıma/kopyalama/silme işlemleri her zaman geri alınabilir veya en azından loglanabilir şekilde tasarlanmalı; kullanıcı verisi (fotoğraflar) kalıcı olarak kaybolmamalıdır.
* **Kopya Algılama:** Mevcut SHA-256/MD5 tabanlı duplicate detection mantığı korunmalı; yeni özellikler bu mekanizmayı bozmamalıdır.

---

## 6. Kodlama Standartları

### Python versiyonu için
* PEP 8 stiline uyulmalı.
* Fonksiyon ve değişken isimleri `snake_case`.
* Dosya işlemleri (`shutil`, `os`, EXIF okuma vb.) her zaman `try-except` içinde ele alınmalı; kullanıcıya anlaşılır hata mesajı, log dosyasına ise detaylı hata basılmalı.
* GUI ve iş mantığı (dosya tarama/taşıma/EXIF okuma) mümkün olduğunca ayrı fonksiyon/modüllere bölünmeli.

### Go versiyonu için
* Standart Go konvansiyonları (`gofmt`, `camelCase` fonksiyon/değişken isimleri, exported isimler için `PascalCase`).
* Hata yönetimi Go'nun `error` dönüş idiomu ile yapılmalı (panic'ten kaçınılmalı, kullanıcıya güvenilir hata mesajı dönülmeli).

### Genel (her iki versiyon için)
* Karmaşık algoritmalar (EXIF ayrıştırma, kopya algılama, klasörleme mantığı) için kısa ve öz Türkçe yorum satırları kullanılmalı.
* Placeholder/eksik kod bloğu bırakılmamalı; üretilen kod işlevsel ve eksiksiz olmalı.
* Yeni bir dosya formatı/uzantı desteği eklenirken mevcut "format odaklı filtreleme" mantığına entegre edilmeli, ayrı bir sistem kurulmamalı.

---

## 7. Kritik İş Akışları (Key Workflows)

### Dosya Tarama ve Klasörleme
1. Kullanıcı kaynak ve hedef klasörü seçer (GUI'de sürükle-bırak veya CLI'de argüman olarak).
2. Uygulama kaynak klasördeki dosyaları tarar, desteklenen formatlara göre filtreler.
3. Fotoğraflar için EXIF verisi okunur (çekim tarihi); EXIF yoksa dosya sistemi tarihi fallback olarak kullanılmalıdır (bu davranış AI tarafından yeni kod eklenirken korunmalı).
4. SHA-256/MD5 hash hesaplanarak kopya dosyalar tespit edilir.
5. Dosyalar Yıl/Ay/Gün klasör hiyerarşisine taşınır/kopyalanır.
6. İşlem sonunda kullanıcıya hangi dosyanın nereye taşındığına dair detaylı bir liste sunulur ve `lume_app.log`'a yazılır.

### CLI Kullanımı
```powershell
.\Lume_LITE.exe "C:\KaynakKlasor" "C:\HedefKlasor"
```

---

## 8. AI Asistanına Özel Talimatlar

1. **Bağlam Kontrolü:** Hangi versiyona (Python/Go/CLI) kod yazacağını her zaman netleştir; üç versiyonun kod tabanını birbirine karıştırma.
2. **Gerçek Dizin Yapısını Kullan:** Bölüm 3'teki dizin haritası dışında klasör/dosya varsayma.
3. **Ağ Bağlantısı Ekleme:** Hiçbir surette dış servise istek atan, telemetri gönderen veya internet bağlantısı gerektiren kod önerme — bu projenin temel gizlilik ilkesine aykırıdır.
4. **Eksiksiz Kod:** `# ... mevcut kod buraya gelecek ...` gibi placeholder bırakma.
5. **Versiyonlar Arası Tutarlılık:** Python ve Go versiyonlarında aynı özellik isteniyorsa, davranış (klasörleme mantığı, kopya algılama, log formatı) iki dilde de tutarlı olmalı.
6. **Güvenlik Taraması Anlayışı:** Proje sahibi düzenli güvenlik taraması yaptığını belirtmiştir; yeni eklenen dosya işleme kodları (özellikle yol/path işlemleri) path traversal ve benzeri dosya sistemi güvenlik açıklarına karşı dikkatli yazılmalıdır.

---

## 9. Proje Durumu

* **Son sürüm:** v2.2 (17 Temmuz 2026)
* **Geliştirme notu:** Uygulamada kararlılık, yüksek performans (klasör oluşturma önbelleği, boyut bazlı erken elenen kopya tespiti), asenkron kilit senkronizasyonu (data race çözümleri) ve Yüksek DPI (High-DPI aware) netlik ayarları entegre edilmiştir. Güvenlik açıkları (path traversal, symlink/junction koruması) giderilmiştir. Geliştirme notları doğrultusunda güncel/iteratif kod değişiklikleri devam edebilir.
