# 3B — Kimlik ve erişim teslimi

Tarih: 21 Ekim 2026

## Sonraki düzenleme — e-postasız özel yönetici

Proje sahibinin talebiyle ayrı kullanıcı adı kimliği eklendi; seçilen hesap yerel veritabanında oluşturuldu. E-posta alanı NULL, e-posta doğrulama kaydı yoktur; sahte adres kullanılmaz. Parola CLI'ın gizli girişinden alınıp Argon2id özetiyle saklandı; kaynak koduna veya bu rapora yazılmadı.

Şema 4 kimlik türlerini veritabanında ayırır. Kullanıcı adı hesabı yalnızca yönetici olabilir. Öğrenciler okul alan adı + e-posta doğrulaması + yönetici onayı ve normal parola kuralına tabidir. Özel yönetici aynı CSRF, deneme sınırı ve sunucu oturumu korumalarını kullanır. E-posta ile sıfırlama özel hesabı hedefleyemez. Bu kimlik web arayüzünden oluşturulamaz.

Yedi yeni güvenlik senaryosu dahil 18 ana PostgreSQL kimlik entegrasyon senaryosu geçti; Go test/vet ve web derlemesi geçti. Tam izole tarayıcı testi bu düzenlemede geçici migration yürütülebilir dosyasının Windows Application Control engeli nedeniyle başlayamadı; aşağıdaki önceki tam tarayıcı sonucu 3B ilk teslimine aittir. Güncel özel hesabın girişi, yönetici erişimi, e-postasız profil, 320 px görünüm, HttpOnly oturumu ve çıkışı çalışan yerel uygulamada Chrome ile ayrıca doğrulandı. Parolanın kaynak dosyalarında bulunmadığı kontrol edildi. Güvenlik ayarları değiştirilmedi.

Aşağıdaki bölümler ilk 3B tesliminin kaydıdır; güncel şema 4 ve özel yönetici kararı bu ekle birlikte okunmalıdır.

## Tamamlanan

Proje sahibinin devam talimatı ve kesinleştirdiği `stu.koc.k12.tr` + e-posta doğrulaması + yönetici onayı kuralı uygulandı. Backend, arayüz ve bağımsız güvenlik testleri GPT‑6.1 Sol alt işleri olarak yürütüldü; API sözleşmesi, SMTP, birleştirme, tarayıcı kontrolleri ve teslim incelemesi ana ajan tarafından tamamlandı.

- PostgreSQL hesap kaydı; okul alan adının sunucuda tam eşleşmeyle denetlenmesi. Açık kayıt her zaman öğrenci rolü oluşturur.
- Ayrı e-posta sahipliği ve öğrenci uygunluğu; ikisinden biri eksikken giriş kapalı.
- Argon2id parola saklama; süreli ve iptal edilebilir sunucu oturumu; giriş/çıkış; tek kullanımlık doğrulama ve sıfırlama tokenları.
- Parola sıfırlamada tüm oturumların iptali; rol değişiminde ilgili hesabın yeniden giriş yapması.
- Yönetici başvuru listesi, onay/ret ve rol değişimi; mevcut parola teyidi, açık onay penceresi ve son erişilebilir yönetici koruması.
- Gizli parola girişiyle çalışan ilk yönetici CLI; mevcut hesabı sessizce yükseltmez, e-posta doğrulamasını atlamaz.
- Gerçek yönetim/hesap olaylarının sunucu işlem günlüğü; parola, token veya doğrulama URL'si loglanmaz.
- Origin/CSRF/JSON kontrolleri, sınırlı istek gövdesi, deneme sınırları ve eşzamanlı Argon2 bellek sınırı.
- Mevcut HTML tasarımıyla uyumlu React kayıt, giriş, yardım bağlantıları, doğrulama/sıfırlama, hesap ve başvuru ekranları.
- Yerel Mailpit SMTP yakalama, SMTP hatasında yeniden gönderme, gerçek gönderim için zorunlu STARTTLS.
- Şema sürümü 3; SQL sorguları ve OpenAPI tipleri; güncellenmiş çalışma/test komutları ve CI.

Kök HTML/CSS/JS prototipi değiştirilmedi. Gerçek uygulamada avantaj kataloğu ve 12 yönetim bölümü henüz sunucuya bağlanmadığı için tamamlanmış gibi gösterilmez.

## Test kaydı

Ortam: Windows, Chrome, PostgreSQL 18; gerçek hesaplar yerine `_test` veritabanında her çalıştırmaya özel şema ve sentetik kullanıcılar. Yerel e-posta ağ dışına çıkmadı. Testlerin açtığı süreçler kapatıldı; test şemaları temizlendi.

| Kontrol | Sonuç |
| --- | --- |
| Go birim testleri, `go vet`, Go derlemesi | Geçti |
| PostgreSQL kimlik/erişim entegrasyonu | 11 ana test senaryosu geçti; yarış koşulları dahil |
| Argon2/token, kesin alan adı, JSON/CSRF/çerez ve yapılandırma testleri | Geçti |
| SMTP testleri | UTF-8 ileti/token aktarımı, header enjeksiyonu reddi, TLS gerekliliği, iptal ve loopback sınırı geçti |
| TypeScript, ESLint, Prettier, Vite derlemesi | Geçti |
| Gerçek SMTP + Chrome hesap akışı | CLI yönetici doğrulaması, öğrenci kayıt/doğrulama/onay/giriş, rol erişimi, sıfırlama, token tekrar reddi ve hata/yeniden deneme geçti |
| Mobil ve masaüstü | 1440/390/320 px giriş ekranları; 320 px yönetim; taşma yok; azaltılmış hareket ve çalışma zamanı hata kontrolü geçti |
| HTML prototipi | Önceki 93 kontrol geçti; çalışma zamanı hatası yok |
| Sistem durumu | Gerçek yerel API/şema 3, 1440/390/320 px, ağ hatası ve yeniden deneme geçti |
| Migration testi, Windows | Test yürütülebilir dosyası Windows Application Control tarafından engellendi; güvenlik ayarı değiştirilmedi. Yeni şema kurulumu ve uygulamanın şema 3 kontrolü tarayıcı testinde geçti. |
| Migration testi, Linux CI | Temiz şema, sürüm yükseltme/veri koruma, kısıtlar ve tekrar çalıştırma geçti. Kimlik entegrasyonu, Go test/vet/build ve web kontrolleri de geçti. [GitHub çalıştırması](https://github.com/momerefe/Kabine101/actions/runs/39636546882). |

Windows engeli yerel `tools/dev.ps1 check` komutunun migration adımında tekrar görülebilir. Tam otomatik kontrol Linux CI'da yürütüldü; işletim sisteminin güvenlik ayarı kapatılmadı. Chrome test yardımcılarında da DevTools port dosyasının yazım sırasındaki kısa dosya kilidine karşı bekleme düzeltildi.

Kimlik entegrasyonu: yanlış alan adı ve istemci rol enjeksiyonu, onay/doğrulama sıraları, ret, öğrenci → yönetici erişim reddi, CSRF/Origin, süresi dolan/iptal edilen oturum, token amacı/tek kullanım, veritabanında token/parola özeti, tüm oturumları kapatan sıfırlama, paralel token tüketimi, iki yöneticinin aynı anda yetki bırakması ve SMTP arızasında hesap varlığını açıklamayan yanıtlar.

## Henüz tamamlanmayanlar

- Gerçek okul e-posta sunucusuna teslimat, üretim SMTP hesabı, HTTPS/ters vekil ve yayın kurulumu. Yerel posta kutusu e-posta sahipliğinin üretimde doğrulandığı anlamına gelmez.
- Canlı ilk yönetici hesabı. Kullanıcının seçtiği ad/adres ve özel parola CLI üzerinden girilecek; hazır ortak parola yok.
- İşletme/avantaj API'leri (3C), gerçek öğrenci kataloğu ve hesap bazlı favoriler (3D), yönetim ekranlarının tamamı (3E).
- Gerçek avantaj kullanımı doğrulaması, native mobil uygulamalar ve genel yayın.
- Üretim için veritabanı yetkilerinin ayrılması, saklama/temizleme planı, yedek/geri dönüş ve pilot kontrolleri.

Deneme sınırları tek API sürecinin belleğindedir; yeniden başlatmada sıfırlanır. Rol işlemleri oturumları iptal eder. Başvuru kararı yalnızca bekleyen başvuruda uygulanır; etkin hesabı kapatma veya reddedilmiş başvuruyu yeniden açma arayüzü bu teslimde yoktur. Bunlar çalışıyor gibi gösterilmez. Çoklu API süreci veya kapsamlı yetki ağacı eklenmedi.

## İnceleme ve sonraki iş

[README](../README.md) üç terminalle çalıştırmayı, ilk yönetici oluşturmayı ve öğrenci test akışını açıklar. Sonraki teslim 3C: iş ortakları, işletme ve avantaj oluşturma/düzenleme/yayın API'leri ve mevcut yönetici formlarının gerçek veriye bağlanması. Gerçek sponsor bilgisi uydurulmaz; geliştirme kayıtları açıkça örnek olarak tutulur.
