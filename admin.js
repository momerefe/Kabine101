/* Yönetici arayüzü. Bu rol ayrımı yalnızca demo gezinmesidir.
 * İşlemler yerel demo verisini değiştirir; sunucu yetkilendirmesi sağlamaz.
 */
window.createKocAdmin = function createKocAdmin(context) {
  'use strict';
  const { store, main, escapeHTML: e, icon, notify, getUser, config, categories } = context;
  const nav = [
    ['overview', 'Yönetici Genel Bakış', 'grid'],
    ['partners', 'İş Ortaklığı Takibi', 'people'],
    ['businesses', 'İşletme Detayları', 'building'],
    ['benefits', 'Avantaj Yönetimi', 'ticket'],
    ['students', 'Öğrenci Doğrulama', 'shield'],
    ['published', 'Yayımlanmış Avantajlar', 'check'],
    ['archived', 'Süresi Dolan / Askıdaki', 'pause'],
    ['usage', 'Kullanım Kayıtları', 'qr'],
    ['analytics', 'Analitik', 'chart'],
    ['users', 'Kullanıcı ve Rol Yönetimi', 'user'],
    ['audit', 'İşlem Günlüğü', 'list'],
    ['settings', 'Platform Ayarları', 'settings']
  ].map(([id, label, glyph]) => ({ id, label, glyph }));
  const partnershipLabels = { discussing: 'Görüşülüyor', contacted: 'İletişim kuruldu', awaiting: 'Anlaşma bekleniyor', active: 'Aktif', ended: 'Sonlandırıldı' };
  const statusLabels = { active: 'Aktif', draft: 'Taslak', expired: 'Süresi doldu', suspended: 'Askıda', scheduled: 'Yakında' };
  const userLabels = { active: 'Aktif', pending: 'Doğrulama bekliyor', rejected: 'Reddedildi' };
  const images = [['assets/cafe.svg', 'Kahve'], ['assets/fashion.svg', 'Giyim'], ['assets/tech.svg', 'Teknoloji'], ['assets/fitness.svg', 'Spor'], ['assets/culture.svg', 'Etkinlik'], ['assets/books.svg', 'Kitap']];
  let section = 'overview';
  let pinned = false;
  let opener = null;

  const badge = (key, label) => `<span class="status-badge status-${e(key)}">${e(label || statusLabels[key] || key)}</span>`;
  const categoryName = id => categories.find(item => item.id === id)?.name || 'Diğer avantajlar';
  const prettyDate = value => value ? new Intl.DateTimeFormat('tr-TR', { dateStyle: 'medium', ...(value.includes('T') ? { timeStyle: 'short' } : {}), timeZone: config.TIME_ZONE }).format(new Date(value.includes('T') ? value : `${value}T12:00:00+03:00`)) : 'Belirtilmedi';
  const options = (entries, selected) => entries.map(([value, label]) => `<option value="${e(value)}" ${value === selected ? 'selected' : ''}>${e(label)}</option>`).join('');

  function heading(title, description, actions = '') {
    return `<div class="page-heading admin-heading"><div><div class="admin-eyebrow">YÖNETİM ALANI <span class="demo-tag">DEMO</span></div><h1>${e(title)}</h1><p>${e(description)}</p></div><div class="heading-actions">${actions}</div></div>`;
  }
  function demoNotice(text = 'Bu alandaki veriler örnektir. İşlemler yalnızca bu tarayıcıdaki prototipi değiştirir.') {
    return `<div class="admin-demo-notice">${icon('info')}<span>${e(text)}${store.persistent ? '' : ' Tarayıcıya kayıt yapılamıyor; değişiklikler yalnızca açık oturumda kalır.'}</span></div>`;
  }
  function empty(title = 'Henüz kayıt yok.', description = 'Yeni bir demo kayıt eklediğinde burada görünecek.') {
    return `<div class="admin-empty">${icon('list')}<h2>${e(title)}</h2><p>${e(description)}</p></div>`;
  }
  function table(headers, rows, label) {
    return rows.length ? `<div class="table-scroll" tabindex="0" role="region" aria-label="${e(label)}"><table class="admin-table"><thead><tr>${headers.map(text => `<th scope="col">${e(text)}</th>`).join('')}</tr></thead><tbody>${rows.join('')}</tbody></table></div>` : empty('Eşleşen kayıt bulunamadı.', 'Aramayı veya durum filtresini değiştirmeyi dene.');
  }
  function toolbar(placeholder, filterOptions = '') {
    return `<div class="admin-toolbar"><div class="catalog-search">${icon('search')}<label class="sr-only" for="admin-search">${e(placeholder)}</label><input id="admin-search" type="search" maxlength="100" placeholder="${e(placeholder)}"></div>${filterOptions ? `<label class="admin-filter-label"><span class="sr-only">Durum filtresi</span><select id="admin-filter"><option value="all">Tüm durumlar</option>${filterOptions}</select></label>` : ''}</div>`;
  }
  function field(name, label, value = '', type = 'text', help = '', required = false) {
    const id = `edit-${name}`;
    const control = type === 'textarea'
      ? `<textarea id="${id}" name="${name}" rows="3" maxlength="2500" ${required ? 'required' : ''}>${e(value)}</textarea>`
      : `<input id="${id}" name="${name}" type="${type}" value="${e(value)}" ${type === 'number' ? 'min="0" step="0.01"' : 'maxlength="500"'} ${required ? 'required' : ''}>`;
    return `<div class="field"><label for="${id}">${e(label)}${required ? ' <span class="required-mark">*</span>' : ''}</label>${control}${help ? `<small>${e(help)}</small>` : ''}</div>`;
  }
  function selectField(name, label, entries, selected, help = '') {
    return `<div class="field"><label for="edit-${name}">${e(label)}</label><select id="edit-${name}" name="${name}">${options(entries, selected)}</select>${help ? `<small>${e(help)}</small>` : ''}</div>`;
  }
  function group(title, description, content) {
    return `<section class="surface form-group"><div class="form-group-heading"><h2>${e(title)}</h2>${description ? `<p>${e(description)}</p>` : ''}</div>${content}</section>`;
  }
  function error(form, message) {
    const box = form.querySelector('.form-error');
    box.textContent = message;
    box.hidden = false;
    box.scrollIntoView({ block: 'center', behavior: 'auto' });
    box.focus();
  }

  function setMenu(open) {
    document.body.classList.toggle('admin-menu-open', open);
    document.getElementById('menu-scrim').hidden = !open;
    const button = document.getElementById('admin-menu-button');
    button.setAttribute('aria-expanded', String(open));
    button.setAttribute('aria-label', open ? 'Yönetim menüsünü kapat' : 'Yönetim menüsünü aç');
    if (open) document.querySelector('#desktop-nav a[aria-current=page]')?.focus();
  }

  function navigation(active) {
    const sidebar = document.querySelector('.sidebar');
    sidebar.setAttribute('aria-label', 'Yönetici ana menüsü');
    sidebar.querySelector('.brand').href = '#admin/overview';
    sidebar.querySelector('.sidebar-label').innerHTML = '<span>YÖNETİCİ PANELİ</span><button class="menu-pin" data-admin="pin-menu" aria-label="Menüyü sabitle" aria-pressed="false">' + icon('pin') + '</button>';
    document.getElementById('desktop-nav').innerHTML = nav.map(item => `<a href="#admin/${item.id}" class="nav-link ${active === item.id ? 'active' : ''}" title="${e(item.label)}" aria-label="${e(item.label)}" ${active === item.id ? 'aria-current="page"' : ''}>${icon(item.glyph)}<span>${e(item.label)}</span></a>`).join('');
    const pin = sidebar.querySelector('.menu-pin');
    pin.setAttribute('aria-pressed', String(pinned));
    pin.setAttribute('aria-label', pinned ? 'Menü sabitlemesini kaldır' : 'Menüyü sabitle');
    document.getElementById('mobile-nav').innerHTML = '';
    document.getElementById('breadcrumb-current').textContent = nav.find(item => item.id === active)?.label || 'Avantaj önizlemesi';
  }

  function auditRows(items) {
    return items.map(item => `<tr><td><strong>${e(item.type)}</strong><small>${e(item.subject)}</small></td><td>${e(prettyDate(item.at))}</td><td>${e(item.actor)} <span class="demo-tag">DEMO</span></td></tr>`);
  }

  function renderOverview() {
    const counts = [
      ['building', 'İşletme', store.businesses.length, 'businesses'],
      ['ticket', 'Aktif avantaj', store.benefits.filter(item => store.statusOf(item) === 'active').length, 'published'],
      ['shield', 'Doğrulama bekleyen', store.users.filter(item => item.verification === 'pending').length, 'students'],
      ['clock', 'Süresi dolan', store.benefits.filter(item => store.statusOf(item) === 'expired').length, 'archived']
    ];
    main.innerHTML = `${heading('Yönetici Genel Bakış', `Merhaba, ${getUser().name}. Demo platformunu buradan yönet.`, '<a class="btn btn-primary" href="#admin/benefits?new=1">' + icon('plus') + 'Avantaj oluştur</a>')}${demoNotice('Özetler bu tarayıcıdaki örnek kayıtlardan hesaplanır. Kullanım veya finansal istatistik sunulmaz.')}<div class="admin-stats">${counts.map(([glyph, label, count, destination]) => `<a class="surface stat-card" href="#admin/${destination}"><span class="stat-icon">${icon(glyph)}</span><strong>${count}</strong><span>${label}</span><small>DEMO KAYITLAR</small></a>`).join('')}</div><section class="surface admin-surface"><div class="section-heading"><h2>Son yönetim işlemleri</h2><a class="text-link" href="#admin/audit">Tümünü gör${icon('arrow')}</a></div>${store.state.audit.length ? table(['İşlem', 'Tarih', 'İşlemi yapan'], auditRows(store.state.audit.slice(0, 5)), 'Son demo yönetim işlemleri') : empty('Henüz bir yönetim işlemi yapılmadı.', 'Eklediğin veya düzenlediğin demo kayıtlar burada görünecek.')}</section><div class="admin-overview-note">${icon('shield')}Rol ayrımı ve işlem kayıtları arayüz demosudur; gerçek kimlik doğrulaması veya güvenilir denetim kaydı değildir.</div>`;
  }

  function listBusinesses(partners = false) {
    main.innerHTML = `${heading(partners ? 'İş Ortaklığı Takibi' : 'İşletme Detayları', partners ? 'Görüşmeleri, durumları ve son notları bir arada tut.' : 'Örnek işletme bilgilerini görüntüle ve düzenle.', '<a href="#admin/businesses?new=1" class="btn btn-primary">' + icon('plus') + 'İşletme ekle</a>')}${demoNotice()}${toolbar('İşletme veya not ara…', options(Object.entries(partnershipLabels), ''))}<div id="admin-results"></div>`;
    renderListResults();
  }

  function listBenefits() {
    const title = { benefits: 'Avantaj Yönetimi', published: 'Yayımlanmış Avantajlar', archived: 'Süresi Dolan ve Askıdaki Avantajlar' }[section];
    const description = { benefits: 'Taslak oluştur, koşulları düzenle ve yayımlamadan önce kontrol et.', published: 'Şu anda öğrenci kullanımına açık demo avantajlar.', archived: 'Süresi dolan veya askıya alınan avantajları koşullarını kontrol ederek yeniden düzenle.' }[section];
    main.innerHTML = `${heading(title, description, '<a href="#admin/benefits?new=1" class="btn btn-primary">' + icon('plus') + 'Avantaj oluştur</a>')}${demoNotice()}${toolbar('İşletme, başlık veya kategori ara…', options(Object.entries(statusLabels), ''))}<div id="admin-results"></div>`;
    renderListResults();
  }

  function renderListResults() {
    const target = document.getElementById('admin-results');
    if (!target) return;
    const query = (document.getElementById('admin-search')?.value || '').trim().toLocaleLowerCase('tr-TR');
    const filter = document.getElementById('admin-filter')?.value || 'all';
    const matches = text => String(text).toLocaleLowerCase('tr-TR').includes(query);
    let headers, rows;
    if (section === 'businesses' || section === 'partners') {
      const partners = section === 'partners';
      const records = store.businesses.filter(item => matches(`${item.name} ${item.notes} ${item.contactPerson}`) && (filter === 'all' || item.partnershipStatus === filter));
      headers = partners ? ['İşletme', 'İletişim kişisi', 'Son görüşme', 'Durum', 'Notlar', 'İşlemler'] : ['İşletme', 'Kategori', 'İletişim / Şubeler', 'İş ortaklığı', 'İşlemler'];
      rows = records.map(item => `<tr><td><strong>${e(item.name)}</strong><small>${e(item.number)}</small></td>${partners ? `<td>${e(item.contactPerson || 'Belirtilmedi')}</td><td>${e(prettyDate(item.lastContact))}</td><td>${badge(item.partnershipStatus, partnershipLabels[item.partnershipStatus])}</td><td class="note-cell">${e(item.notes || 'Not yok')}</td>` : `<td>${e(categoryName(item.category))}</td><td>${e(item.contact || 'İletişim belirtilmedi')}<small>${e(item.branches || 'Şube belirtilmedi')}</small></td><td>${badge(item.partnershipStatus, partnershipLabels[item.partnershipStatus])}</td>`}<td><div class="row-actions"><a href="#admin/businesses?view=${item.id}">Görüntüle</a><a href="#admin/businesses?edit=${item.id}">${partners ? 'Takibi düzenle' : 'Düzenle'}</a></div></td></tr>`);
    } else if (['benefits', 'published', 'archived'].includes(section)) {
      const records = store.benefits.filter(item => {
        const status = store.statusOf(item);
        const inSection = section === 'published' ? status === 'active' : section === 'archived' ? ['expired', 'suspended'].includes(status) : true;
        return inSection && (filter === 'all' || status === filter) && matches(`${item.business} ${item.title} ${categoryName(item.category)}`);
      });
      headers = ['İşletme / Avantaj', 'Kategori', 'Yayın tarihi', 'Durum / Geçerlilik', 'İşlemler'];
      rows = records.map(item => {
        const status = store.statusOf(item);
        const reason = status === 'expired' ? 'Bitiş tarihi geçti.' : status === 'suspended' ? item.suspensionReason || (item.publicationStatus === 'suspended' ? 'Yönetici tarafından askıya alındı.' : 'İş ortaklığı aktif değil.') : '';
        return `<tr><td><strong>${e(item.business)}</strong><small>${e(item.title)}</small></td><td>${e(categoryName(item.category))}</td><td>${e(prettyDate(item.publishedAt))}</td><td>${badge(status)}<small>${e(reason || (item.validUntil ? `Bitiş: ${prettyDate(item.validUntil)}` : 'Bitiş tarihi belirtilmedi'))}</small></td><td><div class="row-actions"><a href="#benefit/${item.id}">Önizle</a><a href="#admin/benefits?edit=${item.id}">${['expired', 'suspended'].includes(status) ? 'İncele / yenile' : 'Düzenle'}</a>${['active', 'scheduled'].includes(status) ? `<button data-admin="suspend" data-id="${item.id}">Yayından kaldır</button>` : ''}</div></td></tr>`;
      });
    } else if (section === 'students') {
      const records = store.users.filter(item => item.role === 'student' && matches(`${item.name} ${item.email}`) && (filter === 'all' || item.verification === filter));
      headers = ['Örnek öğrenci', 'Okul e-postası', 'Doğrulama durumu', 'İşlemler'];
      rows = records.map(item => `<tr><td><strong>${e(item.name)}</strong></td><td>${e(item.email)}</td><td>${badge(item.verification === 'approved' ? 'active' : item.verification, { pending: 'Bekliyor', approved: 'Demo onaylandı', rejected: 'Demo reddedildi' }[item.verification])}</td><td>${item.verification === 'pending' ? `<div class="row-actions"><button data-admin="approve" data-id="${item.id}">Onayla</button><button data-admin="reject" data-id="${item.id}">Reddet</button></div>` : '<span class="muted">İşlem tamamlandı</span>'}</td></tr>`);
    } else if (section === 'users') {
      const records = store.users.filter(item => matches(`${item.name} ${item.email}`) && (filter === 'all' || item.role === filter));
      headers = ['Örnek kullanıcı', 'E-posta', 'Rol', 'Hesap durumu', 'İşlemler'];
      rows = records.map(item => `<tr><td><strong>${e(item.name)}</strong></td><td>${e(item.email)}</td><td>${badge(item.role === 'admin' ? 'admin' : 'student', item.role === 'admin' ? 'Yönetici' : 'Öğrenci')}</td><td>${badge(item.status, userLabels[item.status])}</td><td>${item.id === getUser().id ? '<span class="muted">Açık demo oturumu</span>' : `<button class="table-button" data-admin="role" data-id="${item.id}">Rolü düzenle</button>`}</td></tr>`);
    } else if (section === 'audit') {
      headers = ['İşlem / Kayıt', 'Tarih', 'İşlemi yapan'];
      rows = auditRows(store.state.audit.filter(item => matches(`${item.type} ${item.subject} ${item.actor}`)));
    }
    target.innerHTML = `<p class="table-count" role="status" aria-live="polite">${rows.length} demo kayıt</p>${table(headers, rows, 'Demo yönetim kayıtları')}`;
  }

  function businessDetail(id) {
    const business = store.businesses.find(item => item.id === id);
    if (!business) { main.innerHTML = empty('İşletme bulunamadı.'); return; }
    main.innerHTML = `${heading(business.name, `${business.number} · Kurgusal işletme`, `<a href="#admin/businesses" class="btn btn-secondary">Listeye dön</a><a href="#admin/businesses?edit=${id}" class="btn btn-primary">${icon('edit')}Düzenle</a>`)}<div class="admin-business-detail"><img src="${e(business.image)}" alt="${e(business.imageAlt)}"><section class="surface"><h2>İşletme bilgileri</h2><p>${e(business.description)}</p><dl class="detail-facts"><div><dt>Kategori</dt><dd>${e(categoryName(business.category))}</dd></div><div><dt>İş ortaklığı</dt><dd>${badge(business.partnershipStatus, partnershipLabels[business.partnershipStatus])}</dd></div>${[['İletişim kişisi', business.contactPerson], ['İletişim', business.contact], ['Şubeler', business.branches], ['Son görüşme', business.lastContact ? prettyDate(business.lastContact) : ''], ['Notlar', business.notes]].filter(([, value]) => value).map(([label, value]) => `<div><dt>${label}</dt><dd>${e(value)}</dd></div>`).join('')}</dl></section></div>${demoNotice()}`;
  }

  function businessForm(id) {
    const business = id ? store.businesses.find(item => item.id === id) : { category: categories[0].id, image: images[0][0], partnershipStatus: 'discussing' };
    if (!business) { main.innerHTML = empty('İşletme bulunamadı.'); return; }
    main.innerHTML = `${heading(id ? 'İşletmeyi düzenle' : 'İşletme ekle', 'Yalnızca örnek bilgiler gir. Zorunlu alanlar yıldız ile işaretlidir.', '<a href="#admin/businesses" class="btn btn-secondary">Listeye dön</a>')}<form id="business-form" data-id="${e(id || '')}" novalidate><div class="form-error" role="alert" tabindex="-1" hidden></div>${group('İşletme kimliği', 'Öğrencilerin göreceği temel bilgiler.', `${field('name', 'İşletme adı', business.name, 'text', '', true)}<div class="form-columns">${selectField('category', 'Kategori', categories.map(item => [item.id, item.name]), business.category)}${selectField('image', 'Örnek işletme görseli', images, business.image)}</div>${field('description', 'Açıklama', business.description, 'textarea')}`)}${group('İletişim ve şubeler', 'Bilgi verilmediyse boş bırak. Gerçek kişisel veri girme.', `<div class="form-columns">${field('contactPerson', 'Örnek iletişim kişisi', business.contactPerson)}${field('contact', 'Örnek iletişim bilgisi', business.contact)}</div>${field('branches', 'İlgili şubeler', business.branches)}`)}${group('İş ortaklığı takibi', 'Görüşme notları yalnızca yönetim alanında gösterilir.', `<div class="form-columns">${selectField('partnershipStatus', 'İş ortaklığı durumu', Object.entries(partnershipLabels), business.partnershipStatus)}${field('lastContact', 'Son görüşme tarihi', business.lastContact, 'date')}</div>${field('notes', 'Notlar', business.notes, 'textarea')}`)}<div class="form-actions"><a class="btn btn-secondary" href="#admin/businesses">Vazgeç</a><button class="btn btn-primary" type="submit">Demo işletmeyi kaydet${icon('check')}</button></div></form>`;
  }

  function methodFields(benefit = {}) {
    const method = document.getElementById('edit-method')?.value || benefit.method || 'code';
    return `<div data-method-fields="code" ${method !== 'code' ? 'hidden' : ''}>${field('code', 'Demo indirim kodu', benefit.code, 'text', 'Gerçek bir işletmede geçerli olmayan örnek kod kullan.')}</div><div data-method-fields="link" ${method !== 'link' ? 'hidden' : ''}>${field('url', 'Örnek avantaj bağlantısı', benefit.url, 'url', 'Yalnızca https:// veya http:// bağlantıları kabul edilir. Örnek: https://example.org/')}</div><div data-method-fields="qr" ${method !== 'qr' ? 'hidden' : ''}>${selectField('qrImage', 'Demo QR görseli', [['assets/demo-qr.svg', 'Yerel örnek QR']], 'assets/demo-qr.svg', 'Bu sabit QR yalnızca DEMO-KOCBENEFITS metnini içerir. Özel kod veya doğrulama üretilmez.')}</div><div data-method-fields="membership" ${method !== 'membership' ? 'hidden' : ''}>${field('membershipLabel', 'Örnek kart adı', benefit.membershipLabel || 'Demo öğrenci kartı', 'text', 'Kart yalnızca görsel bir örnektir; gerçek üyelik veya öğrenci doğrulaması sağlamaz.')}</div>`;
  }

  function benefitForm(id) {
    const benefit = id ? store.benefits.find(item => item.id === id) : { businessId: store.businesses[0]?.id, method: 'code', conditions: [], instructions: [], publicationStatus: 'draft' };
    if (!benefit) { main.innerHTML = empty('Avantaj bulunamadı.'); return; }
    main.innerHTML = `${heading(id ? 'Avantajı düzenle' : 'Avantaj oluştur', 'Koşulları tamamla, taslak kaydet veya kontrol ederek yayımla.', '<a href="#admin/benefits" class="btn btn-secondary">Listeye dön</a>')}<form id="benefit-form" data-id="${e(id || '')}" novalidate><div class="form-error" role="alert" tabindex="-1" hidden></div>${group('1. Temel bilgiler', 'İşletme, avantaj başlığı ve sunulan ayrıcalık.', `${selectField('businessId', 'İlgili işletme', store.businesses.map(item => [item.id, `${item.name} (${partnershipLabels[item.partnershipStatus]})`]), benefit.businessId)}${field('title', 'Avantaj başlığı', benefit.title, 'text', '', true)}<div class="form-columns">${field('discount', 'İndirim veya ayrıcalık', benefit.discount, 'text', 'Örnek: %20 veya 1 ders hediye', true)}${selectField('badge', 'Avantaj türü', [['İNDİRİM', 'İndirim'], ['HEDİYE', 'Hediye / ayrıcalık']], benefit.badge || 'İNDİRİM')}</div>${field('description', 'Tam açıklama', benefit.description, 'textarea')}`)}${group('2. Kapsam ve koşullar', 'Öğrencinin bilmesi gereken koşulları açıkça belirt.', `${field('conditions', 'Kullanım koşulları', (benefit.conditions || []).join('\n'), 'textarea', 'Her koşulu ayrı satıra yaz. Yayımlamak için en az bir koşul gerekli.')}${field('products', 'Geçerli ürünler / hizmetler', benefit.products)}${field('branches', 'Geçerli şubeler', benefit.branches)}${field('exceptions', 'İstisnalar', benefit.exceptions, 'textarea')}<div class="form-columns">${field('minimumSpend', 'Minimum harcama (TL)', benefit.minimumSpend ?? '', 'number', 'Şart yoksa boş bırak.')}${field('verification', 'Gerekli öğrenci doğrulaması', benefit.verification, 'text', 'Bu prototip gerçek doğrulama yapmaz.')}</div>`)}${group('3. Tarihler ve kullanım yöntemi', 'Her avantaj için yalnızca ilgili yöntem gösterilir.', `<div class="form-columns">${field('validFrom', 'Geçerlilik başlangıcı', benefit.validFrom, 'date')}${field('validUntil', 'Geçerlilik bitişi', benefit.validUntil, 'date')}</div>${selectField('method', 'Kullanım yöntemi', [['code', 'İndirim kodu'], ['link', 'Bağlantı'], ['qr', 'QR kodu'], ['membership', 'Üyelik kartı']], benefit.method)}${methodFields(benefit)}${field('instructions', 'Adım adım kullanım talimatları', (benefit.instructions || []).join('\n'), 'textarea', 'Her adımı ayrı satıra yaz.')}${field('supportEmail', 'Örnek destek e-postası (varsa)', benefit.supportEmail, 'email', 'Destek kanalı tanımlanmadıysa boş bırak.')}`)}${group('4. Yayın kontrolü', 'Yayınlamak, bu tarayıcıdaki öğrenci kataloğunu günceller.', `<p class="current-publication">Mevcut durum: ${badge(id ? store.statusOf(benefit) : 'draft')}</p><label class="check-field"><input name="reviewed" type="checkbox"><span>Kullanım koşullarını, tarihleri ve işletme anlaşmasını bu demo yayını için yeniden kontrol ettim.</span></label><p class="field-help">Taslak kaydetmek öğrenci görünürlüğünü kapatır. Süresi dolmuş avantajı yayımlamak için geçerlilik dönemini yenilemelisin.</p>`)}<div class="form-actions"><a class="btn btn-secondary" href="#admin/benefits">Vazgeç</a><button type="submit" class="btn btn-secondary" name="intent" value="draft">Taslak kaydet</button><button type="submit" class="btn btn-primary" name="intent" value="publish">İncele ve yayımla${icon('check')}</button></div></form>`;
  }

  function students() {
    main.innerHTML = `${heading('Öğrenci Doğrulama Bekleme Listesi', 'Örnek kayıtları incele ve demo onay kararını ver.')}${demoNotice('Onay ve ret işlemleri yalnızca demo hesap durumunu değiştirir. E-posta onayı veya gerçek öğrenci doğrulaması yapılmaz.')}${toolbar('Örnek ad veya okul e-postası ara…', options([['pending', 'Bekliyor'], ['approved', 'Demo onaylandı'], ['rejected', 'Demo reddedildi']], ''))}<div id="admin-results"></div>`;
    document.getElementById('admin-filter').value = 'pending';
    renderListResults();
  }

  function usage() {
    main.innerHTML = `${heading('Kullanım Kayıtları', 'Yalnızca güvenilir biçimde doğrulanmış kullanımlar bu alanda yer almalı.')}${demoNotice('Gerçek bir kullanım doğrulama sistemi yok. Kod, bağlantı, QR veya kart görüntüleme kullanım olarak kaydedilmez.')}<section class="surface admin-surface">${empty('Doğrulanmış kullanım kaydı bulunmuyor.', 'Bu ekran bir demo yer tutucudur. Sahte öğrenci veya işletme kullanım kaydı üretilmez.')}<div class="usage-schema"><span>Avantaj</span><span>İşletme</span><span>Doğrulama tarihi</span><span>Kayıt durumu</span></div><p class="field-help">Kayıt alanları yalnızca tasarım örneğidir. İşletme onayının yöntemi sonraki aşamada belirlenecek.</p></section>`;
  }

  function analytics() {
    const totalViews = Object.values(store.state.views).reduce((sum, value) => sum + Number(value || 0), 0);
    const currentFavorites = context.getFavorites();
    const items = [...store.benefits].sort((a, b) => (store.state.views[b.id] || 0) - (store.state.views[a.id] || 0));
    const max = Math.max(1, ...items.map(item => store.state.views[item.id] || 0));
    main.innerHTML = `${heading('Analitik', 'Bu tarayıcıdaki demo etkileşimlerine sade bir bakış.')}${demoNotice('Görüntülemeler yalnızca bu tarayıcıdaki öğrenci detay ziyaretleridir. Favori sayısı mevcut tarayıcı listesidir; toplam öğrenci performansı veya gerçek kullanım değildir.')}<div class="admin-stats analytics-stats"><div class="surface stat-card">${icon('compass')}<strong>${totalViews}</strong><span>Demo detay görüntüleme</span></div><div class="surface stat-card">${icon('heart')}<strong>${currentFavorites.size}</strong><span>Bu tarayıcıdaki favoriler</span></div><div class="surface stat-card">${icon('shield')}<strong class="stat-text">Kayıt yok</strong><span>Doğrulanmış kullanım sistemi yok</span></div></div><section class="surface"><h2>Avantaj görüntülemeleri <span class="demo-tag">DEMO</span></h2><div class="bar-chart" aria-label="Yerel demo avantaj görüntülemeleri">${items.map(item => `<div class="bar-row"><span>${e(item.business)}<small>${e(item.title)}</small></span><div class="bar-track"><div style="width:${((store.state.views[item.id] || 0) / max) * 100}%"></div></div><strong>${Number(store.state.views[item.id]) || 0}</strong></div>`).join('')}</div><p class="field-help">Yönetici önizlemeleri görüntüleme sayacına dahil edilmez. Bu veriler tarayıcıdan değiştirilebilir ve güvenilir analitik değildir.</p></section>`;
  }

  function users() {
    main.innerHTML = `${heading('Kullanıcı ve Rol Yönetimi', 'İki temel rol: öğrenci ve yönetici.')}${demoNotice('Rol değişiklikleri sonraki demo girişini etkiler. Bu arayüz gerçek yetki vermez; güvenli rol denetimi sunucuda yapılmalıdır.')}${toolbar('Örnek ad veya e-posta ara…', options([['student', 'Öğrenci'], ['admin', 'Yönetici']], ''))}<div id="admin-results"></div>`;
    renderListResults();
  }

  function audit() {
    main.innerHTML = `${heading('İşlem Günlüğü', 'Bu prototipte gerçekleştirdiğin yönetim işlemleri.')}${demoNotice('Zamanlar cihaz saatinden alınır. Kayıtlar bu tarayıcıda tutulur ve değiştirilebilir; gerçek denetim günlüğü değildir.')}${toolbar('İşlem, kayıt veya demo yönetici ara…')}<div id="admin-results"></div>`;
    renderListResults();
  }

  function settings() {
    main.innerHTML = `${heading('Platform Ayarları', 'Değiştirilebilir örnek platform bilgileri.')}${demoNotice('Bu ayarlar yalnızca prototipi değiştirir. Alan adı kontrolü e-posta doğrulaması veya gerçek güvenlik sağlamaz.')}<form id="settings-form" novalidate><div class="form-error" role="alert" tabindex="-1" hidden></div>${group('Temel bilgiler', 'Gerçek okul alan adı ve iletişim bilgileri henüz verilmedi.', `${field('PLATFORM_NAME', 'Platform adı', store.settings.PLATFORM_NAME, 'text', '', true)}${field('SCHOOL_EMAIL_DOMAIN', 'İzin verilen okul e-posta alan adı', store.settings.SCHOOL_EMAIL_DOMAIN, 'text', '@ işareti olmadan gir. Varsayılan ogrenci.example bir örnektir.', true)}${field('supportEmail', 'Örnek temel iletişim e-postası (varsa)', store.settings.supportEmail, 'email', 'Destek kanalı tanımlanmadıysa boş bırak.')}<label class="check-field"><input name="showFeatured" type="checkbox" ${store.settings.showFeatured ? 'checked' : ''}><span>Öğrenci Genel Bakış ekranında öne çıkan avantajları göster.</span></label>`)}<div class="form-actions"><button type="submit" class="btn btn-primary">Demo ayarları kaydet${icon('check')}</button></div></form>`;
  }

  function render(path, params) {
    section = path.replace('admin/', '') || 'overview';
    document.title = `${nav.find(item => item.id === section)?.label || 'Yönetim'} · ${config.PLATFORM_NAME}`;
    navigation(section);
    if (section === 'businesses' && (params.has('edit') || params.has('new'))) businessForm(params.get('edit'));
    else if (section === 'businesses' && params.has('view')) businessDetail(params.get('view'));
    else if (section === 'benefits' && (params.has('edit') || params.has('new'))) benefitForm(params.get('edit'));
    else ({ overview: renderOverview, partners: () => listBusinesses(true), businesses: () => listBusinesses(), benefits: listBenefits, published: listBenefits, archived: listBenefits, students, usage, analytics, users, audit, settings }[section] || (() => { main.innerHTML = empty('Yönetim sayfası bulunamadı.'); }))();
    setMenu(false);
    main.classList.remove('page-enter');
    void main.offsetWidth;
    main.classList.add('page-enter');
  }

  function confirmAction({ title, message, label = 'Onayla', content = '', onConfirm }) {
    const dialog = document.getElementById('admin-dialog');
    opener = document.activeElement;
    dialog.innerHTML = `<form method="dialog" id="confirmation-form"><div class="dialog-heading"><span class="demo-tag">DEMO İŞLEM</span><button type="button" data-admin="close-dialog" class="dialog-close" aria-label="Pencereyi kapat">${icon('close')}</button></div><h2 id="dialog-title">${e(title)}</h2><p>${e(message)}</p>${content}<div class="dialog-actions"><button type="button" class="btn btn-secondary" data-admin="close-dialog">Vazgeç</button><button type="submit" class="btn btn-primary">${e(label)}</button></div></form>`;
    dialog.querySelector('form').addEventListener('submit', event => {
      event.preventDefault();
      const values = new FormData(event.target);
      dialog.close();
      onConfirm(values);
    }, { once: true });
    dialog.showModal();
    dialog.querySelector('[data-admin=close-dialog]').focus();
  }

  function complete(type, subject, destination) {
    store.record(type, subject, getUser().name);
    const saved = store.save();
    notify(saved ? `${type}. Demo veriler güncellendi.` : 'İşlem uygulandı; tarayıcıya kaydedilemedi.');
    if (destination) {
      if (location.hash === destination) context.renderRoute(); else location.hash = destination;
    } else context.renderRoute();
  }

  function safeURL(value) {
    try { const url = new URL(value); return ['http:', 'https:'].includes(url.protocol); } catch { return false; }
  }
  const emailValid = value => !value || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
  const lines = value => value.split('\n').map(line => line.trim()).filter(Boolean);

  document.addEventListener('submit', event => {
    if (getUser()?.role !== 'admin') return;
    const form = event.target;
    if (!['business-form', 'benefit-form', 'settings-form'].includes(form.id)) return;
    event.preventDefault();
    const values = Object.fromEntries(new FormData(form));
    Object.keys(values).forEach(key => { values[key] = values[key].trim(); });
    form.querySelector('.form-error').hidden = true;
    if (form.id === 'business-form') {
      if (values.name.length < 2) return error(form, 'En az 2 karakterli bir işletme adı gir.');
      if (!categories.some(item => item.id === values.category) || !images.some(([path]) => path === values.image) || !partnershipLabels[values.partnershipStatus]) return error(form, 'Kategori, görsel veya iş ortaklığı durumu geçersiz.');
      const existing = store.businesses.find(item => item.id === form.dataset.id);
      const save = () => {
        const record = { ...values, imageAlt: `${values.name} için örnek işletme illüstrasyonu` };
        if (existing?.partnershipStatus === 'active' && values.partnershipStatus !== 'active') {
          store.benefits.filter(item => item.businessId === existing.id && item.publicationStatus === 'published').forEach(item => {
            item.publicationStatus = 'suspended';
            item.suspensionReason = 'İş ortaklığı aktif durumdan çıkarıldı. Yeniden yayın için koşullar kontrol edilmeli.';
          });
        }
        if (existing) Object.assign(existing, record);
        else store.businesses.push({ ...record, id: store.uid('biz'), number: `İŞL-${String(store.businesses.length + 1).padStart(3, '0')}` });
        store.benefits.forEach(store.enrich);
        complete(existing ? 'İşletme bilgisi değiştirildi' : 'İşletme eklendi', values.name, '#admin/businesses');
      };
      if (existing?.partnershipStatus === 'active' && values.partnershipStatus !== 'active' && store.benefits.some(item => item.businessId === existing.id && item.publicationStatus === 'published')) {
        confirmAction({ title: 'İş ortaklığı durumunu değiştir?', message: 'Bu işletmenin yayımlanmış avantajları öğrenci kullanımına kapanacak. Tekrar kullanım için iş ortaklığını aktif yapman gerekecek.', label: 'Durumu değiştir', onConfirm: save });
      } else save();
    }
    if (form.id === 'benefit-form') {
      const existing = store.benefits.find(item => item.id === form.dataset.id);
      const publish = event.submitter?.value === 'publish';
      const business = store.businesses.find(item => item.id === values.businessId);
      if (!business || values.title.length < 3 || !values.discount) return error(form, 'İşletme, en az 3 karakterli bir başlık ve sunulan ayrıcalık gerekli.');
      if (!['code', 'link', 'qr', 'membership'].includes(values.method)) return error(form, 'Geçerli bir kullanım yöntemi seç.');
      if (values.validFrom && values.validUntil && values.validFrom > values.validUntil) return error(form, 'Bitiş tarihi başlangıç tarihinden önce olamaz.');
      if (values.minimumSpend && (!Number.isFinite(Number(values.minimumSpend)) || Number(values.minimumSpend) < 0)) return error(form, 'Minimum harcama sıfır veya daha büyük olmalı.');
      if (values.url && values.method === 'link' && !safeURL(values.url)) return error(form, 'Bağlantı http:// veya https:// ile başlamalı.');
      if (!emailValid(values.supportEmail)) return error(form, 'Destek adresini geçerli bir örnek e-posta olarak gir veya boş bırak.');
      if (publish) {
        if (!values.reviewed) return error(form, 'Yayımlamadan önce koşulları, tarihleri ve anlaşmayı kontrol ettiğini onayla.');
        if (business.partnershipStatus !== 'active') return error(form, 'Yayımlamak için işletmenin iş ortaklığı durumu aktif olmalı.');
        if (values.validUntil && values.validUntil < store.today()) return error(form, 'Bu avantajın süresi dolmuş. Yeni bir geçerlilik dönemi belirle.');
        if (!values.description || !lines(values.conditions).length || !lines(values.instructions).length) return error(form, 'Yayımlamak için açıklama, kullanım koşulları ve talimatları tamamla.');
        if (values.method === 'code' && !values.code) return error(form, 'Kod yöntemi için örnek indirim kodu gir.');
        if (values.method === 'link' && !values.url) return error(form, 'Bağlantı yöntemi için örnek URL gir.');
      }
      const save = () => {
        const record = {
          ...existing, id: existing?.id || store.uid('avantaj'), businessId: business.id,
          title: values.title, description: values.description, discount: values.discount, badge: values.badge,
          conditions: lines(values.conditions), instructions: lines(values.instructions),
          products: values.products, branches: values.branches, exceptions: values.exceptions,
          minimumSpend: values.minimumSpend === '' ? null : Number(values.minimumSpend),
          validFrom: values.validFrom || null, validUntil: values.validUntil || null,
          verification: values.verification, method: values.method,
          code: values.method === 'code' ? values.code : '', url: values.method === 'link' ? values.url : '',
          qrImage: values.method === 'qr' ? 'assets/demo-qr.svg' : '', qrContent: values.method === 'qr' ? 'DEMO-KOCBENEFITS' : '',
          membershipLabel: values.method === 'membership' ? values.membershipLabel : '', supportEmail: values.supportEmail,
          publicationStatus: publish ? 'published' : 'draft', publishedAt: publish ? store.today() : existing?.publishedAt || null,
          suspensionReason: ''
        };
        store.enrich(record);
        if (existing) Object.assign(existing, record); else store.benefits.push(record);
        complete(publish ? 'Avantaj yayımlandı' : existing ? 'Avantaj düzenlendi / taslak kaydedildi' : 'Avantaj oluşturuldu', values.title, '#admin/benefits');
      };
      if (publish || existing?.publicationStatus === 'published') confirmAction({ title: publish ? 'Demo avantajı yayımla?' : 'Avantajı taslağa al?', message: publish ? 'Bu avantaj, tarihleri uygunsa bu tarayıcıdaki öğrenci kataloğunda görünecek. Gerçek bir sponsorluk yayımlanmaz.' : 'Taslak kaydetmek bu avantajı öğrenci kataloğundan kaldırır.', label: publish ? 'Demo olarak yayımla' : 'Taslak kaydet', onConfirm: save });
      else save();
    }
    if (form.id === 'settings-form') {
      const domain = values.SCHOOL_EMAIL_DOMAIN.toLowerCase();
      if (!values.PLATFORM_NAME || values.PLATFORM_NAME.length > 40) return error(form, 'Platform adı 1–40 karakter arasında olmalı.');
      if (domain.length > 120 || !/^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/.test(domain)) return error(form, 'Geçerli bir alan adı gir; @, boşluk veya bağlantı ekleme.');
      if (!emailValid(values.supportEmail)) return error(form, 'Geçerli bir örnek iletişim e-postası gir veya boş bırak.');
      const apply = () => {
        const oldDomain = store.settings.SCHOOL_EMAIL_DOMAIN;
        Object.assign(store.settings, { PLATFORM_NAME: values.PLATFORM_NAME, SCHOOL_EMAIL_DOMAIN: domain, supportEmail: values.supportEmail, showFeatured: values.showFeatured === 'on' });
        // Örnek kullanıcı adreslerini yeni demo alan adıyla tutarlı tut.
        if (oldDomain !== domain) store.users.forEach(item => { item.email = `${item.email.split('@')[0]}@${domain}`; });
        Object.assign(config, store.settings);
        context.updateBrand();
        complete('Platform ayarları güncellendi', values.PLATFORM_NAME);
      };
      if (domain !== store.settings.SCHOOL_EMAIL_DOMAIN) confirmAction({ title: 'Demo alan adını değiştir?', message: 'Örnek kullanıcı adresleri de yeni alan adına taşınacak. Sonraki demo girişleri yeni alan adıyla yapılacak. E-posta doğrulaması yapılmaz.', label: 'Demo alan adını değiştir', onConfirm: apply });
      else apply();
    }
  });

  document.addEventListener('input', event => {
    if (getUser()?.role === 'admin' && event.target.id === 'admin-search') renderListResults();
  });
  document.addEventListener('change', event => {
    if (getUser()?.role !== 'admin') return;
    if (event.target.id === 'admin-filter') renderListResults();
    if (event.target.id === 'edit-method') {
      main.querySelectorAll('[data-method-fields]').forEach(element => { element.hidden = element.dataset.methodFields !== event.target.value; });
    }
  });
  document.addEventListener('click', event => {
    const mobileButton = event.target.closest('#admin-menu-button');
    if (mobileButton && getUser()?.role === 'admin') { setMenu(!document.body.classList.contains('admin-menu-open')); return; }
    if (event.target.closest('#menu-scrim')) { setMenu(false); document.getElementById('admin-menu-button').focus(); return; }
    const button = event.target.closest('[data-admin]');
    if (!button || getUser()?.role !== 'admin') return;
    const action = button.dataset.admin;
    if (action === 'pin-menu') {
      pinned = !pinned;
      document.body.classList.toggle('admin-nav-pinned', pinned);
      button.setAttribute('aria-pressed', String(pinned));
      button.setAttribute('aria-label', pinned ? 'Menü sabitlemesini kaldır' : 'Menüyü sabitle');
    }
    if (action === 'close-dialog') document.getElementById('admin-dialog').close();
    if (action === 'suspend') {
      const benefit = store.benefits.find(item => item.id === button.dataset.id);
      if (!benefit) return;
      confirmAction({ title: 'Avantajı yayından kaldır?', message: `${benefit.title} öğrenci kataloğundan kaldırılıp askıya alınacak. Yeniden yayımlamak için koşulların kontrol edilmesi gerekecek.`, label: 'Askıya al', onConfirm: () => { benefit.publicationStatus = 'suspended'; benefit.suspensionReason = 'Yönetici tarafından askıya alındı.'; complete('Avantaj askıya alındı', benefit.title); } });
    }
    if (['approve', 'reject'].includes(action)) {
      const student = store.users.find(item => item.id === button.dataset.id && item.verification === 'pending');
      if (!student) return;
      const approve = action === 'approve';
      confirmAction({ title: approve ? 'Örnek öğrenciyi onayla?' : 'Örnek öğrenciyi reddet?', message: `${student.name} için yalnızca demo hesap durumu değişecek. Gerçek e-posta veya öğrenci doğrulaması yapılmayacak.`, label: approve ? 'Demo olarak onayla' : 'Demo olarak reddet', onConfirm: () => { student.verification = approve ? 'approved' : 'rejected'; student.status = approve ? 'active' : 'rejected'; complete(approve ? 'Öğrenci hesabı onaylandı' : 'Öğrenci hesabı reddedildi', student.name); } });
    }
    if (action === 'role') {
      const account = store.users.find(item => item.id === button.dataset.id);
      if (!account || account.id === getUser().id) return;
      confirmAction({ title: 'Demo rolünü düzenle', message: `${account.name} için rol değişikliği sonraki demo girişinde uygulanır. Gerçek yetkilendirme yapılmaz.`, content: selectField('role', 'Örnek rol', [['student', 'Öğrenci'], ['admin', 'Yönetici']], account.role), label: 'Demo rolünü kaydet', onConfirm: data => { account.role = data.get('role') === 'admin' ? 'admin' : 'student'; complete('Kullanıcı rolü değiştirildi', account.name); } });
    }
  });
  document.getElementById('admin-dialog').addEventListener('close', () => { if (opener?.isConnected) opener.focus({ preventScroll: true }); });
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && document.body.classList.contains('admin-menu-open')) {
      setMenu(false); document.getElementById('admin-menu-button').focus();
    }
    if (event.key === 'Tab' && document.body.classList.contains('admin-menu-open')) {
      const controls = [...document.querySelectorAll('.sidebar a, .sidebar button')].filter(element => element.getClientRects().length);
      const first = controls[0], last = controls.at(-1);
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    }
  });
  return { render, navigation, setMenu, nav };
};
