# Yönetici onaylı alışverişler ve aylık tasarruf

Proje sahibinin seçimiyle tasarrufun kaynağı yönetici onaylı alışveriş kayıtlarıdır. Öğrencinin kendi beyanıyla tutar girmesi veya kod/QR açılışlarından kullanım çıkarılması bu kapsamda değildir.

## İşleyiş

1. Yönetici **Kullanım Kayıtları** bölümünde öğrenci e-postası, avantaj, alışveriş tarihi, indirim öncesi tutar, ödenen tutar ve fiş/referans bilgisini girer. TRY tutarları en fazla iki ondalık basamakla kabul edilir.
2. Kayıt önce bekler. Yönetici gerçek alışverişi ve ilgili KoçBenefits indiriminin uygulandığını kontrol ederek ayrıca onaylar veya gerekçeyle reddeder. Aynı yönetici kayıt girip inceleyebilir; ikinci bir yönetici zorunluluğu eklenmez.
3. Onaylı tutar öğrencinin **Genel Bakış** sayfasında ilgili ayın işletme bazlı tasarrufuna ve **Kullanım Geçmişi** bölümüne yansır. Örneğin gerçek bir alışverişte indirim öncesi 250,00 TL ve ödenen 200,00 TL kaydedilmişse tasarruf 50,00 TL olur; bu örnek sisteme veri olarak eklenmez.
4. İade veya hata durumunda yönetici gerekçeyle onayı geri alır. Bu kayıt tasarruf ve öğrenci geçmişinden çıkar; yönetim tarafında ve işlem günlüğünde korunur.

Ay, onay tarihinden değil alışveriş tarihinden belirlenir. Geçen aya ait bir alışveriş bugün onaylanırsa bu ayın toplamına girmez. Kullanım Geçmişi önceki aylardaki onaylı kayıtları da gösterir. Onaylı alışveriş bulunmaması sıfır toplamdır; API hatası sıfır tasarruf gibi gösterilmez.

## Kayıt ve erişim kuralları

- Yalnızca uygunluğu onaylı, e-postası doğrulanmış aktif öğrenci için kayıt oluşturulur. İşletmenin aktif, avantajın yayımlanmış ve alışveriş tarihinin avantajın geçerlilik aralığında olması gerekir. Geleceğe kayıt girilemez.
- İndirim öncesi tutar sıfırdan büyük; ödenen tutar sıfır ile indirim öncesi tutar arasında olmalıdır. İşlem başına üst sınır 9.999.999,99 TL'dir. Veritabanında tam kuruş kullanılır; metindeki indirim yüzdesinden tutar hesaplanmaz.
- Aynı işletmede fiş/referans, boşluk ve harf farkları normalize edilerek tekildir. Şubelerde aynı numara kullanılıyorsa referansa şube bilgisi eklenir. Bu kontrol yanlış veya farklı referansla yeniden girişin yerini tutmaz; yönetici belgeyi kontrol etmelidir.
- Alışverişin öğrenci, işletme, avantaj, tarih ve tutarları değiştirilemez. İşletme/avantaj adları kayıt anındaki haliyle korunur. Hata düzeltmek için yanlış kayıt reddedilir veya onayı geri alınır; yeni kayıtta eski kayda atıf içeren ayrı bir düzeltme referansı kullanılır.
- Öğrenci istekleri oturum sahibine göre sınırlandırılır; başka öğrencinin kimliği istekle seçilemez. Fiş/referans ve inceleme gerekçeleri öğrenciye gönderilmez. Tüm yönetim değişiklikleri CSRF ve sunucu yetkilendirmesiyle korunur.
- Liste ekranları en fazla 1000 kayıt döndürür ve daha fazla varsa belirtir. Aylık tasarruf hesabı bu sınıra bağlı değildir; ayın tüm onaylı kayıtlarından hesaplanır.

## Yerel çalıştırma

Bu değişiklik şema sürümünü **6** yapar. Yeni API'yi çalıştırmadan önce, proje kökünde:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 db-up
```

Mevcut API terminalini `Ctrl+C` ile kapatıp yeniden başlat:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 api -LanAddress 192.168.1.94
```

Açık Vite kaynak değişikliklerini alır; tarayıcıyı yenile. Vite kapalıysa ayrı terminalde:

```powershell
powershell -ExecutionPolicy Bypass -File tools/dev.ps1 web -LanAddress 192.168.1.94
```

IP değiştiyse iki komutta da güncel yerel IP kullanılmalıdır. Yerel sunucular proje sahibi tarafından başlatılır; bu geliştirme sırasında ajan başlatmaz. Gerçek veritabanına örnek alışveriş eklenmez.

## Sınırlar

Fiş dosyası yükleme, POS/kasa entegrasyonu, otomatik alışveriş doğrulaması, kısmi iade muhasebesi ve öğrenci başvuru formu eklenmez. Gösterilen tasarruf, yöneticinin kontrol edip onayladığı kayıtlara dayanır. Tam onay geri alma vardır; kısmi düzeltme ayrı ve açık bir kayıt gerektirir.
