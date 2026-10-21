# 3A Proje ve veri temeli teslimi

Tarih: 21 Ekim 2026
Durum: Yerel uygulama ve kontroller tamamlandı; proje sahibi incelemesine sunuldu.

## Tamamlanan

- Go/Chi API; `/api/health` ve şema sürümünü de kontrol eden `/api/ready`.
- PostgreSQL 18 üzerinde iki migration, sqlc sorgusu ve geliştirmeye özel tekrar çalıştırılabilir demo seed.
- Ayrı geliştirme/test veritabanları ve süper kullanıcı olmayan yerel uygulama hesabı. Bağlantı bilgileri Git dışındaki `.env` dosyasında.
- OpenAPI'den Go tipleri ve TypeScript şeması; tipli istemci kullanan React bağlantı ekranı.
- HTML prototipinin öğrenci ve yönetici arayüzleri korunarak çalışır bırakıldı.
- PowerShell çalışma komutları, kurulum/test belgeleri, MIT lisansı ve GitHub CI tanımı.

## Yerel doğrulama

| Kontrol | Sonuç |
| --- | --- |
| Go birim testleri | Geçti; yapılandırma, hata yanıtında sır sızmaması, sağlık/hazırlık ayrımı |
| PostgreSQL entegrasyon testi | Geçti; sıfır kurulum, 1 → 2 yükseltme, mevcut kaydın korunması ve tekrar migration |
| Veri kısıtları | Geçti; hatalı tarih, incelemesiz yayın, uygunsuz URL, mükerrer/yetim favori reddi ve hesap silme ilişkisi |
| Demo seed | İki çalıştırmada aynı işletme ve taslak; hesap oluşturulmaz |
| Go vet ve derleme | Geçti |
| TypeScript, ESLint ve Vite derlemesi | Geçti |
| Yeni ekran tarayıcı kontrolü | Windows Chrome; canlı API/DB, bağlantı hatası/yeniden deneme, 320/390/1440 px taşma yok, çalışma zamanı hatası yok |
| HTML prototipi regresyonu | 38 öğrenci + 55 yönetici kontrolü geçti; mobil/masaüstü menü ve dosyadan açılış korundu |

Komutlar [README](../README.md) içindedir. Tarayıcı ekran görüntüleri ve raporlar Git dışındaki `tests/artifacts/` dizinindedir. Bu kontroller henüz geliştirilmemiş kimlik doğrulamanın güvenli olduğunu kanıtlamaz.

## Eksikler ve sonraki teslim

3B gerçek kayıt/giriş, e-posta doğrulaması ve yetkilendirme henüz uygulanmadı. Yönetim ve öğrenci ekranları hâlâ HTML demosudur; PostgreSQL verilerine bağlanmaları 3C/3D teslimleridir. Şema tek başına uygulama erişim güvenliği sağlamaz. Üretim dağıtımı yapılmadı.

3B öncesinde gerçek okul e-posta alan adı ve öğrenci uygunluk/onay kuralı gereklidir. Üretim e-posta ve barındırma hizmetleri ayrıca belirlenecektir.

## GitHub durumu

Yerel `main` dalında ilk commit oluşturuldu ve `origin` adresi `https://github.com/momerefe/Kabine101.git` olarak ayarlandı. `.env`, yerel araçlar ve test çıktıları Git dışında bırakıldı; gönderilecek dosyalarda yerel bağlantı parolalarının bulunmadığı kontrol edildi.

Gönderim denemesi Git kimlik doğrulaması bulunmadığı için başarısız oldu. Bağlı GitHub uygulamasının hesabı `celikbros`; depo için yalnızca okuma yetkisi var. `momerefe` hesabıyla GitHub bağlantısı veya uygun yazma erişimi gerektiğinden uzaktaki dosyalar, CI çalıştırması ve yönetim ayarları tamamlanmadı. Yerelde geçen kontroller uzaktaki CI çalışmış gibi sunulmaz.
