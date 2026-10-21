# KoçBenefits — öğrenci ve yönetici arayüz prototipi

Bu belge yalnızca kök dizindeki HTML demosunu anlatır. Yeni Go/PostgreSQL/React uygulamasının kurulumu ve durumu için [README.md](README.md) dosyasını kullan.

Mevcut KoçBenefits tasarımı üzerinde HTML, CSS ve saf JavaScript ile geliştirilmiş yerel demo. Öğrenci panelinin altı bölümü korunur; ayrı yönetici girişi, 12 yönetim bölümü ve geliştirilmiş avantaj detayları eklenmiştir. Framework, derleme, backend veya veritabanı gerekmez.

## Proje anayasası ve yol haritası

- [ANAYASA.md](ANAYASA.md): Ürün kapsamı, mevcut HTML tasarımını koruma kuralları, öğrenci/yönetici akışları, güvenlik ve geliştirme ilkeleri.
- [ROADMAP.md](ROADMAP.md): Prototip kabulünden gerçek uygulama ve kabine devrine kadar iş sırası, bağımlılıklar, test ve kabul koşulları.

İki belge de 21 Ekim 2026 tarihli inceleme taslağıdır. Teknik önerileri kaydeder; teknoloji seçimini kesinleştirmez veya gerçek uygulama geliştirmesini başlatmaz. Mevcut HTML prototipi tasarım referansı olarak korunur.

## Yerel olarak çalıştırma

1. Bu klasördeki **index.html** dosyasına çift tıkla ve güncel bir tarayıcıda aç.
2. Öğrenci tarafı için **Örnek öğrenciyle incele** butonuna bas.
3. Yönetici tarafı için giriş ekranının üstünden **Yönetici → Örnek yöneticiyle incele** yolunu kullan.
4. Paneller arasında geçmek için **Çıkış yap**, ardından diğer demo girişini seç.

Başlangıçtaki örnek hesaplar:

| Demo rolü | Örnek e-posta | Parola |
| --- | --- | --- |
| Öğrenci | `deniz@ogrenci.example` | İstenmez |
| Yönetici | `yonetici@ogrenci.example` | İstenmez |

`ogrenci.example` **okulun gerçek alan adı değildir**. Gerçek kişisel veya sponsorluk bilgisi girmen gerekmez. Yönetici formu yalnızca örnek kullanıcı listesinde yönetici rolüne sahip aktif hesapları kabul eder. Öğrenci formuna bir demo yönetici hesabı girilirse yönlendirme hesabın rolüne göre yönetim paneline yapılır. Bu davranış güvenli kimlik doğrulaması değildir.

Yeni bir örnek öğrenci kaydı, demo doğrulama bekleme listesine eklenir. Yönetici demosunda onaylanana kadar o örnek hesapla giriş yapılamaz; e-posta gönderilmez. Hızlı öğrenci incelemesi için hazır öğrenci butonunu kullanabilirsin.

İstersen, Python bilgisayarında zaten kuruluysa klasörde terminal açıp şu komutu çalıştır:

```powershell
python -m http.server 5500 --bind 127.0.0.1
```

Tarayıcıda **http://127.0.0.1:5500** adresini aç. Durdurmak için terminalde **Ctrl+C** kullan. Bu komut yalnızca statik dosyaları yerel olarak sunar; uygulama backend’i değildir. Yerel sunucu, pano ve tarayıcı depolaması davranışını daha tutarlı kılabilir.

Prototip ve görseller internet olmadan açılır. Harici örnek avantaj bağlantısını açmak için internet gerekir. Başlangıç verilerindeki bağlantılar example.org alan adını kullanır.

## Hızlı inceleme senaryosu

1. Yönetici olarak gir; **İşletme Detayları** bölümünde işletme eklemeyi/düzenlemeyi dene. Yayımlanacak avantajın işletmesi **Aktif** olmalı.
2. **Avantaj Yönetimi → Avantaj oluştur** bölümünde örnek bilgileri gir ve **Taslak kaydet**.
3. Taslağı yeniden aç; açıklama, koşullar, talimatlar ve kullanım yöntemini tamamla. Koşulları kontrol ettiğini işaretle, **İncele ve yayımla** butonuna bas ve onay penceresini onayla.
4. Çıkış yapıp öğrenci olarak gir. Yeni avantajı katalogda gör; favorilere ekle ve detay bağlantısını kopyala.
5. Yöneticiye dönüp avantajı askıya al. Öğrenci kataloğundan kalktığını ve doğrudan detay bağlantısında kullanımın kapandığını incele.

Yayımlama burada yalnızca **aynı tarayıcıdaki öğrenci kataloğunu** değiştirir. İnternette yayın veya gerçek sponsorluk duyurusu yapılmaz.

## Öğrenci paneli ve detaylar

Altı ana bölüm: **Genel Bakış, Avantajları Keşfet, Favorilerim, Kullanım Geçmişi, Profil ve Hesap, Yardım ve Destek**. Arama, kategori filtresi, favorilere ekleme/çıkarma, profil adı düzenleme ve çıkış çalışır.

Her avantajın benzersiz bir hash adresi vardır: `index.html#benefit/mola`. Yeni avantajlara da benzersiz kimlik verilir. **Bağlantıyı kopyala** butonu tam demo adresini panoya yazar. Pano izni yoksa bağlantıyı seçilebilir bir alanda gösterir. Giriş yapılmamışsa önce giriş ekranı açılır; girişten sonra aynı avantaja dönülür.

Dosya bağlantısı yalnızca aynı bilgisayardaki dosya konumunda çalışır. `localhost` bağlantısı o bilgisayardaki yerel sunucuya aittir. Başka bilgisayarda aynı dosyalar gerekir; yönetici tarafından oluşturulan tarayıcı verileri cihazlar arasında paylaşılmaz. Bu prototip genel erişime açık bir URL altyapısı kurmaz.

Detay sayfasında işletme kimliği, görseli, avantaj başlığı, açıklama, açık durum etiketi ve ilgili kullanım yöntemi gösterilir. Sağlanmışsa ürün/hizmet kapsamı, şubeler, istisnalar, minimum harcama, başlangıç/bitiş tarihi, öğrenci doğrulama koşulu ve örnek iletişim bilgisi görünür. Koşullar kapalı menülere saklanmaz. Mobilde kullanım öğesi uzun koşullardan ve görselden önce gelir.

Dört kullanım yöntemi desteklenir; aynı anda yalnızca seçilen yöntem gösterilir:

- **Kod:** demo kodunu gösterme, kopyalama ve pano engelinde elle kopyalama açıklaması.
- **Bağlantı:** http/https bağlantısını güvenli yeni sekmede açma.
- **QR:** yerel `assets/demo-qr.svg` görselini gösterme. Bu sabit QR yalnızca `DEMO-KOCBENEFITS` metnini kodlar. Özel QR üretimi veya doğrulama yapılmaz.
- **Üyelik kartı:** örnek adla, açıkça geçersiz demo olarak etiketlenmiş görsel kart. Kimlik/üyelik doğrulaması veya gerçek hak sağlamaz. Bu yöntemi avantaj formundan seçerek deneyebilirsin.

Süresi dolmuş, askıdaki, başlangıç tarihi gelmemiş veya taslak avantajlarda kullanım kapalıdır. Taslakları yalnızca yönetici demo oturumu önizleyebilir. Süresi dolmuş yayımlanmış örnekler katalogda etiketlenir; askıya alınanlar katalogdan kalkar. Favorilerde kayıtlı bir askıdaki avantajın durumu açıkça gösterilir.

Tarih hesabı **Europe/Istanbul** saat dilimine göre yapılır. Bitiş günü boyunca avantaj geçerli sayılır. Tarihi belirtilmemiş bir kayıtta bu durum açıkça yazılır; bu, süresiz gerçek indirim taahhüdü değildir.

## Yönetici panelinin 12 bölümü

| Bölüm | Çalışan demo etkileşimi |
| --- | --- |
| Yönetici Genel Bakış | Örnek kayıtlardan hesaplanan işletme, aktif avantaj, bekleyen öğrenci ve süresi dolan avantaj sayıları; son yönetim işlemleri |
| İş Ortaklığı Takibi | İşletme/kişi/not araması, durum filtresi, görüşme tarihleri ve notları düzenleme |
| İşletme Detayları | İşletme ekleme, görüntüleme ve düzenleme; kategori, yerel görsel, açıklama, iletişim, şubeler |
| Avantaj Yönetimi | Gruplanmış form, taslak, dört yöntem, koşullar, tarihler, kapsam ve kontrollü yayımlama |
| Öğrenci Doğrulama | Bekleyen örnek kayıtları arama, filtreleme, onaylama veya reddetme; onay pencereleri |
| Yayımlanmış Avantajlar | Şu anda aktif avantajları arama, önizleme, düzenleme ve onayla askıya alma |
| Süresi Dolan / Askıdaki | Durum nedenini görme; tarih ve koşulları kontrol ederek yeniden yayımlama |
| Kullanım Kayıtları | Açık demo bilgi ekranı ve boş durum; sahte kullanım kaydı yok |
| Analitik | Yalnızca bu tarayıcıdaki öğrenci detay ziyaretleri ve mevcut favoriler; sade çubuk grafik |
| Kullanıcı ve Rol Yönetimi | Örnek hesapları arama/filtreleme, iki rolü görüntüleme ve onayla demo rolü değiştirme |
| İşlem Günlüğü | Gerçekleştirilen yerel demo yönetim işlemleri, cihaz saati, örnek yönetici, arama |
| Platform Ayarları | Platform adı, örnek okul alan adı, isteğe bağlı örnek iletişim ve öne çıkanlar tercihi |

Yayından kaldırma, yayımlama, mevcut yayını taslağa alma, öğrenci onay/ret ve rol değişikliklerinde onay pencereleri vardır. Yeniden yayımlama için koşulların kontrol edildiği kutu tekrar işaretlenmelidir. Aktif iş ortaklığı kapatılırsa ilgili yayımlanmış avantajlar da askıya alınır; anlaşmayı yeniden aktif yapmak avantajları otomatik yayımlamaz.

Masaüstü yönetici menüsü simge ağırlıklı başlar; fareyle veya klavye odağıyla genişler. Fare ayrılınca ve menüde odak yoksa daralır. İğne butonu menüyü sabitler. Menü içerik üzerine açılır, sayfayı sıkıştırmaz. Mobilde üstteki menü butonuyla açılır; bir bölüm seçerek, dış alana dokunarak veya Escape ile kapanır. Menüde klavye odağı tutulur. Öğrencinin mobil alt menüsü korunmuştur. Kısa geçişler **hareketi azaltma** tercihinde devre dışıdır.

## Demo verilerinin saklanması ve ayarlar

- **Açık oturum:** bellektedir. Sayfa yenilenince yeniden giriş gerekir. Profil ekranındaki ad değişikliği yalnızca o oturumda geçerlidir.
- **Yönetim verileri:** işletmeler, avantajlar, örnek kullanıcılar, ayarlar, demo görüntülemeler ve işlem günlüğü bu tarayıcının `localStorage` alanında tutulur.
- **Favoriler:** ayrı bir tarayıcı kaydıdır; gerçek öğrenci hesabına bağlı değildir ve demo öğrencileri arasında ortaktır.
- Depolama engellenirse değişiklikler bellekte çalışır; kalıcılık sağlanamadığı arayüzde belirtilir.

Yönetici **Platform Ayarları** ekranından alan adını değiştirebilir. Örnek kullanıcı adresleri de yeni alan adına taşınır; sonraki girişler yeni alan adıyla yapılır. İlk değerler `config.js`, başlangıç avantajları/kategorileri/isteğe bağlı detay alanları `data.js`, başlangıç yönetim kayıtları `demo-store.js` içindedir. Gerçek okul alan adı ve destek adresi bilinmediğinden örnek alan adı ve boş iletişim kullanılır.

Kaydedilmiş demo verisi varsa başlangıç dosyalarından önce o veri yüklenir. Başlangıç dosyalarını düzenlediğinde temiz bir başlangıç görmek için tarayıcı konsolunda yalnızca bu prototipin kayıtlarını kaldırıp sayfayı yenileyebilirsin:

```js
localStorage.removeItem('kocbenefits.demo.management.v1');
localStorage.removeItem('kocbenefits.demo.favorites.v1');
location.reload();
```

Bu işlem o tarayıcıdaki demo değişikliklerini ve demo favorilerini sıfırlar. `file://` ve `http://127.0.0.1:5500` adresleri farklı depolama alanlarıdır; birinde yaptığın değişiklik diğerinde görünmeyebilir.

## Güvenlik ve tamamlanmayan gerçek işlevler

Bu bir **arayüz prototipidir**. Tarayıcıdaki alan adı ve rol kontrolü aşılabilir. E-posta sahipliği, gerçek yönetici kimliği veya öğrenci statüsü doğrulanmaz. Parola alınmaz. Onay/ret, rol düzenleme, yayın durumu ve işlem günlüğü yalnızca demo davranışıdır; güvenilir veya değiştirilemez kayıt değildir. Gerçek uygulamada bunların sunucuda korunması ve kullanıcıya göre erişim denetimi uygulanması gerekir.

Kod görüntüleme/kopyalama, bağlantı açma, QR veya kart gösterme **kullanım değildir**. Öğrenci geçmişi ve yönetici kullanım kayıtları sahte kayıt üretmez. Analitik yalnızca tarayıcı demo etkileşimlerini sayar; işletme kullanım doğrulaması veya gerçek performans analizi yapmaz. Destek talebi gönderimi ve gerçek iletişim hizmeti yoktur. Gerçek kullanıcı veya sponsorluk verilerini bu demoda saklama.

Backend, veritabanı, e-posta doğrulaması, gerçek yetkilendirme, işletme kullanım doğrulaması, genel URL altyapısı, gerçek analitik, güvenilir denetim günlüğü ve internette yayın geliştirilmemiştir. Teknik öneriler anayasa ve yol haritasında değerlendirmeye sunulmuştur; teknoloji kararları kesinleştirilmemiş ve gerçek uygulama geliştirmesine başlanmamıştır.

## Dosyalar ve kontroller

```text
index.html       Ortak uygulama çerçevesi ve ikonlar
ANAYASA.md       Ürün kapsamı, tasarım ve geliştirme kuralları taslağı
ROADMAP.md       Aşamalar, bağımlılıklar ve kabul koşulları taslağı
styles.css       Mevcut öğrenci tasarımı
admin.css        Yönetim, menü ve genişletilmiş detay bileşenleri
config.js        Başlangıç prototip ayarları
data.js          Örnek kategoriler, avantajlar ve detay alanları
demo-store.js    Paylaşılan yerel demo verileri ve durum hesabı
app.js           Öğrenci paneli, demo girişler, yönlendirme ve avantaj detayları
admin.js         12 yönetim bölümü, formlar, onaylar ve yönetim etkileşimleri
assets/          Yerel görseller ve sabit örnek QR
tests/           Öğrenci ve yönetici tarayıcı kontrolleri
tools/           İsteğe bağlı sabit QR varlığı üretimi
```

Python ve Chrome kuruluysa isteğe bağlı geliştirici kontrolü:

```powershell
python tests/run_browser_checks.py
```

Geçici bir Chrome profili ve yerel dosya sunucusu kullanır; normal tarayıcı profilini değiştirmez. Öğrenci ve yönetici senaryolarını, gerçek pano kopyalamasını, menü davranışını, 320/390 px mobil taşma kontrollerini, hareketi azaltma tercihini, depolamayı ve dosyaya çift tıklayarak açılışı denetler. Sonuç ve ekran görüntüleri `tests/artifacts/` içine yazılır. Test için yaratılan kayıtlar geçici tarayıcı profilinde kalır. Son doğrulama Chrome ile yapılmıştır; diğer tarayıcılarda ayrıca cihaz testi yapılmamıştır.

QR görseli yerel olarak yüklenir; test Chrome’unda QR çözme API’si desteklenmediğinden otomatik tarama kontrolü yapılmaz. Bu, demo QR’nin işletme doğrulaması sağlamadığı ayrımını değiştirmez.
