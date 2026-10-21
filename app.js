/* KoçBenefits · Aşama 1
 * Saf HTML/CSS/JS. Sunucuya istek gönderilmez, hesap oluşturulmaz.
 * Oturum yalnızca bellekte; favoriler bu tarayıcıda localStorage'da tutulur.
 */
(() => {
  'use strict';

  const store = window.KOC_STORE;
  const config = { ...window.KOC_CONFIG, ...store?.settings };
  const benefits = store?.benefits || window.KOC_BENEFITS;
  const categories = window.KOC_CATEGORIES;
  const authView = document.getElementById('auth-view');
  const appView = document.getElementById('app-view');
  const main = document.getElementById('main-content');
  const toastElement = document.getElementById('toast');
  const navigation = [
    { id: 'overview', label: 'Genel Bakış', mobile: 'Genel Bakış', icon: 'grid' },
    { id: 'discover', label: 'Avantajları Keşfet', mobile: 'Keşfet', icon: 'compass' },
    { id: 'favorites', label: 'Favorilerim', mobile: 'Favoriler', icon: 'heart' },
    { id: 'history', label: 'Kullanım Geçmişi', mobile: 'Geçmiş', icon: 'clock' },
    { id: 'profile', label: 'Profil ve Hesap', mobile: 'Profil', icon: 'user' },
    { id: 'help', label: 'Yardım ve Destek', mobile: 'Yardım', icon: 'help' }
  ];
  let user = null;
  let authMode = 'login';
  let authAudience = 'student';
  let admin;
  let pendingDetailHash = /^#benefit\/[a-zA-Z0-9-]+$/.test(location.hash) ? location.hash : '';
  let route = 'overview';
  let toastTimeout;
  let storageAvailable = true;
  let pendingLogin = false;
  let favorites = new Set();
  const revealed = new Set();

  function escapeHTML(value) {
    return String(value ?? '').replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
  }
  const icon = name => `<svg class="icon" aria-hidden="true"><use href="#i-${name}"/></svg>`;
  const demoTag = '<span class="demo-tag">DEMO</span>';
  const sampleEmail = () => `${config.DEMO_EMAIL_LOCAL_PART}@${config.SCHOOL_EMAIL_DOMAIN}`;
  const categoryFor = id => categories.find(category => category.id === id);
  const methodLabels = { code: 'İndirim kodu', link: 'Avantaj bağlantısı', qr: 'QR kodu', membership: 'Üyelik kartı' };
  const methodIcons = { code: 'copy', link: 'link', qr: 'qr', membership: 'user' };
  const benefitStatus = benefit => store.statusOf(benefit);
  const studentCatalog = () => benefits.filter(item => item.publicationStatus === 'published' && store.businessFor(item)?.partnershipStatus === 'active');
  const statusLabel = { active: 'Aktif', expired: 'Süresi doldu', suspended: 'Askıya alındı', draft: 'Taslak', scheduled: 'Yakında' };
  const statusBadge = benefit => `<span class="status-badge status-${benefitStatus(benefit)}">${statusLabel[benefitStatus(benefit)]}</span>`;
  const brandText = () => config.PLATFORM_NAME === 'KoçBenefits' ? 'Koç<span class="brand-light">Benefits</span>' : escapeHTML(config.PLATFORM_NAME);

  function updateBrand() {
    const brand = document.querySelector('.sidebar .brand > span:last-child');
    brand.innerHTML = `${brandText()}<small>ÖĞRENCİYE AYRICALIK</small>`;
    document.querySelector('.app-footer > span:first-child').innerHTML = `${escapeHTML(config.PLATFORM_NAME)} <span class="footer-dot">•</span> Öğrencilerden, öğrencilere.`;
  }

  function today() {
    const parts = new Intl.DateTimeFormat('en-US', { timeZone: config.TIME_ZONE, year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date());
    const part = type => parts.find(item => item.type === type).value;
    return `${part('year')}-${part('month')}-${part('day')}`;
  }
  const isExpired = benefit => Boolean(benefit.validUntil && benefit.validUntil < today());
  const dateLabel = date => date ? new Intl.DateTimeFormat('tr-TR', { day: 'numeric', month: 'long', year: 'numeric', timeZone: config.TIME_ZONE }).format(new Date(`${date}T12:00:00+03:00`)) : 'Bu demo için tarih belirtilmedi';

  function notify(message) {
    clearTimeout(toastTimeout);
    toastElement.innerHTML = `${icon('check')}<span>${escapeHTML(message)}</span>`;
    toastElement.hidden = false;
    toastTimeout = setTimeout(() => { toastElement.hidden = true; }, 4200);
  }

  function loadFavorites() {
    try {
      const saved = JSON.parse(localStorage.getItem(config.FAVORITES_STORAGE_KEY) || '[]');
      if (Array.isArray(saved)) favorites = new Set(saved.filter(id => benefits.some(benefit => benefit.id === id)));
    } catch { storageAvailable = false; }
  }

  function saveFavorites() {
    try {
      localStorage.setItem(config.FAVORITES_STORAGE_KEY, JSON.stringify([...favorites]));
      return true;
    } catch {
      storageAvailable = false;
      return false;
    }
  }

  function membershipCard() {
    return `<div class="card-back" aria-hidden="true"></div><div class="membership-card" aria-hidden="true"><div class="card-brand">${brandText()}</div><small>ÖĞRENCİYE AYRICALIK</small><div class="membership-card-bottom"><strong>Öğrencilerden, öğrencilere.<br><small>DEMO ÖĞRENCİ KARTI</small></strong><span>✦</span></div></div>`;
  }

  function renderAuth() {
    const isRegister = authMode === 'register';
    const isAdmin = authAudience === 'admin';
    authView.hidden = false;
    appView.hidden = true;
    document.body.classList.remove('is-admin', 'admin-menu-open', 'admin-nav-pinned');
    document.getElementById('menu-scrim').hidden = true;
    document.title = `${isAdmin ? 'Yönetici girişi' : isRegister ? 'Kayıt ol' : 'Giriş yap'} · ${config.PLATFORM_NAME}`;
    authView.innerHTML = `<div class="auth-shell">
      <section class="auth-story" aria-label="KoçBenefits hakkında">
        <div class="brand"><span class="brand-mark">k<span>✦</span></span><span>${brandText()}<small>ÖĞRENCİYE AYRICALIK</small></span></div>
        <div class="auth-story-copy"><div class="eyebrow">ÖĞRENCİLERDEN, ÖĞRENCİLERE.</div><h1>Okullu olmanın<br>bir de <em>bu yanı var.</em></h1><p>Sevdiğin yerler, yeni keşifler ve sana özel avantajlar. Hepsi tek bir yerde.</p></div>
        <div class="auth-visual">${membershipCard()}<div class="auth-floating-note">${icon('gift')}Küçük ayrıcalıklar. Güzel bir okul hayatı.</div></div>
        <p class="auth-story-footer">Koç Okulu 10. sınıf kabinesi tarafından tasarlanan okul projesi.</p>
      </section>
      <section class="auth-form-area" aria-label="Demo giriş ve kayıt">
        <div class="auth-content"><div class="audience-switch" aria-label="Giriş alanı"><button data-auth-audience="student" aria-pressed="${!isAdmin}" class="${!isAdmin ? 'active' : ''}">${icon('user')}Öğrenci</button><button data-auth-audience="admin" aria-pressed="${isAdmin}" class="${isAdmin ? 'active' : ''}">${icon('shield')}Yönetici</button></div>${demoTag}<h2>${isAdmin ? 'Yönetim alanına hoş geldin.' : isRegister ? 'Aramıza hoş geldin.' : 'Yeniden merhaba.'}</h2><p class="auth-description">${isAdmin ? 'Örnek yönetici hesabıyla işletmeleri ve avantajları yönet.' : isRegister ? 'Okul e-postanla başlayıp avantajları keşfet.' : 'Okul e-postanla giriş yap,<br>avantajların seni bekliyor.'}${pendingDetailHash ? '<br>Girişten sonra paylaşılan avantaja döneceksin.' : ''}</p>
          ${isAdmin ? `<form id="admin-auth-form" novalidate><div id="admin-auth-error" class="form-error" role="alert" hidden></div><div class="field"><label for="admin-email">Demo yönetici e-postası</label><input id="admin-email" type="email" maxlength="120" autocomplete="off" placeholder="${escapeHTML(store.users.find(item => item.role === 'admin' && item.status === 'active')?.email || '')}"><small>Yalnızca demo kullanıcı listesinde yönetici rolü verilen aktif örnek hesaplar kabul edilir.</small></div><button type="submit" class="btn btn-primary btn-full">Demo yönetici girişi${icon('arrow')}</button><p class="auth-security">${icon('shield')}Bu giriş parola istemez ve gerçek yetki koruması sağlamaz.</p></form><div class="auth-divider">veya hızlıca göz at</div><button class="btn btn-secondary btn-full" data-action="sample-admin-login">Örnek yöneticiyle incele${icon('arrow')}</button>` : `
          <div class="auth-tabs" aria-label="Giriş veya kayıt seçimi"><button class="auth-tab ${!isRegister ? 'active' : ''}" data-auth-mode="login" aria-pressed="${!isRegister}">Giriş yap</button><button class="auth-tab ${isRegister ? 'active' : ''}" data-auth-mode="register" aria-pressed="${isRegister}">Kayıt ol</button></div>
          <form id="auth-form" class="auth-form" novalidate>
            <div id="auth-error" class="form-error" role="alert" hidden></div>
            ${isRegister ? '<div class="field"><label for="auth-name">Ad soyad</label><input id="auth-name" name="name" type="text" placeholder="Örnek ad soyad" autocomplete="off" maxlength="50" required><small>Prototipte örnek bir ad kullanabilirsin.</small></div>' : ''}
            <div class="field"><label for="auth-email">Okul e-posta adresi</label><input id="auth-email" name="email" type="email" placeholder="${escapeHTML(sampleEmail())}" autocomplete="off" autocapitalize="none" spellcheck="false" maxlength="120" required aria-describedby="email-domain-note"><small id="email-domain-note">Demo alan adı: <strong>@${escapeHTML(config.SCHOOL_EMAIL_DOMAIN)}</strong><br>Gerçek okul alan adı henüz tanımlanmadı; bu değer yalnızca örnektir.</small></div>
            <button type="submit" class="btn btn-primary btn-full">${isRegister ? 'Demo kayıt akışını dene' : 'Demo giriş yap'}${icon('arrow')}</button>
            <p class="auth-security">${icon('shield')}Bu prototip parola istemez ve gerçek hesap oluşturmaz. E-posta gönderimi veya doğrulaması yapılmaz.</p>
          </form>
          <div class="auth-divider">veya hızlıca göz at</div><button class="btn btn-secondary btn-full" data-action="sample-login">Örnek öğrenciyle incele${icon('arrow')}</button>
          `}
          <p class="auth-disclaimer"><strong>Yalnızca arayüz demosu.</strong> Alan adı kontrolü tarayıcıda yapılır ve güvenli kimlik doğrulama sağlamaz. Gerçek sürümde sunucu denetimi ve okul e-postası onayı gerekecek. Gerçek kişisel bilgilerini girmen gerekmez.</p>
        </div><div class="auth-bottom">KoçBenefits · Aşama 1 arayüz prototipi</div>
      </section></div>`;
  }

  function validateEmail(email) {
    // Alt alan adı, benzer alan adı veya sonuna eklenmiş başka bir alan kabul edilmez.
    const trimmed = email.trim();
    if (!/^[A-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Z0-9.-]+$/i.test(trimmed)) return false;
    const [local, domain] = trimmed.split('@');
    return Boolean(local && !local.startsWith('.') && !local.endsWith('.') && !local.includes('..') && domain.toLowerCase() === config.SCHOOL_EMAIL_DOMAIN.toLowerCase());
  }

  function showFormError(id, message, input) {
    const box = document.getElementById(id);
    box.textContent = message;
    box.hidden = false;
    if (input) { input.setAttribute('aria-invalid', 'true'); input.focus(); }
  }

  function login(email, name, role = 'student', id = null) {
    const account = store.users.find(item => item.email.toLowerCase() === email.toLowerCase());
    user = { name, email: email.trim(), role: account?.role || role, id: account?.id || id };
    document.body.classList.toggle('is-admin', user.role === 'admin');
    document.getElementById('admin-menu-button').hidden = user.role !== 'admin';
    document.getElementById('header-search').hidden = user.role === 'admin';
    document.getElementById('panel-label').textContent = user.role === 'admin' ? 'Yönetim paneli' : 'Öğrenci paneli';
    document.querySelector('.account-button').href = user.role === 'admin' ? '#admin/users' : '#profile';
    document.querySelector('.mobile-brand').href = user.role === 'admin' ? '#admin/overview' : '#overview';
    revealed.clear();
    authView.hidden = true;
    appView.hidden = false;
    authView.innerHTML = '';
    const destination = pendingDetailHash || (user.role === 'admin' ? '#admin/overview' : '#overview');
    pendingDetailHash = '';
    if (location.hash === destination) renderRoute();
    else location.hash = destination;
  }

  function logout() {
    user = null;
    pendingLogin = false;
    revealed.clear();
    clearTimeout(toastTimeout);
    toastElement.hidden = true;
    document.getElementById('global-search').value = '';
    authMode = 'login';
    authAudience = 'student';
    pendingDetailHash = '';
    admin.setMenu(false);
    document.getElementById('admin-dialog').close();
    history.replaceState(null, '', `${location.pathname}${location.search}`);
    renderAuth();
    window.scrollTo(0, 0);
    document.getElementById('auth-email').focus({ preventScroll: true });
  }

  function renderNavigation(activeRoute) {
    if (user.role === 'admin') {
      admin.navigation(activeRoute.replace('admin/', ''));
      document.getElementById('header-name').textContent = user.name.split(' ')[0];
      document.getElementById('header-avatar').textContent = user.name.charAt(0).toLocaleUpperCase('tr-TR');
      return;
    }
    document.querySelector('.sidebar').setAttribute('aria-label', 'Öğrenci ana menüsü');
    document.querySelector('.sidebar .brand').href = '#overview';
    document.querySelector('.sidebar-label').textContent = 'ÖĞRENCİ PANELİ';
    document.getElementById('desktop-nav').innerHTML = navigation.map(item => `<a href="#${item.id}" class="nav-link ${activeRoute === item.id ? 'active' : ''}" ${activeRoute === item.id ? 'aria-current="page"' : ''}>${icon(item.icon)}<span>${item.label}</span>${item.id === 'history' ? demoTag : ''}</a>`).join('');
    document.getElementById('mobile-nav').innerHTML = navigation.map(item => `<a href="#${item.id}" data-route="${item.id}" class="${activeRoute === item.id ? 'active' : ''}" ${activeRoute === item.id ? 'aria-current="page"' : ''} ${item.id === 'history' ? 'aria-label="Kullanım geçmişi, demo bilgi ekranı"' : ''}>${item.id === 'history' ? '<span class="mobile-demo-dot">DEMO</span>' : ''}${icon(item.icon)}<span>${item.mobile}</span></a>`).join('');
    const label = navigation.find(item => item.id === activeRoute)?.label || 'Avantaj detayı';
    document.getElementById('breadcrumb-current').textContent = label;
    document.getElementById('header-name').textContent = user.name.split(' ')[0];
    document.getElementById('header-avatar').textContent = user.name.charAt(0).toLocaleUpperCase('tr-TR');
  }

  function favoriteButton(benefit, detail = false) {
    const selected = favorites.has(benefit.id);
    return `<button class="${detail ? 'detail-save' : 'favorite-toggle'}" data-favorite="${benefit.id}" aria-pressed="${selected}" aria-label="${escapeHTML(benefit.business)}: ${selected ? 'favorilerden çıkar' : 'favorilere ekle'}" title="${selected ? 'Favorilerden çıkar' : 'Favorilere ekle'}">${icon('heart')}${detail ? `<span>${selected ? 'Kaydedildi' : 'Kaydet'}</span>` : ''}</button>`;
  }

  function benefitCard(benefit) {
    const category = categoryFor(benefit.category);
    const expired = isExpired(benefit);
    const state = benefitStatus(benefit);
    return `<article class="benefit-card ${expired ? 'is-expired' : ''}" data-card="${benefit.id}">
      <a class="card-image-link" href="#benefit/${benefit.id}" tabindex="-1" aria-hidden="true"><img src="${escapeHTML(benefit.image)}" alt="" loading="lazy" decoding="async" width="720" height="420"><span class="offer-badge"><strong>${escapeHTML(benefit.discount)}</strong>${escapeHTML(benefit.badge)}</span>${state !== 'active' ? `<span class="expired-label">${statusLabel[state]}</span>` : ''}</a>
      ${favoriteButton(benefit)}
      <div class="card-content"><div class="card-meta">${category.short}${demoTag}</div><h3><a href="#benefit/${benefit.id}">${escapeHTML(benefit.business)}</a></h3><p>${escapeHTML(benefit.title)}</p><div class="card-bottom"><span class="card-method">${icon(methodIcons[benefit.method])}${methodLabels[benefit.method]}</span>${icon('arrow')}</div></div>
    </article>`;
  }

  function pageHeading(title, description, aside = '') {
    return `<div class="page-heading"><div><h1>${title}</h1><p>${description}</p></div>${aside}</div>`;
  }

  function renderOverview() {
    const featured = config.showFeatured === false ? [] : studentCatalog().filter(benefit => benefit.featured && benefitStatus(benefit) === 'active');
    main.innerHTML = `<div class="welcome-heading"><div class="greeting-line">${icon('gift')}Öğrencilerden, öğrencilere.</div>${pageHeading(`Merhaba, ${escapeHTML(user.name.split(' ')[0])}.`, 'Küçük ayrıcalıklar, güzel keşifler. Bugün ne yapmak istersin?', `<span class="quiet-chip">${icon('shield')}Okuluna özel bir deneyim</span>`)}</div>
      <section class="hero" aria-label="KoçBenefits'i keşfet"><div class="hero-copy"><div class="eyebrow">KOÇBENEFITS İLE TANIŞ</div><h2>Okullu olmanın<br>bir de <em>bu yanı var.</em></h2><p>Kahve molandan hafta sonu planına,<br>sana özel avantajlar burada.</p><a href="#discover" class="btn btn-primary">Avantajları keşfet${icon('arrow')}</a></div><div class="hero-art">${membershipCard()}<span class="sparkle" aria-hidden="true">✦</span><span class="sparkle small" aria-hidden="true">✧</span><span class="hero-caption">BİRLİKTE, DAHA AVANTAJLI.</span></div></section>
      <section aria-labelledby="category-heading"><div class="section-heading"><h2 id="category-heading">Ne keşfetmek istersin?</h2></div><div class="category-grid">${categories.map(category => `<a href="#discover?category=${category.id}" class="category-tile" aria-label="${category.name}">${icon(category.icon)}<span>${category.short}</span></a>`).join('')}</div></section>
      <section aria-labelledby="featured-heading"><div class="section-heading"><div><h2 id="featured-heading">Bir göz at, sevebilirsin.</h2><p class="section-subtitle">Senin için seçtiğimiz örnek avantajlar.</p></div><a class="text-link" href="#discover">Tümünü gör${icon('arrow')}</a></div>${featured.length ? `<div class="benefit-grid overview-featured">${featured.map(benefitCard).join('')}</div>` : emptyState('clock', 'Yeni avantajlar için takipte kal.', 'Öne çıkan örnek avantajların süresi dolmuş olabilir.', '<a href="#discover" class="btn btn-primary">Kataloğu incele</a>')}</section>
      <aside class="how-strip">${icon('ticket')}<div><h3>Bir avantajı kullanmak çok kolay.</h3><p>Keşfet. Koşulları incele. İşletmeye uygun yöntemi seç.</p></div><a href="#help" class="text-link">Nasıl çalışır?${icon('arrow')}</a></aside>
      <p class="demo-footnote">${icon('info')}Buradaki tüm işletmeler ve indirimler örnektir. Gerçek sponsorlukları temsil etmez.</p>`;
  }

  function emptyState(iconName, heading, description, action = '') {
    return `<div class="empty-state"><div class="empty-icon">${icon(iconName)}</div><h2>${heading}</h2><p>${description}</p>${action}</div>`;
  }

  function renderDiscover(params) {
    const category = categories.some(item => item.id === params.get('category')) ? params.get('category') : 'all';
    main.innerHTML = `${pageHeading('Avantajları Keşfet', 'Yeni favorini bul. Sana uygun avantajı kolayca keşfet.')}
      <section class="catalog-toolbar" aria-label="Avantaj arama ve filtreleme"><div class="catalog-search">${icon('search')}<label class="sr-only" for="catalog-query">İşletme veya avantaj ara</label><input id="catalog-query" type="search" placeholder="İşletme veya avantaj ara…" maxlength="80" value="${escapeHTML(params.get('q') || '')}"></div><div class="filter-row" role="group" aria-label="Kategori filtresi"><button class="filter-chip ${category === 'all' ? 'active' : ''}" data-category="all" aria-pressed="${category === 'all'}">Tümü</button>${categories.map(item => `<button class="filter-chip ${category === item.id ? 'active' : ''}" data-category="${item.id}" aria-pressed="${category === item.id}">${item.name}</button>`).join('')}</div></section>
      <div id="catalog-results"></div><p class="demo-footnote">${icon('info')}Tüm içerikler demo. Süresi dolan avantajlar etiketlenir ve kullanıma kapatılır.</p>`;
    updateResults();
  }

  function updateResults() {
    const input = document.getElementById('catalog-query');
    if (!input) return;
    const query = input.value.trim().toLocaleLowerCase('tr-TR');
    const selected = main.querySelector('[data-category].active')?.dataset.category || 'all';
    const filtered = studentCatalog().filter(benefit => {
      const text = `${benefit.business} ${benefit.title} ${categoryFor(benefit.category).name}`.toLocaleLowerCase('tr-TR');
      return (selected === 'all' || benefit.category === selected) && text.includes(query);
    }).sort((a, b) => Number(isExpired(a)) - Number(isExpired(b)));
    const filteredTitle = selected === 'all' ? 'Tüm avantajlar' : categoryFor(selected).name;
    document.getElementById('catalog-results').innerHTML = `<div class="results-heading"><h2>${filteredTitle}</h2><span role="status" aria-live="polite">${filtered.length} örnek avantaj</span></div>${filtered.length ? `<div class="benefit-grid">${filtered.map(benefitCard).join('')}</div>` : emptyState('search', 'Henüz bir eşleşme yok.', 'Farklı bir kelime deneyebilir veya kategori filtresini temizleyebilirsin.', '<button class="btn btn-secondary" data-action="reset-filters">Aramayı ve filtreyi temizle</button>')}`;
    // Geri dönüşte aynı arama ve kategori korunur. Her tuş vuruşu geçmişe yeni kayıt eklemez.
    const params = new URLSearchParams();
    if (selected !== 'all') params.set('category', selected);
    if (input.value.trim()) params.set('q', input.value.trim());
    const queryString = params.toString();
    history.replaceState(null, '', `#discover${queryString ? `?${queryString}` : ''}`);
  }

  function renderFavorites() {
    const selected = benefits.filter(benefit => favorites.has(benefit.id) && benefit.publicationStatus !== 'draft');
    main.innerHTML = `${pageHeading('Favorilerim', 'Beğendiğin avantajlar, yeniden keşfetmen için burada.')}${selected.length ? `<div class="results-heading"><h2>Kaydettiklerin</h2><span>${selected.length} örnek avantaj</span></div><div class="benefit-grid">${selected.map(benefitCard).join('')}</div>` : emptyState('heart', 'Güzel keşiflerini burada biriktir.', 'Bir avantajın kalbine dokun. Daha sonra kolayca ulaşmak istediğin tüm avantajları burada bulabilirsin.', `<a href="#discover" class="btn btn-primary">Avantajları keşfet${icon('arrow')}</a>`)}<p class="demo-footnote">${icon('info')}${storageAvailable ? 'Favoriler yalnızca bu tarayıcıda saklanır. Gerçek bir hesaba bağlı değildir; demo öğrenciler arasında ortaktır.' : 'Tarayıcı depolaması kullanılamıyor. Favoriler yalnızca bu açık oturum boyunca tutulacak.'}</p>`;
  }

  function renderHistory() {
    main.innerHTML = `${pageHeading('Kullanım Geçmişi', 'Doğrulanmış avantaj kullanımlarının yer alacağı bölüm.', demoTag)}<div class="notice history-notice">${icon('info')}<div><strong>Bu prototipte gerçek kullanım takibi bulunmuyor.</strong><br>Bu bölüm, gelecekteki işleyişi açıklayan bir demo bilgi ekranıdır.</div></div>${emptyState('clock', 'Henüz doğrulanmış bir kullanım yok.', 'Kod görüntülemek, bağlantı açmak veya QR göstermek bir avantajın kullanıldığını kanıtlamaz. Bu işlemler burada kayıt oluşturmaz.', '<a href="#discover" class="btn btn-secondary">Avantajlara göz at</a>')}<div class="history-explanation"><h3>Gerçek kullanım nasıl kaydedilecek?</h3><p>İşletmeden güvenilir bir kullanım onayı alınmasını sağlayan yöntem sonraki aşamada birlikte kararlaştırılacak. Bu prototip kullanım sayısı, tasarruf miktarı veya örnek geçmiş kayıtları üretmez.</p></div>`;
  }

  function renderProfile() {
    main.innerHTML = `${pageHeading('Profil ve Hesap Ayarları', 'Örnek hesap bilgilerini görüntüle ve demo profilini düzenle.')}<div class="profile-grid"><section class="surface profile-summary" aria-label="Örnek öğrenci profili"><span class="avatar">${escapeHTML(user.name.charAt(0).toLocaleUpperCase('tr-TR'))}</span><h2>${escapeHTML(user.name)}</h2><p class="profile-email">${escapeHTML(user.email)}</p>${demoTag}<hr><p>Bu bilgiler yalnızca açık demo oturumunda kullanılır. Sayfayı yenilediğinde yeniden giriş yapman gerekir.</p></section><section class="surface"><h2>Hesap bilgileri</h2><form id="profile-form" novalidate><div id="profile-error" class="form-error" role="alert" hidden></div><div class="field"><label for="profile-name">Ad soyad</label><input id="profile-name" name="name" value="${escapeHTML(user.name)}" maxlength="50" autocomplete="off" required><small>Gerçek adını kullanmak zorunda değilsin.</small></div><div class="field"><label for="profile-email">Okul e-posta adresi</label><input id="profile-email" value="${escapeHTML(user.email)}" type="email" readonly><small>Demo oturumunun e-posta adresi. E-posta doğrulanmadı.</small></div><div class="field"><label for="allowed-domain">İzin verilen örnek alan adı</label><input id="allowed-domain" value="@${escapeHTML(config.SCHOOL_EMAIL_DOMAIN)}" readonly><small>Gerçek okul alan adı bilinmiyor. Geliştirme ayarı config.js dosyasında.</small></div><button class="btn btn-primary" type="submit">Demo değişiklikleri kaydet${icon('check')}</button></form></section></div><div class="notice profile-note">${icon('shield')}<div><strong>Demo oturumu, gerçek hesap güvenliği sağlamaz.</strong><br>Bu aşamada parola, e-posta doğrulaması veya hesap silme servisi yoktur. Gerçek kayıt ve güvenli erişim sonraki aşamanın kapsamındadır.</div></div><section class="surface logout-card"><div><h2>Oturum</h2><p>Çıkış yapmak örnek profil bilgilerini bellekten temizler. Tarayıcıdaki demo favorileri korunur.</p></div><button class="btn btn-secondary" data-action="logout">${icon('logout')}Çıkış yap</button></section>`;
  }

  function renderHelp() {
    const questions = [
      ['Avantajlardan nasıl yararlanabilirim?', 'Avantajları Keşfet bölümünden bir işletme seç. Detay sayfasındaki kullanım koşullarını ve tarihi incele. İşletmeye göre indirim kodu, bağlantı veya QR gösterilir. Bu prototipte bunların tamamı demo amaçlıdır.'],
      ['İndirim kodunu nasıl kopyalarım?', 'Kodla çalışan bir avantajın detayında “Demo kodunu görüntüle” butonuna, ardından “Kodu kopyala” butonuna bas. Tarayıcı kopyalamaya izin vermezse kod seçilir ve elle kopyalaman istenir. Demo kodları gerçek işletmelerde geçerli değildir.'],
      ['Koşulları ve son geçerlilik tarihini nerede bulurum?', 'Her avantajın detayında “Kullanım koşulları” alanı ve varsa geçerlilik tarihi yer alır. Tarih belirtilmemişse bu durum açıkça yazılır. Süresi dolmuş avantajlar katalogda etiketlenir ve kullanılamaz.'],
      ['Kod veya QR görüntülediğimde kullanılmış sayılır mı?', 'Hayır. Kodun açılması, bağlantıya tıklanması ya da QR gösterilmesi kullanım onayı değildir. Prototip bu işlemleri kullanım geçmişine kaydetmez. Gerçek takip için işletme tarafından güvenilir doğrulama yöntemi gereklidir.'],
      ['Favorilerim nerede tutuluyor?', 'Favoriler bu tarayıcının yerel depolamasında tutulur. Gerçek bir öğrenci hesabıyla ilişkilendirilmez ve demo oturumları arasında ortaktır. Tarayıcı verilerini silersen favoriler de silinir. Depolama engellenmişse sadece açık oturum boyunca korunur.'],
      ['Okul e-postam gerçekten doğrulanıyor mu?', `Hayır. Bu prototip yalnızca @${escapeHTML(config.SCHOOL_EMAIL_DOMAIN)} örnek alan adını tarayıcıda kontrol eder. Bir e-posta göndermez ve adresin sana ait olduğunu doğrulamaz. Gerçek uygulamada sunucu kontrolü ve e-posta onayı ayrıca geliştirilecek.`],
      ['Bir sorunla karşılaşırsam ne yapmalıyım?', 'Gerçek bir destek ekibi veya iletişim kanalı henüz tanımlanmadı. Bu aşamada tasarım değerlendirmesi için karşılaştığın sorunu not alabilirsin. Yardım talebi gönderme ya da destek kaydı oluşturma işlevi bulunmuyor.']
    ];
    main.innerHTML = `${pageHeading('Yardım ve Destek', 'Aradığın yanıtı bul, avantajları kolayca keşfet.')}<div class="help-layout"><section aria-label="Sık sorulan sorular"><div class="faq-list">${questions.map(([question, answer], index) => `<details ${index === 0 ? 'open' : ''}><summary>${question}</summary><p>${answer}</p></details>`).join('')}</div></section><aside class="surface support-card">${icon('help')}<h2>Bir sorunun mu var?</h2><p>Bu prototipte henüz bir destek kanalı tanımlanmadı. Gerçek iletişim bilgileri belirlendiğinde bu alanda paylaşılacak.</p>${demoTag}</aside></div><div class="help-footer">${icon('info')}Bu uygulama Koç Okulu 10. sınıf kabinesi için hazırlanan bir arayüz prototipidir. Görünen işletmeler, indirimler ve kullanıcı bilgileri örnektir.</div>`;
  }

  function redemptionContent(benefit) {
    const state = benefitStatus(benefit);
    if (state !== 'active') return `<button class="btn btn-subtle btn-full" disabled>${statusLabel[state]}</button><p class="redeem-note">${icon('clock')}${state === 'draft' ? 'Bu avantaj yalnızca yönetici önizlemesinde görünür.' : state === 'scheduled' ? 'Bu örnek avantajın geçerlilik dönemi henüz başlamadı.' : 'Bu örnek avantaj kullanıma açık değil.'}</p>`;
    const isRevealed = revealed.has(benefit.id);
    if (benefit.method === 'code') return `${isRevealed ? `<div class="code-box"><label for="discount-code">DEMO İNDİRİM KODUN</label><input id="discount-code" type="text" value="${escapeHTML(benefit.code)}" readonly aria-label="Demo indirim kodu"></div><button class="btn btn-primary btn-full copy-button" data-action="copy-code" data-id="${benefit.id}">${icon('copy')}Kodu kopyala</button><p id="copy-hint" class="copy-hint" role="status" hidden></p>` : `<button class="btn btn-primary btn-full" data-action="reveal" data-id="${benefit.id}">${icon('copy')}Demo kodunu görüntüle</button>`}<p class="redeem-note">${icon('info')}Bu kod örnektir; işletmede geçerli değildir. Görüntülemek veya kopyalamak kullanım kaydı oluşturmaz.</p>`;
    if (benefit.method === 'link') {
      let safe = false;
      try { safe = ['https:', 'http:'].includes(new URL(benefit.url).protocol); } catch { /* Eksik taslak bağlantısı. */ }
      return safe ? `<a href="${escapeHTML(benefit.url)}" target="_blank" rel="noopener noreferrer" class="btn btn-primary btn-full">${icon('link')}Örnek bağlantıyı aç</a><p class="redeem-note">${icon('info')}Yeni sekmede örnek bağlantı açılır. Gerçek indirim veya kullanım kaydı oluşturulmaz.</p>` : '<p class="form-error">Örnek bağlantı henüz tanımlanmamış veya geçersiz.</p>';
    }
    if (benefit.method === 'membership') return `${isRevealed ? `<div class="demo-member-pass" tabindex="-1"><span class="demo-tag">GEÇERSİZ ÖRNEK KART</span><strong>${escapeHTML(benefit.membershipLabel || 'Demo öğrenci kartı')}</strong><span>${escapeHTML(user.name)}</span><small>Kimlik veya üyelik doğrulaması yapmaz.</small></div>` : `<button class="btn btn-primary btn-full" data-action="reveal" data-id="${benefit.id}">${icon('user')}Demo üyelik kartını göster</button>`}<p class="redeem-note">${icon('info')}Bu kart yalnızca görsel bir örnektir; üyelik, giriş hakkı veya indirim sağlamaz.</p>`;
    return `${isRevealed ? `<div class="qr-box"><img src="assets/demo-qr.svg" width="180" height="180" alt="DEMO-KOCBENEFITS metnini içeren örnek QR kodu"><p>Örnek içerik: <strong>DEMO-KOCBENEFITS</strong></p></div>` : `<button class="btn btn-primary btn-full" data-action="reveal" data-id="${benefit.id}">${icon('qr')}Demo QR kodunu göster</button>`}<p class="redeem-note">${icon('info')}Bu QR gerçek bir hak sağlamaz; işletme doğrulaması veya kullanım takibi yapmaz.</p>`;
  }

  function renderBenefit(id) {
    const benefit = benefits.find(item => item.id === id);
    if (!benefit || (benefit.publicationStatus === 'draft' && user.role !== 'admin')) return renderNotFound();
    const state = benefitStatus(benefit);
    const isAdmin = user.role === 'admin';
    if (!isAdmin) store.view(id);
    document.title = `${benefit.business} · ${config.PLATFORM_NAME}`;
    const facts = [
      ['Geçerli ürün / hizmetler', benefit.products], ['Geçerli şubeler', benefit.branches],
      ['İstisnalar', benefit.exceptions], ['Minimum harcama', benefit.minimumSpend != null ? `${new Intl.NumberFormat('tr-TR').format(benefit.minimumSpend)} TL (demo koşul)` : ''],
      ['Gerekli öğrenci doğrulaması', benefit.verification]
    ].filter(([, value]) => value);
    const supportEmail = benefit.supportEmail || store.settings.supportEmail;
    main.innerHTML = `<div class="detail-toolbar"><a href="${isAdmin ? '#admin/benefits' : lastCatalogHash}" class="back-link" ${!isAdmin ? 'data-action="back-to-catalog"' : ''}>${icon('left')}${isAdmin ? 'Yönetime dön' : 'Avantajlara dön'}</a><div class="detail-toolbar-actions">${isAdmin ? `<a class="btn btn-secondary" href="#admin/benefits?edit=${benefit.id}">${icon('edit')}Düzenle</a>` : ''}<button class="btn btn-secondary share-button" data-action="share-benefit" data-id="${benefit.id}">${icon('link')}Bağlantıyı kopyala</button></div></div><div id="share-fallback" hidden><label for="share-url">Paylaşılabilir demo bağlantısı</label><input id="share-url" readonly><p role="status">Otomatik kopyalama kullanılamıyor. Seçili bağlantıyı elle kopyalayabilirsin.</p></div>${isAdmin ? '<div class="admin-demo-notice">Yönetici önizlemesi · Taslaklar öğrenci kataloğunda görünmez.</div>' : ''}<div class="detail-layout advantage-detail" data-benefit-state="${state}"><section class="detail-summary"><div class="detail-business-row"><span class="business-monogram" aria-hidden="true">${escapeHTML(benefit.brandInitial)}</span><div><div class="detail-business-name">${escapeHTML(benefit.business)}</div><div class="card-meta">${escapeHTML(benefit.businessNumber || benefit.businessId)} · ${categoryFor(benefit.category).name}</div></div>${!isAdmin ? favoriteButton(benefit, true) : ''}</div><div class="detail-status-line">${statusBadge(benefit)}${demoTag}</div><h1 class="detail-title">${escapeHTML(benefit.title)}</h1>${benefit.description ? `<p class="detail-description">${escapeHTML(benefit.description)}</p>` : ''}</section><aside class="redeem-panel ${state !== 'active' ? 'expired-panel' : ''}" aria-label="Demo avantajı kullan">${demoTag}<div class="redeem-discount">${escapeHTML(benefit.discount)}</div><h2>${methodLabels[benefit.method]} ile kullanım</h2>${benefit.instructions?.length ? `<ol class="usage-steps">${benefit.instructions.map(step => `<li>${escapeHTML(step)}</li>`).join('')}</ol>` : ''}<div id="redemption-content">${redemptionContent(benefit)}</div></aside><img class="detail-image" src="${escapeHTML(benefit.image)}" alt="${escapeHTML(benefit.imageAlt)}" width="720" height="420"><section class="conditions"><h2>Kullanım koşulları</h2>${benefit.conditions?.length ? `<ul>${benefit.conditions.map(condition => `<li>${escapeHTML(condition)}</li>`).join('')}</ul>` : '<p class="detail-description">Bu taslak için koşullar henüz girilmedi.</p>'}${facts.length ? `<dl class="detail-facts">${facts.map(([label, value]) => `<div><dt>${label}</dt><dd>${escapeHTML(value)}</dd></div>`).join('')}</dl>` : ''}${benefit.validFrom ? `<p class="date-info">${icon('clock')}Başlangıç: ${dateLabel(benefit.validFrom)}</p>` : ''}<p class="date-info ${state === 'expired' ? 'expired' : ''}">${icon('clock')}${state === 'expired' ? 'Süresi doldu: ' : benefit.validUntil ? 'Son geçerlilik: ' : ''}${dateLabel(benefit.validUntil)}</p>${supportEmail ? `<div class="detail-support"><strong>Demo iletişim bilgisi</strong><p>${escapeHTML(supportEmail)}</p><small>Gerçek destek hizmeti tanımlanmamıştır.</small></div>` : ''}</section></div><p class="demo-footnote">${icon('info')}${escapeHTML(benefit.business)} kurgusal bir işletmedir. Bu sayfa gerçek bir sponsorluk veya indirim sunmaz.</p>`;
  }

  function renderNotFound() {
    main.innerHTML = emptyState('search', 'Bu sayfayı bulamadık.', 'Bağlantı değişmiş veya örnek avantaj kaldırılmış olabilir.', '<a href="#discover" class="btn btn-primary">Avantajlara dön</a>');
  }

  let lastCatalogHash = '#discover';
  function renderRoute() {
    if (!user) return;
    const [path, search = ''] = location.hash.slice(1).split('?');
    route = path || 'overview';
    const isAdmin = user.role === 'admin';
    const activeRoute = route.startsWith('benefit/') ? (isAdmin ? 'admin/benefits' : 'discover') : route;
    const item = navigation.find(nav => nav.id === activeRoute);
    document.title = `${item?.label || 'Sayfa bulunamadı'} · ${config.PLATFORM_NAME}`;
    renderNavigation(activeRoute);
    if (route.startsWith('admin/')) {
      if (isAdmin) admin.render(route, new URLSearchParams(search));
      else main.innerHTML = emptyState('shield', 'Bu bölüm yönetici demosuna ait.', 'Öğrenci demo oturumundan yönetim ekranlarına geçilemez. Bu ayrım gerçek sunucu güvenliği sağlamaz.', '<a href="#overview" class="btn btn-primary">Öğrenci paneline dön</a>');
    }
    else if (isAdmin && !route.startsWith('benefit/')) { location.hash = 'admin/overview'; return; }
    else if (route === 'overview') renderOverview();
    else if (route === 'discover') { renderDiscover(new URLSearchParams(search)); lastCatalogHash = location.hash; }
    else if (route === 'favorites') renderFavorites();
    else if (route === 'history') renderHistory();
    else if (route === 'profile') renderProfile();
    else if (route === 'help') renderHelp();
    else if (route.startsWith('benefit/')) renderBenefit(route.slice(8));
    else renderNotFound();
    if (route === 'overview' && config.showFeatured === false) main.querySelector('[aria-labelledby="featured-heading"]')?.remove();
    if (!route.startsWith('admin/')) {
      main.classList.remove('page-enter'); void main.offsetWidth; main.classList.add('page-enter');
    }
    window.scrollTo(0, 0);
    main.focus({ preventScroll: true });
  }

  async function copyCode(id, button) {
    const benefit = benefits.find(item => item.id === id);
    if (!benefit || benefitStatus(benefit) !== 'active' || benefit.method !== 'code' || !revealed.has(id)) return;
    let copied = false;
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(benefit.code);
        copied = true;
      }
    } catch { /* file:// ve izin reddinde aşağıdaki elle kopyalama akışını kullan. */ }
    if (!document.getElementById('discount-code')) return;
    if (!copied) {
      const field = document.getElementById('discount-code');
      field.focus();
      field.select();
      field.setSelectionRange(0, field.value.length);
      try { copied = document.execCommand('copy'); } catch { copied = false; }
    }
    if (copied) {
      button.innerHTML = `${icon('check')}Kod kopyalandı`;
      notify('Demo kodu panoya kopyalandı.');
    } else {
      const hint = document.getElementById('copy-hint');
      hint.textContent = 'Otomatik kopyalama kullanılamıyor. Seçili kodu Ctrl+C / ⌘C ile veya telefonda basılı tutup kopyala.';
      hint.hidden = false;
    }
  }

  async function shareBenefit(id, button) {
    const url = new URL(location.href);
    url.hash = `benefit/${id}`;
    let copied = false;
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(url.href);
        copied = true;
      }
    } catch { /* Pano izni yoksa seçilebilir bağlantı sun. */ }
    if (!button.isConnected) return;
    if (!copied) {
      const box = document.getElementById('share-fallback');
      box.hidden = false;
      const input = document.getElementById('share-url');
      input.value = url.href;
      input.focus(); input.select();
      try { copied = document.execCommand('copy'); } catch { copied = false; }
      if (copied) { box.hidden = true; button.focus({ preventScroll: true }); }
    }
    if (copied) {
      button.innerHTML = `${icon('check')}Bağlantı kopyalandı`;
      notify(location.protocol === 'file:' ? 'Yerel demo bağlantısı kopyalandı; başka bilgisayarda aynı proje dosyaları gerekir.' : 'Demo avantaj bağlantısı kopyalandı. Girişten sonra açılır.');
    }
  }

  document.addEventListener('click', event => {
    const audienceButton = event.target.closest('[data-auth-audience]');
    if (audienceButton && !user) { authAudience = audienceButton.dataset.authAudience; renderAuth(); document.querySelector(`[data-auth-audience="${authAudience}"]`).focus(); return; }
    const authTab = event.target.closest('[data-auth-mode]');
    if (authTab) { authMode = authTab.dataset.authMode; renderAuth(); document.querySelector(`[data-auth-mode="${authMode}"]`).focus(); return; }
    const favoriteControl = event.target.closest('[data-favorite]');
    if (favoriteControl && user?.role === 'student') {
      const id = favoriteControl.dataset.favorite;
      if (!benefits.some(benefit => benefit.id === id)) return;
      const shouldAdd = !favorites.has(id);
      if (shouldAdd) favorites.add(id); else favorites.delete(id);
      const saved = saveFavorites();
      if (route === 'favorites') {
        renderFavorites();
        main.focus({ preventScroll: true });
      } else {
        document.querySelectorAll(`[data-favorite="${id}"]`).forEach(button => {
          const business = benefits.find(benefit => benefit.id === id).business;
          button.setAttribute('aria-pressed', String(shouldAdd));
          button.setAttribute('aria-label', `${business}: ${shouldAdd ? 'favorilerden çıkar' : 'favorilere ekle'}`);
          button.title = shouldAdd ? 'Favorilerden çıkar' : 'Favorilere ekle';
          if (button.classList.contains('detail-save')) button.querySelector('span').textContent = shouldAdd ? 'Kaydedildi' : 'Kaydet';
        });
      }
      notify(saved ? shouldAdd ? 'Favorilerine eklendi.' : 'Favorilerinden çıkarıldı.' : 'Değişiklik bu oturumda uygulandı; tarayıcıya kaydedilemedi.');
      return;
    }
    const categoryButton = event.target.closest('[data-category]');
    if (categoryButton && user) {
      main.querySelectorAll('[data-category]').forEach(button => {
        const selected = button === categoryButton;
        button.classList.toggle('active', selected);
        button.setAttribute('aria-pressed', String(selected));
      });
      updateResults();
      lastCatalogHash = location.hash;
      return;
    }
    const target = event.target.closest('[data-action]');
    if (!target) return;
    const action = target.dataset.action;
    if (action === 'sample-login') {
      const account = store.users.find(item => item.role === 'student' && item.status === 'active');
      if (account) login(account.email, account.name, account.role, account.id);
      else notify('Aktif örnek öğrenci yok. Yönetici demosunda bir öğrenci kaydını onaylayabilirsin.');
      return;
    }
    if (action === 'sample-admin-login') {
      const account = store.users.find(item => item.role === 'admin' && item.status === 'active');
      if (account) login(account.email, account.name, account.role, account.id);
      return;
    }
    if (!user) return;
    if (action === 'logout') logout();
    if (action === 'back-to-catalog') { event.preventDefault(); location.hash = lastCatalogHash; }
    if (action === 'reset-filters') {
      document.getElementById('catalog-query').value = '';
      main.querySelectorAll('[data-category]').forEach(button => {
        const selected = button.dataset.category === 'all';
        button.classList.toggle('active', selected);
        button.setAttribute('aria-pressed', String(selected));
      });
      updateResults();
      lastCatalogHash = location.hash;
      document.getElementById('catalog-query').focus();
    }
    if (action === 'reveal') {
      const benefit = benefits.find(item => item.id === target.dataset.id);
      if (!benefit || benefitStatus(benefit) !== 'active') return;
      revealed.add(benefit.id);
      document.getElementById('redemption-content').innerHTML = redemptionContent(benefit);
      if (benefit.method === 'code') document.getElementById('discount-code').focus();
      else { const box = main.querySelector('.qr-box, .demo-member-pass'); box.setAttribute('tabindex', '-1'); box.focus({ preventScroll: true }); }
    }
    if (action === 'copy-code') copyCode(target.dataset.id, target);
    if (action === 'share-benefit') shareBenefit(target.dataset.id, target);
  });

  document.addEventListener('input', event => {
    event.target.removeAttribute('aria-invalid');
    if (event.target.id === 'catalog-query') { updateResults(); lastCatalogHash = location.hash; }
  });

  document.addEventListener('submit', event => {
    if (event.target.id === 'admin-auth-form') {
      event.preventDefault();
      const input = document.getElementById('admin-email');
      const account = store.users.find(item => item.email.toLowerCase() === input.value.trim().toLowerCase() && item.role === 'admin' && item.status === 'active');
      if (!validateEmail(input.value) || !account) { showFormError('admin-auth-error', 'Bu alan adıyla kayıtlı aktif bir demo yönetici hesabı kullan. Öğrenci hesapları yönetim panelini açamaz.', input); return; }
      login(account.email, account.name, account.role, account.id);
    }
    if (event.target.id === 'auth-form') {
      event.preventDefault();
      if (pendingLogin) return;
      const emailInput = document.getElementById('auth-email');
      const email = emailInput.value.trim();
      if (!validateEmail(email)) {
        showFormError('auth-error', `Bu demoda yalnızca @${config.SCHOOL_EMAIL_DOMAIN} alan adı kabul edilir. Örneğin: ${sampleEmail()}`, emailInput);
        return;
      }
      const nameInput = document.getElementById('auth-name');
      const name = nameInput ? nameInput.value.trim().replace(/\s+/g, ' ') : config.DEMO_NAME;
      if (name.length < 2 || name.length > 50) { showFormError('auth-error', 'Lütfen 2–50 karakter uzunluğunda örnek bir ad gir.', nameInput); return; }
      let account = store.users.find(item => item.email.toLowerCase() === email.toLowerCase());
      if (!account && authMode === 'register') {
        account = { id: store.uid('student'), name, email, role: 'student', status: 'pending', verification: 'pending' };
        store.users.push(account); store.save();
        const box = document.getElementById('auth-error');
        box.className = 'notice'; box.setAttribute('role', 'status'); box.hidden = false;
        box.textContent = 'Örnek kayıt, demo doğrulama bekleme listesine eklendi. Gerçek e-posta gönderilmedi. Yönetici demosunda onaylayabilir veya “Örnek öğrenciyle incele” ile tasarıma göz atabilirsin.';
        return;
      }
      if (!account || account.status !== 'active') {
        const box = document.getElementById('auth-error'); box.className = 'form-error'; box.setAttribute('role', 'alert');
        showFormError('auth-error', !account ? 'Bu demo hesabı kayıtlı değil. Kayıt sekmesini veya örnek öğrenci butonunu kullan.' : account.status === 'pending' ? 'Bu örnek hesap yönetici demo onayını bekliyor. Gerçek e-posta doğrulaması yapılmadı.' : 'Bu örnek hesap reddedilmiş. Başka bir örnek hesapla tasarımı inceleyebilirsin.', emailInput);
        return;
      }
      pendingLogin = true;
      const submit = event.target.querySelector('[type="submit"]');
      submit.disabled = true;
      submit.textContent = 'Demo açılıyor…';
      event.target.setAttribute('aria-busy', 'true');
      requestAnimationFrame(() => { pendingLogin = false; login(email, nameInput ? name : account.name, account.role, account.id); });
    }
    if (event.target.id === 'header-search') {
      event.preventDefault();
      if (!user) return;
      const query = document.getElementById('global-search').value.trim();
      const hash = `#discover${query ? `?q=${encodeURIComponent(query)}` : ''}`;
      if (location.hash === hash) renderRoute(); else location.hash = hash;
    }
    if (event.target.id === 'profile-form') {
      event.preventDefault();
      if (!user) return;
      const input = document.getElementById('profile-name');
      const name = input.value.trim().replace(/\s+/g, ' ');
      if (name.length < 2 || name.length > 50) { showFormError('profile-error', 'Lütfen 2–50 karakter uzunluğunda örnek bir ad gir.', input); return; }
      user.name = name;
      renderNavigation('profile');
      renderProfile();
      main.querySelector('[type="submit"]').focus({ preventScroll: true });
      notify('Örnek profil bu oturum için güncellendi.');
    }
  });

  document.addEventListener('error', event => {
    if (event.target instanceof HTMLImageElement && !event.target.dataset.failed) {
      event.target.dataset.failed = 'true';
      if (event.target.closest('.qr-box')) {
        event.target.parentElement.innerHTML = '<p role="alert">Demo QR görseli yüklenemedi. assets/demo-qr.svg dosyasının yerinde olduğunu kontrol et.</p>';
      } else {
        event.target.src = 'assets/placeholder.svg';
        event.target.alt = 'Örnek avantaj görseli yüklenemedi';
      }
    }
  }, true);

  window.addEventListener('hashchange', () => {
    if (user) renderRoute();
    else if (/^#benefit\/[a-zA-Z0-9-]+$/.test(location.hash)) { pendingDetailHash = location.hash; renderAuth(); }
  });

  if (!window.KOC_CONFIG || !store || !window.createKocAdmin || !Array.isArray(benefits) || !Array.isArray(categories)) {
    authView.innerHTML = '<div class="empty-state"><h1>Örnek içerik yüklenemedi.</h1><p>config.js ve data.js dosyalarının index.html ile aynı klasörde olduğundan emin ol.</p><button class="btn btn-primary" onclick="location.reload()">Tekrar dene</button></div>';
    return;
  }
  admin = window.createKocAdmin({ store, main, escapeHTML, icon, notify, getUser: () => user, config, categories, renderRoute, updateBrand, getFavorites: () => favorites });
  loadFavorites();
  updateBrand();
  renderAuth();
})();
