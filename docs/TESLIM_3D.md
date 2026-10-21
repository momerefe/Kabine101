# 3D — Öğrenci deneyimi ve avantaj detayları

Tarih: 21 Ekim 2026. Proje sahibi 3D kapsamını onayladı; HTML prototipi tasarım referansı olarak korundu.

## Çalışan akışlar

- **Genel Bakış:** Öne çıkan aktif avantajlar ve kategori erişimi. Ölçülmemiş tasarruf, kullanım veya popülerlik sayısı yoktur.
- **Keşfet:** Yayımlanmış ve iş ortaklığı aktif işletmelerin avantajları, işletme/avantaj metninde arama ve altı kategori filtresi. Aktif, yakında ve süresi dolmuş kayıtlar ayrı etiketlenir.
- **Favoriler:** Hesaba bağlı ekleme/çıkarma; oturum kapanıp açılınca aynı hesapta kalır. Erişimi kaldırılan avantaj, özel alanları olmayan “artık erişilemiyor” kaydı olarak çıkarılabilir. Başka hesabın kimliği istemciden seçilemez.
- **Detay:** Kalıcı `#benefit/<UUID>` adresi, giriş sonrası aynı avantaja dönüş, paylaşım, açıklama, işletme kimliği, koşullar, istisnalar, ürün/şube kapsamı, varsa minimum harcama, tarihler ve doğrulama gereksinimi. Sağlanmayan alanlar gizlenir; kullanım koşulları kapalı menü arkasında değildir.
- **Kullanım:** Yalnızca ilgili kod, bağlantı, QR veya üyelik bilgisi. Kod ve paylaşım için kopyalama; izin yoksa görünür metin üzerinden elle kopyalama. QR gerçek içerikten tarayıcıda üretilir; üçüncü tarafa veri gönderilmez.
- **Profil:** Gerçek ad/e-posta, hesap durumu, rol, parola yenileme talebine erişim ve çıkış.
- **Yardım:** Kullanım açıklamaları ve platform ayarında tanımlıysa destek bilgisi. Bilgi yoksa açıkça belirtilir; adres veya destek ekibi uydurulmaz.
- **Geçmiş:** Doğrulanmış kullanım altyapısı olmadığı için açıklayıcı boş ekran. Kod açmak, QR göstermek veya bağlantıya gitmek geçmiş kaydı oluşturmaz.

## Erişim ve veri sınırları

`internal/student` modülü mevcut Go/Chi uygulamasına eklendi. Tek API, tek PostgreSQL ve şema sürümü **5** korunur; migration yoktur. Mevcut `favorites` tablosu kullanılır. SQL/sqlc ve OpenAPI/TypeScript tipleri güncellendi.

Öğrenci API'si yalnızca oturumu ve hesap uygunluğu doğrulanmış hesaplara açıktır. Yönetici de yayımlanmış öğrenci görünümüne erişebilir; öğrenci uç noktaları yöneticiye dahi taslak açmaz. Yönetim API'si ve taslak önizlemesi ayrı yetki kontrolünü korur.

Liste, detay ve favoriler; iş ortaklığı notlarını, iletişim kişisini, parola/oturum bilgilerini veya kod/bağlantı/QR/kart içeriğini içermez. Kullanım bilgisi ayrı `GET /api/student/benefits/{id}/redemption` isteğinden alınır. Hesap, yayın durumu, işletme aktifliği ve Europe/Istanbul gününe göre tarihler yeniden denetlenir. Bitiş günü dahildir. Askıda, taslak, yakında veya süresi dolmuş avantajdan kullanım içeriği alınamaz. Aynı kilit sırası yönetim işlemleriyle eşzamanlı değişiklikleri korur.

Favori değişiklikleri CSRF korumalı ve tekrarlanabilirdir. Özet yanıtlarında ve bütün API yanıtlarında `no-store` kullanılır. Kullanım içeriği Query veri önbelleğine veya localStorage'a yazılmaz; sayfadan ayrılma, hesap değişimi, odak kaybı ve durum yenilenmesinde kapatılır. Daha önce öğrenci tarafından görülen bir kodu uzaktan geri alma veya tek kullanımlık hak garantisi verilmez.

Listeler filtrelenen ilk 1000 kayıtla sınırlıdır ve sınır aşımı belirtilir. Gelişmiş sayfalama eklenmedi. Tarih/yayın kararlarının asıl kaynağı sunucudur; katalogdaki durum yenilemesi kullanım izninin yerine geçmez. Katalog/detay ön planda 60 saniyede ve odağa dönüşte yenilenir.

## Doğrulama

- Go birim/entegrasyon, vet ve derleme; TypeScript, ESLint, Prettier ve web derlemesi GitHub Actions'ta birlikte denetlenir.
- Altı öğrenci PostgreSQL entegrasyon senaryosu: erişim durumlarının her istekte yeniden denetlenmesi, özel alanların sızmaması, taslakların kapalı kalması, filtreler, hesaplar arası favori ayrımı/oturumlar arası kalıcılık, CSRF, dört kullanım yöntemi, İstanbul tarih sınırları, askı ve ortaklık kapanışı, ayarlar ve 1000 kayıt sınırı.
- `tests/run_student_ui_checks.py`: Kullanıcının açtığı Vite ile Chrome ekran/dokunma emülasyonunda **87 yerleşim kontrolü**. 320/390/768/1024/1440 px, yatay görünüm, altı bölüm, dört yöntem, süresi dolmuş/yakında/eksik detay, arama, kategori, favoriler, hata/yeniden deneme, kopyalama alternatifi, uzun metin ve giriş sonrası dönüş. API yanıtları yalnızca bu ayrı tarayıcı profilinde sentetiktir; geliştirme veritabanı değiştirilmez.
- `TestStudentBrowser` + `tests/run_student_real_checks.py`: GitHub Actions içinde derlenmiş gerçek web arayüzü, gerçek Go API ve ayrı PostgreSQL şemasıyla uçtan uca kabul. Paylaşılan detaydan giriş, gerçek favori kaydı ve yeni oturumda kalıcılık, dört kullanım yöntemi, kopyalama, profil/parola talebi ve boş geçmiş sınanır. Bu test yerel sunucuları başlatmak için kullanılmadı.
- Önceki kimlik ve yönetim API testleri aynı CI'da çalışır. Önceki isteğe bağlı tarayıcı betikleri yeni öğrenci yönlendirmesine uyarlandı; yeniden çalıştırılmayan eski tarayıcı paketi bu teslimin yeni kanıtı olarak gösterilmez.

Yeni öğrenci ekranında fiziksel telefon/Safari testi henüz yapılmadı. Proje sahibinin önceki telefon kontrolü önceki sürüme aittir. Ekran görüntüleri Git dışındaki `tests/artifacts/student/` ve CI kabulünde `tests/artifacts/student-real/` altına yazılır.

## Yerelde inceleme

Backend/frontend süreçlerini proje sahibi başlatır. Yeni API için açık backend terminalinde `Ctrl+C` ardından:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 api -LanAddress 192.168.1.94
```

Frontend açıksa sayfayı yenile. Gerekirse kendin başlat:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 web -LanAddress 192.168.1.94
```

Bilgisayar ve telefonda **http://192.168.1.94:5173/#overview** adresini kullan; telefon aynı Wi-Fi'da olmalıdır. IP değişirse iki komutta da yeni adresi kullan. API yalnızca loopback'te kalır. Yeni migration veya örnek veri yükleme gerekmez.

Onaylanmış/doğrulanmış öğrenciyle giriş yap. Yönetimden aktif iş ortaklığına ait avantaj yayımla; öğrenci kataloğunda görünecektir. Gerçek sponsor anlaşması yoksa test içeriğini açıkça test olarak adlandır. Prototip örnekleri veya otomatik test kayıtları gerçek sponsor olarak eklenmez. Yeni kurulumda frontend bağımlılıkları için `pnpm install --frozen-lockfile` gerekir; bu bilgisayarda QR paketi kuruldu.

Arayüz kontrolünü frontend açıkken tekrarlamak için:

```powershell
$env:KOC_STUDENT_URL = 'http://192.168.1.94:5173'
python tests/run_student_ui_checks.py
Remove-Item Env:KOC_STUDENT_URL
```

## Sınırlar ve sonraki kabul noktası

Gerçek SMTP teslimatı, pilot, üretim HTTPS/hosting, dosya yükleme, doğrulanmış kullanım ve native mobil uygulamalar bu teslimde eklenmedi. Statik QR veya üyelik metninin işletmede kabulü gerçek sponsor anlaşmasına bağlıdır; otomatik doğrulama iddiası yoktur. Çok uzun içerik QR kapasitesini aşarsa açık hata gösterilir.

Sonraki teslim **3E yönetim bütünlüğü ve ölçümler**. Önceden kabul edilen Yönetici Genel Bakış hazırdır; diğer yönetim modülleri ve ölçümler ayrıca değerlendirilecektir.
