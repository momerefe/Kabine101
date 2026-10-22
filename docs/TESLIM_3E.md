# 3E — Yönetim bütünlüğü ve sade analitik

Bu teslim proje sahibinin açık devam onayıyla işlem günlüğü, platform ayarları, kullanıcı/rol yönetimi ve analitik ekranlarını tamamlar. Mevcut öğrenci paneli ve yönetici görsel kimliği korunur. Tasarruf ve alışveriş onay sistemi kapsam dışındadır.

## Ekranlar

**İşlem Günlüğü:** Gerçek kayıtlar en yeniden eskiye gösterilir. İşlemi yapan kişi, işlem türü, ilgili kayıt ve İstanbul saatine göre tarih bulunur. Arama, işlem türü, başlangıç/bitiş günü ve 50 satırlık sayfalama vardır. Arayüz/API üzerinden günlük değiştirme veya silme sunulmaz. Günlüğün kişi/kayıt adları mevcut kayıtlardan okunur; tarihsel ad kopyası tutulmaz.

**Platform Ayarları:** Platform adı, destek iletişimi (boş, e-posta, HTTPS bağlantısı veya telefon) ve öğrenci genel bakışındaki öne çıkan öneriler yönetilir. Katalog erişimi öneri tercihinden etkilenmez. `stu.koc.k12.tr` alan adı sunucu yapılandırmasından salt okunur gösterilir. İki yöneticinin eşzamanlı değişikliklerinde eski form yeni ayarları ezemez; girilen bilgiler korunur ve güncel değerlerle karşılaştırılabilir.

**Öğrenci Doğrulama / Kullanıcı ve Rol Yönetimi:** Mevcut başvuru/rol işleyişine arama ve filtreler, hesap devre dışı bırakma ve yeniden etkinleştirme eklenir. Hassas işlemde mevcut yönetici parolası ve açık onay gerekir. Durum/rol değişiklikleri hedef hesabın oturumlarını kapatır. Son etkin ve uygun yönetici kapatılamaz; e-postasız özel yönetici hesabı öğrenciye dönüştürülemez. Bekleyen/reddedilen hesaplarda etkinleştirme yolu e-posta ve yönetici onayını atlayamaz. Listeler 1000 satırla sınırlıdır; daha fazla eşleşme varsa filtreyle daraltma istenir.

**Analitik:** Son 30 İstanbul takvim günündeki öğrenci avantaj detay görüntülenmeleri ve mevcut favori sayıları ayrı gösterilir. Aynı öğrenci aynı avantajı aynı gün tekrar açarsa sayı artmaz; başka bir gün açarsa o gün için bir görüntülenme oluşur. Yönetici önizlemeleri sayılmaz. Tablo en çok 20 avantajı listeler; toplamlar bu sınıra bağlı değildir. Yüklenemeyen veri sıfır istatistik olarak sunulmaz. Görüntülenme, indirim kullanımı veya satın alma kanıtı değildir.

**Kullanım Kayıtları:** Doğrulanmış kullanım altyapısı olmadığını açıklayan mevcut boş durum korunur. Sahte geçmiş oluşturulmaz.

## Yerel güncelleme

Sunucuları proje sahibi kendi terminallerinden başlatır. Bu teslim yeni **000006_management_insights** migration'ını içerir. API terminalinde `Ctrl+C` ile mevcut süreci durdurduktan sonra proje kökünde:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 db-up
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 api -LanAddress 192.168.1.94
```

Açık Vite'ye bağlı tarayıcıyı yenile. Frontend kapalıysa ayrı terminalde:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 web -LanAddress 192.168.1.94
```

IP değiştiyse iki komutta güncel bilgisayar IP'sini kullan. Ajan bu geliştirme için yerel API/frontend başlatmaz ve gerçek veritabanına örnek veri eklemez. Yeni analitik geçmişe dönük sayı üretmez; ölçüm yeni API ve arayüz etkin olduktan sonra oluşur.

## Doğrulama

- Yerelde Go build/vet, TypeScript/ESLint, üretim derlemesi ve biçim kontrolü geçti.
- Proje sahibinin açık Vite sunucusuna bağlanan, tüm API çağrılarını örnek yanıtlarla karşılayan kontrollerde öğrenci panelinin 87 ve yönetici genel bakışının 17 yerleşim/etkileşim senaryosu geçti. Yerel uygulama verisi değiştirilmedi.
- GitHub Actions üzerinde Go birim testleri ve PostgreSQL 18 ile migration, kimlik doğrulama, yönetim ve öğrenci entegrasyon testleri geçti. Ayar sürüm çakışmaları, işlem günlüğünün filtreleri/veri sınırları, günlük görüntülenme tekilleştirmesi, analitik toplamları, hesap oturumlarının iptali ve eşzamanlı son-yönetici koruması sınandı.
- Gerçek API'ye bağlı öğrenci tarayıcı kabul testi, 21 yerleşim ve giriş/favori/kullanım öğesi akışıyla geçti.
- Gerçek API'ye bağlı 3E tarayıcı kabul testi; ayar kaydetme/çakışma çözümü, günlük araması/sayfalaması, öğrenci görüntülenmesi/favorisinin analitiğe yansıması, yanlış/doğru parola ile hesap kapatma ve eski oturumların iptali akışlarıyla geçti. Ayarlar, günlük, analitik, hesap listesi ve onay penceresi 320/390/1440 px boyutlarında kontrol edildi. Dar ekranda üst çubuk taşması bu kontrol sırasında düzeltildi.
- Son kaynak sürümü `6ec0320` için [GitHub Actions kontrollerinin tamamı başarılı](https://github.com/momerefe/Kabine101/actions/runs/39674712805); OpenAPI/sqlc tarafından üretilen dosyalar da yeniden üretimle karşılaştırıldı.

Tarayıcı ölçüleri gerçek cihaz testinin yerine geçmez; proje sahibinin telefonunda son görsel kabul beklenir. Yerel veritabanı güncellemesi ve API yeniden başlatması yukarıdaki komutlarla proje sahibi tarafından yapılmalıdır.

## Kapsam sınırı ve sonraki adım

### 22 Ekim 2026 — Sponsorluk Yönetimi düzenlemesi

Proje sahibinin isteğiyle işletme detayları, avantaj yönetimi, yayımlanmış avantajlar ve süresi dolan/askıdaki kayıtlar tek **Sponsorluk Yönetimi** menüsünde toplandı. İçeride dört görünüm bulunur: İşletmeler, Tüm avantajlar, Yayındakiler ve Süresi dolan / askıdaki. Eski adresler ve genel bakıştan gelen durum filtreleri korunur. Ana menü 9 bölüme indirildi; mevcut tasarım ve mobil açılır menü kullanılır.

Öğrenci **Avantajları Keşfet** ekranına elle yenileme eklendi. Yayın işlemi mevcut ortak PostgreSQL kayıtlarını kullanır; ayrı aktarım veya demo içerik gerekmez. Yayımlama sonrası açık katalog otomatik yenilenmeyle veya **Avantajları yenile** düğmesiyle güncellenir. İşletmenin aktif olması gerekir; taslak/askıdaki kayıtlar öğrenciye açılmaz. Yayımlanmış kayıtların yönetici detayında öğrenci detay sayfasına bağlantı bulunur.

`2b8791b` sürümünün [GitHub Actions kontrollerinin tamamı geçti](https://github.com/momerefe/Kabine101/actions/runs/39754493174). Ayrı Chrome profillerindeki yönetici ve normal öğrenci oturumlarıyla taslak gizliliği → yönetici yayını → öğrenci katalog/detay erişimi → askıya alma → öğrenci erişiminin kapanması gerçek API üzerinde sınandı. Dokuz ana menü bağlantısı ve dört sponsorluk görünümü 320/390/1440 px boyutlarında kontrol edildi. Yerelde öğrenci arayüzünün 87 ve genel bakışın 17 kontrolü, TypeScript/ESLint, derleme ve biçim kontrolleri geçti.

Bu düzenleme yeni migration veya backend yeniden başlatması gerektirmez. Önceki 3E kurulumunun şema 6 ile çalışması gerekir. Açık Vite sunucusunda tarayıcıyı yenilemek yeterlidir; yerel sunucular ajan tarafından başlatılmadı ve gerçek hesaplara test kaydı eklenmedi.

### 22 Ekim 2026 — Yayın gereksinimlerinin açıklanması

İşletme ve avantaj formlarında zorunlu alanlar kırmızı yıldızla gösterilir. Avantaj formu taslak kaydı ile yayın gereksinimlerini ayırır: işletme, başlık ve kullanım yöntemiyle taslak kaydedilebilir; yayın için açıklama, indirim/ayrıcalık, koşullar, kullanım adımları ve seçilen yöntemin bilgisi tamamlanmalıdır. İsteğe bağlı alanlar yıldızla işaretlenmez.

Yayına hazırlık alanı eksik bilgileri ve işletmenin iş ortaklığı durumunu açıklar. Eksik alanla yayın denenirse ilgili alana odaklanılır; kayıt kendiliğinden yayımlanmaz veya işletme aktif yapılmaz. Sunucu da kendi kontrollerini sürdürür ve genel hata yerine eksik alanların adlarını döndürür.

`f5f4d9b` için [GitHub Actions kontrollerinin tamamı geçti](https://github.com/momerefe/Kabine101/actions/runs/39759058534). Go birim/entegrasyon testleri ve gerçek API ile tarayıcı testleri; eksik ayrıcalıkla taslak kaydetmeyi, açıklayıcı yayın engelini, kırmızı yıldızları, aktif olmayan işletme kontrolünü ve tamamlanan yayının normal öğrenciye açılmasını doğruladı. Form 320/390/1440 px boyutlarında kontrol edildi; TypeScript/ESLint, üretim derlemesi ve biçim kontrolü de geçti.

Bu düzeltme migration gerektirmez. Güncel sunucu hata mesajlarını almak için proje sahibi açık API terminalinde `Ctrl+C` ile durdurup kullandığı API başlatma komutunu yeniden çalıştırmalı; tarayıcı yenilenmelidir. Yerel sunucular ajan tarafından başlatılmaz ve mevcut sponsorluk bilgileri değiştirilmez.

### Aktif işletmeden yayınlama

Aktif işletmelerin satırında **Detay** ve **Düzenle** yanında kırmızı **Yayınla** düğmesi bulunur. İşletme formunun sonunda da aynı akışa erişilir; kaydedilmemiş değişiklik varsa **Kaydet ve yayına devam et** önce işletme kaydını günceller. Bu akış işletmenin avantajlarını getirir. Yönetici yayımlayacağı avantajı seçer veya henüz yoksa işletmesi önseçili yeni bir avantaj oluşturur. Aktif işletme tek başına indirim içeriği sağlamaz; boş bilgilerle veya bütün avantajları birden yayımlama yapılmaz.

Avantaj formunun sonunda kırmızı **Yayınla**, yeni/değişmiş kayıtta **Kaydet ve yayımla** bulunur. Koşulların incelendiğine dair tek onaydan sonra gerekiyorsa kayıt taslak olarak saklanır ve dönen sürümle yayın isteği yapılır. Yayın başarısız olursa kaydedilmiş taslak ve sürümü formda korunur; yeniden yayın denenebilir. Eksik bilgiler, aktif olmayan işletme, süresi dolmuş tarih, yetki ve sürüm çakışması kontrolleri korunur. Başarılı yayın ortak veritabanından öğrenci kataloğuna yansır; açık öğrenci ekranında **Avantajları yenile** ile hemen alınabilir.

Avantaj listesi API'sine isteğe bağlı `business_id` filtresi eklendi; SQL sonucu sınırlandırılmadan önce uygulanır. Bu güncelleme migration gerektirmez, ancak yeni filtre için API proje sahibi tarafından yeniden başlatılmalıdır.

`fe4d60c` için [tüm GitHub Actions kontrolleri geçti](https://github.com/momerefe/Kabine101/actions/runs/39760869052). Gerçek API ile tarayıcı testleri; aktif işletme satırı/formundaki kırmızı düğmeleri, işletmesi önseçili oluşturmayı, tek onayla kaydet-yayımla işlemini, yayın hatası sonrası taslak/sürüm korunmasını ve tekrar denemeyi, birden fazla avantajdan yalnız seçilenin yayımlanmasını ve normal öğrenci erişimini doğruladı. 320/390/1440 px yerleşimleri kontrol edildi. İşletme filtresinin yetkilendirme, geçersiz kimlik ve SQL sınırı öncesi uygulanması entegrasyon testlerinden geçti. Yerel sunucular başlatılmadı; gerçek sponsorluk kayıtları değiştirilmedi.

### Avantaj bilgilerine görsel yükleme

Proje sahibinin isteğiyle avantaj başına isteğe bağlı tek JPG/PNG eklenebilir. **Avantaj bilgileri → Avantaj görseli** alanında dosya seçilince önizlenir; **Taslak olarak kaydet** veya **Kaydet ve yayımla** ile kalıcı kayda girer. Görsel değiştirme ve kaldırma da aynı kayıt/onay akışını kullanır. Yayımlanan görsel öğrenci kartında ve detayında görünür; özel görsel kaldırılırsa işletme görseli kullanılır. JPEG fotoğrafların tarayıcıda okunan yönü yüklemeden önce piksellere aktarılır.

Sınır dosya başına 5 MiB, 16 milyon piksel ve kenar başına 8192 pikseldir. Go sunucusu JPEG/PNG içeriğini doğrulayıp yeniden kodlar. Görseller ayrı PostgreSQL tablosunda saklanır; katalog yanıtlarına ikili veri eklenmez. Taslak/askı görselleri öğrenciye açılmaz; yetki, CSRF, kayıt sürümü ve işlem günlüğüyle aynı transaction korunur. Hata olursa eski görsel korunur; formdaki yeni seçim yeniden denenebilir.

Bu teslim **000007_benefit_images** migration'ını gerektirir. Proje sahibi API terminalinde `Ctrl+C` ile durdurduktan sonra proje kökünde:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 db-up
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 api
```

Bu komutlar son kullanılan localhost düzeni içindir. Açık frontend sayfası yenilenir. LAN kullanılıyorsa mevcut API/frontend adres eşleşmesi korunarak aynı `-LanAddress` değeriyle başlatılır. Ajan yerel migration veya sunucu başlatma işlemi yapmaz. Yeni yedekler görselleri de içerir; üretim kapasite ve geri yükleme denemesi pilot hazırlığında değerlendirilir.

`e0015db` için [tüm GitHub Actions kontrolleri geçti](https://github.com/momerefe/Kabine101/actions/runs/39808440043). JPEG/PNG doğrulaması ve boyut sınırları, atomik kayıt/sürüm çakışması, taslak/yayın/askı erişimi ve oturum iptali entegrasyon testleriyle doğrulandı. Tarayıcı testinde dosya seçme/önizleme, JPEG yönü, taslak kaydetme, öğrenci kartı/detayında gerçek görsel yükleme, başarısız değiştirme sonrası yeniden deneme ve kaldırma akışları geçti. Mobil/masaüstü yerleşimleri, TypeScript/ESLint, derleme ve üretilen dosya kontrolleri de başarılıdır. Yerel veritabanına migration uygulanmadı; proje sahibi yukarıdaki komutlarla günceller.

Gerçek e-posta teslimatı, barındırma, TLS, yedekleme/geri yükleme denemesi ve pilot kullanıcı grubu 3F kapsamında ayrıca değerlendirilecektir. Bu teslim otomatik yayına alma veya yeni bir aşama onayı değildir.
