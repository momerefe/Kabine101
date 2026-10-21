# Teknik kararlar ve ilk teslim

Tarih: 21 Ekim 2026

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

## Sonraki teslimden önce

3B için gerçek okul e-posta alan adı, öğrenci uygunluk kuralı ve yönetici onayının hangi durumda gerektiği belirlenmelidir. Üretim e-posta hizmeti, dosya deposu, barındırma ve yayın kararı bu teslim kapsamında kesinleştirilmemiştir. `ogrenci.example` gerçek giriş kuralı olarak üretime taşınamaz.
