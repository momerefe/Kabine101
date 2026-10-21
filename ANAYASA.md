# KoçBenefits Proje Anayasası

Sürüm: 1.4
Tarih: 21 Ekim 2026
Durum: Proje sahibi Yönetici Genel Bakış'ı kabul edip 3D öğrenci deneyimi ve avantaj detaylarını onayladı. 3D bu onayla gerçek katalog, hesap bazlı favoriler ve koşullu kullanım bilgileriyle incelemeye hazırlandı. Normal kayıt alan adı `stu.koc.k12.tr`; e-posta doğrulaması ve ayrıca yönetici onayı zorunludur. Özel e-postasız yönetici istisnası normal hesaplara uygulanmaz. [Güncel teslim raporu](docs/TESLIM_3D.md) ve [teknik kararlar](docs/TEKNIK_KARARLAR.md) esas alınır; sonraki teslimlerin kabul noktaları korunur.

KoçBenefits, Koç Okulu öğrencilerinin anlaşmalı işletmelerin avantajlarını bulmasını ve kullanım koşullarını kolayca anlamasını sağlar. Kabine, işletmeleri ve avantajları sade bir yönetim panelinden yönetir. Başarı ölçütü; öğrencinin geçerli bir avantaja ulaşabilmesi, yöneticinin doğru bilgiyi güvenilir biçimde yayımlayabilmesi ve sistemin sonraki kabineye devredilebilmesidir.

Bu anayasa ürünün kapsamını, tasarımını ve geliştirme kurallarını belirler. İş sırası ve aşama kabul koşulları [ROADMAP.md](ROADMAP.md), mevcut HTML demosunun çalıştırılması [README.md](README.md) içindedir.

## 1 Karar ve kapsam yönetimi

- Son ürün, teknoloji ve yayın kararları proje sahibine aittir. Önerinin olumlu karşılanması, geliştirme veya yayına alma onayı sayılmaz.
- Üç ana aşama korunur: **HTML prototipi → teknik mimari kararı → gerçek uygulama**. Her ana aşamaya geçiş açık onay gerektirir. Gerçek uygulamanın alt teslimleri yol haritasındaki kabul noktalarında değerlendirilir.
- Onaylanan teslim içindeki rutin uygulama, düzeltme ve test işleri için tekrar tekrar izin istenmez. Kapsamı, maliyeti veya kullanıcı davranışını değiştiren kararlar somut etkileriyle sunulur.
- Yeni özellik veya bağımlılık eklenmeden önce hangi mevcut ihtiyacı karşıladığı, bakım yükü ve daha basit seçeneği açıklanır. Onaysız kapsam genişletilmez.
- Bir karar değişirse gerekçesi, etkilenen ekranlar/veriler ve varsa geçiş işi belgelenir. Anayasa ve yol haritası birlikte güncellenir.
- İlerideki bir olasılık için bugünden çok kurumlu sistem, genel kural motoru veya ayrı servisler kurulmaz.

## 2 HTML prototipi tasarım referansıdır

Başlangıç noktası bu klasördeki çalışan prototiptir. Gerçek uygulama geliştirilirken ekranlar ilgisiz bir görsel kimlikle yeniden tasarlanmaz.

| Kaynak | Korunacak referans |
| --- | --- |
| [index.html](index.html), [styles.css](styles.css) | Öğrenci yerleşimi, ortak kabuk, tipografi ve kartlar |
| [admin.css](admin.css), [admin.js](admin.js) | Yönetici menüsü, tablolar, gruplu formlar, onay pencereleri |
| [app.js](app.js) | Öğrenci akışları, giriş sonrası yönlendirme, avantaj detayları |
| [data.js](data.js), [demo-store.js](demo-store.js) | Örnek içerik, yayın durumları ve demo senaryoları |
| [tests/checks.js](tests/checks.js), [tests/admin-checks.js](tests/admin-checks.js) | Mevcut davranışları karşılaştırmak için test senaryoları |

Mevcut kırmızı vurgu `#c82b3c`, açık zemin `#faf9f6`, koyu metin `#252a36`, sade sistem yazı tipi ve yuvarlatılmış kart dili esas alınır. Okunabilirlik ve erişilebilirlik düzeltmeleri aynı kimlik içinde yapılır. Bileşenler ortaklaştırılır; ayrı bir tasarım sistemi ürünü geliştirilmez.

Öğrenci mobil alt navigasyonu korunur. Yönetici menüsü masaüstünde dar başlar; fare ve klavye odağıyla açılır, sabitlenebilir. Mobilde dokunarak açılır, seçimle ve kapatma kontrolüyle kapanır. Açık menü içeriği sıkıştırmaz. Aktif bölüm belli olur, klavye odağı kaybolmaz ve hareketi azaltma tercihi uygulanır.

HTML dosyaları gerçek uygulamanın tasarımı kabul edilene kadar çalışır referans olarak tutulur. Framework dönüşümü, mevcut tasarımın ve kullanıcı akışlarının korunmasıyla kabul edilir.

## 3 Öğrenci kapsamı

Altı ana bölüm korunur:

| Bölüm | İlk gerçek sürümde beklenen davranış |
| --- | --- |
| Genel Bakış | Gerçek yayımlanmış avantajlar, öne çıkanlar ve kategori erişimi; uydurma tasarruf veya kullanım sayısı yok |
| Avantajları Keşfet | Arama, kategori filtresi, açık durum etiketleri ve detay sayfasına erişim |
| Favorilerim | Öğrenci hesabına bağlı ekleme/çıkarma; boş ve artık erişilemeyen avantaj durumları |
| Kullanım Geçmişi | Doğrulanmış kullanım sistemi yokken açıklayıcı boş ekran; sahte kayıt yok |
| Profil ve Hesap | Ad, e-posta, temel hesap işlemleri ve anlaşılır çıkış |
| Yardım ve Destek | Kullanım açıklamaları ve yalnızca tanımlanmış gerçek iletişim bilgileri |

Kategoriler: restoranlar ve kafeler, giyim ve moda, teknoloji, spor ve fitness, eğlence ve etkinlikler, diğer avantajlar. Gereksiz alt menüler eklenmez.

## 4 Yönetici kapsamı

Mevcut 12 bölüm korunur. Bir bölümün bulunması, arkasında doğrulanmamış bir hizmetin var olduğu anlamına gelmez.

| Bölüm | Sınır ve sorumluluk |
| --- | --- |
| Yönetici Genel Bakış | Mevcut kayıtlardan özetler ve son işlemler |
| İş Ortaklığı Takibi | Durum, isteğe bağlı kişi/iletişim, görüşme tarihi ve notlar; basit takip |
| İşletme Detayları | Ekleme, görüntüleme, düzenleme; kategori, açıklama, görsel ve şubeler |
| Avantaj Yönetimi | Gruplu form, taslak, yayın ve koşul kontrolü |
| Öğrenci Doğrulama | Hesap, e-posta doğrulaması ve varsa öğrenci uygunluk incelemesinin ayrı durumları |
| Yayımlanmış Avantajlar | O anda aktif avantajlar, arama, düzenleme ve askıya alma |
| Süresi Dolan ve Askıdaki | Nedenin görünmesi ve kontrollü yeniden yayımlama |
| Kullanım Kayıtları | Doğrulama altyapısı yokken açıklayıcı boş durum |
| Analitik | Tanımı belli, gerçekten toplanan görüntüleme/favori ölçümleri; bunları kullanım diye sunmama |
| Kullanıcı ve Rol Yönetimi | Öğrenci/yönetici rolleri ve hesap durumu; sunucuda yetki kontrolü |
| İşlem Günlüğü | Önemli yönetim değişiklikleri, yapan kişi ve sunucu zamanı |
| Platform Ayarları | Gerekli platform bilgileri ve kontrollü ayar değişiklikleri |

Yayımlama, askıya alma, öğrenci onay/ret ve rol değişikliklerinde anlaşılır onay adımı bulunur. Onay penceresi sunucudaki erişim kontrolünün yerine geçmez. Öğrenci parolası veya parola özeti hiçbir yönetim ekranına gönderilmez.

## 5 Avantajın yaşam döngüsü

Yayın kararı ile tarih geçerliliği ayrı kavramlardır. Taslak, yayımlanmış ve askıya alınmış yayın durumları; başlangıç/bitiş tarihi ve işletmenin iş ortaklığı durumu ile birlikte değerlendirilir.

| Görünen durum | Öğrenciye davranış |
| --- | --- |
| Taslak | Katalogda ve öğrenci detay erişiminde kapalı; yalnızca yetkili yönetici önizler |
| Yakında | Başlangıç gelmediği açık; kullanım kapalı |
| Aktif | Yayımlanmış, işletmesi aktif ve tarihleri uygun; ilgili yöntem kullanılabilir |
| Süresi doldu | Aktif avantaj gibi sunulmaz; varsa katalog/favori/detay kaydı etiketli, kullanım kapalı |
| Askıya alınmış | Katalogdan çıkar; mevcut favori veya bağlantıda durum gösterilebilir, kullanım kapalı |

- İlk yayın ve yeniden yayın için koşullar gözden geçirilir. Geçmiş bitiş tarihi olan avantaj yayımlanamaz.
- İş ortaklığı aktif durumdan çıkarılırsa bağlı yayımlanmış avantajlar askıya alınır. İş ortaklığını yeniden açmak avantajları kendiliğinden yayımlamaz.
- Bu kurallar gerçek uygulamada doğrudan API çağrısıyla da korunur. Kullanım verisi istenirken durum tekrar denetlenir.
- Tarih anlamı **Europe/Istanbul** esasına göre belirlenir; mevcut prototipte bitiş günü dahildir. Tarih belirtilmemişse açık yazılır. İşlem zamanları sunucuda kaydedilir ve arayüzde tutarlı saat diliminde gösterilir.
- Sürenin dolması bir zamanlayıcının çalışmasına bağlı olamaz; okuma ve kullanım yetkisi kontrolünde tarih değerlendirilir.

## 6 Avantaj detayları ve kullanım

Her avantajın kalıcı, benzersiz bir adresi olur. Adın değişmesi kimliği değiştirmez. Prototipteki `#benefit/mola` gibi adresler işlevsel referanstır; gerçek URL biçimi mimari aşamada kararlaştırılır. Giriş gerekirse girişten sonra aynı avantaja dönülür. Bağlantıyı bilmek erişim hakkı sağlamaz.

Telefonda başlık, işletme, indirim/ayrıcalık, durum ve kullanım öğesi kolay bulunur. Koşullar, geçerli ürün/hizmet/şubeler, istisnalar, minimum harcama ve tarihler sağlandıkları ölçüde açık görünür. Önemli koşullar kapalı menüler arkasına saklanmaz. Sağlanmayan bilgi uydurulmaz.

Yalnızca avantajın seçili yöntemi gösterilir: **indirim kodu, bağlantı, QR veya üyelik kartı**. Kopyalama başarısız olursa elle kopyalama yolu bulunur. QR/kart için gerçek işletme kabul biçimi belirlenmeden demo görsel gerçek hak gibi sunulamaz. Tek kullanımlık kupon veya dinamik QR ayrıca kararlaştırılmadıkça kapsamda değildir.

**Kod görüntüleme veya kopyalama, bağlantı açma, QR ya da kart gösterme doğrulanmış kullanım değildir.** Kullanım geçmişinin açılması için işletmede kullanımın nasıl doğrulanacağı, kim tarafından kaydedileceği ve tekrar kayıtların nasıl önleneceği ayrıca onaylanmalıdır. Bu altyapı ilk sürümün zorunlu bağımlılığı değildir.

## 7 Demo ile gerçek sistemin ayrılması

| Prototip davranışı | Gerçek uygulamada gerekli karşılığı |
| --- | --- |
| Hazır örnek hesapla giriş | Gerçek kimlik doğrulaması; üretimde demo giriş yok |
| Tarayıcıdaki alan adı kontrolü | Sunucuda alan adı kontrolü ve e-posta sahipliği doğrulaması |
| Yönetici demosunda öğrenci onayı | E-posta doğrulamasından ayrı, kararlaştırılmış öğrenci uygunluk süreci |
| Tarayıcıda ortak favoriler | Her hesabın kendi favorileri; başka hesaptan erişim engeli |
| Yerel yönetim kayıtları | Sunucuda kalıcı ve erişimi denetlenen kayıtlar |
| Alan adı değişince örnek e-postaların değişmesi | Mevcut kullanıcı e-postaları kendiliğinden yeniden yazılmaz; geçiş ayrıca değerlendirilir |
| Tarayıcı görüntüleme sayıları ve günlüğü | Tanımı belirli ölçüm, sunucuda yönetim işlem kaydı ve erişim sınırı |
| Sabit örnek QR ve geçersiz kart | İşletmeyle kararlaştırılan yöntem; doğrulama kapsamının açık belirtilmesi |

Örnek işletmeler, kodlar, destek adresleri ve kullanıcılar üretime gerçek kayıt olarak taşınmaz. Gerçek okul alan adı `stu.koc.k12.tr` olarak kesinleşmiştir; `ogrenci.example` yalnızca izole testlerde kullanılır.

## 8 Güvenlik ve veri sorumluluğu

- Öğrenci kaydı yalnızca izin verilen okul alan adıyla açılır. Alan adı benzerliği veya alt alan adı otomatik izin sayılmaz. Gerçek adres ve uygunluk politikası okuldan öğrenilir.
- E-posta sahipliği, öğrenci statüsü ve hesap etkinliği ayrı tutulur. Yönetici onayı e-posta sahipliği kontrolünü atlatamaz. Personel/mezunların aynı alan adını kullanıp kullanmadığı netleşmeden yalnızca alan adıyla öğrenci statüsü varsayılmaz.
- Oturumlar ve yetkiler sunucuda denetlenir. İstemcinin gönderdiği rol veya kullanıcı kimliği güvenilir kabul edilmez. Paylaşılan detay adresleri ve doğrudan API istekleri de aynı kontrole tabidir.
- Parola saklama, doğrulama/sıfırlama bağlantıları, oturum süresi ve iptali, CSRF koruması ve giriş sınırlaması gerçek kimlik doğrulama tesliminin parçasıdır. Bağlantılar süreli ve tek kullanımlı olur. Okulun ortak internetini gereksiz yere engelleyen sınırlar kurulmaz.
- İlk yöneticinin oluşturulması, rol verme ve son yetkili yöneticiyi kaybetmeme akışı tanımlanır. Açık kayıtla yönetici olunamaz.
- Proje sahibinin açık talebiyle özel yönetici hesabı e-postasız, kullanıcı adıyla oluşturulabilir. Bu hesap yalnızca operatör CLI'sından açılır; öğrenci olamaz, e-posta doğrulanmış gibi gösterilmez ve parola kaynak koduna yazılmaz. Normal okul hesaplarının alan adı, e-posta doğrulaması, yönetici onayı ve parola kuralları bu istisnadan etkilenmez.
- Gerekli kişisel bilgilerle sınırlı kalınır. Parolalar, oturum anahtarları ve doğrulama bağlantıları loglara yazılmaz. İş ortaklığı iletişim bilgileri öğrenciye gereksiz yere açılmaz.
- Yükleme varsa izin verilen dosya türü, boyut ve erişim kuralları uygulanır. Dosyalar çalıştırılabilir içerik olarak sunulmaz. Genel dosya paylaşım sistemi kurulmaz.
- Yönetim günlüğü normal kullanıcılarca değiştirilemez; uygulamada düzenleme/silme sunulmaz. Saklama süresi ve işletim erişimi yayından önce belirlenir; mutlak değiştirilemezlik iddiasında bulunulmaz.
- Üretim sırları kaynak koduna konmaz. Veritabanı ve varsa görsellerin yedekleme/geri yükleme süreci denenir. Veri saklama, hesap kapatma ve destek sorumlusu yayından önce belirlenir.

## 9 Teknik yaklaşım önerisi

Aşağıdaki temel önceki değerlendirmeyi kaydeder. Geliştirmeyi başlatma talimatı kapsamında 3A'da kullanılan araçlar ve sürüm kaynakları [teknik karar kaydında](docs/TEKNIK_KARARLAR.md) belirtilmiştir. Diğer araçlar ilgili teslimde ihtiyaç olduğunda eklenir; üretim hizmetleri ve maliyetleri ayrıca kararlaştırılır.

| Alan | Önerilen başlangıç |
| --- | --- |
| Sunucu | Go + Chi; modüler monolit, başlangıçta tek uygulama süreci |
| Veri | Tek PostgreSQL 18; SQL + sqlc + pgx; golang-migrate |
| Arayüz | React + TypeScript + Vite; aynı projede öğrenci ve yönetici alanları |
| İstemci araçları | TanStack Router ve Query; gerekli yönetim tablolarında Table; React Hook Form + Zod |
| Görsel bileşenler | Tailwind + Radix; mevcut tasarımı koruyan ortak bileşenler |
| API | Tek OpenAPI sözleşmesi, üretilen Go tanımları ve TypeScript istemcisi; sunucuda iş kuralı doğrulaması |
| Kimlik ve oturum | Argon2id; PostgreSQL oturumları; rastgele oturum kimliği, HttpOnly/Secure/SameSite çerezler |
| İşletim | slog; SMTP, yerelde Mailpit; Linux + systemd ve tek ters vekil olarak Caddy önerisi |
| Geliştirme ve test | pnpm, ESLint/Prettier; Go testleri, Vitest/Testing Library ve Playwright; ihtiyaç varsa MSW |

SQLite tüm uygulama için alternatif olarak mimari aşamada değerlendirilebilir. PostgreSQL seçilirse yalnızca ayarlar için ikinci SQLite veritabanı eklenmez. Veri katmanı seçiminin sürücü ve sorgulara etkisi değerlendirilmeden değişiklik yapılmaz.

Zustand, i18next, CEL, çok kurumlu RLS, ayrı worker/scheduler, genel outbox altyapısı, MinIO, kapsamlı karantina/tarama sistemi, native Android/iOS ve gelişmiş kullanım doğrulaması başlangıç paketine dahil değildir. İlgili ihtiyaç doğarsa ayrı değerlendirilir. Ertelenen bir araç, gerekli güvenlik kontrolünü veya güvenilir hata yönetimini erteleme gerekçesi olamaz.

## 10 Tamamlanma ve bakım kuralları

Bir teslim ancak ilgili kullanıcı akışı uçtan uca çalıştığında, kabul testleri geçtiğinde ve sınırlamaları yazıldığında tamamlanmış sayılır. Sadece ekranın görünmesi yeterli değildir.

- Yüklenme, boş liste, sonuç bulunamaması, erişim reddi ve hata durumları ele alınır. Form hatasında girilen bilgiler gereksiz yere kaybolmaz.
- Mobilde 320 ve 390 piksel genişliklerde sayfa taşması kontrol edilir. Geniş tablo kendi alanında kayabilir. Masaüstü, dokunma, klavye ve hareketi azaltma kontrolleri yapılır.
- Yetkisiz doğrudan API erişimi, başka hesabın favorileri, yayın durumu, tarih sınırları ve oturum iptali test edilir. Prototip testleri gerçek güvenlik testlerinin yerine geçmez.
- Güncel test sonucu, bilinen eksikler, demo kalan işlevler ve çalıştırma adımları teslimde sunulur. Önceki test raporu yeni kod test edilmiş gibi kullanılamaz.
- Bağımlılık sürümleri sabitlenir; güncellemeler ve veritabanı değişiklikleri kontrollü uygulanır. Sırf araç sayısını artırmak için paket eklenmez.
- Sonraki kabine için kurulum, güncelleme, yedekten dönüş ve yönetici devri anlaşılır biçimde belgelenir.

## 11 Açık kararlar

1. Prototip tasarımının kabulü ve varsa son düzeltmeler.
2. Okul alan adı `stu.koc.k12.tr` ve e-posta doğrulaması + yönetici onayı normal hesaplar için kesinleştirildi. Ayrı özel yönetici yerelde kullanıcı adıyla oluşturuldu. Gerçek e-posta hizmeti ve üretim hesabının kurulumu yayına hazırlıkta tanımlanacak.
3. Teknik temel, dosya depolama seçimi, sunucu/e-posta hizmeti ve bütçe.
4. Gerçek sponsor koşulları ve her avantajın işletmede kabul edilen kullanım yöntemi.
5. Destek ve işletim sorumluları, veri saklama süreleri ve pilot kullanıcı grubu.

Bu bilgiler uydurularak doldurulmaz. Her karar, [yol haritasında](ROADMAP.md) ilgili teslimin başlaması veya yayına çıkması için gereken noktada kesinleştirilir.
