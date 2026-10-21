# KoçBenefits

Koç Okulu öğrencileri için avantaj platformu. Mevcut HTML prototipi tasarım referansıdır; gerçek uygulamanın **3A proje ve veri temeli** yerelde tamamlanmış ve incelemeye sunulmuştur. Gerçek kayıt/giriş ve öğrenci/yönetici API'leri henüz uygulanmamıştır.

## İki çalışma biçimi

| Bölüm | Çalıştırma | Durum |
| --- | --- | --- |
| HTML prototipi | Kök dizindeki `index.html` dosyasını aç | Altı öğrenci ve 12 yönetici bölümü, tarayıcıda demo |
| Gerçek uygulama temeli | Go API + Vite web arayüzü | PostgreSQL şeması, sağlık/hazırlık API'leri, bağlantı ekranı |

Prototip verileri PostgreSQL'e otomatik aktarılmaz. Prototipteki giriş güvenli kimlik doğrulaması değildir. Ayrıntılı demo anlatımı [PROTOTIP.md](PROTOTIP.md) içindedir.

## Gereksinimler

- Go: `go.mod` içindeki sürüm.
- Node.js 24 LTS ve pnpm: `package.json` içindeki sürüm.
- PostgreSQL 18; veritabanı hazırlama için `psql`.
- Python 3 yalnızca yerel veritabanı hazırlama ve mevcut prototip testi için.

Bu bilgisayarda Go, Node ve pnpm proje içindeki Git dışında kalan `.tools/` dizinine kurulmuştur. PowerShell yardımcısı bunları otomatik bulur. Başka bilgisayarda araçları standart biçimde kur; `.tools/` GitHub'dan gelmez.

## Yerel kurulum

Depoyu indirdikten sonra proje kökünde çalış:

```powershell
pnpm install --frozen-lockfile
go mod download
python tools/init_local_db.py
```

Son komut yerel `postgres` parolasını gizli girişle ister; proje için süper kullanıcı olmayan `kocbenefits_local` rolünü, `kocbenefits_dev` ve `kocbenefits_test` veritabanlarını oluşturur. Rastgele uygulama parolasını Git dışında kalan `.env` dosyasına yazar. Var olan proje hesabını/veritabanını veya `.env` dosyasını değiştirmez. **Bu bilgisayarda bu hazırlık zaten yapılmıştır.**

Mevcut veritabanına bağlanıyorsan `.env.example` dosyasını `.env` olarak kopyalayıp bağlantıları düzenleyebilirsin. Test bağlantısı ayrı, adı `_test` ile biten veritabanına gitmelidir. `.env` veya veritabanı yedeği Git'e eklenmez.

Şemayı uygula:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 db-up
```

API için bir terminal:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 api
```

Web arayüzü için ikinci terminal:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 web
```

**http://127.0.0.1:5173** adresini aç. API **http://127.0.0.1:8080** üzerinde çalışır. İki süreç terminalde `Ctrl+C` ile durdurulur. Execution policy seçimi yalnızca ilgili PowerShell sürecini etkiler.

Araçlar PATH üzerindeyse Windows/Linux/macOS'ta karşılıkları: `go run ./cmd/db up`, `go run ./cmd/api`, `pnpm dev`. `.env` Go komutları tarafından okunur; Vite'a veritabanı parolası aktarılmaz. Geliştirme sunucuları yalnızca yerel makineye bağlanır.

## Şema ve örnek veri

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 db-version
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 seed-demo
```

Örnek veri komutu isteğe bağlıdır ve yalnızca `APP_ENV=development` altında çalışır. Kurgusal işletme ile yayımlanmamış avantaj taslağı ekler; hesap, gerçek sponsor veya kullanım kaydı oluşturmaz. Tekrar çalıştırmak kayıtları çoğaltmaz. API başlangıcı migration veya seed çalıştırmaz.

`db/migrations` sürümlü değişiklikleri içerir. Mevcut migration uygulandıktan sonra düzenlenmez; yeni sürüm eklenir. CLI otomatik geri alma/sıfırlama komutu sunmaz. Üretim veri geçişleri ayrıca planlanır.

## Kontroller ve kod üretimi

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 generate
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 check
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 build
```

`check`: Go testleri, ayrı PostgreSQL test veritabanında migration/kısıt testleri, `go vet`, TypeScript ve ESLint. Test yalnızca kendisinin oluşturduğu benzersiz şemayı temizler; geliştirme verisini silmez. `build`: Go derlemesi ve `dist/` içinde web çıktısı. Bu bir dağıtım işlemi değildir.

Araçlar PATH üzerindeyken kod üretimi:

```text
go tool oapi-codegen --config api/codegen.yaml api/openapi.yaml
go tool sqlc generate
pnpm generate:api
```

Kaynaklar `api/openapi.yaml` ve `db/queries` dosyalarıdır. `*.gen.*` ve `internal/database` içeriği elle düzenlenmez. Üretilen dosyalar Git'e alınır; CI yeniden üreterek fark olmadığını kontrol eder. Go araçları `go.mod` ile sabitlenmiştir.

HTML prototip testi ayrı çalışır: `python tests/run_browser_checks.py`. Bu test gerçek backend güvenliğini test etmez.

Yeni başlangıç ekranı için API ve Vite açıkken Windows/Chrome üzerinde `python tests/run_foundation_checks.py` çalıştırılabilir. Bağlantı, hata/yeniden deneme ve 320/390/1440 px taşma kontrolü yapar. [3A test raporu](docs/TESLIM_3A.md).

## API ve klasörler

- `GET /api/health`: API süreci çalışıyor mu; veritabanından bağımsız.
- `GET /api/ready`: Veritabanı erişimi, beklenen migration sürümü ve temel sorgu; hazır değilse `503`. Bağlantı bilgileri yanıta yazılmaz.
- `cmd/api`, `cmd/db`: API ve migration/örnek veri komutları.
- `internal`: Yapılandırma, HTTP ve üretilen veri/API tipleri.
- `db`: Migration ve sqlc sorguları.
- `web`: React başlangıcı; mevcut tasarımın `styles.css` değişkenlerini kullanır.
- Kök HTML/CSS/JS ve `assets`: Çalışır tasarım prototipi.
- `.github`: CI ve pull request şablonu; otomatik yayın yok.

## Plan ve sınırlar

- [ANAYASA.md](ANAYASA.md): Kapsam, tasarım ve güvenlik kuralları.
- [ROADMAP.md](ROADMAP.md): Teslimler ve kabul koşulları.
- [Teknik kararlar](docs/TEKNIK_KARARLAR.md): Geliştirme başlangıcı ve bu teslimin kararları.

`ogrenci.example` hâlâ örnek alan adıdır. Gerçek alan adı, öğrenci uygunluğu ve e-posta hizmeti netleşmeden gerçek kayıt açılmaz. Üretimde örnek alan adı reddedilir. Yerel DB rolü şema sahibidir; üretimde uygulama ve migration yetkileri ayrılacaktır. Kimlik doğrulama, iş kurallarının API'lere bağlanması, gerçek kullanım doğrulaması ve yayın henüz tamamlanmamıştır.

## Lisans

Kaynak kodu [MIT](LICENSE) lisanslıdır. Okul/işletme marka hakları ve gerçek sponsorluk veya kişisel veriler lisansla paylaşılmış sayılmaz.
