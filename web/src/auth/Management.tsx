import {
  useEffect,
  useRef,
  useState,
  type ReactNode,
  type FormEvent,
} from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { components } from '../api/schema.gen';
type Business = components['schemas']['Business'];
type Benefit = components['schemas']['Benefit'];
type Session = components['schemas']['SessionResponse'];
const nav = [
  ['overview', 'Yönetici Genel Bakış', '▦'],
  ['partners', 'İş Ortaklığı Takibi', '♧'],
  ['businesses', 'İşletme Detayları', '▤'],
  ['benefits', 'Avantaj Yönetimi', '◇'],
  ['students', 'Öğrenci Doğrulama', '✓'],
  ['published', 'Yayımlanmış Avantajlar', '☑'],
  ['archived', 'Süresi Dolan / Askıdaki', 'Ⅱ'],
  ['usage', 'Kullanım Kayıtları', '▣'],
  ['analytics', 'Analitik', '▥'],
  ['users', 'Kullanıcı ve Rol Yönetimi', '♙'],
  ['audit', 'İşlem Günlüğü', '≡'],
  ['settings', 'Platform Ayarları', '⚙'],
];
const categories = {
  food: 'Restoranlar ve kafeler',
  fashion: 'Giyim ve moda',
  tech: 'Teknoloji',
  sport: 'Spor ve fitness',
  culture: 'Eğlence ve etkinlikler',
  other: 'Diğer avantajlar',
};
const partnership = {
  discussing: 'Görüşülüyor',
  contacted: 'İletişim kuruldu',
  awaiting: 'Anlaşma bekleniyor',
  active: 'Aktif',
  ended: 'Sonlandırıldı',
};
const imageUrls = Object.fromEntries(
  Object.entries(
    import.meta.glob('../../../assets/*.svg', {
      eager: true,
      query: '?url',
      import: 'default',
    }),
  ).map(([path, url]) => ['assets/' + path.split('/').at(-1), String(url)]),
);
const states = {
  draft: 'Taslak',
  published: 'Yayımlanmış',
  active: 'Aktif',
  scheduled: 'Yakında',
  expired: 'Süresi doldu',
  suspended: 'Askıda',
};
function date(value: string) {
  if (!value) return 'Belirtilmedi';
  return new Intl.DateTimeFormat('tr-TR', {
    dateStyle: 'medium',
    ...(value.includes('T') ? { timeStyle: 'short' as const } : {}),
    timeZone: 'Europe/Istanbul',
  }).format(new Date(value.includes('T') ? value : `${value}T12:00:00+03:00`));
}
function check<T>(result: {
  data?: T;
  error?: unknown;
  response: Response;
}): T {
  if (result.error || !result.data)
    throw new Error(
      result.response.status === 409
        ? 'Kayıt başka bir işlemle değişti. Girdiğin bilgiler korundu. Güncel kaydı açıkça yeniden yükle ve değişikliklerini karşılaştır.'
        : typeof result.error === 'object' &&
            result.error &&
            'message' in result.error
          ? String(result.error.message)
          : 'İşlem tamamlanamadı. Oturumunu ve bağlantını kontrol edip yeniden dene.',
    );
  return result.data;
}
export function Management({
  session,
  accounts,
  logout,
}: {
  session: Session;
  accounts: ReactNode;
  logout: ReactNode;
}) {
  const [hash, setHash] = useState(location.hash);
  const [open, setOpen] = useState(false);
  const [pinned, setPinned] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const menu = useRef<HTMLElement>(null);
  const section = hash.split('/')[1] || 'overview';
  function close() {
    setOpen(false);
    trigger.current?.focus();
  }
  useEffect(() => {
    const update = () => {
      setHash(location.hash);
      setOpen(false);
    };
    window.addEventListener('hashchange', update);
    return () => window.removeEventListener('hashchange', update);
  }, []);
  useEffect(() => {
    document.body.classList.add('is-admin');
    document.body.classList.toggle('admin-nav-pinned', pinned);
    document.body.classList.toggle('admin-menu-open', open);
    return () =>
      document.body.classList.remove(
        'is-admin',
        'admin-nav-pinned',
        'admin-menu-open',
      );
  }, [open, pinned]);
  useEffect(() => {
    if (!open) return;
    menu.current?.querySelector<HTMLElement>('[aria-current="page"]')?.focus();
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        close();
      }
      if (e.key === 'Tab') {
        const items = Array.from(
          menu.current?.querySelectorAll<HTMLElement>('a,button') || [],
        );
        if (e.shiftKey && document.activeElement === items[0]) {
          e.preventDefault();
          items.at(-1)?.focus();
        } else if (!e.shiftKey && document.activeElement === items.at(-1)) {
          e.preventDefault();
          items[0]?.focus();
        }
      }
    };
    window.addEventListener('keydown', key);
    return () => window.removeEventListener('keydown', key);
  }, [open]);
  return (
    <>
      <aside ref={menu} className="sidebar" aria-label="Yönetici ana menüsü">
        <a className="brand" href="#admin/overview">
          <span className="brand-mark">
            k<span>✦</span>
          </span>
          <span>KoçBenefits</span>
        </a>
        <div className="sidebar-label">
          <span>YÖNETİCİ PANELİ</span>
          <button
            className="menu-pin"
            aria-label="Menüyü sabitle"
            aria-pressed={pinned}
            onClick={() => setPinned(!pinned)}
          >
            ⌖
          </button>
        </div>
        <nav id="desktop-nav">
          {nav.map(([id, label]) => (
            <a
              key={id}
              className={`nav-link ${section === id ? 'active' : ''}`}
              href={`#admin/${id}`}
              aria-label={label}
              title={label}
              aria-current={section === id ? 'page' : undefined}
              onClick={() => {
                if (window.matchMedia('(max-width:760px)').matches) close();
              }}
            >
              <MenuIcon section={id} />
              <span>{label}</span>
            </a>
          ))}
        </nav>
        <button className="mobile-menu-close btn btn-secondary" onClick={close}>
          Menüyü kapat
        </button>
        <div className="sidebar-bottom">
          <a className="nav-link" href="#account" aria-label="Hesabım">
            <b className="icon">♙</b>
            <span>Hesabım</span>
          </a>
        </div>
      </aside>
      <button
        className="menu-scrim"
        aria-label="Menüyü kapat"
        hidden={!open}
        onClick={close}
      />
      <div className="app-body">
        <header className="topbar">
          <button
            ref={trigger}
            className="admin-menu-button"
            aria-expanded={open}
            aria-controls="desktop-nav"
            onClick={() => (open ? close() : setOpen(true))}
          >
            ☰ Menü
          </button>
          <span>
            Yönetim /{' '}
            {nav.find((n) => n[0] === section)?.[1] || 'Sayfa bulunamadı'}
          </span>
          <div className="management-account">
            {session.user?.name} {logout}
          </div>
        </header>
        <main
          className="main-content management-content"
          id="management-content"
        >
          <div className="page-heading admin-heading">
            <div>
              <div className="admin-eyebrow">YÖNETİM ALANI</div>
              <h1>
                {nav.find((n) => n[0] === section)?.[1] || 'Sayfa bulunamadı'}
              </h1>
            </div>
          </div>
          {section === 'students' || section === 'users' ? (
            accounts
          ) : [
              'partners',
              'businesses',
              'benefits',
              'published',
              'archived',
            ].includes(section) ? (
            <Records
              key={section}
              section={section}
              csrf={session.csrf_token}
              accountId={session.user!.id}
            />
          ) : section === 'overview' ? (
            <section className="surface management-intro">
              <h2>İş ortaklıklarından yayına.</h2>
              <p>
                İşletmeyi ekle, avantaj taslağını hazırla ve koşulları
                inceleyerek yayımla.
              </p>
              <div className="account-actions">
                {nav
                  .filter((n) =>
                    ['partners', 'businesses', 'benefits', 'students'].includes(
                      n[0],
                    ),
                  )
                  .map(([id, label]) => (
                    <a
                      key={id}
                      className="btn btn-secondary"
                      href={`#admin/${id}`}
                    >
                      {label} →
                    </a>
                  ))}
              </div>
            </section>
          ) : (
            <section className="surface management-intro">
              <h2>
                {section === 'usage'
                  ? 'Doğrulanmış kullanım kaydı bulunmuyor.'
                  : 'Sonraki teslim'}
              </h2>
              <p>
                {section === 'usage'
                  ? 'Kod veya QR görüntülemek doğrulanmış kullanım sayılmaz. Kullanım doğrulama altyapısı henüz mevcut değil.'
                  : 'Bu bölüm 3E yönetim bütünlüğü tesliminde gerçek verilerle tamamlanacak.'}
              </p>
              <a className="text-link" href="#admin/businesses">
                İşletmelere dön →
              </a>
            </section>
          )}
        </main>
      </div>
    </>
  );
}
const blankBusiness = (): Business => ({
  id: crypto.randomUUID(),
  revision: 0,
  name: '',
  category: 'food',
  description: '',
  image_path: 'assets/cafe.svg',
  contact_person: '',
  contact_info: '',
  branches: '',
  partnership_status: 'discussing',
  last_contact: '',
  notes: '',
  created_at: '',
  updated_at: '',
});
const blankBenefit = (): Benefit => ({
  id: crypto.randomUUID(),
  revision: 0,
  business_id: '',
  business_name: '',
  category: 'food',
  title: '',
  description: '',
  perk: '',
  conditions: [],
  instructions: [],
  products: '',
  branches: '',
  exceptions: '',
  minimum_spend: '',
  currency: 'TRY',
  verification_requirement: '',
  valid_from: '',
  valid_until: '',
  method: 'code',
  code: '',
  target_url: '',
  qr_content: '',
  membership_label: '',
  support_contact: '',
  featured: false,
  publication_status: 'draft',
  status: 'draft',
  suspension_reason: '',
  reviewed_at: '',
  published_at: '',
  created_at: '',
  updated_at: '',
});
function Records({
  section,
  csrf,
  accountId,
}: {
  section: string;
  csrf: string;
  accountId: string;
}) {
  const businessMode = section === 'partners' || section === 'businesses';
  const client = useQueryClient();
  const [q, setQ] = useState('');
  const [category, setCategory] = useState('');
  const [status, setStatus] = useState('');
  const [editing, setEditing] = useState<Business | Benefit | null>(null);
  const [detail, setDetail] = useState<Business | Benefit | null>(null);
  const [notice, setNotice] = useState('');
  const businesses = useQuery({
    queryKey: [
      'management-businesses',
      accountId,
      q,
      category,
      status,
      businessMode,
    ],
    queryFn: async () =>
      check(
        await api.GET('/api/admin/businesses', {
          params: {
            query: businessMode
              ? {
                  q,
                  category: category || undefined,
                  status: status || undefined,
                }
              : {},
          },
        }),
      ),
  });
  const benefits = useQuery({
    queryKey: ['management-benefits', accountId, q, category, status, section],
    enabled: !businessMode,
    queryFn: async () =>
      check(
        await api.GET('/api/admin/benefits', {
          params: {
            query: {
              q,
              category: category || undefined,
              status:
                status ||
                (['published', 'archived'].includes(section)
                  ? section
                  : undefined),
            },
          },
        }),
      ),
  });
  const listing = businessMode ? businesses : benefits;
  const items = businessMode
    ? businesses.data?.businesses
    : benefits.data?.benefits;
  return (
    <>
      <div className="heading-actions">
        <button
          className="btn btn-primary"
          onClick={() => {
            setEditing(businessMode ? blankBusiness() : blankBenefit());
            setDetail(null);
          }}
        >
          {businessMode ? 'İşletme ekle' : 'Avantaj taslağı ekle'} +
        </button>
      </div>
      {notice && (
        <div className="auth-notice" role="status">
          {notice}
        </div>
      )}
      <div className="admin-toolbar">
        <label className="field">
          Ara
          <input
            type="search"
            value={q}
            maxLength={100}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Ad, başlık veya açıklama"
          />
        </label>
        <label className="field">
          Kategori
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            <option value="">Tüm kategoriler</option>
            {Object.entries(categories).map(([v, l]) => (
              <option key={v} value={v}>
                {l}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          Durum
          <select value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">Tüm durumlar</option>
            {Object.entries(
              businessMode
                ? partnership
                : section === 'published'
                  ? {
                      active: 'Aktif',
                      scheduled: 'Yakında',
                      expired: 'Süresi doldu',
                    }
                  : section === 'archived'
                    ? { expired: 'Süresi doldu', suspended: 'Askıda' }
                    : states,
            ).map(([v, l]) => (
              <option key={v} value={v}>
                {l}
              </option>
            ))}
          </select>
        </label>
      </div>
      {listing.data?.truncated && (
        <div className="auth-notice" role="status">
          İlk 1000 eşleşme gösteriliyor. Arama ve filtrelerle sonuçları daralt.
        </div>
      )}
      {listing.isPending ? (
        <p role="status">Kayıtlar yükleniyor…</p>
      ) : listing.isError ? (
        <div role="alert" className="form-error">
          {friendlyError(listing.error)}
          <button
            className="btn btn-secondary"
            onClick={() => void listing.refetch()}
          >
            Yeniden dene
          </button>
        </div>
      ) : items?.length ? (
        <div
          className="table-scroll"
          tabIndex={0}
          role="region"
          aria-label="Yönetim kayıtları"
        >
          <table className="admin-table">
            <thead>
              <tr>
                <th scope="col">
                  {businessMode ? 'İşletme' : 'Avantaj / işletme'}
                </th>
                <th scope="col">Durum</th>
                <th scope="col">Tarihler</th>
                <th scope="col">İşlemler</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id}>
                  <td>
                    <strong>{'name' in item ? item.name : item.title}</strong>
                    <small>
                      {'business_name' in item
                        ? item.business_name
                        : categories[item.category]}
                    </small>
                  </td>
                  <td>
                    <span className="status-badge">
                      {'partnership_status' in item
                        ? partnership[item.partnership_status]
                        : states[item.status]}
                    </span>
                    {'publication_status' in item && (
                      <small>Yayın: {states[item.publication_status]}</small>
                    )}
                    {'suspension_reason' in item && item.suspension_reason && (
                      <small>{item.suspension_reason}</small>
                    )}
                  </td>
                  <td>
                    {'valid_from' in item ? (
                      <>
                        <small>Başlangıç: {date(item.valid_from)}</small>
                        <small>Bitiş: {date(item.valid_until)}</small>
                      </>
                    ) : (
                      <small>Son görüşme: {date(item.last_contact)}</small>
                    )}
                    {'published_at' in item && (
                      <small>Yayın: {date(item.published_at)}</small>
                    )}
                    <small>Güncelleme: {date(item.updated_at)}</small>
                  </td>
                  <td>
                    <div className="account-actions">
                      <button
                        className="btn btn-secondary"
                        onClick={() => setDetail(item)}
                      >
                        Detay
                      </button>
                      <button
                        className="btn btn-secondary"
                        onClick={() => {
                          setEditing(item);
                          setDetail(null);
                        }}
                      >
                        Düzenle
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <section className="admin-empty">
          <h2>Eşleşen kayıt bulunamadı.</h2>
          <p>Arama ve filtreleri değiştir veya yeni kayıt ekle.</p>
        </section>
      )}
      {detail && (
        <section className="surface management-intro">
          <button className="btn btn-secondary" onClick={() => setDetail(null)}>
            Detayı kapat
          </button>
          <h2>{'name' in detail ? detail.name : detail.title}</h2>
          <dl className="account-details">
            {Object.entries(detail)
              .filter(
                ([key]) =>
                  ![
                    'code',
                    'target_url',
                    'qr_content',
                    'membership_label',
                  ].includes(key) ||
                  ('method' in detail &&
                    {
                      code: 'code',
                      link: 'target_url',
                      qr: 'qr_content',
                      membership: 'membership_label',
                    }[detail.method] === key),
              )
              .map(([key, value]) => (
                <div key={key}>
                  <dt>{fieldLabels[key] || key}</dt>
                  <dd>
                    {key === 'image_path' && imageUrls[String(value)] ? (
                      <img
                        className="management-image"
                        src={imageUrls[String(value)]}
                        alt="İşletme görseli"
                      />
                    ) : key.endsWith('_at') ||
                      ['valid_from', 'valid_until', 'last_contact'].includes(
                        key,
                      ) ? (
                      date(String(value))
                    ) : key === 'category' ? (
                      categories[value as keyof typeof categories]
                    ) : key === 'partnership_status' ? (
                      partnership[value as keyof typeof partnership]
                    ) : ['status', 'publication_status'].includes(key) ? (
                      states[value as keyof typeof states]
                    ) : key === 'method' ? (
                      {
                        code: 'İndirim kodu',
                        link: 'Bağlantı',
                        qr: 'QR',
                        membership: 'Üyelik kartı',
                      }[value as Benefit['method']]
                    ) : Array.isArray(value) ? (
                      value.join('\n') || 'Belirtilmedi'
                    ) : typeof value === 'boolean' ? (
                      value ? (
                        'Evet'
                      ) : (
                        'Hayır'
                      )
                    ) : (
                      String(value) || 'Belirtilmedi'
                    )}
                  </dd>
                </div>
              ))}
          </dl>
        </section>
      )}
      {editing && (
        <Editor
          key={editing.id}
          initial={editing}
          csrf={csrf}
          businesses={businesses.data?.businesses || []}
          onClose={() => setEditing(null)}
          onSaved={async (message) => {
            setNotice(message);
            setEditing(null);
            await client.invalidateQueries({
              queryKey: ['management-businesses'],
            });
            await client.invalidateQueries({
              queryKey: ['management-benefits'],
            });
          }}
        />
      )}
    </>
  );
}
const fieldLabels: Record<string, string> = {
  name: 'İşletme adı',
  category: 'Kategori',
  description: 'Açıklama',
  image_path: 'Hazır görsel',
  contact_person: 'İlgili kişi',
  contact_info: 'İletişim bilgileri',
  branches: 'Geçerli şubeler',
  partnership_status: 'İş ortaklığı durumu',
  last_contact: 'Son görüşme',
  notes: 'Notlar',
  business_id: 'İşletme',
  business_name: 'İşletme adı',
  title: 'Avantaj başlığı',
  perk: 'İndirim / ayrıcalık',
  conditions: 'Koşullar (her satıra bir koşul)',
  instructions: 'Kullanım adımları (her satıra bir adım)',
  products: 'Ürün / hizmet kapsamı',
  exceptions: 'İstisnalar',
  minimum_spend: 'Minimum harcama (TRY)',
  currency: 'Para birimi',
  verification_requirement: 'Doğrulama gereksinimi',
  valid_from: 'Başlangıç tarihi',
  valid_until: 'Bitiş tarihi (dahil)',
  method: 'Kullanım yöntemi',
  code: 'İndirim kodu',
  target_url: 'Hedef bağlantı',
  qr_content: 'QR içeriği',
  membership_label: 'Üyelik kartı metni',
  support_contact: 'Destek iletişimi',
  featured: 'Öne çıkar',
  publication_status: 'Yayın durumu',
  status: 'Geçerlilik durumu',
  suspension_reason: 'Askıya alma nedeni',
  reviewed_at: 'Koşul inceleme zamanı',
  published_at: 'Yayın zamanı',
  created_at: 'Oluşturulma zamanı',
  updated_at: 'Güncelleme zamanı',
  revision: 'Kayıt sürümü',
  id: 'Kayıt kimliği',
};
function Editor({
  initial,
  csrf,
  businesses,
  onClose,
  onSaved,
}: {
  initial: Business | Benefit;
  csrf: string;
  businesses: Business[];
  onClose: () => void;
  onSaved: (message: string) => Promise<void>;
}) {
  const [value, setValue] = useState(initial);
  const [baseline, setBaseline] = useState(initial);
  const business = 'name' in value;
  const dirty = JSON.stringify(value) !== JSON.stringify(baseline);
  const [action, setAction] = useState<'save' | 'publish' | 'suspend' | null>(
    null,
  );
  const [reason, setReason] = useState('');
  const [reviewed, setReviewed] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const editor = useRef<HTMLElement>(null);
  useEffect(() => {
    editor.current?.focus();
    editor.current?.scrollIntoView({ block: 'start', behavior: 'auto' });
  }, []);
  const mutation = useMutation({
    mutationFn: async () => {
      const params = {
        path: { id: value.id },
        header: { 'X-CSRF-Token': csrf, Origin: location.origin },
      };
      if ('name' in value) {
        check(
          await api.PUT('/api/admin/businesses/{id}', {
            params,
            body: writeBody(value),
          }),
        );
      } else if (action === 'publish') {
        check(
          await api.POST('/api/admin/benefits/{id}/publish', {
            params,
            body: { revision: value.revision, conditions_reviewed: true },
          }),
        );
      } else if (action === 'suspend') {
        check(
          await api.POST('/api/admin/benefits/{id}/suspend', {
            params,
            body: { revision: value.revision, reason },
          }),
        );
      } else {
        check(
          await api.PUT('/api/admin/benefits/{id}', {
            params,
            body: writeBody(value),
          }),
        );
      }
    },
    onSuccess: () =>
      onSaved(
        action === 'publish'
          ? 'Avantaj yayımlandı.'
          : action === 'suspend'
            ? 'Avantaj askıya alındı.'
            : business
              ? 'İşletme kaydedildi.'
              : 'Avantaj taslak olarak kaydedildi.',
      ),
    onError: (e) => {
      setError(friendlyError(e));
      setAction(null);
    },
  });
  async function reload() {
    if (
      !window.confirm(
        'Girilen değişiklikler yerine sunucudaki güncel kayıt yüklenecek. Devam edilsin mi?',
      )
    )
      return;
    setLoading(true);
    try {
      const params = { path: { id: value.id } };
      const next = business
        ? check(await api.GET('/api/admin/businesses/{id}', { params }))
            .business
        : check(await api.GET('/api/admin/benefits/{id}', { params })).benefit;
      setValue(next);
      setBaseline(next);
      setError('');
      setAction(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Kayıt yüklenemedi.');
    } finally {
      setLoading(false);
    }
  }
  function field(key: string, type = 'text', required = false) {
    const current = value[key as keyof typeof value];
    return (
      <label className="field" key={key}>
        {fieldLabels[key]}
        {required ? ' *' : ''}
        {type === 'textarea' ? (
          <textarea
            name={key}
            id={`edit-${key}`}
            rows={3}
            required={required}
            maxLength={2500}
            value={
              Array.isArray(current)
                ? current.join('\n')
                : String(current ?? '')
            }
            onChange={(e) =>
              setValue({
                ...value,
                [key]: Array.isArray(current)
                  ? e.target.value.split('\n')
                  : e.target.value,
              })
            }
          />
        ) : (
          <input
            name={key}
            id={`edit-${key}`}
            type={type}
            required={required}
            step={type === 'number' ? '0.01' : undefined}
            min={type === 'number' ? '0' : undefined}
            maxLength={2500}
            value={String(current ?? '')}
            onChange={(e) => setValue({ ...value, [key]: e.target.value })}
          />
        )}
      </label>
    );
  }
  function select(key: string, entries: Record<string, string>) {
    return (
      <label className="field" key={key}>
        {fieldLabels[key]}
        <select
          name={key}
          id={`edit-${key}`}
          disabled={key === 'business_id' && value.revision > 0}
          required
          value={String(value[key as keyof typeof value] ?? '')}
          onChange={(e) => setValue({ ...value, [key]: e.target.value })}
        >
          {Object.entries(entries).map(([v, l]) => (
            <option key={v} value={v}>
              {l}
            </option>
          ))}
        </select>
      </label>
    );
  }
  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setReviewed(false);
    setAction('save');
  }
  const needsWarning = business
    ? 'name' in initial &&
      initial.partnership_status === 'active' &&
      value.partnership_status !== 'active'
    : 'publication_status' in initial &&
      initial.publication_status === 'published';
  return (
    <section
      ref={editor}
      tabIndex={-1}
      className="management-editor"
      aria-label="Kayıt formu"
    >
      <h2>{value.revision ? 'Kaydı düzenle' : 'Yeni kayıt'}</h2>
      <p>
        İsteğe bağlı tarihler boş bırakılabilir. Tarihler Europe/Istanbul saat
        dilimindedir.
      </p>
      {!business && 'publication_status' in value && (
        <div className="auth-notice">
          Yayın: {states[value.publication_status]} · Durum:{' '}
          {states[value.status]}
          <br />
          Başlangıç: {date(value.valid_from)} · Bitiş: {date(value.valid_until)}
          <br />
          Koşul inceleme: {date(value.reviewed_at)} · Yayın:{' '}
          {date(value.published_at)}
          {value.suspension_reason && (
            <p>Askı nedeni: {value.suspension_reason}</p>
          )}
        </div>
      )}
      {error && (
        <div role="alert" className="form-error">
          {error}
          <button
            type="button"
            className="btn btn-secondary"
            disabled={loading}
            onClick={() => void reload()}
          >
            Güncel kaydı yeniden yükle
          </button>
        </div>
      )}
      <form onSubmit={submit}>
        <fieldset disabled={mutation.isPending || loading || action !== null}>
          {business ? (
            <>
              <Group title="İşletme bilgileri">
                {field('name', 'text', true)}
                {select('category', categories)}
                {field('description', 'textarea')}
                {select(
                  'image_path',
                  Object.fromEntries(
                    [
                      'cafe',
                      'fashion',
                      'tech',
                      'fitness',
                      'culture',
                      'books',
                    ].map((x) => [`assets/${x}.svg`, x]),
                  ),
                )}
                {field('branches', 'textarea')}
              </Group>
              <Group title="İş ortaklığı takibi">
                {select('partnership_status', partnership)}
                {field('contact_person')}
                {field('contact_info')}
                {field('last_contact', 'date')}
                {field('notes', 'textarea')}
              </Group>
            </>
          ) : (
            <>
              <Group title="Avantaj bilgileri">
                {value.revision > 0 && (
                  <p>
                    İşletme sonradan değiştirilemez. Başka bir işletme için yeni
                    avantaj oluştur.
                  </p>
                )}
                {select('business_id', {
                  '': 'İşletme seç',
                  ...Object.fromEntries(
                    businesses.map((b) => [
                      b.id,
                      `${b.name} · ${partnership[b.partnership_status]}`,
                    ]),
                  ),
                })}
                {field('title', 'text', true)}
                {field('perk')}
                {field('description', 'textarea')}
                {field('valid_from', 'date')}
                {field('valid_until', 'date')}
                <label className="confirmation-check">
                  <input
                    type="checkbox"
                    checked={'featured' in value && value.featured}
                    onChange={(e) =>
                      setValue({ ...value, featured: e.target.checked })
                    }
                  />
                  Öne çıkar
                </label>
              </Group>
              <Group title="Koşullar ve kapsam">
                {field('conditions', 'textarea')}
                {field('products', 'textarea')}
                {field('branches', 'textarea')}
                {field('exceptions', 'textarea')}
                {field('minimum_spend', 'number')}
                {field('verification_requirement', 'textarea')}
              </Group>
              <Group title="Kullanım yöntemi">
                {select('method', {
                  code: 'İndirim kodu',
                  link: 'Bağlantı',
                  qr: 'QR',
                  membership: 'Üyelik kartı',
                })}
                {'method' in value &&
                  field(
                    {
                      code: 'code',
                      link: 'target_url',
                      qr: 'qr_content',
                      membership: 'membership_label',
                    }[value.method],
                    value.method === 'link' ? 'url' : 'text',
                    false,
                  )}
                {field('instructions', 'textarea')}
                {field('support_contact')}
                <p className="foundation-note">
                  QR ve kart alanları işletmenin kabul edeceği içeriği kaydeder.
                  Bu ekran doğrulanmış kullanım veya gerçek kullanım hakkı
                  üretmez.
                </p>
              </Group>
            </>
          )}
          <div className="account-actions">
            <button type="submit" className="btn btn-primary">
              {business ? 'Kaydet' : 'Taslak olarak kaydet'}
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onClose}
            >
              Vazgeç
            </button>
            {!business && dirty && (
              <p role="status">Önce değişiklikleri taslak olarak kaydet.</p>
            )}
            {!business && value.revision > 0 && (
              <>
                <button
                  type="button"
                  className="btn btn-secondary"
                  disabled={dirty}
                  onClick={() => {
                    if (dirty) {
                      setError(
                        'Yayın işleminden önce değişiklikleri taslak olarak kaydet.',
                      );
                      return;
                    }
                    setReviewed(false);
                    setAction('publish');
                  }}
                >
                  Koşulları incele ve yayımla
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  disabled={dirty}
                  onClick={() => {
                    if (dirty) {
                      setError(
                        'Askıya almadan önce değişiklikleri taslak olarak kaydet.',
                      );
                      return;
                    }
                    setAction('suspend');
                  }}
                >
                  Askıya al
                </button>
              </>
            )}
          </div>
        </fieldset>
      </form>
      {action && (
        <Confirmation
          pending={mutation.isPending}
          onClose={() => setAction(null)}
          onConfirm={() => mutation.mutate()}
          ready={
            action === 'publish'
              ? reviewed
              : action === 'suspend'
                ? !!reason.trim()
                : true
          }
        >
          <h2>
            {action === 'publish'
              ? 'Yayını onayla'
              : action === 'suspend'
                ? 'Askıya almayı onayla'
                : 'Kaydetmeyi onayla'}
          </h2>
          <p>
            {action === 'save'
              ? needsWarning
                ? business
                  ? 'Aktif iş ortaklığını kapatmak bağlı yayımlanmış avantajları askıya alır. Yeniden açmak otomatik yayımlamaz.'
                  : 'Yayımlanmış avantajı düzenlemek yayını kapatır ve taslak kaydeder. Yeniden koşul incelemesi ve yayın gerekir.'
                : business
                  ? 'İşletme bilgileri kaydedilecek.'
                  : 'Avantaj yalnızca taslak olarak kaydedilecek.'
              : action === 'publish'
                ? 'İşletmenin aktif olduğunu, tarihleri, koşulları ve kullanım yöntemini kontrol et.'
                : 'Avantaj öğrenci erişiminden kaldırılacak.'}
          </p>
          {action === 'publish' && (
            <>
              <p>
                {'title' in value && value.title} ·{' '}
                {'valid_from' in value && date(value.valid_from)} —{' '}
                {'valid_until' in value && date(value.valid_until)}
              </p>
              <ul>
                {'conditions' in value &&
                  value.conditions.map((c, i) => <li key={i}>{c}</li>)}
              </ul>
              <label className="confirmation-check">
                <input
                  type="checkbox"
                  checked={reviewed}
                  onChange={(e) => setReviewed(e.target.checked)}
                />
                Koşulları, kapsamı ve kullanım bilgilerini inceledim; yayını
                onaylıyorum.
              </label>
            </>
          )}
          {action === 'suspend' && (
            <label className="field">
              Askıya alma nedeni
              <textarea
                required
                value={reason}
                maxLength={2500}
                onChange={(e) => setReason(e.target.value)}
              />
            </label>
          )}
        </Confirmation>
      )}
    </section>
  );
}
function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="surface form-group">
      <div className="form-group-heading">
        <h3>{title}</h3>
      </div>
      <div className="management-grid">{children}</div>
    </section>
  );
}
function Confirmation({
  children,
  pending,
  ready,
  onClose,
  onConfirm,
}: {
  children: ReactNode;
  pending: boolean;
  ready: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const previous = document.activeElement as HTMLElement;
    const el = ref.current;
    el?.showModal();
    return () => {
      el?.close();
      previous?.focus();
    };
  }, []);
  return (
    <dialog
      ref={ref}
      className="account-dialog"
      onCancel={(e) => {
        if (pending) e.preventDefault();
        else onClose();
      }}
    >
      {children}
      <div className="account-actions">
        <button
          className="btn btn-secondary"
          disabled={pending}
          onClick={onClose}
        >
          Vazgeç
        </button>
        <button
          className="btn btn-primary"
          disabled={pending || !ready}
          onClick={onConfirm}
        >
          {pending ? 'İşlem sürüyor…' : 'Onayla'}
        </button>
      </div>
    </dialog>
  );
}

function writeBody(value: Business): components['schemas']['BusinessWrite'];
function writeBody(value: Benefit): components['schemas']['BenefitWrite'];
function writeBody(
  value: Business | Benefit,
):
  | components['schemas']['BusinessWrite']
  | components['schemas']['BenefitWrite'] {
  if ('name' in value) {
    const {
      revision,
      name,
      category,
      description,
      image_path,
      contact_person,
      contact_info,
      branches,
      partnership_status,
      last_contact,
      notes,
    } = value;
    return {
      revision,
      name,
      category,
      description,
      image_path,
      contact_person,
      contact_info,
      branches,
      partnership_status,
      last_contact,
      notes,
    };
  }
  const {
    revision,
    business_id,
    title,
    description,
    perk,
    conditions,
    instructions,
    products,
    branches,
    exceptions,
    minimum_spend,
    currency,
    verification_requirement,
    valid_from,
    valid_until,
    method,
    code,
    target_url,
    qr_content,
    membership_label,
    support_contact,
    featured,
  } = value;
  return {
    revision,
    business_id,
    title,
    description,
    perk,
    conditions: conditions.filter((c) => c.trim()),
    instructions: instructions.filter((c) => c.trim()),
    products,
    branches,
    exceptions,
    minimum_spend,
    currency,
    verification_requirement,
    valid_from,
    valid_until,
    method,
    code: method === 'code' ? code : '',
    target_url: method === 'link' ? target_url : '',
    qr_content: method === 'qr' ? qr_content : '',
    membership_label: method === 'membership' ? membership_label : '',
    support_contact,
    featured,
  };
}

function MenuIcon({ section }: { section: string }) {
  const paths: Record<string, ReactNode> = {
    overview: (
      <>
        <rect x="3" y="3" width="7" height="7" rx="1" />
        <rect x="14" y="3" width="7" height="7" rx="1" />
        <rect x="3" y="14" width="7" height="7" rx="1" />
        <rect x="14" y="14" width="7" height="7" rx="1" />
      </>
    ),
    partners: (
      <>
        <circle cx="9" cy="7" r="3" />
        <path d="M3 21v-3a6 6 0 0 1 12 0v3M16 4a3 3 0 0 1 0 6m2 5a5 5 0 0 1 3 5" />
      </>
    ),
    businesses: (
      <>
        <path d="M4 21V3h16v18M2 21h20M9 21v-5h6v5M8 7h1m6 0h1M8 11h1m6 0h1" />
      </>
    ),
    benefits: (
      <path d="M3 6h18v4a2 2 0 0 0 0 4v4H3v-4a2 2 0 0 0 0-4zM15 7v2m0 2v2m0 2v2" />
    ),
    students: (
      <>
        <path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6zM8 12l3 3 5-6" />
      </>
    ),
    published: (
      <>
        <circle cx="12" cy="12" r="9" />
        <path d="m7 12 3 3 7-7" />
      </>
    ),
    archived: (
      <>
        <circle cx="12" cy="12" r="9" />
        <path d="M9 8v8m6-8v8" />
      </>
    ),
    usage: (
      <path d="M3 3h6v6H3zm12 0h6v6h-6zM3 15h6v6H3zm12 0h3v3h3v3h-6zm6-3v3" />
    ),
    analytics: <path d="M3 3v18h18M7 17v-4m5 4V9m5 8V5" />,
    users: (
      <>
        <circle cx="12" cy="7" r="4" />
        <path d="M4 21v-2a8 8 0 0 1 16 0v2" />
      </>
    ),
    audit: <path d="M8 6h13M8 12h13M8 18h13M3 6h1m-1 6h1m-1 6h1" />,
    settings: (
      <>
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v3m0 14v3M2 12h3m14 0h3M5 5l2 2m10 10 2 2M5 19l2-2M17 7l2-2" />
      </>
    ),
  };
  return (
    <svg
      className="icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {paths[section] || paths.users}
    </svg>
  );
}

function friendlyError(error: unknown) {
  return error instanceof Error && error.name !== 'TypeError'
    ? error.message
    : 'İşlem tamamlanamadı. Bağlantını kontrol edip yeniden dene.';
}
