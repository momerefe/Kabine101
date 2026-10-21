# Yönetici Genel Bakış — 21 Ekim 2026

Proje sahibinin isteğiyle 3E'nin Genel Bakış bölümü öne alındı. Öğrenci kataloğu/favoriler ve diğer tamamlanmamış yönetim modülleri kendi teslimlerinde kalır.

## Tamamlanan

- Dört özet kartı: toplam işletme, aktif avantaj, aktif öğrenci ve yönetici onayı bekleyen başvurular. Sayılar gerçek veritabanındaki tüm ilgili kayıtları kapsar; liste ekranlarının 1000 kayıt sınırından etkilenmez.
- Bekleyen başvurular, avantaj taslakları ve yaklaşan bitişlere hızlı erişim. Avantaj bağlantıları ilgili durum filtresiyle açılır.
- Aktif, yakında başlayacak, taslak, süresi dolmuş ve askıdaki avantajların anlaşılır dağılımı; aktif/görüşülen/sonlandırılmış iş ortaklığı özeti.
- Bugün dahil yedi gün içinde bitecek aktif avantajların bitiş tarihine göre ilk beşi ve toplam sayısı.
- En son altı işletme, avantaj, öğrenci onayı veya rol işlemi. Gerçek denetim kayıtlarından işlem, kayıt adı, yapan kişi ve zaman gösterilir; parola/token olayları, iletişim bilgileri ve kullanım kodları özete taşınmaz.
- Son güncelleme zamanı, elle yenileme ve sayfa ön plandayken 60 saniyelik yenileme. Kayıt değişiklikleri özet önbelleğini geçersiz kılar. Açılış, boş liste ve hata durumları vardır; geçici bağlantı hatasında son veri açık uyarıyla korunur, 401/403 erişim kaybında gizlenir.
- Masaüstünde dört kart/iki sütun; telefonda iki kart/tek içerik sütunu. Mevcut kırmızı vurgu, açık zemin ve navigasyon korunur.

## Sayıların anlamı

Aktif öğrenci; öğrenci rolünde, hesabı aktif, uygunluğu onaylanmış ve e-postası doğrulanmış hesaptır. Onay bekleyen başvurular ile e-posta bekleyenler farklı aşamalardır ve örtüşebilir; bu ayrım ekranda belirtilir. Yayımlanmış avantaj toplamı aktif, yakında başlayacak ve süresi dolmuş kayıtları içerir. Süresi dolmuş kayıtlar aktif sayılmaz. Tarihler Europe/Istanbul gününe göre hesaplanır; bitiş günü dahildir.

`GET /api/admin/overview` yalnızca yönetici oturumuna açıktır ve `no-store` yanıtı verir. Bütün özetler aynı salt okunur PostgreSQL işleminden ve aynı tarih anından gelir. SQL/sqlc ve OpenAPI tipleri kullanıldı. Yeni migration, servis veya bağımlılık eklenmedi.

Son işlemlerde adlar kayıtların güncel adıdır; bu sayfa değiştirilemez tarihsel ad arşivi veya tam işlem günlüğü ekranı değildir. Kaydı silinmiş aktör “Sistem”, silinmiş hedef açıklayıcı boş ad ile gösterilir. Kullanım, gelir, tasarruf, gerçek zamanlı çevrimiçi kullanıcı veya ölçülmemiş performans rakamı üretilmez.

## Kontroller

- TypeScript, ESLint, biçim ve web derlemesi.
- Go entegrasyon senaryoları: anonim/öğrenci erişimi ve geri alınan yönetici rolü, boş yanıt, 1000'i aşan tam sayımlar, İstanbul tarih sınırları, yayın durumları, öğrenci doğrulama aşamaları, liste sınırları/sıralama ve özel bilgilerin özete sızmaması.
- `tests/run_overview_ui_checks.py`: 320–1920 px ve yatay görünüm; toplam 17 yerleşim kontrolü. Dolu/boş veri, uzun metin, ilk yükleme hatası, yenileme hatası, yeniden deneme, yetki kaybı ve durum filtrelerine geçişler.
- Tarayıcı kontrolü proje sahibinin zaten açtığı Vite üzerinden çalışır. Ayrı Chrome test profilinde **tüm API yanıtları sentetiktir**; gerçek hesap veya geliştirme veritabanı değiştirilmez. Backend doğruluğu ayrı Go/PostgreSQL entegrasyon testlerinin konusudur. Fiziksel telefon testi proje sahibinin önceki sürüm gözlemidir; yeni ekran için tarayıcı emülasyonu kullanıldı.

Arayüz kontrolünü tekrar etmek için frontend açıkken:

```powershell
$env:KOC_OVERVIEW_URL = 'http://192.168.1.94:5173'
python tests/run_overview_ui_checks.py
Remove-Item Env:KOC_OVERVIEW_URL
```

Go testleri GitHub Actions'ta ayrı PostgreSQL test veritabanında çalışır. Bu geliştirmede yerel backend/frontend başlatılmadı veya yeniden başlatılmadı. Ekran görüntüleri Git dışında `tests/artifacts/overview/` altındadır.

## İnceleme

Yeni API için mevcut backend terminalinde `Ctrl+C` ardından:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 api -LanAddress 192.168.1.94
```

Açık Vite kaynak değişikliklerini alır; sayfayı yenilemek yeterlidir. Gerekirse frontend'i kendin `powershell -ExecutionPolicy Bypass -File tools/dev.ps1 web -LanAddress 192.168.1.94` ile başlat. Bilgisayar ve telefonda **http://192.168.1.94:5173/#admin/overview** adresini kullan. IP adresi değiştiyse iki komutta da güncel adresi kullan. Mevcut şema sürümü 5 olduğu için migration gerekmiyor.
