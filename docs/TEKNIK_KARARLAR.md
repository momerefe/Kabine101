# Teknik kararlar

Tarih: 21 Ekim 2026

## 3C devam kararı

Proje sahibi 3B ve özel yönetici hesabını kabul ederek plana uygun devam edilmesini istedi. Bu teslim iş ortaklığı, işletme ve avantaj yönetimini kapsar; öğrenci kataloğu/favoriler 3D'de, diğer yönetim modüllerinin tamamlanması 3E'de kalır.

- Tek Go API, PostgreSQL ve mevcut React/CSS temeli korunur; yeni framework veya hizmet eklenmez. HTML prototipi tasarım referansı olarak değişmeden kalır.
- İşletme ve avantaj değişiklikleri kayıt sürümüyle denetlenir. İstemcinin oluşturduğu sabit UUID ve yeni kayıtta sürüm 0, tekrar gönderimde ikinci kayıt oluşmasını önler; eski sürümle güncelleme 409 yanıtı verir. Form içeriği korunarak güncel kayıtla karşılaştırma istenir.
- Avantaj düzenlemesi taslak olarak kaydedilir. İlk yayın ve yeniden yayın ayrı bir koşul kontrolü ve açık onay gerektirir. Yayınlanmış kaydı düzenleyip kaydetmenin yayını durdurduğu formda belirtilir.
- İş ortaklığını kapatma, bağlı yayımlanmış avantajları aynı veritabanı işlemi içinde askıya alır. Yeniden açma otomatik yayın yapmaz. İşletme/yayın değişiklikleri ve günlük kaydı birlikte tamamlanır.
- Tarihler İstanbul gününe göre değerlendirilir; bitiş günü dahildir. Sürenin dolması zamanlayıcıya bağlı değildir.
- Görsel yükleme/depolama kararı verilmediğinden mevcut yerel görsellerden seçim sunulur. Dosya yükleme, harici görsel URL'si veya tarama altyapısı eklenmez. Gerçek sponsor içerikleri uydurulmaz.
- Backend, React ekranları ve bağımsız PostgreSQL kabul testleri GPT‑6.1 Sol alt işlerine ayrılır. Ana ajan bütünleştirme, tarayıcı kontrolü ve teslimi yürütür.

## 3B devam kararı

**Sonraki hesap kararı:** Proje sahibi normal hesaplardan ayrı, e-postasız bir özel yönetici istedi. Şema 4, `email` veya `username` kimliğinden tam birini zorunlu kılar. Kullanıcı adı kimliği yalnızca yönetici olabilir; öğrenci rolüne dönüştürülemez. Ayrı yönetici giriş uç noktası aynı oturum, CSRF, deneme sınırı ve Argon2id kontrollerini kullanır. Özel yönetici parolası için 8–128 karakter; normal hesaplarda 15–128 korunur. Seçilen gerçek parola hiçbir kaynak veya dokümanda tutulmaz; yalnızca veritabanındaki özeti kullanılır. Bu istisna okul hesabını e-posta doğrulamasından muaf tutmaz.

Proje sahibi CelikPanel değerlendirmesinin ardından planlandığı şekilde devam edilmesini ve alt işlerin GPT‑6.1 Sol'e verilmesini onayladı. Ardından **`stu.koc.k12.tr`** okul alan adını ve **e-posta doğrulaması + yönetici onayı** kuralını belirledi. Aşağıdaki 3A bölümleri önceki teslimin tarihsel kaydıdır; güncel işlevler 3B teslim raporundadır.

- Go/Chi + PostgreSQL/sqlc/migrate korunur. CelikPanel kaynak kodu veya lisanslı parçaları kopyalanmaz.
- Parolalar Go `x/crypto/argon2` Argon2id ile; 64 MiB, üç geçiş, dört paralellik, rastgele salt kullanılarak saklanır. En fazla iki eşzamanlı parola işi bellek tüketimini sınırlar. Parolalar 15–128 Unicode karakter olabilir; zorunlu sembol kuralı eklenmez.
- Oturum tokenı 32 rastgele bayttan üretilir, veritabanında SHA256 özeti tutulur. JWT veya refresh-token zinciri bu web uygulamasının ihtiyacı değildir. 24 saatlik sunucu oturumu iptal edilebilir; rol/hesap uygunluğu her korunan istekte sunucudan okunur.
- Doğrulama 24 saat, sıfırlama 30 dakika; ayrı amaçlarla, atomik tüketilen tek kullanımlık tokenlar. Sıfırlama bütün oturumları kapatır. Bağlantılardaki token URL fragment'ında taşınır; GET isteği hesabı değiştirmez, kullanıcı açıkça form gönderir.
- `Origin`, JSON içerik türü ve CSRF çerezi/başlığı birlikte denetlenir. Üretim çerezleri `__Host-`, `Secure`, `HttpOnly`, `SameSite=Lax`; yerel HTTP ayrı davranır. Giriş ve hassas işlemler sınırlanır. Üretimde güvenilecek ters vekil ve ortak okul IP bütçesi ayrıca sınanır.
- Yönetici onay/ret ve rol işlemleri parola teyidi ve işlem günlüğü kullanır. Son erişilebilir yönetici korunur. İlk yönetici yalnızca yerel/operatör CLI ile oluşturulur; e-posta doğrulamasını yine tamamlar.
- Basit SMTP gönderimi aynı API sürecindedir; worker/outbox kurulmaz. Yerelde yalnızca loopback Mailpit; uzak sunucuda doğrulanan STARTTLS. İstek kabulü gerçek teslimat garantisi değildir; SMTP hatasında yeniden gönderme mümkündür. Gerçek SMTP hizmeti henüz seçilmedi.
- React ve mevcut CSS ile sınırlı hesap ekranları taşındı. Bu teslimde yeni bir router/form/durum paketi eklemek gerekmedi. Daha kapsamlı katalog ve yönetim yönlendirmesi kendi tesliminde değerlendirilecek.
- Güvenlik kontrolleri ayrı PostgreSQL test şemaları ve gerçek HTTP istekleriyle; kullanıcı akışı ayrı test şemasında Chrome + Mailpit ile doğrulanır. Hiçbir test gerçek öğrenci e-postasına ileti göndermez.

Başvuru kaynakları: [OWASP kimlik doğrulama](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html), [OWASP oturum yönetimi](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html), [Go Argon2](https://pkg.go.dev/golang.org/x/crypto/argon2), [Mailpit kurulumu](https://mailpit.axllent.org/docs/install/). Bunlar tam güvenlik denetimi veya yayına hazır olma iddiası değildir.

## 3A başlangıç kaydı

Proje sahibi PostgreSQL kurulumunu ve `momerefe/Kabine101` deposunu bildirmiş, GitHub/README/lisans düzenlemelerine izin vermiş ve geliştirmeyi başlatmıştır. Bu talimat, daha önce görüşülen sade teknik temel üzerinden **3A teslimine başlama** yetkisi olarak uygulanmıştır. HTML prototipi görsel referans olarak aynen korunmuştur; sonraki ürün teslimlerinin kabul noktaları devam eder.

## Bu teslimde kullanılanlar

- Go + Chi; tek API uygulaması, JSON slog ve kontrollü kapanış.
- PostgreSQL 18; pgx, sqlc ve golang-migrate. Yerelde kurulu sunucu kullanılır.
- React + TypeScript + Vite; TanStack Query. Arayüz henüz başlangıç/bağlantı ekranıdır.
- OpenAPI'den Go yanıt tipleri ve TypeScript şeması; openapi-fetch ile tipli istemci.
- pnpm; Go testleri, gerçek PostgreSQL üzerinde entegrasyon testi, TypeScript/ESLint ve derleme kontrolleri.

Kesin bağımlılık sürümleri `go.mod`, `go.sum`, `package.json` ve `pnpm-lock.yaml` içindedir. TypeScript 5.9, kod üretimi ve lint araçlarının ortak uyum aralığında olduğu için seçilmiştir. Go araç bağımlılıklarında sqlc'nin kendi CEL/SQLite paketlerinin bulunması uygulamaya kural motoru veya ikinci veritabanı eklenmesi anlamına gelmez.

Router, form, Radix/Tailwind ve diğer arayüz araçları ilgili ekran tesliminde eklenir. Şimdiden kullanılmayan paketler kurulmaz. Native mobil, işçi/zamanlayıcı, kullanım doğrulaması ve çok kurumlu yapı ertelenmiştir.

## Veriler ve erişim

Şema: `accounts`, `sessions`, `account_tokens`, `businesses`, `benefits`, `favorites`, `audit_events`, `platform_settings`. E-posta sahipliği ile öğrenci uygunluğu ayrı alanlardır. Oturum/doğrulama kayıtları yalnızca özetlenmiş token saklayacak biçimdedir; kimlik doğrulama kodu 3B'de yazılacaktır.

Yayın durumu ile tarihten hesaplanacak geçerlilik ayrıdır. Şema tarih aralığı, rol/durum değerleri, yabancı anahtar ve favori benzersizliğini korur. Yayın iş kurallarının tamamı 3C'de sunucuda uygulanacak; bugün yalnızca sağlık/hazırlık API'leri vardır. Öğrenci veya yönetim verisi dışarı açılmaz.

Yerel `kocbenefits_local` hesabı süper kullanıcı değildir, iki proje veritabanının sahibidir. Bu **geliştirme düzenidir**; üretim öncesi migration sahibi ile sınırlı uygulama hesabı ayrılır. PostgreSQL yönetici parolası kaynakta ve `.env` dosyasında saklanmaz; uygulamanın ayrı rastgele parolası yalnızca Git dışında kalan `.env` içindedir.

## Lisans ve GitHub

Kullanıcının verdiği lisans düzenleme yetkisi kapsamında kaynak kodu için MIT seçilmiştir. Lisans, bildirim korunarak kullanım/değiştirme/dağıtıma izin verir. Okul ve işletme adlarına ilişkin marka izni veya gerçek sponsor/veri paylaşım izni sağlamaz. [MIT açıklaması](https://choosealicense.com/licenses/mit/)

CI dosyası test ve derlemeyi tanımlar; otomatik dağıtım yapmaz. Depo görünürlüğü değiştirilmez. Bağlı hesabın depo ayarlarını değiştirme veya gönderme yetkisi yoksa yerel hazırlık tamamlanır ve erişim eksikliği teslimde açıkça belirtilir.

## 3A sonunda açık kalanlar

3A sonunda okul alan adı ve uygunluk kuralı açıktı; yukarıdaki 3B kararıyla kesinleşti. Üretim e-posta hizmeti, dosya deposu, barındırma ve yayın kararı hâlâ kesinleşmedi. `ogrenci.example` yalnızca izole testlerde kullanılır ve üretime taşınamaz.
