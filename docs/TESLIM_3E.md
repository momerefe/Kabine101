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

## Kapsam sınırı ve sonraki adım

Gerçek e-posta teslimatı, barındırma, TLS, yedekleme/geri yükleme denemesi ve pilot kullanıcı grubu 3F kapsamında ayrıca değerlendirilecektir. Bu teslim otomatik yayına alma veya yeni bir aşama onayı değildir.
