# 3C — İş ortaklığı ve avantaj yönetimi

Tarih: 21 Ekim 2026

## Tamamlanan

Proje sahibi 3B ve özel yönetici hesabını kabul edip plana uygun devam edilmesini istedi. İş ortaklığı, işletme ve avantaj ekranları mevcut HTML tasarımından React'e taşındı ve PostgreSQL'e bağlandı. Backend, ekranlar ve bağımsız kabul testleri GPT‑6.1 Sol alt işleriyle yürütüldü; bütünleştirme ve gerçek tarayıcı kabulü ana ajan tarafından tamamlandı.

- Yöneticiye özel işletme ekleme, görüntüleme ve düzenleme; iş ortaklığı durumu, görüşme tarihi, iletişim ve notlar.
- Gruplu avantaj formu; eksik bilgileri taslakta saklama, yalnızca seçili yöntemin bilgilerini kaydetme; kod/bağlantı/QR/kart alanları.
- Koşulların açıkça onaylanmasıyla yayın; tarih ve işletme durumu denetimi; gerekçeli askıya alma; yeniden yayın için yeniden inceleme.
- Aktif iş ortaklığını kapatınca bağlı yayımlanmış avantajları aynı işlemde askıya alma; yeniden açılınca kendiliğinden yayımlamama.
- Sunucuda kayıt sürümü kontrolü; tekrar gönderimde kopya oluşmaması; çakışmada form içeriğinin korunması. Kaydedilmemiş koşullarla yayın onayı verilemez.
- Arama/kategori/durum filtreleri; yayın, başlangıç ve bitiş tarihleri; Türkçe durumlar ve askı nedenleri; boş/yüklenme/hata/yeniden deneme ekranları.
- 12 bölümlü yönetici menüsü; masaüstünde dar ray, fare/klavye ile genişleme, sabitleme; mobilde açma/kapatma ve odak yönetimi.
- Mevcut hesap onay/rol akışları, kullanıcı adıyla özel yönetici girişi ve öğrenci kuralları korunur. Yönetici bağlantısıyla girişe gelindiğinde hedef bölüme dönülür.
- Şema 5, SQL sorguları, OpenAPI ve üretilen tipler; önemli yönetim değişiklikleriyle birlikte atomik işlem günlüğü.

HTML prototipi dosyaları değiştirilmedi. Görseller mevcut yerel varlıklardan seçilir. Yeni framework, veritabanı, hizmet veya runtime bağımlılığı eklenmedi.

## Doğrulama

Testler ayrı `_test` veritabanında her çalıştırmaya özel şemalarda ve sentetik hesaplarla yürütüldü. Gerçek sponsor, öğrenci veya geliştirme hesabı değiştirilmedi. Tarayıcı testinin kendi sunucusu ve şeması sonunda kapatıldı/temizlendi; geliştirme servisleri ayrı tutuldu.

| Kontrol | Sonuç |
| --- | --- |
| Go birim testleri ve vet | Geçti; İstanbul gece yarısı ve bitiş gününün dahil olması ayrıca doğrulandı |
| Migration entegrasyonu | Temiz kurulum, güncelleme, veri koruma ve tekrar çalıştırma geçti; bu çalıştırmada önceki Windows engeli görülmedi |
| Önceki kimlik/erişim entegrasyonu | 18 ana senaryo geçti |
| Yönetim API entegrasyonu | 5 ana senaryo; yetki/CSRF, tarih/yöntem doğrulaması, yayın önkoşulları, tekrar gönderim, eşzamanlı düzenleme, iş ortaklığı kapatma/yayın yarışı, audit hatasında tüm işlemin geri alınması geçti |
| TypeScript, ESLint, Prettier ve web derlemesi | Geçti |
| Gerçek API ile Chrome | İşletme oluşturma, kısmi taslak, düzenleme, yayın/askı/yeniden yayın, iş ortaklığı kapanışı, çakışmada form koruma, arama ve ağ hatasından toparlanma geçti |
| Menü ve mobil | 12 bölüm erişimi, dar/geniş masaüstü menüsü, içerik konumunun sabit kalması, mobil seçim/Escape, 390/320 px taşma ve azaltılmış hareket geçti |
| Hesap regresyonu | E-postasız özel yönetici ve doğrulanmış/onaylanmış test öğrencisiyle tarayıcı girişi/çıkışı; öğrencinin yönetim ekranına erişememesi geçti |

Tarayıcı kontrolü `internal/management/browser_integration_test.go` üzerinden gerçek HTTP API ve `dist` arayüzüyle çalışır; tarayıcı yanıtları mock değildir. SMTP doğrulama/sıfırlama akışının tamamı bu teslimin tarayıcı testinde tekrar edilmedi; önceki rapor ve güncel Go kimlik/SMTP testleri ayrı kanıtlardır. İlk tarayıcı denemesinde eksik taslağın HTML zorunlu alanına takılması bulundu, düzeltildi ve yeniden test edildi.

## İnceleme

[Yerel yönetici girişi](http://127.0.0.1:5173/#login/admin) üzerinden mevcut hesabınla giriş yap. İşletme Detayları → işletme ekle → Avantaj Yönetimi → taslak kaydet → koşulları incele ve yayımla akışı hazırdır. Detaylı komutlar ve test tekrarı [README](../README.md) içindedir.

Yerel ortamda kayıt eklemek gerçek geliştirme veritabanını değiştirir. Deneme içeriklerini test olarak adlandır. Test otomasyonunun kayıtları ise yalnızca ayrı test şemasında yaşar.

## Sınırlar ve sıradaki teslim

- **3D:** Öğrencinin gerçek kataloğu, kalıcı avantaj detay adresleri ve hesap bazlı favoriler henüz tamamlanmadı. Yönetimde yayımlamak, öğrenci kataloğunun tamamlandığı anlamına gelmez.
- **3E:** Yönetici özetleri, analitik, işlem günlüğü ekranı ve platform ayarları henüz tamamlanmadı; menüde açıklamalı durum gösterilir. İşlem günlüğünün sunucuda kaydedilmesi ile ekranının hazır olması ayrıdır.
- Görsel yükleme yoktur. Avantajın bağlı işletmesi sonradan değişmez; farklı işletme için ayrı kayıt gerekir. Listeler filtrelenmiş 1000 kayıtla sınırlıdır ve sınırda açık uyarı gösterir; sınırsız tablo/paginasyon altyapısı kurulmadı.
- QR/kart içeriği saklanır; işletmede kullanım doğrulaması veya tek kullanımlık hak üretilmez. Sahte kullanım geçmişi/istatistik yoktur.
- Gerçek sponsor koşulları, okul SMTP teslimatı, üretim hizmetleri, pilot ve yayın onayı hâlâ gereklidir. Genel yayına alma yapılmadı.

3C incelemeye hazırdır. Sonraki kabul noktası, proje sahibinin değerlendirmesinin ardından 3D öğrenci deneyimi ve avantaj detaylarıdır.
