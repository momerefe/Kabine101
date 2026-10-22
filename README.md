# KoçBenefits

Koç Okulu öğrencileri için avantaj platformu. Mevcut HTML prototipi tasarım referansı olarak korunur. **3D öğrenci deneyimi ve avantaj detayları** gerçek PostgreSQL kayıtlarıyla çalışır: öğrenci kataloğu, arama/kategori filtresi, hesap bazlı favoriler ve kalıcı avantaj detayları. İş ortaklığı ve avantaj yönetimi, gerçek hesap, e-posta doğrulaması, yönetici onayı ve oturum akışları korunur. [3D teslim bilgileri](docs/TESLIM_3D.md).

Normal hesaplar için izin verilen okul alan adı **`stu.koc.k12.tr`**. Öğrenci girişinde **hem e-posta doğrulaması hem yönetici onayı** gerekir. Proje sahibinin istediği özel yönetici hesabı ise e-postasız, kullanıcı adıyla giriş yapar; bu istisna normal hesaplara uygulanmaz. Yerel ortamda e-postalar Mailpit'te yakalanır; gerçek okul adreslerine gönderilmez. İşletme ve avantaj yönetimi yalnızca yönetici oturumuna açıktır. Öğrenci API'si yalnızca oturumu ve hesap uygunluğu doğrulanmış hesaplara açıktır.

## Prototip ve gerçek uygulama

**Yönetici Genel Bakış:** Gerçek işletme, avantaj ve öğrenci özetleri; bekleyen işler, durum dağılımı, bugün dahil yedi gün içinde bitecek aktif avantajlar ve son yönetim işlemleri hazırdır. Sayfa açıkken dakikada bir yenilenir, elle de yenilenebilir. [Teslim ve test bilgileri](docs/TESLIM_GENEL_BAKIS.md).

**3E yönetim ekranları:** İşlem günlüğünde arama/tarih/işlem filtresi ve sayfalama; platform adı, destek iletişimi ve öne çıkan öneri ayarları; hesap arama/filtreleri ve gerekçesi onay ekranında açıklanan devre dışı bırakma/yeniden etkinleştirme akışları bulunur. Analitik, son 30 günün öğrenci görüntülenmelerini ve mevcut favori sayılarını gösterir. Okul alan adı bu ekrandan değiştirilemez. [3E teslim ve çalıştırma bilgileri](docs/TESLIM_3E.md).

**Avantaj görseli:** Avantaj bilgileri formundan isteğe bağlı tek JPG/PNG seçilebilir, önizlenebilir, değiştirilebilir ve kaldırılabilir. En fazla 5 MB/16 megapiksel desteklenir. Görsel, avantajı kaydetme veya kaydet-yayımla işlemiyle kalıcılaşır; yayımlanan avantajın öğrenci kartında ve detayında görünür. Yeni **şema 7** gerekir: API'yi kendi terminalinde durdur, `tools/dev.ps1 db-up` komutunu çalıştır ve API'yi yeniden başlat. [Görsel teslimi ve kurulum](docs/TESLIM_3E.md).

| Bölüm | Çalıştırma | Durum |
| --- | --- | --- |
| HTML prototipi | Kök dizindeki `index.html` dosyasını aç | Altı öğrenci ve 12 yönetici bölümü, demo veriler |
| Gerçek uygulama | Go API + Vite + yerel Mailpit | Gerçek hesap/oturum, yönetim, öğrenci kataloğu, favoriler ve avantaj detayları |

Prototipteki hesaplar, favoriler ve sponsorlar gerçek veritabanına aktarılmaz. Ayrıntılar [PROTOTIP.md](PROTOTIP.md) içinde. Gerçek yönetici paneli 9 ana bölümden oluşur; işletmeler ve avantajların yayın durumları **Sponsorluk Yönetimi** altında toplanır. Eski yönetim bağlantıları ilgili görünümü açmaya devam eder. Doğrulanmış kullanım sistemi bulunmadığından kullanım geçmişi açıklayıcı boş durumdur; kod/QR açmak kullanım kaydı oluşturmaz. Tasarruf ve alışveriş onay sistemi kapsam dışında kalır.

## Bu bilgisayarda çalıştırma

Go, Node, pnpm ve Mailpit Git dışında kalan `.tools/` dizinine kurulmuştur. PostgreSQL ve `.env` hazırdır. Önce yeni şemayı uygula:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 db-up
```

Ardından **üç ayrı terminalde**, proje kökünde:

```powershell
# 1. Yerel e-posta kutusu
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 mail
```

```powershell
# 2. Go API
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 api
```

```powershell
# 3. Web arayüzü
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 web
```

- Uygulama: **http://127.0.0.1:5173**
- Yerel posta kutusu: **http://127.0.0.1:8025**
- API hazırlık kontrolü: **http://127.0.0.1:8080/api/ready**

Adreslerde `127.0.0.1` kullan; `APP_ORIGIN` kaynak kontrolü nedeniyle `localhost` farklı bir kaynak sayılır. Terminalde `Ctrl+C` ilgili süreci kapatır. Execution policy seçimi yalnızca o PowerShell sürecine uygulanır.

### Aynı Wi-Fi üzerinden telefondan erişim

Backend ve frontend'i proje sahibi kendi terminallerinde başlatır. Önce açık API ve web terminallerini `Ctrl+C` ile durdur. Bilgisayarın Wi-Fi IPv4 adresini `ipconfig` ile kontrol et. Aşağıdaki örnekte bu adres `192.168.1.94`; adres değişirse iki komutta da yeni adresi kullan:

```powershell
# Birinci terminal: API aynı bilgisayarda kalır; izin verilen site adresi ayarlanır.
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 api -LanAddress 192.168.1.94
```

```powershell
# İkinci terminal: Vite yalnızca belirtilen ağ adresinde dinler.
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 web -LanAddress 192.168.1.94
```

Hem bilgisayarda hem telefonda **http://192.168.1.94:5173** adresini kullan. Telefon aynı Wi-Fi'da olmalı. API istekleri Vite üzerinden geçer; 8080'i ağa açmak gerekmez. `-LanAddress` yalnızca o başlatma için geçerlidir; `.env` değişmez. Bu seçenek olmadan komutlar önceki `127.0.0.1` davranışını korur. IP adresi bu bilgisayara atanmış değilse komut açık bir hata verir. Vite'nin [resmî host ayarı](https://vite.dev/config/server-options#server-host) kullanılır; kaynak/CSRF kontrolü kaldırılmaz.

Bilgisayarda aynı IP adresiyle açılıyor fakat telefonda açılmıyorsa Windows Güvenlik Duvarı veya Wi-Fi istemci yalıtımı engelliyor olabilir. Mevcut Wi-Fi profili **Genel/Public** olarak ayarlıysa yalnızca Özel/Private ağlara verilen Node.js izni yeterli olmaz. Güvenilir yerel ağda gerekirse **yönetici PowerShell** içinde yalnızca yerel alt ağdan bu IP ve porta izin ver:

```powershell
New-NetFirewallRule -Name 'KocBenefits-LAN-5173' -DisplayName 'KocBenefits yerel test 5173' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 5173 -LocalAddress 192.168.1.94 -RemoteAddress LocalSubnet -Profile Any
```

Bu kural otomatik eklenmez; güvenlik duvarı kapatılmaz. Daha önce aynı adla eklediysen tekrar oluşturma. Kuralı kaldırmak için `Remove-NetFirewallRule -Name 'KocBenefits-LAN-5173'` kullan. IP değişirse eski kuralı kaldırıp yeni IP ile oluştur. Misafir Wi-Fi veya erişim noktası cihazları birbirinden yalıtıyorsa ağ ayarını ayrıca kontrol et.

Bu HTTP bağlantısı aynı güvenilir ağda geliştirme içindir; genel yayın değildir. Mailpit bilgisayarda **http://127.0.0.1:8025** adresinde kalır. Telefondaki e-posta doğrulama/sıfırlama bağlantısını denemek için Mailpit'teki bağlantıyı telefona aktar. Aynı IP adresini kullanmıyorsan giriş/kayıt istekleri kaynak kontrolünden reddedilir.

### İlk yönetici

**Özel yönetici:** Yönetici girişindeki **Kullanıcı adı veya okul e-postası** alanından giriş yapar. Hesabın e-postası ve e-posta doğrulama kaydı yoktur; uygulamada sahte bir adres üretilmez. Proje sahibinin seçtiği bilgiler yalnızca yerel veritabanındaki hesapta kullanılır; parola kaynak koduna veya README'ye eklenmez. Bu hesap açık kayıtla oluşturulamaz veya öğrenci rolüne dönüştürülemez.

Başka bir kurulumda özel yönetici oluşturmak için, örnek değerleri değiştirerek:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 admin-create -Name "Yönetici" -Username "yonetici"
```

Parola komut satırı argümanıyla değil, gizli girişle alınır. Özel yönetici için 8–128, normal hesaplar için 15–128 karakter kuralı uygulanır; iki türde de aynı Argon2id koruması kullanılır. E-postasız hesaba e-posta üzerinden parola yenileme uygulanmaz. CLI mevcut hesabı veya parolasını değiştirmez.

**Okul e-postalı yönetici oluşturma** da desteklenir:

Kayıt ekranı yönetici oluşturmaz. Mailpit ve veritabanı hazırken yeni bir terminalde aşağıdaki örneği **kendi adın ve okul adresinle değiştirerek** çalıştır:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 admin-create -Name "Ad Soyad" -Email "ADRESIN@stu.koc.k12.tr"
```

Parola gizli olarak iki kez istenir; 15–128 karakter olmalıdır. Komut yeni bir yönetici hesabı oluşturur ve e-posta doğrulaması ister. Mevcut hesabı yöneticiye yükseltmez. Mailpit'teki bağlantıyı açıp doğruladıktan sonra **Yönetici** girişinden oturum aç. Gönderim sırasında hata olduysa hesap oluşturulmuş olabilir; giriş ekranındaki doğrulama e-postasını yeniden gönderme seçeneğini kullan.

### Öğrenci akışı

Giriş ve kayıt formunda okul alan adı kutunun içinde sabittir. Öğrenci yalnızca `@` öncesini yazar; tam adres arayüzde sunucunun bildirdiği okul alan adıyla birleştirilir. Aynı okulun tam adresi yapıştırılırsa ilk kısma dönüştürülür. Sunucudaki alan adı ve e-posta doğrulaması kuralları ayrıca uygulanır.

1. Okul adresiyle kayıt ol.
2. Yerel posta kutusundaki bağlantıyı aç ve e-posta doğrulama düğmesine bas.
3. Yönetici hesabında başvuruyu inceleyip onayla. İşlem mevcut yönetici parolası ve onay ister.
4. Öğrenci hesabıyla giriş yap. Genel Bakış ve Keşfet üzerinden yayımlanmış avantajlara ulaş; karttaki kalple favori ekle/çıkar.
5. Avantajın detayında koşulları ve tarihleri oku. Aktif avantajın ilgili kullanım düğmesine bas; sunucu o anda yayın, tarih ve iş ortaklığı durumunu yeniden denetler.
6. Paylaşım bağlantısını kopyala. Bağlantıyı oturumsuz açan öğrenci girişten sonra aynı avantaja döner. Favoriler çıkış/giriş sonrasında aynı hesapta kalır.

E-posta doğrulaması ve yönetici kararı farklı sırada tamamlanabilir; ikisi de tamamlanmadan giriş açılamaz. Rol değişikliği ilgili hesabın oturumlarını kapatır. Son erişilebilir yönetici öğrenci rolüne geçirilemez. Parola sıfırlama tüm eski oturumları kapatır ve yeniden giriş ister.

### İşletme ve avantaj yönetimini deneme

1. [Yönetici girişini](http://127.0.0.1:5173/#login/admin) aç. **Sponsorluk Yönetimi → İşletmeler → İşletme ekle** üzerinden kayıt oluştur. Gerçek anlaşma girilmiyorsa ad ve açıklamada test olduğunu belirt.
2. **İş Ortaklığı Takibi** bölümünden durum, iletişim kişisi, görüşme tarihi ve notları düzenle. Yayımlama için işletme aktif olmalıdır.
3. **Sponsorluk Yönetimi → Tüm avantajlar → Avantaj taslağı ekle**. İşletme, başlık ve kullanım yöntemini belirle. Diğer bilgiler tamamlanmadan taslak kaydedilebilir.
4. Açıklama, ayrıcalık, koşullar, kullanım adımları ve seçili yöntemin bilgisini tamamlayıp taslağı kaydet. Ardından **Düzenle → Koşulları incele ve yayımla** ile onayla. Kaydedilmemiş değişiklikler varken yayın yapılamaz.
5. Aynı alandaki **Yayındakiler** ve **Süresi dolan / askıdaki** görünümlerinden durum ve tarihleri kontrol et. Askıya alma neden ister; yeniden yayın yeniden koşul incelemesi ister.
6. E-posta doğrulaması ve yönetici onayı tamamlanmış normal öğrenci hesabıyla aynı uygulamada **Avantajları Keşfet** bölümünü aç. Yayımlanan avantaj ortak veritabanından görünür; ayrı bir aktarım gerekmez. Açık katalog odaklanınca ve görünürken yaklaşık 60 saniyede yenilenir; **Avantajları yenile** düğmesiyle hemen kontrol edilebilir. Taslak/askıdaki avantajlar görünmez; işletme aktif olmalıdır. Yakında/süresi dolmuş avantajlar etiketlenir ve kullanım öğeleri kapalıdır.

Yayımlanmış avantajı düzenleyip kaydetmek onu taslağa çevirir; onay penceresi bunu açıklar. İş ortaklığını aktif durumdan çıkarmak bağlı yayımlanmış avantajları birlikte askıya alır; tekrar aktif etmek otomatik yayın yapmaz. Bitiş günü İstanbul saatine göre dahildir. Eşzamanlı düzenlemede eski sürüm reddedilir; form bilgileri korunur. Kayıt sürümünü yenilemeden tekrar göndermek ikinci kayıt oluşturmaz.

Görseller mevcut hazır seçeneklerle sınırlıdır; dosya yükleme yoktur. QR/kart alanları işletmenin vereceği metni kaydeder, gerçek kullanım doğrulaması üretmez. Avantajın işletmesi kayıttan sonra değişmez; farklı işletme için ayrı avantaj oluşturulur. Listeler filtrelenmiş ilk 1000 kaydı gösterir; sınır aşılırsa aramayı daraltma uyarısı görünür.

## Başka bilgisayarda kurulum

Gereksinimler: `go.mod` içindeki Go sürümü, Node.js 24 LTS, `package.json` içindeki pnpm sürümü ve PostgreSQL 18. Python 3 yerel hazırlık ve tarayıcı testleri içindir.

```powershell
pnpm install --frozen-lockfile
go mod download
python tools/init_local_db.py
python tools/install_mailpit.py
```

Veritabanı hazırlığı `postgres` parolasını gizli girişle ister; projeye özel, süper kullanıcı olmayan `kocbenefits_local` rolünü ve `kocbenefits_dev` / `kocbenefits_test` veritabanlarını oluşturur. Rastgele uygulama parolasını yalnızca Git dışında kalan `.env` dosyasına yazar. Mevcut rol/veritabanı veya `.env` varsa hiçbirini değiştirmez; `.env.example` üzerinden elle yapılandır.

Mailpit yardımcı yükleyicisi Windows x64 içindir; sabit sürümün SHA256 özetini kontrol eder ve servis kurmaz. Diğer sistemlerde [resmî kurulum](https://mailpit.axllent.org/docs/install/) kullanılabilir. Linux/macOS'ta karşılıklar: `go run ./cmd/db up`, `go run ./cmd/api`, `pnpm dev`; Mailpit'i `--listen 127.0.0.1:8025 --smtp 127.0.0.1:1025` ile başlat.

## Yapılandırma ve güvenlik sınırları

- `.env.example` tüm ayarları gösterir. `.env`, parolalar, tokenlar ve yedekler Git'e eklenmez.
- `SCHOOL_EMAIL_DOMAIN=stu.koc.k12.tr`, `ELIGIBILITY_POLICY=manual` kabul edilen kuraldır. `REGISTRATION_OPEN=false` yeni kayıtları kapatır.
- `LOCAL_EMAIL=true` yalnızca geliştirme/test ve loopback SMTP içindir. Yerel posta kutusu bir **test aracıdır**; yerel makineye erişenler test e-postalarını görebilir.
- Gerçek gönderim için `LOCAL_EMAIL=false`, gerçek SMTP bilgileri ve doğrulanmış gönderici gerekir. Uzak SMTP doğrulanan STARTTLS bağlantısı kullanır. SMTP teslimi ve okul adresine gerçek ulaşma henüz sınanmadı.
- Üretimde HTTPS `APP_ORIGIN`, gerçek okul alan adı ve SMTP kimlik bilgileri zorunludur. Oturum ve CSRF çerezleri `__Host-`, `Secure`, `HttpOnly`, `SameSite=Lax` kullanır. Yerel HTTP'de `Secure` kapalıdır.
- Oturumlar 24 saat; doğrulama bağlantıları 24 saat; sıfırlama bağlantıları 30 dakika geçerlidir. Veritabanında tokenların yalnızca SHA256 özetleri saklanır. Parolalar Argon2id ile özetlenir.
- Kayıt ve e-posta talepleri hesabın varlığını açıklamayan yanıt verir. Talebin alınması teslim garantisi değildir; ileti gelmezse yeniden gönderilebilir. SMTP hataları token/e-posta içermeden sunucuya kaydedilir.
- Deneme sınırları tek API sürecinin belleğindedir; yeniden başlatmada sıfırlanır. İstemciden gelen `X-Forwarded-For` güvenilir sayılmaz. Üretim ters vekili ve okul ortak IP sınırları pilot öncesinde ayarlanacaktır.
- Yerel DB rolü şema sahibidir. Üretimde migration ve uygulama yetkileri ayrılacak; TLS, yedekleme, kayıt saklama/temizleme ve gerçek teslimat kontrolleri yayından önce tamamlanacaktır.

## Kontroller

Masaüstü, telefon ve dokunmatik tablet kontrollerinin kapsamı ve tekrar çalıştırma komutları: [Responsive kontrol raporu](docs/RESPONSIVE_KONTROL.md).

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 generate
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 check
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 build
```

`check`: Go birim testleri, ayrı test veritabanında migration, kimlik/erişim ve yönetim testleri, `go vet`, TypeScript ve ESLint. `build`: Go ve web derlemesi; dağıtım yapmaz. `generate`: OpenAPI Go/TypeScript tipleri ve SQL'den sqlc kodu. Kaynaklar `api/openapi.yaml` ve `db/queries`; üretilen dosyalar elle değiştirilmez.

```powershell
python tests/run_browser_checks.py
python tests/run_auth_checks.py
```

İlk komut HTML prototipini sınar. İkinci komut Windows/Chrome, Go, Node, Mailpit ve `_test` ile biten `TEST_DATABASE_URL` ister. Önce API, web ve Mailpit terminallerini kapat; 8080/5173/8025/1025 portları boş olmalı. Test kendi süreçlerini, ayrı PostgreSQL şemasını ve sentetik hesaplarını oluşturur; sonunda kapatır ve yalnızca kendi şemasını temizler. Gerçek geliştirme hesaplarına dokunmaz. SMTP yakalama, doğrulama, yönetici onayı, parola sıfırlama ve mobil ekranları sınar. Ekran görüntüleri Git dışındaki `tests/artifacts/` içine yazılır.

Yeni yönetim ekranlarının gerçek API ile izole tarayıcı kabulü, geliştirme servislerini kapatmadan çalıştırılabilir:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 build
$env:KOC_BROWSER_TEST = '1'
.tools/go/bin/go.exe test -tags=integration ./internal/management -run TestManagementBrowser -count=1 -v
Remove-Item Env:KOC_BROWSER_TEST
```

Bu test Python ve Chrome ister; `dist` arayüzünü rastgele yerel portta sunar, ayrı `_test` şemasını ve sentetik hesaplarını oluşturur, sonunda kendi kaynaklarını temizler. Gerçek geliştirme kayıtlarına dokunmaz. Chrome farklı konumdaysa `KOC_CHROME` ayarlanabilir. `tests/run_management_checks.py` doğrudan çalıştırılmaz; Go testinin başlattığı tarayıcı adımıdır.

API ve Vite zaten açıkken `python tests/run_foundation_checks.py`, `#system` hazırlık/hata ekranını denetler. Güncel sonuçlar [3C teslim raporunda](docs/TESLIM_3C.md); önceki hesap teslimi [3B raporunda](docs/TESLIM_3B.md).

## Şema ve örnek veri

Migration sürümü **7**. Mevcut migration değiştirilmez; yeni sürüm eklenir. API otomatik migration veya seed çalıştırmaz. `tools/dev.ps1 seed-demo` yalnızca geliştirme ortamında kurgusal işletme ve yayımlanmamış avantaj taslağı ekler; hesap veya gerçek sponsor oluşturmaz. Avantaj görselleri ayrı bir PostgreSQL tablosunda saklanır ve veritabanı yedeğine dahildir. Mailpit'in kendi yerel dosyası uygulamanın veritabanı değildir; uygulama yalnızca PostgreSQL kullanır.

## Plan ve lisans

- [ANAYASA.md](ANAYASA.md): Ürün, tasarım ve güvenlik kuralları.
- [ROADMAP.md](ROADMAP.md): Teslimler ve kabul koşulları.
- [Teknik kararlar](docs/TEKNIK_KARARLAR.md): Seçimler ve gerekçeleri.

**3E yönetim bütünlüğü ve ölçümler** incelemeye sunulmuştur. Proje sahibinin kabulünden sonraki adım **3F pilot ve yayına hazırlık** olacaktır. Doğrulanmış kullanım, native mobil ve genel yayına alma bu teslimde yoktur. CelikPanel kaynak kodu kopyalanmamıştır.

Kaynak kodu [MIT](LICENSE) lisanslıdır. Okul/işletme marka hakları ve gerçek sponsorluk veya kişisel veriler bu lisansla paylaşılmış sayılmaz.
