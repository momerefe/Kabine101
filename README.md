# KoçBenefits

Koç Okulu öğrencileri için avantaj platformu. Mevcut HTML prototipi tasarım referansı olarak korunur. **3B kimlik ve erişim teslimi** gerçek PostgreSQL hesaplarıyla çalışır: kayıt, e-posta doğrulaması, yönetici onayı, giriş/çıkış, parola sıfırlama ve temel rol yönetimi.

İzin verilen okul alan adı **`stu.koc.k12.tr`**. Giriş için **hem e-posta doğrulaması hem yönetici onayı** gerekir. Yerel ortamda e-postalar Mailpit'te yakalanır; gerçek okul adreslerine gönderilmez. Avantaj ve işletme API'leri henüz bu hesaplara bağlı değildir.

## Prototip ve gerçek uygulama

| Bölüm | Çalıştırma | Durum |
| --- | --- | --- |
| HTML prototipi | Kök dizindeki `index.html` dosyasını aç | Altı öğrenci ve 12 yönetici bölümü, demo veriler |
| Gerçek uygulama | Go API + Vite + yerel Mailpit | Gerçek hesap ve oturum, başvuru onayı ve rol işlemleri |

Prototipteki hesaplar, favoriler ve sponsorlar gerçek veritabanına aktarılmaz. Ayrıntılar [PROTOTIP.md](PROTOTIP.md) içinde. Gerçek arayüz şu an hesap işlemlerini kapsar; kataloğu tamamlanmış gibi göstermez.

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

### İlk yönetici

Kayıt ekranı yönetici oluşturmaz. Mailpit ve veritabanı hazırken yeni bir terminalde aşağıdaki örneği **kendi adın ve okul adresinle değiştirerek** çalıştır:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 admin-create -Name "Ad Soyad" -Email "ADRESIN@stu.koc.k12.tr"
```

Parola gizli olarak iki kez istenir; 15–128 karakter olmalıdır. Komut yeni bir yönetici hesabı oluşturur ve e-posta doğrulaması ister. Mevcut hesabı yöneticiye yükseltmez. Mailpit'teki bağlantıyı açıp doğruladıktan sonra **Yönetici** girişinden oturum aç. Gönderim sırasında hata olduysa hesap oluşturulmuş olabilir; giriş ekranındaki doğrulama e-postasını yeniden gönderme seçeneğini kullan.

### Öğrenci akışı

1. Okul adresiyle kayıt ol.
2. Yerel posta kutusundaki bağlantıyı aç ve e-posta doğrulama düğmesine bas.
3. Yönetici hesabında başvuruyu inceleyip onayla. İşlem mevcut yönetici parolası ve onay ister.
4. Öğrenci hesabıyla giriş yap. Hesap sayfası gerçek sunucu oturumunu gösterir.

E-posta doğrulaması ve yönetici kararı farklı sırada tamamlanabilir; ikisi de tamamlanmadan giriş açılamaz. Rol değişikliği ilgili hesabın oturumlarını kapatır. Son erişilebilir yönetici öğrenci rolüne geçirilemez. Parola sıfırlama tüm eski oturumları kapatır ve yeniden giriş ister.

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

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 generate
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 check
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 build
```

`check`: Go birim testleri, ayrı test veritabanında migration ve kimlik/erişim testleri, `go vet`, TypeScript ve ESLint. `build`: Go ve web derlemesi; dağıtım yapmaz. `generate`: OpenAPI Go/TypeScript tipleri ve SQL'den sqlc kodu. Kaynaklar `api/openapi.yaml` ve `db/queries`; üretilen dosyalar elle değiştirilmez.

```powershell
python tests/run_browser_checks.py
python tests/run_auth_checks.py
```

İlk komut HTML prototipini sınar. İkinci komut Windows/Chrome, Go, Node, Mailpit ve `_test` ile biten `TEST_DATABASE_URL` ister. Önce API, web ve Mailpit terminallerini kapat; 8080/5173/8025/1025 portları boş olmalı. Test kendi süreçlerini, ayrı PostgreSQL şemasını ve sentetik hesaplarını oluşturur; sonunda kapatır ve yalnızca kendi şemasını temizler. Gerçek geliştirme hesaplarına dokunmaz. SMTP yakalama, doğrulama, yönetici onayı, parola sıfırlama ve mobil ekranları sınar. Ekran görüntüleri Git dışındaki `tests/artifacts/` içine yazılır.

API ve Vite zaten açıkken `python tests/run_foundation_checks.py`, `#system` hazırlık/hata ekranını denetler. Sonuçlar ve bilinen sınırlar [3B teslim raporunda](docs/TESLIM_3B.md).

## Şema ve örnek veri

Migration sürümü **3**. Mevcut migration değiştirilmez; yeni sürüm eklenir. API otomatik migration veya seed çalıştırmaz. `tools/dev.ps1 seed-demo` yalnızca geliştirme ortamında kurgusal işletme ve yayımlanmamış avantaj taslağı ekler; hesap veya gerçek sponsor oluşturmaz. Mailpit'in kendi yerel dosyası uygulamanın veritabanı değildir; uygulama yalnızca PostgreSQL kullanır.

## Plan ve lisans

- [ANAYASA.md](ANAYASA.md): Ürün, tasarım ve güvenlik kuralları.
- [ROADMAP.md](ROADMAP.md): Teslimler ve kabul koşulları.
- [Teknik kararlar](docs/TEKNIK_KARARLAR.md): Seçimler ve gerekçeleri.

Sıradaki teslim **3C işletme ve avantaj yönetimi**. Öğrenci kataloğu, hesap bazlı favoriler ve diğer yönetim bölümleri sonraki teslimlerdir. Doğrulanmış kullanım, native mobil ve genel yayına alma bu teslimde yoktur. CelikPanel kaynak kodu kopyalanmamıştır.

Kaynak kodu [MIT](LICENSE) lisanslıdır. Okul/işletme marka hakları ve gerçek sponsorluk veya kişisel veriler bu lisansla paylaşılmış sayılmaz.
