# Masaüstü ve mobil kontrolü — 21 Ekim 2026

Mevcut React uygulaması ve tasarım referansı olan HTML prototipi kontrol edildi. Tasarım dili korundu; yeni özellik, framework veya servis eklenmedi.

## Düzeltilenler

- Dokunmatik tabletlerde yönetici menüsü düğmeyle açılıp kapanır. Seçim ve Escape menüyü kapatır; kapalı menü klavye odağı almaz. Açık menüde Tab/Shift+Tab görünür kontroller arasında dolaşır. Ekran düzeni değiştiğinde menü durumu sıfırlanır.
- Kısa yatay ekranlarda menü kaydırılabilir; daraltılmış masaüstü menüsündeki gereksiz yatay kaydırma çubuğu kaldırıldı.
- Uzun bölüm adları başlık alanını taşırmaz. Uzun kayıtlar ve onay pencereleri ekran içinde kalır. Yönetim form alanları 16 px yazı boyutu kullanır; onay pencerelerinin erişilebilir başlığı vardır.
- Sistem durumu ekranındaki düğme ve dönüş bağlantısına aralık eklendi.
- HTML öğrenci prototipinde 320 px genişlikte dekoratif kartın metin ve düğmenin üzerine binmesi giderildi.

## Kontrol kapsamı

Sonuç: gerçek uygulamada **322 sayfa/ekran boyutu kontrolü**, öğrenci prototipinde **50 kontrol** geçti; JavaScript hatası görülmedi. Önceki HTML prototipi kabul paketi de 38 öğrenci ve 55 yönetici kontrolüyle geçti. Gerçek API üzerinden yönetim işlemleri regresyonu, TypeScript, ESLint, Prettier ve web derlemesi başarılıdır. Yönetim testinde dinamik seçeneklerin yüklenmesi, ekran boyutu değişikliğinin işlenmesi ve menü kapandıktan sonra odağın dönmesi beklenerek zamanlama kaynaklı hatalar giderildi.

| Alan | Kontrol |
| --- | --- |
| Gerçek uygulama | Öğrenci/yönetici girişi, kayıt, parola ve e-posta ekranları, eksik token, sistem durumu, hesaplar; 12 yönetici bölümü |
| Yönetim içeriği | İşletme formu/detayı; kod, bağlantı, QR ve üyelik kartı avantaj formları/detayları; yayın ve öğrenci onay pencereleri |
| Ekranlar | 320×740, 360×800, 390×844, 430×932, 768×1024, 1024×768, 1440×1000, 1920×1080, 844×390; ayrıca 820×1180 dokunmatik menü ve 390×430 kısa pencere |
| Etkileşim | Dokunmayla menü açma/seçme, klavye odağı ve Escape; gerçek test hesaplarıyla giriş/çıkış; işletme/avantaj işlemleri ve arama regresyonu |
| HTML prototipi | 320, 390, 768, 1024 ve 1440 px; altı öğrenci bölümü ve dört avantaj detay durumu |

Ekran taşması, pencere sınırları, kırık görseller ve JavaScript hataları otomatik denetlendi; seçili telefon, yatay ekran ve masaüstü ekran görüntüleri ayrıca incelendi. Geniş tablolar sayfayı taşırmak yerine kendi alanlarında yatay kaydırılır.

Testler Chrome'un ekran boyutu/dokunma emülasyonunda çalışır. Fiziksel iPhone/Android, Safari ve gerçek ekran klavyesi testleri yapılmadı; kısa pencere kontrolü gerçek klavye testinin yerine geçmez. Bu rapor her cihazda kusursuzluk garantisi değildir.

## Tekrarlama

HTML prototipi:

```powershell
python tests/run_browser_checks.py
python tests/prototype_responsive_checks.py
```

Gerçek uygulama için önce README'deki yerel kurulum ve web derlemesi tamamlanmalıdır. Ayrı `_test` veritabanı, Python ve Chrome gerekir:

```powershell
$env:KOC_BROWSER_TEST = '1'
$env:KOC_RESPONSIVE_TEST = '1'
.tools/go/bin/go.exe test -tags=integration ./internal/management -run TestManagementBrowser -count=1 -v
Remove-Item Env:KOC_RESPONSIVE_TEST
Remove-Item Env:KOC_BROWSER_TEST
```

`KOC_RESPONSIVE_TEST` olmadan aynı Go testi yönetim işlemlerinin tarayıcı regresyonunu çalıştırır. Go sistemde kuruluysa `.tools/go/bin/go.exe` yerine `go` kullanılabilir. Chrome yolu `KOC_CHROME` ile değiştirilebilir. Test kendi portlarını, sentetik hesaplarını ve PostgreSQL şemasını oluşturur; sonunda temizler. Geliştirme kayıtları değiştirilmez. Tarayıcı betikleri doğrudan geliştirme sunucusuna bağlanmak için tasarlanmamıştır.

Bu Windows oturumunda yeni Go test yürütülebilir dosyası Application Control tarafından engellendi. Sistem politikası değiştirilmedi. Tarayıcı kontrolleri mevcut derlenmiş gerçek API ile, ayrı test şemasını oluşturan geçici yerel düzenek üzerinden yürütüldü; HTTP yanıtları taklit edilmedi. Bu, yukarıdaki Go komutunun yerelde başarıyla çalıştığı anlamına gelmez.

Ekran görüntüleri ve ayrıntılı raporlar Git dışında `tests/artifacts/responsive/` ve `tests/artifacts/prototype-responsive/` altındadır.

## Kapsam sınırı

3D öğrenci kataloğu ve hesap bazlı favoriler ile 3E'nin tamamlanmamış yönetim ekranları bu görünüm çalışmasında geliştirilmedi. Öğrenci prototipinin başarılı olması bu özelliklerin gerçek API ile tamamlandığı anlamına gelmez. Sunucu yetkilendirmesi, gerçek e-posta teslimi ve yayına alma için önceki teslim raporlarındaki sınırlar geçerlidir.
