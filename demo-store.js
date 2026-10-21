/* Paylaşılan tarayıcı demosu. Veritabanı veya güvenlik sınırı değildir.
 * Yönetim verileri yalnızca bu tarayıcıya kaydedilir; gerçek kişisel veri girmeyin.
 */
(() => {
  'use strict';
  const defaults = window.KOC_CONFIG;
  if (!defaults || !Array.isArray(window.KOC_BENEFITS)) return;
  const clone = value => JSON.parse(JSON.stringify(value));
  const settings = {
    SCHOOL_EMAIL_DOMAIN: defaults.SCHOOL_EMAIL_DOMAIN,
    PLATFORM_NAME: defaults.PLATFORM_NAME,
    supportEmail: '',
    showFeatured: true
  };
  const seedBenefits = window.KOC_BENEFITS.map((item, index) => ({
    ...clone(item),
    ...clone(window.KOC_BENEFIT_DETAILS?.[item.id] || {}),
    businessId: `biz-${item.id}`,
    publicationStatus: 'published',
    publishedAt: '2026-09-01',
    businessNumber: `İŞL-${String(index + 1).padStart(3, '0')}`
  }));
  const seedBusinesses = seedBenefits.map(item => ({
    id: item.businessId, number: item.businessNumber, name: item.business,
    image: item.image, imageAlt: item.imageAlt, category: item.category,
    description: `${item.business}, prototip için hazırlanmış kurgusal bir işletmedir.`,
    contactPerson: '', contact: '', branches: item.branches || '',
    partnershipStatus: 'active', lastContact: '', notes: ''
  }));
  seedBusinesses.push({ id: 'biz-kiyi', number: 'İŞL-007', name: 'Kıyı Atölye (Demo)', image: 'assets/books.svg', imageAlt: 'Kitap illüstrasyonu', category: 'other', description: 'İş ortaklığı görüşmelerini incelemek için örnek potansiyel işletme.', contactPerson: '', contact: '', branches: '', partnershipStatus: 'contacted', lastContact: '2026-10-15', notes: 'Örnek not: teklif yanıtı bekleniyor.' });
  const domain = settings.SCHOOL_EMAIL_DOMAIN;
  const seed = {
    version: 1, settings, businesses: seedBusinesses, benefits: seedBenefits,
    users: [
      { id: 'user-admin', name: defaults.DEMO_ADMIN_NAME, email: `${defaults.DEMO_ADMIN_LOCAL_PART}@${domain}`, role: 'admin', status: 'active', verification: 'approved' },
      { id: 'user-deniz', name: defaults.DEMO_NAME, email: `${defaults.DEMO_EMAIL_LOCAL_PART}@${domain}`, role: 'student', status: 'active', verification: 'approved' },
      { id: 'user-a', name: 'Örnek Öğrenci A', email: `ogrenci.a@${domain}`, role: 'student', status: 'pending', verification: 'pending' },
      { id: 'user-b', name: 'Örnek Öğrenci B', email: `ogrenci.b@${domain}`, role: 'student', status: 'pending', verification: 'pending' },
      { id: 'user-c', name: 'Örnek Öğrenci C', email: `ogrenci.c@${domain}`, role: 'student', status: 'pending', verification: 'pending' }
    ],
    audit: [], views: {}
  };
  let state = clone(seed);
  let persistent = true;
  try {
    const saved = JSON.parse(localStorage.getItem(defaults.DEMO_STORAGE_KEY) || 'null');
    if (saved && saved.version === 1 && ['businesses', 'benefits', 'users', 'audit'].every(key => Array.isArray(saved[key])) && saved.settings && saved.views) {
      state = saved;
    }
  } catch { persistent = false; }

  function save() {
    try {
      localStorage.setItem(defaults.DEMO_STORAGE_KEY, JSON.stringify(state));
      persistent = true;
    } catch { persistent = false; }
    return persistent;
  }

  function today() {
    const parts = new Intl.DateTimeFormat('en-US', { timeZone: defaults.TIME_ZONE, year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date());
    const part = type => parts.find(item => item.type === type).value;
    return `${part('year')}-${part('month')}-${part('day')}`;
  }

  function businessFor(benefit) { return state.businesses.find(item => item.id === benefit.businessId); }

  function statusOf(benefit) {
    if (benefit.publicationStatus === 'draft') return 'draft';
    if (benefit.publicationStatus === 'suspended' || businessFor(benefit)?.partnershipStatus !== 'active') return 'suspended';
    if (benefit.validUntil && benefit.validUntil < today()) return 'expired';
    if (benefit.validFrom && benefit.validFrom > today()) return 'scheduled';
    return 'active';
  }

  function enrich(benefit) {
    const business = businessFor(benefit);
    if (business) Object.assign(benefit, {
      business: business.name, category: business.category, image: business.image,
      imageAlt: business.imageAlt, businessNumber: business.number,
      brandInitial: business.name.charAt(0).toLocaleLowerCase('tr-TR')
    });
    return benefit;
  }
  state.benefits.forEach(enrich);

  function record(type, subject, actor) {
    state.audit.unshift({ id: uid('log'), type, subject, actor: actor || 'Demo yönetici', at: new Date().toISOString() });
  }

  function uid(prefix) {
    const token = window.crypto?.randomUUID?.() || `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 9)}`;
    return `${prefix}-${token}`;
  }

  window.KOC_STORE = {
    state, benefits: state.benefits, businesses: state.businesses, users: state.users,
    settings: state.settings, save, today, statusOf, enrich, businessFor, record, uid,
    get persistent() { return persistent; },
    // Görüntüleme yalnızca bu tarayıcıdaki demo ziyaretidir; doğrulanmış kullanım değildir.
    view(id) { state.views[id] = (Number(state.views[id]) || 0) + 1; save(); }
  };
})();
