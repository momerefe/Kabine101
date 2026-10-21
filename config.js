// Yalnızca arayüz prototipi ayarlarıdır. Güvenlik veya yetkilendirme sağlamaz.
window.KOC_CONFIG = Object.freeze({
  SCHOOL_EMAIL_DOMAIN: 'ogrenci.example', // Gerçek okul alan adı öğrenildiğinde değiştirin.
  DEMO_NAME: 'Deniz Örnek',
  DEMO_EMAIL_LOCAL_PART: 'deniz',
  PLATFORM_NAME: 'KoçBenefits',
  DEMO_ADMIN_LOCAL_PART: 'yonetici',
  DEMO_ADMIN_NAME: 'Ada Yönetici',
  DEMO_STORAGE_KEY: 'kocbenefits.demo.management.v1',
  FAVORITES_STORAGE_KEY: 'kocbenefits.demo.favorites.v1',
  // Tarih karşılaştırması her gün Europe/Istanbul saat dilimine göre yapılır.
  TIME_ZONE: 'Europe/Istanbul'
});
