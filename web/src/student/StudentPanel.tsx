import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { components } from '../api/schema.gen';
import {
  categories,
  copyText,
  methodLabels,
  safeURL,
  statusLabels,
  studentRequest,
  type Benefit,
  type Catalog,
  type Redemption,
} from './student-api';
import '../student.css';
import qrcode from 'qrcode-generator';

qrcode.stringToBytes = (value) => Array.from(new TextEncoder().encode(value));

const images = Object.fromEntries(
  Object.entries(
    import.meta.glob(
      '../../../assets/{cafe,fashion,tech,fitness,culture,books,placeholder}.svg',
      { eager: true, query: '?url', import: 'default' },
    ),
  ).map(([path, url]) => ['assets/' + path.split('/').at(-1), String(url)]),
);
const sections = [
  ['overview', 'Genel Bakış', 'Genel Bakış', 'grid'],
  ['discover', 'Avantajları Keşfet', 'Keşfet', 'compass'],
  ['favorites', 'Favorilerim', 'Favoriler', 'heart'],
  ['history', 'Kullanım Geçmişi', 'Geçmiş', 'clock'],
  ['profile', 'Profil ve Hesap', 'Profil', 'user'],
  ['help', 'Yardım ve Destek', 'Yardım', 'help'],
];
function Icon({ name }: { name: string }) {
  const paths: Record<string, ReactNode> = {
    grid: (
      <>
        <rect x="3" y="3" width="7" height="7" rx="1" />
        <rect x="14" y="3" width="7" height="7" rx="1" />
        <rect x="3" y="14" width="7" height="7" rx="1" />
        <rect x="14" y="14" width="7" height="7" rx="1" />
      </>
    ),
    compass: (
      <>
        <circle cx="12" cy="12" r="9" />
        <path d="m16 8-3 5-5 3 3-5Z" />
      </>
    ),
    heart: <path d="M12 21 3 12a5.5 5.5 0 0 1 9-7 5.5 5.5 0 0 1 9 7Z" />,
    clock: (
      <>
        <circle cx="12" cy="12" r="9" />
        <path d="M12 6v6l4 3" />
      </>
    ),
    user: (
      <>
        <circle cx="12" cy="8" r="4" />
        <path d="M4 21v-2a8 8 0 0 1 16 0v2" />
      </>
    ),
    help: (
      <>
        <circle cx="12" cy="12" r="9" />
        <path d="M9 9a3 3 0 1 1 5 2c-2 1-2 2-2 3m0 3h.01" />
      </>
    ),
    arrow: <path d="M4 12h16m-6-6 6 6-6 6" />,
    search: (
      <>
        <circle cx="10" cy="10" r="6" />
        <path d="m15 15 6 6" />
      </>
    ),
  };
  return (
    <svg className="icon" viewBox="0 0 24 24" aria-hidden="true">
      {paths[name] || paths.compass}
    </svg>
  );
}
function Empty({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="empty-state">
      <Icon name="compass" />
      <h2>{title}</h2>
      <p>{children}</p>
    </div>
  );
}
function Heading({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="page-heading">
      <div>
        <h1>{title}</h1>
        <p>{children}</p>
      </div>
    </div>
  );
}
function Contact({ value }: { value?: string }) {
  if (!value) return <p>Henüz bir destek iletişim bilgisi tanımlanmadı.</p>;
  const href = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
    ? `mailto:${value}`
    : /^\+?[\d ()-]{7,}$/.test(value)
      ? `tel:${value.replace(/[ ()-]/g, '')}`
      : safeURL(value);
  return href ? (
    <a className="text-link" href={href} rel="noopener noreferrer">
      {value}
    </a>
  ) : (
    <p>{value}</p>
  );
}

export function StudentPanel({
  session,
  logout,
}: {
  session: components['schemas']['SessionResponse'];
  logout: ReactNode;
}) {
  const [hash, setHash] = useState(location.hash);
  const [message, setMessage] = useState('');
  useEffect(() => {
    const changed = () => {
      setHash(location.hash);
      setMessage('');
    };
    window.addEventListener('hashchange', changed);
    return () => window.removeEventListener('hashchange', changed);
  }, []);
  const [raw, search = ''] = hash.replace(/^#\/?/, '').split('?');
  const route =
    raw === 'account'
      ? 'profile'
      : !raw || ['login/student', 'login/admin', 'register'].includes(raw)
        ? 'overview'
        : raw;
  const params = new URLSearchParams(search);
  const q = (params.get('q') || '').slice(0, 80);
  const category = categories.some(([id]) => id === params.get('category'))
    ? params.get('category')!
    : '';
  const account = session.user!;
  const accountId = account.id;
  const client = useQueryClient();
  const isDetail = route.startsWith('benefit/');
  const id = isDetail ? route.slice(8) : '';
  const catalog = useQuery({
    queryKey: ['student-catalog', accountId, q, category],
    queryFn: ({ signal }) =>
      studentRequest<Catalog>(
        `catalog?${new URLSearchParams({ q, category })}`,
        { signal },
      ),
    enabled: route === 'overview' || route === 'discover',
    refetchInterval: 60000,
    refetchOnWindowFocus: true,
  });
  const favorites = useQuery({
    queryKey: ['student-favorites', accountId],
    queryFn: ({ signal }) => studentRequest<Catalog>('favorites', { signal }),
    enabled: route === 'favorites',
    refetchInterval: 60000,
    refetchOnWindowFocus: true,
  });
  const settings = useQuery({
    queryKey: ['student-settings', accountId],
    queryFn: ({ signal }) =>
      studentRequest<{ platform_name: string; support_contact: string }>(
        'settings',
        { signal },
      ),
  });
  const favorite = useMutation({
    mutationFn: ({ benefit }: { benefit: Benefit }) =>
      studentRequest(`favorites/${encodeURIComponent(benefit.id)}`, {
        method: benefit.is_favorite ? 'DELETE' : 'PUT',
        headers: {
          'X-CSRF-Token': session.csrf_token,
          'Content-Type': 'application/json',
        },
        body: '{}',
      }),
    onSuccess: () => {
      void client.invalidateQueries({
        queryKey: ['student-catalog', accountId],
      });
      void client.invalidateQueries({
        queryKey: ['student-favorites', accountId],
      });
      void client.invalidateQueries({
        queryKey: ['student-benefit', accountId],
      });
    },
    onError: (e) => setMessage(e.message),
  });
  function Favorite({
    benefit,
    detail = false,
  }: {
    benefit: Benefit;
    detail?: boolean;
  }) {
    return (
      <button
        type="button"
        className={detail ? 'detail-save' : 'favorite-toggle'}
        aria-pressed={benefit.is_favorite}
        aria-label={
          benefit.is_favorite ? 'Favorilerden çıkar' : 'Favorilere ekle'
        }
        disabled={favorite.isPending}
        onClick={() => {
          setMessage('');
          favorite.mutate({ benefit });
        }}
      >
        <Icon name="heart" />
        {detail && (benefit.is_favorite ? 'Kaydedildi' : 'Kaydet')}
      </button>
    );
  }
  function Card({ benefit: b }: { benefit: Benefit }) {
    if (b.status === 'unavailable')
      return (
        <article className="student-unavailable">
          <h3>Artık erişilemeyen avantaj</h3>
          <p>Bu avantajın bilgileri artık gösterilemiyor.</p>
          <Favorite benefit={b} detail />
        </article>
      );
    return (
      <article className="benefit-card">
        <a
          className="card-image-link"
          href={`#benefit/${b.id}`}
          aria-hidden="true"
          tabIndex={-1}
        >
          <img
            src={images[b.image_path] || images['assets/placeholder.svg']}
            alt=""
            width="720"
            height="420"
            loading="lazy"
          />
          {b.perk && (
            <span className="offer-badge">
              <strong>{b.perk}</strong>
            </span>
          )}
          <span className={`student-status student-${b.status}`}>
            {statusLabels[b.status] || 'Kullanıma kapalı'}
          </span>
        </a>
        <Favorite benefit={b} />
        <div className="card-content">
          <div className="card-meta">
            {categories.find(([key]) => key === b.category)?.[1] ||
              'Diğer avantajlar'}
          </div>
          <h3>
            <a href={`#benefit/${b.id}`}>{b.business_name}</a>
          </h3>
          <p>{b.title}</p>
          <div className="card-bottom">
            <span>{methodLabels[b.method] || 'Avantaj'}</span>
            <Icon name="arrow" />
          </div>
        </div>
      </article>
    );
  }
  const list = route === 'favorites' ? favorites : catalog;
  function Cards({ featured = false }: { featured?: boolean }) {
    if (list.isPending) return <Empty title="Avantajlar yükleniyor…" />;
    if (list.isError)
      return (
        <Empty title="Avantajlara ulaşılamadı">
          <span>{list.error.message}</span>
          <button
            className="btn btn-secondary"
            onClick={() => void list.refetch()}
          >
            Yeniden dene
          </button>
        </Empty>
      );
    const items = (list.data?.benefits || []).filter(
      (b) => !featured || (b.featured && b.status === 'active'),
    );
    return (
      <>
        {items.length ? (
          <div className="benefit-grid">
            {items.map((b) => (
              <Card key={b.id} benefit={b} />
            ))}
          </div>
        ) : (
          <Empty
            title={
              route === 'favorites'
                ? 'Henüz kaydettiğin bir avantaj yok.'
                : featured
                  ? 'Yeni keşifler için takipte kal.'
                  : q || category
                    ? 'Aramana uygun bir avantaj bulunamadı.'
                    : 'Henüz yayımlanmış bir avantaj yok.'
            }
          >
            <a className="text-link" href="#discover">
              Avantajları keşfet
            </a>
          </Empty>
        )}
        {list.data?.truncated && (
          <p className="notice">
            Sonuçların bir bölümü gösteriliyor. Arama veya kategori seçerek
            daraltabilirsin.
          </p>
        )}
      </>
    );
  }
  function discover(nextQ: string, nextCategory = category) {
    location.hash =
      'discover?' +
      new URLSearchParams({
        ...(nextQ ? { q: nextQ } : {}),
        ...(nextCategory ? { category: nextCategory } : {}),
      });
  }
  function Search({ header = false }: { header?: boolean }) {
    return (
      <form
        key={hash + (header ? 'header' : 'catalog')}
        className={header ? 'header-search' : 'catalog-search'}
        role="search"
        onSubmit={(e: FormEvent<HTMLFormElement>) => {
          e.preventDefault();
          discover(String(new FormData(e.currentTarget).get('q') || ''));
        }}
      >
        <Icon name="search" />
        <label
          className="sr-only"
          htmlFor={header ? 'student-header-search' : 'student-catalog-search'}
        >
          Avantajlarda ara
        </label>
        <input
          id={header ? 'student-header-search' : 'student-catalog-search'}
          name="q"
          type="search"
          maxLength={80}
          defaultValue={q}
          placeholder="Bir avantaj keşfet…"
        />
        <button type="submit" aria-label="Ara">
          <Icon name="arrow" />
        </button>
      </form>
    );
  }
  const current =
    sections.find(([key]) => key === route)?.[1] ||
    (isDetail ? 'Avantaj detayı' : 'Sayfa bulunamadı');
  const platform = settings.data?.platform_name || 'KoçBenefits';
  const nav = (mobile = false) =>
    sections.map(([key, label, short, icon]) => (
      <a
        key={key}
        href={`#${key}`}
        className={
          route === key || (isDetail && key === 'discover') ? 'active' : ''
        }
        aria-current={route === key ? 'page' : undefined}
      >
        <Icon name={icon} />
        <span>{mobile ? short : label}</span>
      </a>
    ));
  return (
    <div className="student-panel">
      <a
        className="skip-link"
        href="#student-main"
        onClick={(e) => {
          e.preventDefault();
          document.getElementById('student-main')?.focus();
        }}
      >
        İçeriğe geç
      </a>
      <aside className="sidebar">
        <a className="brand" href="#overview">
          <span className="brand-mark">
            k<span>✦</span>
          </span>
          <span>
            {platform}
            <small>ÖĞRENCİYE AYRICALIK</small>
          </span>
        </a>
        <div className="sidebar-label">ÖĞRENCİ PANELİ</div>
        <nav>{nav()}</nav>
        <div className="sidebar-bottom">
          <div className="school-note">
            Birlikte, daha avantajlı.<small>Öğrencilerden, öğrencilere.</small>
          </div>
          {account.role === 'admin' && (
            <a className="text-link" href="#admin/overview">
              Yönetime dön
            </a>
          )}
          {logout}
        </div>
      </aside>
      <div className="app-body">
        <header className="topbar">
          <a className="mobile-brand" href="#overview">
            k<span>✦</span>
          </a>
          <div className="breadcrumb">
            Öğrenci paneli <span>/</span> {current}
          </div>
          <Search header />
          <a className="account-button" href="#profile">
            <span className="avatar">{account.name.slice(0, 1)}</span>
            <span className="account-name">{account.name.split(' ')[0]}</span>
          </a>
        </header>
        <main id="student-main" tabIndex={-1}>
          {message && (
            <p className="notice" role="alert">
              {message}
            </p>
          )}
          {route === 'overview' ? (
            <>
              <Heading title={`Merhaba, ${account.name.split(' ')[0]}.`}>
                Küçük ayrıcalıklar, güzel keşifler. Bugün ne yapmak istersin?
              </Heading>
              <section className="hero">
                <div className="hero-copy">
                  <div className="eyebrow">
                    {platform.toLocaleUpperCase('tr-TR')} İLE TANIŞ
                  </div>
                  <h2>
                    Okullu olmanın
                    <br />
                    bir de <em>bu yanı var.</em>
                  </h2>
                  <p>
                    Kahve molandan hafta sonu planına,
                    <br />
                    sana özel avantajlar burada.
                  </p>
                  <a className="btn btn-primary" href="#discover">
                    Avantajları keşfet
                    <Icon name="arrow" />
                  </a>
                </div>
                <div className="hero-art" aria-hidden="true">
                  <div className="card-back" />
                  <div className="membership-card">
                    <div className="card-brand">{platform}</div>
                    <small>ÖĞRENCİYE AYRICALIK</small>
                    <div className="membership-card-bottom">
                      <strong>
                        Öğrencilerden,
                        <br />
                        öğrencilere.
                      </strong>
                      <span>✦</span>
                    </div>
                  </div>
                  <span className="hero-caption">
                    BİRLİKTE, DAHA AVANTAJLI.
                  </span>
                </div>
              </section>
              <section>
                <div className="section-heading">
                  <h2>Ne keşfetmek istersin?</h2>
                </div>
                <div className="category-grid">
                  {categories.map(([key, label]) => (
                    <a
                      className="category-tile"
                      href={`#discover?category=${key}`}
                      key={key}
                    >
                      <Icon name="compass" />
                      <span>{label}</span>
                    </a>
                  ))}
                </div>
              </section>
              <section>
                <div className="section-heading">
                  <h2>Bir göz at, sevebilirsin.</h2>
                  <a className="text-link" href="#discover">
                    Tümünü gör
                    <Icon name="arrow" />
                  </a>
                </div>
                <Cards featured />
              </section>
              <aside className="how-strip">
                <Icon name="help" />
                <div>
                  <h3>Bir avantajı kullanmak çok kolay.</h3>
                  <p>Keşfet. Koşulları incele. İşletmeye uygun yöntemi seç.</p>
                </div>
                <a className="text-link" href="#help">
                  Nasıl çalışır?
                </a>
              </aside>
            </>
          ) : route === 'discover' ? (
            <>
              <Heading title="Avantajları Keşfet">
                Yeni favorini bul. Sana uygun avantajı kolayca keşfet.
              </Heading>
              <section className="catalog-toolbar">
                <Search />
                <div
                  className="filter-row"
                  role="group"
                  aria-label="Kategori filtresi"
                >
                  {[['', 'Tümü'], ...categories].map(([key, label]) => (
                    <button
                      className={`filter-chip ${category === key ? 'active' : ''}`}
                      aria-pressed={category === key}
                      key={key}
                      onClick={() => discover(q, key)}
                    >
                      {label}
                    </button>
                  ))}
                </div>
              </section>
              <Cards />
            </>
          ) : route === 'favorites' ? (
            <>
              <Heading title="Favorilerim">
                Beğendiğin avantajlar, yeniden keşfetmen için burada.
                Favorilerin hesabına bağlı saklanır.
              </Heading>
              <Cards />
            </>
          ) : route === 'history' ? (
            <>
              <Heading title="Kullanım Geçmişi">
                Doğrulanmış avantaj kullanımlarının yer alacağı bölüm.
              </Heading>
              <Empty title="Henüz doğrulanmış bir kullanım yok.">
                Kod görüntülemek, bağlantı açmak veya QR göstermek kullanım
                onayı değildir. Bu işlemler kullanım kaydı oluşturmaz.
              </Empty>
            </>
          ) : route === 'profile' ? (
            <>
              <Heading title="Profil ve Hesap">
                Hesap bilgilerin ve oturumun.
              </Heading>
              <section className="surface student-profile">
                <div className="avatar">{account.name.slice(0, 1)}</div>
                <h2>{account.name}</h2>
                <dl>
                  <dt>E-posta</dt>
                  <dd>{account.email || 'E-postasız yönetici hesabı'}</dd>
                  <dt>Hesap durumu</dt>
                  <dd>
                    {account.status === 'active'
                      ? 'Aktif'
                      : 'İnceleme gerekiyor'}
                  </dd>
                  <dt>Rol</dt>
                  <dd>{account.role === 'admin' ? 'Yönetici' : 'Öğrenci'}</dd>
                </dl>
                {account.email && (
                  <a className="text-link" href="#forgot-password">
                    Parolamı yenile
                  </a>
                )}
              </section>
              <section className="surface logout-card">
                <div>
                  <h2>Oturum</h2>
                  <p>Bu cihazdaki oturumunu güvenle kapat.</p>
                </div>
                {logout}
              </section>
            </>
          ) : route === 'help' ? (
            <>
              <Heading title="Yardım ve Destek">
                Aradığın yanıtı bul, avantajları kolayca keşfet.
              </Heading>
              <div className="help-layout">
                <section className="faq-list">
                  {[
                    [
                      'Avantajlardan nasıl yararlanabilirim?',
                      'Keşfet bölümünden bir avantaj seç. Koşullarını, geçerlilik tarihlerini ve varsa öğrenci doğrulama gereksinimini incele. Aktif avantajın kullanım düğmesine basarak işletmenin tanımladığı yöntemi aç.',
                    ],
                    [
                      'Kod veya QR görüntülediğimde kullanılmış sayılır mı?',
                      'Hayır. Görüntüleme ve bağlantı açma kullanım onayı değildir; kullanım geçmişinde kayıt oluşturmaz.',
                    ],
                    [
                      'Favorilerim nerede tutuluyor?',
                      'Favorilerin hesabına bağlı olarak sunucuda saklanır. Bir avantaj kaldırıldığında kayıtlı favorini listeden çıkarabilirsin.',
                    ],
                    [
                      'Süresi dolan avantajlar kullanılabilir mi?',
                      'Hayır. Yakında başlayacak veya süresi dolmuş avantajların durumu gösterilir ve kullanım kapalıdır.',
                    ],
                  ].map(([title, text]) => (
                    <details key={title}>
                      <summary>{title}</summary>
                      <p>{text}</p>
                    </details>
                  ))}
                </section>
                <aside className="surface support-card">
                  <Icon name="help" />
                  <h2>Bir sorunun mu var?</h2>
                  {settings.isError ? (
                    <>
                      <p>Destek bilgileri yüklenemedi.</p>
                      <button
                        className="btn btn-secondary"
                        onClick={() => void settings.refetch()}
                      >
                        Yeniden dene
                      </button>
                    </>
                  ) : settings.isPending ? (
                    <p>Destek bilgileri yükleniyor…</p>
                  ) : (
                    <Contact value={settings.data?.support_contact} />
                  )}
                </aside>
              </div>
            </>
          ) : isDetail ? (
            <BenefitDetail
              key={`${accountId}/${id}`}
              id={id}
              accountId={accountId}
              favorite={(b) => <Favorite benefit={b} detail />}
            />
          ) : (
            <Empty title="Sayfa bulunamadı.">
              <a href="#overview" className="text-link">
                Genel bakışa dön
              </a>
            </Empty>
          )}
        </main>
        <footer className="app-footer">
          <span>
            {platform} <span className="footer-dot">•</span> Öğrencilerden,
            öğrencilere.
          </span>
        </footer>
      </div>
      <nav className="mobile-nav" aria-label="Mobil ana menü">
        {nav(true)}
      </nav>
    </div>
  );
}

function BenefitDetail({
  id,
  accountId,
  favorite,
}: {
  id: string;
  accountId: string;
  favorite: (b: Benefit) => ReactNode;
}) {
  const detail = useQuery({
    queryKey: ['student-benefit', accountId, id],
    queryFn: ({ signal }) =>
      studentRequest<{ benefit: Benefit }>(
        `benefits/${encodeURIComponent(id)}`,
        { signal },
      ),
    refetchInterval: 60000,
    refetchOnWindowFocus: true,
  });
  const [payload, setPayload] = useState<Redemption | null>(null);
  const [hint, setHint] = useState('');
  const [manualCopy, setManualCopy] = useState('');
  const revealGeneration = useRef(0);
  const redemption = useMutation({
    mutationKey: ['student-redemption', accountId, id],
    gcTime: 0,
    mutationFn: async () => {
      const generation = revealGeneration.current;
      const data = await studentRequest<Redemption>(
        `benefits/${encodeURIComponent(id)}/redemption`,
      );
      if (
        generation === revealGeneration.current &&
        document.visibilityState === 'visible' &&
        document.hasFocus()
      )
        setPayload(data);
    },
    onError: () => {
      setPayload(null);
      void detail.refetch();
    },
  });
  const reset = redemption.reset;
  useEffect(() => {
    const hide = () => {
      revealGeneration.current += 1;
      setPayload(null);
      setManualCopy('');
      setHint('');
      reset();
    };
    window.addEventListener('blur', hide);
    document.addEventListener('visibilitychange', hide);
    return () => {
      revealGeneration.current += 1;
      window.removeEventListener('blur', hide);
      document.removeEventListener('visibilitychange', hide);
    };
  }, [reset]);
  useEffect(() => {
    revealGeneration.current += 1;
    setPayload(null);
    setManualCopy('');
  }, [detail.dataUpdatedAt, detail.errorUpdatedAt]);
  if (detail.isPending) return <Empty title="Avantaj yükleniyor…" />;
  if (detail.isError)
    return (
      <Empty title="Avantaja ulaşılamadı">
        <span>{detail.error.message}</span>
        <button
          className="btn btn-secondary"
          onClick={() => void detail.refetch()}
        >
          Yeniden dene
        </button>
      </Empty>
    );
  const b = detail.data.benefit;
  const validPayload = b.status === 'active' ? payload : null;
  const copy = async (value: string) => {
    const copied = await copyText(value);
    setManualCopy(copied ? '' : value);
    setHint(
      copied
        ? 'Kopyalandı.'
        : 'Otomatik kopyalanamadı. Metni seçerek elle kopyalayabilirsin.',
    );
  };
  const fields = [
    ['Geçerli ürünler', b.products],
    ['Geçerli şubeler', b.branches],
    ['İstisnalar', b.exceptions],
    [
      'Minimum harcama',
      b.minimum_spend ? `${b.minimum_spend} ${b.currency}` : '',
    ],
    ['Öğrenci doğrulaması', b.verification_requirement],
    ['Başlangıç', b.valid_from],
    ['Son geçerlilik', b.valid_until],
  ];
  return (
    <>
      <a className="text-link student-back" href="#discover">
        ← Avantajlara dön
      </a>
      <div className="student-detail">
        <section className="student-detail-summary">
          <Heading title={b.title}>{b.business_name}</Heading>
        </section>
        <aside className="surface student-redemption">
          <span className={`student-status student-${b.status}`}>
            {statusLabels[b.status] || 'Kullanıma kapalı'}
          </span>
          <h2>{b.perk || b.title}</h2>
          <p>{methodLabels[b.method]}</p>
          {favorite(b)}
          {b.instructions?.length > 0 && (
            <section className="student-quick-instructions">
              <h3>Nasıl kullanılır?</h3>
              <ol>
                {b.instructions.map((s, i) => (
                  <li key={i}>{s}</li>
                ))}
              </ol>
            </section>
          )}
          {b.status !== 'active' ? (
            <p className="notice">Bu avantaj şu anda kullanıma açık değil.</p>
          ) : !validPayload ? (
            <button
              className="btn btn-primary btn-full"
              disabled={redemption.isPending}
              onClick={() => {
                setHint('');
                redemption.mutate();
              }}
            >
              {redemption.isPending
                ? 'Kontrol ediliyor…'
                : b.method === 'code'
                  ? 'Kodu görüntüle'
                  : b.method === 'qr'
                    ? 'QR kodunu göster'
                    : b.method === 'membership'
                      ? 'Üyelik bilgisini göster'
                      : 'Avantaj bağlantısını getir'}
            </button>
          ) : (
            <div className="student-payload">
              {validPayload.method === 'code' && validPayload.code && (
                <>
                  <label htmlFor="student-code">İndirim kodun</label>
                  <input
                    id="student-code"
                    value={validPayload.code}
                    readOnly
                    onFocus={(e) => e.target.select()}
                  />
                  <button
                    className="btn btn-primary btn-full"
                    onClick={() => void copy(validPayload.code!)}
                  >
                    Kodu kopyala
                  </button>
                </>
              )}
              {validPayload.method === 'link' &&
                (safeURL(validPayload.target_url) ? (
                  <a
                    className="btn btn-primary"
                    href={safeURL(validPayload.target_url)}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Avantaj bağlantısını aç
                  </a>
                ) : (
                  <p role="alert">Geçerli bir bağlantı tanımlanmadı.</p>
                ))}
              {validPayload.method === 'membership' && (
                <div className="student-membership">
                  {validPayload.membership_label ||
                    'Üyelik bilgisi tanımlanmadı.'}
                </div>
              )}
              {validPayload.method === 'qr' && (
                <QR content={validPayload.qr_content || ''} />
              )}
            </div>
          )}
          {redemption.isError && (
            <p role="alert" className="notice">
              {redemption.error.message}
            </p>
          )}
          <p className="redeem-note">
            Görüntülemek veya bağlantıyı açmak kullanım onayı değildir. Kullanım
            kaydı oluşturmaz.
          </p>
          <button
            className="btn btn-secondary btn-full"
            onClick={() =>
              void copy(
                `${location.origin}${location.pathname}#benefit/${encodeURIComponent(id)}`,
              )
            }
          >
            Avantajı paylaş · bağlantıyı kopyala
          </button>
          {hint && <p role="status">{hint}</p>}
          {manualCopy && (
            <input
              aria-label="Elle kopyalanacak metin"
              className="student-manual-copy"
              readOnly
              value={manualCopy}
              onFocus={(event) => event.target.select()}
            />
          )}
          {b.support_contact && <Contact value={b.support_contact} />}
        </aside>
        <section className="student-detail-content">
          <p className="student-description">{b.description}</p>
          <section className="surface student-conditions">
            <h2>Kullanım koşulları</h2>
            {b.conditions?.length ? (
              <ul>
                {b.conditions.map((s, i) => (
                  <li key={i}>{s}</li>
                ))}
              </ul>
            ) : (
              <p>Ek kullanım koşulu belirtilmedi.</p>
            )}
            <dl>
              {fields
                .filter(([, value]) => value)
                .map(([label, value]) => (
                  <div key={label}>
                    <dt>{label}</dt>
                    <dd>{value}</dd>
                  </div>
                ))}
            </dl>
          </section>
          <img
            className="student-detail-image"
            src={images[b.image_path] || images['assets/placeholder.svg']}
            alt=""
          />
          <small className="student-metadata">
            İşletme kimliği: {b.business_id}
          </small>
        </section>
      </div>
    </>
  );
}
function QR({ content }: { content: string }) {
  try {
    if (!content) throw new Error('empty');
    const qr = qrcode(0, 'M');
    qr.addData(content, 'Byte');
    qr.make();
    const count = qr.getModuleCount();
    let path = '';
    for (let row = 0; row < count; row++)
      for (let col = 0; col < count; col++)
        if (qr.isDark(row, col)) path += `M${col + 4},${row + 4}h1v1h-1z`;
    return (
      <div className="student-qr">
        <svg
          role="img"
          aria-label="İşletmenin tanımladığı kullanım QR kodu"
          viewBox={`0 0 ${count + 8} ${count + 8}`}
          shapeRendering="crispEdges"
        >
          <rect width={count + 8} height={count + 8} fill="white" />
          <path d={path} fill="black" />
        </svg>
      </div>
    );
  } catch {
    return (
      <p role="alert">
        Tanımlı içerik QR koduna dönüştürülemedi. İşletmenin destek bilgileri
        üzerinden yardım alabilirsin.
      </p>
    );
  }
}
