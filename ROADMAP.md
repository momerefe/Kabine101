# KoçBenefits Geliştirme Yol Haritası

Sürüm: 1.3
Tarih: 21 Ekim 2026
Durum: Proje sahibinin 3B ve özel yönetici hesabını kabul edip devam etme talimatıyla **3C iş ortaklığı ve avantaj yönetimi** tamamlandı ve incelemeye hazırlandı. [3C teslim raporu](docs/TESLIM_3C.md). Okul alan adı `stu.koc.k12.tr`, e-posta doğrulaması ve ayrıca yönetici onayı korunur. Sıradaki teslim 3D; kabul noktaları korunur.

Yol haritası, mevcut HTML öğrenci ve yönetici prototipini çalışan ürüne dönüştürür. Kapsam ve geliştirme kuralları [ANAYASA.md](ANAYASA.md) ile birlikte uygulanır. Ekip kapasitesi, bütçe ve okul bilgileri kesinleşmediğinden takvim tarihi veya teslim süresi taahhüt edilmez; sıra, bağımlılıklar ve tamamlanma ölçütleri belirlenir.

## 1 Mevcut durum ve ana aşamalar

| Aşama | Mevcut durum | Geçiş koşulu |
| --- | --- | --- |
| Aşama 1 HTML prototipi | Çalışır tasarım referansı; geliştirmeye başlama talimatıyla korunarak ilerleniyor | Sonraki ekran teslimlerinde tasarım karşılaştırması |
| Aşama 2 Teknik mimari | 3A için sade teknik temel uygulanıyor; üretim hizmetleri açık | [Teknik kararlar](docs/TEKNIK_KARARLAR.md) |
| Aşama 3 Gerçek uygulama | 3A–3C tamamlandı; 3C incelemeye hazır, 3D sıradaki teslim | Alt teslimin test sonucu ve proje sahibi kabulü |

HTML ekranları tarayıcı demosu olarak korunur. 3A teknik temeli; 3B gerçek hesap, oturum, yerel e-posta doğrulaması ve yönetici onayını; 3C gerçek işletme ve avantaj yönetimini kapsar. Öğrenci kataloğu, favoriler ve doğrulanmış avantaj kullanımı henüz yoktur. Başlangıç ve devam yetkileri teknik karar kaydında belirtilmiştir.

## 2 Aşama 1 prototipin kabulü

**Çıktı:** Mevcut tasarımın kabul edilen hâli ve varsa düzeltme listesi.

- [README.md](README.md) üzerinden öğrenci ve yönetici demosunu incele.
- Öğrencinin altı bölümünü; arama, filtre, favori, kod kopyalama ve detay bağlantısını değerlendir.
- Yönetici tarafında işletme ekleme → avantaj taslağı → koşul kontrolü → yayın → öğrenci kataloğu → askıya alma akışını incele.
- Kod, bağlantı, QR ve üyelik kartının yalnızca ilgili yöntemde göründüğünü kontrol et.
- Mobil detay okunabilirliğini, masaüstü dar/geniş menüyü ve mobil menüyü değerlendir.
- Tasarım düzeltmeleri varsa aynı HTML prototipinde tamamla; mevcut tarayıcı kontrollerini değişiklik sonrası çalıştır.

**Kabul koşulu:** Proje sahibi ekranları ve temel davranışları kabul eder; kalan eksikler demo/gerçek işlev ayrımıyla listelenir. Renk, kart, menü ve detay düzeninin referansı mevcut dosyalardır.

## 3 Aşama 2 teknik kararların kesinleşmesi

**Bağımlılık:** Aşama 1 kabulü. Bu aşamada gerçek backend kurulmaz.

**Çıktı:** Onaylanabilir teknik karar listesi, temel veri ilişkileri, erişim matrisi, işletim yaklaşımı ve ilk sürüm sınırları.

| Karar | Netleştirilecek konu |
| --- | --- |
| Kimler kayıt olabilir | Gerçek alan adı, personel/mezun ayrımı, e-posta doğrulaması ve gerekliyse yönetici incelemesi |
| Mimari | Tek Go uygulaması, tek veritabanı, tek web arayüzü önerisi; araçlar ve sürümler |
| Veri modeli | Hesap, oturum, doğrulama, işletme, avantaj, favori, ayar ve işlem günlüğü ilişkileri |
| Yetkiler | Öğrenci/yönetici erişimleri, ilk yönetici oluşturma, rol değişikliği ve hesap kapatma |
| Avantaj yöntemleri | İlk sponsorların kod/bağlantı/QR/kart ihtiyaçları; doğrulanmış kullanımın kapsam dışında kalması |
| Görseller | Hazır varlık mı yükleme mi; yükleme varsa depo, sınırlar, erişim ve yedekleme |
| Hizmetler | Sunucu, alan adı, SMTP, TLS, ters vekil, bütçe ve bakım sorumlusu |
| Veri sorumluluğu | Saklama/temizleme yaklaşımı, destek sorumlusu ve pilot kapsamı |

Her teknoloji için yalnızca gerekli işlev açıklanır. Ertelenen araçlar anayasanın teknik yaklaşım bölümündedir. Başlatılan teknik temel ve açık kalan üretim kararları teknik karar kaydında ayrılır.

**Kabul koşulu:** Proje sahibi mimariyi, ilk sürüm kapsamını ve işletim maliyet yaklaşımını onaylar. Bilinmeyen gerçek bilgiler açık kalabilir; ancak bu bilgilere bağlı teslim başlatılmaz veya yayına çıkarılmaz.

## 4 Aşama 3 gerçek uygulama teslimleri

Alt teslimler sırayla yapılır. Her teslim sonunda çalışan sonuç, test kanıtı ve kalan işler sunulur; sonraki alt teslim için proje sahibinin kabulü alınır. Onaylı teslim içindeki geliştirme ve hata düzeltmeleri ayrıca izin gerektirmez.

### 3A Proje ve veri temeli

**Bağımlılık:** Mimari onayı.

**İşler:** Onaylanan araçlarla uygulama iskeleti ve yerel çalışma düzeni; aynı uygulamada kimlik/hesap, işletme/avantaj ve yönetim modülleri; veritabanı şeması ve sürümlü değişiklikler; geliştirmeye özel örnek veriler; tek API sözleşmesi ve üretilen tipler; ortam ayarları. Mevcut HTML prototipi çalışır referans olarak korunur.

**Kabul koşulları:** Temiz ortamda belgelenen adımlarla çalışma; boş veritabanına şema kurulumu ve önceki şemadan güncelleme testi; örnek verilerin üretime otomatik eklenmemesi; anlaşılır bağlantı/hata davranışı. Bu teslim güvenli giriş veya öğrenci hizmetinin tamamlandığı anlamına gelmez.

### 3B Kimlik doğrulama ve erişim

**Bağımlılık:** 3A; gerçek alan adı ve öğrenci uygunluk kararı.

**Kesinleşen karar:** `stu.koc.k12.tr` alan adı ve **e-posta doğrulaması + ayrı yönetici onayı**. Yerel e-posta Mailpit'te tutulur; okul sunucusuna ileti gönderilmez. Proje sahibinin devam talimatı bu teslimin geliştirme yetkisidir.

**İş organizasyonu:** Backend/CLI, mevcut tasarımı kullanan React ekranları ve bağımsız PostgreSQL güvenlik testleri GPT‑6.1 Sol alt işlerine ayrıldı. Ana ajan API sözleşmesi, SMTP, birleştirme, tarayıcı testleri ve teslim incelemesini yürütür. İşletme ve avantaj API'leri 3C'de kalır.

**Teslim durumu:** Yerel akışlar ve 11 kimlik/güvenlik entegrasyon senaryosu geçti. Windows'un engellediği migration testinin Linux GitHub CI çalıştırması da geçti. Gerçek okul SMTP teslimatı üretim hizmeti seçildikten sonra pilot öncesinde doğrulanacak. Uygulama henüz genel yayına açılmadı.

**Hesap düzenlemesi:** Proje sahibinin sonraki talebiyle e-postasız özel yönetici kimliği ve ayrı kullanıcı adı girişi eklendi (şema 4). Öğrenci kayıt/doğrulama kuralları korunur. Özel hesap açık kayıttan oluşturulamaz veya öğrenciye dönüştürülemez; gerçek parola kaynak koduna yazılmaz. Yedi ek yönetici kimliği güvenlik senaryosu önceki testlerle birlikte geçti.

**İşler:**

- Prototipteki sade kayıt/giriş ekranlarıyla gerçek kayıt, e-posta doğrulama, giriş, çıkış ve parola sıfırlama.
- Süreli oturumlar, güvenli çerezler, CSRF önlemi, giriş sınırlaması ve gerekli yeniden kimlik doğrulama.
- E-posta doğrulama, öğrenci uygunluğu ve hesap etkinliğinin ayrı değerlendirilmesi; kararlaştırılmışsa yönetici onay/ret süreci.
- Öğrenci/yönetici yönlendirmeleri, ilk yönetici oluşturma, rol değişikliği ve sunucu erişim kontrolleri.
- Önemli hesap ve yetki işlemleri için günlük kaydının başlaması.

**Kabul koşulları:** Yanlış alan adı reddedilir; doğrulanmamış/uygun olmayan veya kapalı hesap korunan içeriğe giremez; öğrenci doğrudan istekle yönetici olamaz veya başka kullanıcı verisine erişemez; süresi dolan/iptal edilen oturum kullanılamaz; doğrulama/sıfırlama bağlantısı tekrar kullanılamaz. Girişten sonra hedef sayfaya dönüş çalışır.

Yerel e-posta testleri gerçek teslimat garantisi sayılmaz. Gerçek okul adresine teslimat pilot öncesinde ayrıca doğrulanır.

### 3C İş ortaklığı ve avantaj yönetimi

**Bağımlılık:** 3B; avantajların alan ve yöntem ihtiyaçlarının netleşmesi.

**Teslim durumu:** Yönetici formları gerçek API/şema 5 ile çalışır. Beş PostgreSQL yönetim kabul senaryosu; yetki/CSRF, yayın koşulları, eşzamanlı düzenleme, iş ortaklığı kapanışı ve işlem günlüğü hatasında geri alma kontrollerini kapsar. Gerçek API ile izole Chrome akışında taslak/yayın/askı, kayıt çakışması, 12 menü, mobil ekranlar ve öğrenci erişim sınırı doğrulandı. Önceki 18 kimlik senaryosu ve migration testi de geçti. [Ayrıntılar](docs/TESLIM_3C.md).

**Sınırlar:** Görseller hazır yerel seçeneklerdir; dosya yükleme eklenmedi. Gerçek sponsor koşulları kullanıcı tarafından girilir; yayınlanan kayıtların öğrenci kataloğuna bağlanması 3D'dedir. İşlem günlüğü sunucuda kaydedilir; günlük ekranı 3E'de tamamlanır.

**İşler:**

- İş ortaklığı, işletme ve avantaj formlarını gerçek verilere bağlama; mevcut tablo ve gruplu form tasarımını taşıma.
- Taslak, yayın, başlangıç/bitiş tarihi, askıya alma ve yeniden yayın kurallarının sunucuda uygulanması.
- Koşullar, ürün/hizmet/şube kapsamı, istisnalar ve minimum harcamanın uygun avantajlarda kullanılması.
- Yayımlanmış ve süresi dolan/askıdaki listeler, arama ve temel filtreler.
- Önemli yönetim işlemlerinin günlüğe yazılması; bağlı değişikliklerin yarım kalmadan kaydedilmesi.
- Görsel yükleme onaylandıysa belirlenen dosya akışı.

**Kabul koşulları:** Taslak öğrenciye sızmaz; geçersiz tarih ve eksik zorunlu alan sunucuda reddedilir; koşul kontrolü ve yayın onayı çalışır; iş ortaklığını kapatma bağlı avantajları askıya alır; yeniden açma otomatik yayın yapmaz; tekrarlanan gönderim ve eşzamanlı düzenleme sessiz veri kaybına neden olmaz; yetkisiz değişiklik engellenir.

### 3D Öğrenci deneyimi ve avantaj detayları

**Bağımlılık:** 3C; ilgili sponsorların gerçek kullanım bilgileri.

**İşler:**

- Genel Bakış ve Keşfet ekranlarını gerçek yayımlanmış avantajlara bağlama.
- Arama/kategori filtresi, hesap bazlı favoriler, profil ve yardım akışları.
- Kalıcı detay adresi, giriş sonrası dönüş ve paylaşım bağlantısı kopyalama.
- Yalnızca ilgili kod, bağlantı, QR veya kart yöntemini gösterme; gerçek anlaşması olmayan yöntemi kullanılabilir gibi sunmama.
- Tarih, koşul ve kullanım öğesini prototipteki mobil öncelik sırasıyla taşıma.
- Doğrulanmış kayıt sistemi bulunmadığı sürece kullanım geçmişinde açıklayıcı boş durum.

**Kabul koşulları:** Öğrenci giriş → keşif → detay → uygun kullanım öğesi akışını tamamlar; favoriler kendi hesabında başka oturumda da görünür ve başka hesaba taşınmaz; kopyalama/elle kopyalama çalışır; süresi dolmuş/askıdaki/yakında avantajın kullanım verisi sunucudan yetkisiz alınamaz; geçersiz detay adresi anlaşılır sonuç verir.

### 3E Yönetim bütünlüğü ve ölçümler

**Bağımlılık:** 3D.

**İşler:** Yönetici Genel Bakış, Kullanıcı ve Rol Yönetimi, İşlem Günlüğü ve Platform Ayarlarının tamamlanması; 12 bölümün tümüne erişim; tanımı belli gerçek görüntüleme/favori ölçümleri; kullanım kayıtlarında açıklayıcı boş durum; alan adı gibi ayarlarda mevcut hesaplara etkinin kontrollü yönetimi.

**Kabul koşulları:** Özetler kayıtlarla uyuşur; demo istatistiği üretimde gösterilmez; QR/kod açmak kullanım sayılmaz; günlük normal arayüzden değiştirilemez; rol değişiklikleri sunucuda korunur ve oturumlara etkisi uygulanır; demo e-posta yeniden yazma davranışı gerçek sisteme taşınmaz. Gerekli hiçbir yönetim ekranı açıklamasız boş veya çalışmayan düğmelerle bırakılmaz.

### 3F Pilot ve yayına hazırlık

**Bağımlılık:** 3A–3E kabulü; gerçek sponsor içerikleri, destek sorumlusu ve hizmet bilgileri.

**İşler:**

- Kabul matrisini test ortamında çalıştırma; gerçek telefonlarda okunabilirlik ve dokunma kontrolü.
- Gerçek okul e-postasına doğrulama ve parola sıfırlama teslimatını deneme.
- Üretim kurulumu, TLS, sır yönetimi, günlükler, hata takibi ve güncelleme adımlarını belgeleme.
- Veritabanı ve varsa görselleri yedekleme; ayrı test ortamında geri yükleme ve kayıtları doğrulama.
- Sorunlu sürümden önceki uygulamaya dönüşü ve şema uyumluluğunu planlama; veri kaybettiren otomatik geri alma varsaymama.
- Proje sahibinin seçtiği küçük öğrenci/yönetici grubuyla pilot; işletme koşullarını ve kullanım talimatlarını doğrulama.

**Kabul koşulları:** Hesap/veri erişimi açığı, yanlış avantaj yayını veya temel akışı durduran hata kalmaz. Pilottaki önemli bulgular çözülür; bilinen düşük etkili eksikler açıkça kabul edilir. Yedekten dönüş denenmiştir; üretimden demo girişleri/verileri ayrılmıştır.

**Yayın kapısı:** Proje sahibinin açık yayın onayı alınır. Yayın sonrası giriş, katalog, detay ve yönetim için kısa kontrol yapılır. Onay verilmeden genel erişime açılmaz.

### 3G Devir ve bakım

**Bağımlılık:** Yayın kabulü.

**Çıktı:** Sonraki kabine için kısa işletim rehberi: kurulum, yönetici devri, avantaj yayımlama/askıya alma, güncelleme, yedekleme, geri yükleme ve destek yönlendirmesi.

**Kabul koşulları:** Yetkili başka biri rehberle temel yönetim görevlerini tamamlar; eski yöneticinin erişimini kapatma ve yeni yöneticiyi yetkilendirme denenir; bakım ve hizmet ödemelerinin sorumlusu belirlenir. Parolalar ve gizli anahtarlar rehbere yazılmaz.

## 5 Prototipten gerçek uygulamaya kabul matrisi

Otomasyon, elle tarayıcı kontrolü ve gerçek cihaz denemeleri ilgili teslimde kaydedilir.

| Alan | Referans ve doğrulanacak sonuç | Teslim |
| --- | --- | --- |
| Görsel kimlik | Mevcut renkler, kartlar, tipografi ve detay yerleşimi korunmuş | 3D–3F |
| Öğrenci navigasyonu | Altı bölüm, mobil alt menü, geri dönüş ve boş durumlar | 3D |
| Yönetici navigasyonu | 12 bölüm; fare, klavye, sabitleme, dokunma ve kapatma | 3C–3F |
| Giriş ve kayıt | Okul alan adı + gerçek e-posta doğrulama + kararlaştırılmış uygunluk | 3B |
| Katalog | Arama/kategori filtreleri, aktif/yakında/süresi dolmuş ayrımı | 3D |
| Favoriler | Hesap bazlı kalıcılık, ekleme/çıkarma, boş/erişilemeyen durum | 3D |
| Avantaj detayı | Kalıcı adres, girişten sonra dönüş, ilgili yöntem, açık koşullar | 3D |
| Pano ve bağlantı | Kod/paylaşım kopyalama, izin yokken alternatif, güvenli dış bağlantı | 3D |
| Yayın akışı | Taslak → inceleme → yayın → askı → yeniden inceleme | 3C |
| Tarih ve iş ortaklığı | İstanbul tarih sınırı; kapanan işletme avantajlarının durması | 3C |
| Doğrulama ve roller | Yönetici onayı e-posta kontrolünü atlamaz; yetkisiz rol değişimi engelli | 3B–3E |
| Kullanım ve analitik | Gösterim/kopyalama kullanım sayılmaz; sayıların kaynağı belli | 3E |
| Güvenlik | Doğrudan API, başka hesap verisi, CSRF, oturum iptali ve dosya kontrolleri | 3B–3F |
| Erişilebilirlik | 320/390 px taşma, gerçek telefon, klavye odağı, azaltılmış hareket | 3D–3F |
| Dayanıklılık | Ağ hatası, tekrar gönderim, eşzamanlı düzenleme, veri kaybı önleme | 3C–3F |
| İşletim | Kurulum, güncelleme, yedekten dönüş ve kabine devri | 3F–3G |

Mevcut `tests/run_browser_checks.py` HTML prototipini kontrol eder. İlgili senaryolar gerçek uygulamanın testlerine taşınır; tarayıcı demosunun başarılı olması API güvenliğinin kanıtı değildir. 3A kapsamında çalıştırılan güncel kontroller teslim raporundadır.

## 6 İlk sürümden sonra değerlendirilecek işler

Aşağıdaki işler ilk yayının koşulu değildir ve ayrıca kapsam/onay gerektirir:

- İşletmede doğrulanan kullanım, buna bağlı öğrenci geçmişi ve yönetici kullanım kayıtları.
- Gerçek ihtiyaç varsa tek kullanımlık kupon veya dinamik QR.
- Android/iOS uygulamaları ve ortak kod yaklaşımı.
- İkinci dil, çok okul desteği veya daha ayrıntılı yönetici izinleri.
- Ölçülen ihtiyaç üzerine ayrı worker/scheduler, outbox veya gelişmiş dosya altyapısı.

Bu özellikler için şimdiden boş servisler, kullanılmayan veri modelleri veya karmaşık yönetim ekranları kurulmaz.

## 7 Teslim raporu ve planın güncellenmesi

Her teslim şu kısa kayıtla kapanır:

1. **Tamamlanan:** Çalışan akışlar ve değişen dosyalar.
2. **Doğrulanan:** Test ortamı, tarih, senaryolar ve sonuçlar.
3. **Eksik veya demo:** Gerçekleşmeyen işlev ve kullanıcıya etkisi.
4. **Karar gereken:** Kapsam, maliyet veya gerçek bilgi gerektiren konu.
5. **Sonraki teslim:** Bağımlılıkları ve kabul koşulları.

Öncelik değişirse iş sırası ve etkilenen kabul koşulları güncellenir. Onaylanmış tasarımın kaybı veya temel güvenliğin ertelenmesi, takvimi kısaltma yöntemi olarak kullanılmaz.
