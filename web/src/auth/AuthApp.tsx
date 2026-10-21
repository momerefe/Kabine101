import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { Management } from './Management';
import type { components } from '../api/schema.gen';

type Session = components['schemas']['SessionResponse'];
type Account = components['schemas']['Account'];
type Mode =
  | 'login/student'
  | 'login/admin'
  | 'register'
  | 'forgot-password'
  | 'resend-verification'
  | 'verify-email'
  | 'reset-password'
  | 'account'
  | 'admin'
  | 'system';
type Route = { mode: Mode; token: string; next: string };
type AuthAction =
  | { kind: 'register'; name: string; email: string; password: string }
  | { kind: 'login'; email: string; password: string }
  | { kind: 'admin-login'; username: string; password: string }
  | { kind: 'logout' }
  | { kind: 'verify-email'; token: string }
  | { kind: 'forgot-password' | 'resend-verification'; email: string }
  | { kind: 'reset-password'; token: string; password: string };
type ReviewAction =
  | { account: Account; kind: 'review'; value: 'approved' | 'rejected' }
  | { account: Account; kind: 'role'; value: 'student' | 'admin' };

const titles: Record<Mode, string> = {
  'login/student': 'Yeniden merhaba.',
  'login/admin': 'Yönetim alanına hoş geldin.',
  register: 'Aramıza hoş geldin.',
  'forgot-password': 'Parolanı yenileyelim.',
  'resend-verification': 'E-postanı doğrula.',
  'verify-email': 'Son bir doğrulama.',
  'reset-password': 'Yeni parolanı belirle.',
  account: 'Hesabın',
  admin: 'Hesap yönetimi',
  system: 'Sistem durumu',
};

function safeNext(value: string | null): string {
  return value &&
    /^(\/account|\/admin(?:\/(overview|partners|businesses|benefits|students|published|archived|usage|analytics|users|audit|settings))?)$/.test(
      value,
    )
    ? value
    : '';
}

function readRoute(): Route {
  const [raw, query = ''] = location.hash.slice(1).split('?');
  const name = raw.replace(/^\//, '');
  const mode: Mode = name.startsWith('admin/')
    ? 'admin'
    : Object.hasOwn(titles, name)
      ? (name as Mode)
      : 'login/student';
  const params = new URLSearchParams(query);
  return {
    mode,
    token:
      mode === 'verify-email' || mode === 'reset-password'
        ? params.get('token') || ''
        : '',
    next: safeNext(params.get('next')),
  };
}

function navigate(mode: Mode | `admin/${string}`, next = '', replace = false) {
  const hash = `#${mode}${safeNext(next) ? `?next=${encodeURIComponent(next)}` : ''}`;
  if (replace || readRoute().token) {
    history.replaceState(
      null,
      '',
      `${location.pathname}${location.search}${hash}`,
    );
    window.dispatchEvent(new HashChangeEvent('hashchange'));
  } else location.hash = hash;
}

function fail(error: unknown): never {
  const message =
    typeof error === 'object' &&
    error !== null &&
    'message' in error &&
    typeof error.message === 'string'
      ? error.message
      : 'İşlem tamamlanamadı. Bağlantını kontrol edip yeniden dene.';
  throw new Error(message);
}

async function getSession(): Promise<Session> {
  const { data, error } = await api.GET('/api/auth/session', {
    credentials: 'same-origin',
  });
  if (error || !data) fail(error);
  return data;
}

async function authRequest(action: AuthAction, csrf: string) {
  const options = {
    credentials: 'same-origin' as const,
    params: { header: { 'X-CSRF-Token': csrf } },
  };
  let result;
  switch (action.kind) {
    case 'register':
      result = await api.POST('/api/auth/register', {
        ...options,
        body: {
          name: action.name,
          email: action.email,
          password: action.password,
        },
      });
      break;
    case 'login':
      result = await api.POST('/api/auth/login', {
        ...options,
        body: { email: action.email, password: action.password },
      });
      break;
    case 'admin-login':
      result = await api.POST('/api/auth/admin-login', {
        ...options,
        body: { username: action.username, password: action.password },
      });
      break;
    case 'logout':
      result = await api.POST('/api/auth/logout', { ...options, body: {} });
      break;
    case 'verify-email':
      result = await api.POST('/api/auth/verify-email', {
        ...options,
        body: { token: action.token },
      });
      break;
    case 'forgot-password':
      result = await api.POST('/api/auth/forgot-password', {
        ...options,
        body: { email: action.email },
      });
      break;
    case 'resend-verification':
      result = await api.POST('/api/auth/resend-verification', {
        ...options,
        body: { email: action.email },
      });
      break;
    case 'reset-password':
      result = await api.POST('/api/auth/reset-password', {
        ...options,
        body: { token: action.token, password: action.password },
      });
      break;
  }
  if (result.error || !result.data) fail(result.error);
  return result.data;
}

function Brand() {
  return (
    <a
      className="brand"
      href="#login/student"
      aria-label="KoçBenefits ana sayfa"
    >
      <span className="brand-mark">
        k<span>✦</span>
      </span>
      <span>
        Koç<span className="brand-light">Benefits</span>
        <small>ÖĞRENCİYE AYRICALIK</small>
      </span>
    </a>
  );
}

function Story() {
  return (
    <section className="auth-story" aria-label="KoçBenefits hakkında">
      <Brand />
      <div className="auth-story-copy">
        <div className="eyebrow">ÖĞRENCİLERDEN, ÖĞRENCİLERE.</div>
        <h1>
          Okullu olmanın <br />
          bir de <em>bu yanı var.</em>
        </h1>
        <p>
          Sevdiğin yerler, yeni keşifler ve sana özel avantajlar. Hepsi tek bir
          yerde.
        </p>
      </div>
      <div className="auth-visual" aria-hidden="true">
        <div className="card-back" />
        <div className="membership-card">
          <div className="card-brand">
            Koç<span>Benefits</span>
          </div>
          <small>ÖĞRENCİYE AYRICALIK</small>
          <div className="membership-card-bottom">
            <strong>
              Öğrencilerden, öğrencilere.
              <br />
              <small>TASARIM GÖRSELİ</small>
            </strong>
            <span>✦</span>
          </div>
        </div>
        <div className="auth-floating-note">
          ✦ Küçük ayrıcalıklar. Güzel bir okul hayatı.
        </div>
      </div>
      <p className="auth-story-footer">
        Koç Okulu 10. sınıf kabinesi tarafından tasarlanan okul projesi.
      </p>
    </section>
  );
}

function Notice({
  children,
  error = false,
}: {
  children: ReactNode;
  error?: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    ref.current?.focus();
  }, [children]);
  return (
    <div
      ref={ref}
      tabIndex={-1}
      className={error ? 'form-error' : 'auth-notice'}
      role={error ? 'alert' : 'status'}
    >
      {children}
    </div>
  );
}

function LocalEmail({ enabled }: { enabled: boolean }) {
  return enabled ? (
    <div className="local-email">
      <span className="demo-tag">YEREL E-POSTA TESTİ</span>
      <p>
        Bu ortamda e-posta gönderimi yerel posta kutusu üzerinden test edilir;
        gerçek adreslere gönderilmez.{' '}
        <a href="http://127.0.0.1:8025" target="_blank" rel="noreferrer">
          Yerel posta kutusunu aç ↗
        </a>
      </p>
    </div>
  ) : null;
}

function Field({
  name,
  label,
  type = 'text',
  hint,
  autoComplete,
  required = true,
  minLength,
  maxLength,
}: {
  name: string;
  label: string;
  type?: string;
  hint?: string;
  autoComplete?: string;
  required?: boolean;
  minLength?: number;
  maxLength?: number;
}) {
  return (
    <div className="field">
      <label htmlFor={name}>{label}</label>
      <input
        id={name}
        name={name}
        type={type}
        required={required}
        autoComplete={autoComplete}
        minLength={minLength}
        maxLength={maxLength}
        autoCapitalize={type === 'email' ? 'none' : undefined}
        spellCheck={type === 'email' ? false : undefined}
        aria-describedby={hint ? `${name}-hint` : undefined}
      />
      {hint && <small id={`${name}-hint`}>{hint}</small>}
    </div>
  );
}

function AuthForm({
  route,
  session,
  refresh,
  flash,
  onFlash,
}: {
  route: Route;
  session: Session;
  refresh: () => Promise<Session>;
  flash: string;
  onFlash: (message: string) => void;
}) {
  const [validation, setValidation] = useState('');
  const [notice, setNotice] = useState('');
  const mutation = useMutation({
    mutationFn: async (action: AuthAction) => {
      try {
        return await authRequest(action, session.csrf_token);
      } finally {
        // Mutation metadata must not retain credentials after the request.
        if ('password' in action) action.password = '';
        if ('token' in action) action.token = '';
      }
    },
    onSuccess: async (data, action) => {
      setValidation('');
      setNotice('message' in data ? data.message : 'Giriş yapıldı.');
      if (action.kind === 'verify-email' || action.kind === 'reset-password') {
        onFlash(
          'message' in data
            ? data.message
            : 'İşlem tamamlandı. Giriş yapabilirsin.',
        );
        // Remove used tokens even if the following session refresh fails.
        navigate('login/student', '', true);
      }
      const updated = await refresh();
      if (
        (action.kind === 'login' || action.kind === 'admin-login') &&
        updated.user?.role === 'admin' &&
        safeNext(route.next).startsWith('/admin/')
      ) {
        navigate(route.next.slice(1) as `admin/${string}`, '', true);
        return;
      }
      if (action.kind === 'login' || action.kind === 'admin-login')
        navigate(
          route.next === '/admin' || route.mode === 'login/admin'
            ? 'admin'
            : route.next === '/account'
              ? 'account'
              : updated.user?.role === 'admin'
                ? 'admin'
                : 'account',
          '',
          true,
        );
    },
  });
  const mode = route.mode;
  const admin = mode === 'login/admin';
  const isLogin = mode === 'login/student' || admin;
  const register = mode === 'register';
  const reset = mode === 'reset-password';
  const verify = mode === 'verify-email';
  const password = isLogin || register || reset;
  const needsEmail = !reset && !verify;
  const label = register
    ? 'Hesap oluştur'
    : isLogin
      ? 'Giriş yap'
      : reset
        ? 'Parolayı yenile'
        : verify
          ? 'E-postamı doğrula'
          : mode === 'forgot-password'
            ? 'Yenileme bağlantısı gönder'
            : 'Doğrulama bağlantısı gönder';

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setValidation('');
    setNotice('');
    onFlash('');
    mutation.reset();
    const form = event.currentTarget;
    const values = new FormData(form);
    const email = String(values.get('email') || '').trim();
    const username = String(values.get('username') || '').trim();
    const passwordValue = String(values.get('password') || '');
    if (
      register &&
      email.split('@')[1]?.toLowerCase() !== session.school_domain.toLowerCase()
    ) {
      setValidation(
        `Kayıt için @${session.school_domain} okul e-posta adresini kullan.`,
      );
      return;
    }
    if (
      (register || reset) &&
      ([...passwordValue].length < 15 || [...passwordValue].length > 128)
    ) {
      setValidation('Parolan 15–128 karakter uzunluğunda olmalı.');
      return;
    }
    let action: AuthAction;
    if (register)
      action = {
        kind: 'register',
        name: String(values.get('name') || '').trim(),
        email,
        password: passwordValue,
      };
    else if (isLogin)
      action =
        admin && !username.includes('@')
          ? { kind: 'admin-login', username, password: passwordValue }
          : {
              kind: 'login',
              email: admin ? username : email,
              password: passwordValue,
            };
    else if (reset)
      action = {
        kind: 'reset-password',
        token: route.token,
        password: passwordValue,
      };
    else if (verify) action = { kind: 'verify-email', token: route.token };
    else
      action = {
        kind:
          mode === 'forgot-password'
            ? 'forgot-password'
            : 'resend-verification',
        email,
      };
    mutation.mutate(action);
    const input = form.elements.namedItem('password');
    if (input instanceof HTMLInputElement) input.value = '';
  }

  return (
    <>
      <div className="audience-switch" aria-label="Giriş alanı">
        <button
          onClick={() => navigate('login/student', route.next)}
          aria-pressed={!admin}
          className={!admin ? 'active' : ''}
        >
          Öğrenci
        </button>
        <button
          onClick={() => navigate('login/admin', route.next)}
          aria-pressed={admin}
          className={admin ? 'active' : ''}
        >
          Yönetici
        </button>
      </div>
      <h2>{titles[mode]}</h2>
      <p className="auth-description">
        {register
          ? `@${session.school_domain} adresinle kayıt ol. E-posta doğrulaması ve yönetici onayından sonra giriş yapabilirsin.`
          : admin
            ? 'Yetkili yönetici hesabın ve parolanla giriş yap.'
            : isLogin
              ? 'Okul e-postan ve parolanla hesabına giriş yap.'
              : reset
                ? 'Hesabın için uzun ve benzersiz bir parola seç.'
                : verify
                  ? 'E-posta adresini doğrulamak için aşağıdaki düğmeye bas. Ardından yönetici onayı gerekir.'
                  : 'Okul e-posta adresini yaz. Uygun bir hesap varsa bağlantı göndereceğiz.'}
        {route.next && ' Girişten sonra istediğin hesap sayfasına döneceksin.'}
      </p>
      {!admin && (isLogin || register) && (
        <div className="auth-tabs" aria-label="Giriş veya kayıt seçimi">
          <button
            className={`auth-tab ${isLogin ? 'active' : ''}`}
            aria-pressed={isLogin}
            onClick={() => navigate('login/student', route.next)}
          >
            Giriş yap
          </button>
          <button
            className={`auth-tab ${register ? 'active' : ''}`}
            aria-pressed={register}
            onClick={() => navigate('register', route.next)}
          >
            Kayıt ol
          </button>
        </div>
      )}
      {flash && <Notice>{flash}</Notice>}
      {validation && <Notice error>{validation}</Notice>}
      {mutation.isError && <Notice error>{mutation.error.message}</Notice>}
      {notice && <Notice>{notice}</Notice>}
      {(verify || reset) && !route.token ? (
        <Notice error>
          Bağlantıda doğrulama anahtarı bulunamadı. Yeni bir bağlantı iste.
        </Notice>
      ) : (
        <form className="auth-form" onSubmit={submit}>
          <fieldset
            disabled={
              mutation.isPending || (register && !session.registration_open)
            }
          >
            {register && (
              <Field
                name="name"
                label="Ad soyad"
                autoComplete="name"
                minLength={1}
                maxLength={120}
              />
            )}
            {needsEmail && (
              <Field
                name={admin ? 'username' : 'email'}
                label={
                  admin
                    ? 'Kullanıcı adı veya okul e-postası'
                    : 'Okul e-posta adresi'
                }
                type={admin ? 'text' : 'email'}
                autoComplete="username"
                maxLength={254}
                hint={
                  admin ? undefined : `Okul alan adı: @${session.school_domain}`
                }
              />
            )}
            {password && (
              <Field
                name="password"
                label={reset ? 'Yeni parola' : 'Parola'}
                type="password"
                autoComplete={isLogin ? 'current-password' : 'new-password'}
                hint={
                  isLogin
                    ? undefined
                    : '15–128 karakter. Boşluk ve Türkçe karakter kullanabilirsin.'
                }
              />
            )}
            {register && !session.registration_open && (
              <Notice>Kayıtlar şu anda kapalı.</Notice>
            )}
            <button className="btn btn-primary btn-full" type="submit">
              {mutation.isPending ? 'İşlem sürüyor…' : label}
              <span aria-hidden="true">→</span>
            </button>
          </fieldset>
        </form>
      )}
      <div className="auth-help">
        {isLogin && (
          <a className="text-link" href="#forgot-password">
            {admin ? 'Okul hesabımın parolasını unuttum' : 'Parolamı unuttum'}
          </a>
        )}
        <a
          className="text-link"
          href={reset ? '#forgot-password' : '#resend-verification'}
        >
          {reset
            ? 'Yeni parola bağlantısı iste'
            : admin
              ? 'Okul hesabı için doğrulama e-postası gönder'
              : 'Doğrulama e-postasını yeniden gönder'}
        </a>
        {!isLogin && (
          <a className="text-link" href="#login/student">
            Girişe dön
          </a>
        )}
      </div>
      <p className="auth-security">
        {admin
          ? 'Parola yenileme ve e-posta doğrulaması yalnızca okul e-postalı hesaplar içindir. Yönetici hesabı bu ekrandan oluşturulamaz.'
          : 'Hesabın e-posta doğrulaması ve yönetici onayı tamamlandığında etkinleşir. Yönetici hesabı bu ekrandan oluşturulamaz.'}
      </p>
      <LocalEmail enabled={session.local_email} />
    </>
  );
}

const statusLabels: Record<Account['status'], string> = {
  pending: 'Onay bekliyor',
  active: 'Etkin',
  rejected: 'Reddedildi',
  disabled: 'Devre dışı',
};
const eligibilityLabels: Record<Account['eligibility_status'], string> = {
  pending: 'Bekliyor',
  approved: 'Onaylandı',
  rejected: 'Reddedildi',
};

function AccountPage({
  session,
  refresh,
  adminPage,
  flash,
}: {
  session: Session;
  refresh: () => Promise<Session>;
  adminPage: boolean;
  flash: string;
}) {
  const user = session.user!;
  const client = useQueryClient();
  const logout = useMutation({
    mutationFn: () => authRequest({ kind: 'logout' }, session.csrf_token),
    onSuccess: async () => {
      client.removeQueries({ queryKey: ['accounts'] });
      client.removeQueries({ queryKey: ['management-businesses'] });
      client.removeQueries({ queryKey: ['management-benefits'] });
      await refresh();
      navigate('login/student', '', true);
    },
  });
  if (adminPage && user.role === 'admin')
    return (
      <Management
        session={session}
        accounts={<AdminAccounts session={session} refresh={refresh} />}
        logout={
          <>
            <button
              className="btn btn-secondary"
              disabled={logout.isPending}
              onClick={() => logout.mutate()}
            >
              {logout.isPending ? 'Çıkılıyor…' : 'Çıkış yap'}
            </button>
            {logout.isError && (
              <Notice error>
                Çıkış yapılamadı. Bağlantını kontrol edip yeniden dene.
              </Notice>
            )}
          </>
        }
      />
    );
  return (
    <main className="foundation account-shell" id="account-content">
      <header className="foundation-header">
        <Brand />
        <div className="account-header-actions">
          {user.role === 'admin' && (
            <a className="text-link" href={adminPage ? '#account' : '#admin'}>
              {adminPage ? 'Hesabım' : 'Hesap yönetimi'}
            </a>
          )}
          <button
            className="btn btn-secondary"
            disabled={logout.isPending}
            onClick={() => logout.mutate()}
          >
            {logout.isPending ? 'Çıkılıyor…' : 'Çıkış yap'}
          </button>
        </div>
      </header>
      {logout.isError && <Notice error>{logout.error.message}</Notice>}
      {flash && <Notice>{flash}</Notice>}
      {adminPage && user.role === 'admin' ? (
        <AdminAccounts session={session} refresh={refresh} />
      ) : (
        <section className="foundation-card">
          <p className="eyebrow">HESABIN</p>
          <h1>Merhaba, {user.name}.</h1>
          <p>KoçBenefits hesabına giriş yaptın.</p>
          <dl className="account-details">
            <div>
              <dt>{user.username ? 'Kullanıcı adı' : 'Okul e-postası'}</dt>
              <dd>{user.username || user.email}</dd>
            </div>
            <div>
              <dt>Hesap durumu</dt>
              <dd>
                {user.username ? 'Yönetici hesabı' : statusLabels[user.status]}
              </dd>
            </div>
            {!user.username && (
              <>
                <div>
                  <dt>E-posta doğrulaması</dt>
                  <dd>{user.email_verified ? 'Doğrulandı' : 'Bekliyor'}</dd>
                </div>
                <div>
                  <dt>Yönetici onayı</dt>
                  <dd>{eligibilityLabels[user.eligibility_status]}</dd>
                </div>
              </>
            )}
            <div>
              <dt>Rol</dt>
              <dd>{user.role === 'admin' ? 'Yönetici' : 'Öğrenci'}</dd>
            </div>
          </dl>
          <div className="auth-notice">
            <strong>3B · Hesap ve güvenli giriş hazır.</strong>
            <p>
              Avantaj kataloğu ve diğer öğrenci hizmetleri sonraki teslimatta bu
              hesaba bağlanacak. Bu ekranda henüz kullanılabilir bir avantaj
              bulunmuyor.
            </p>
          </div>
          {adminPage && user.role !== 'admin' && (
            <Notice error>Hesap yönetimi için yönetici yetkisi gerekir.</Notice>
          )}
        </section>
      )}
      <LocalEmail enabled={session.local_email} />
      <footer>Öğrencilerden, öğrencilere.</footer>
    </main>
  );
}

function AdminAccounts({
  session,
  refresh,
}: {
  session: Session;
  refresh: () => Promise<Session>;
}) {
  const client = useQueryClient();
  const [action, setAction] = useState<ReviewAction | null>(null);
  const [notice, setNotice] = useState('');
  const accounts = useQuery({
    queryKey: ['accounts', session.user?.id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/admin/accounts', {
        credentials: 'same-origin',
      });
      if (error || !data) fail(error);
      return data.accounts;
    },
  });
  return (
    <section className="foundation-card">
      <p className="eyebrow">YÖNETİM · 3B</p>
      <h1>Hesap yönetimi</h1>
      <p>
        Öğrenci başvurularını incele ve hesap rollerini yönet. E-posta
        doğrulanmadan verilen onay hesabı etkinleştirmez.
      </p>
      <div className="auth-notice">
        İşletme ve avantaj yönetimine soldaki menüden ulaşabilirsin.
      </div>
      {notice && <Notice>{notice}</Notice>}
      {accounts.isPending ? (
        <p role="status">Hesaplar yükleniyor…</p>
      ) : accounts.isError ? (
        <>
          <Notice error>
            Hesaplar alınamadı. Oturumun veya bağlantın sona ermiş olabilir.
          </Notice>
          <button
            className="btn btn-secondary"
            onClick={() => void accounts.refetch()}
          >
            Yeniden dene
          </button>
        </>
      ) : (
        <>
          <p className="account-count">
            {accounts.data.length} hesap ·{' '}
            {
              accounts.data.filter(
                (account) =>
                  !account.username && account.eligibility_status === 'pending',
              ).length
            }{' '}
            başvuru bekliyor
          </p>
          <div className="admin-account-list">
            {accounts.data.map((account) => (
              <article className="admin-account" key={account.id}>
                <div className="admin-account-heading">
                  <h2>{account.name}</h2>
                  <span className={`account-status status-${account.status}`}>
                    {statusLabels[account.status]}
                  </span>
                </div>
                <p className="account-email">
                  {account.username || account.email}
                </p>
                <dl className="account-meta">
                  {account.username ? (
                    <div>
                      <dt>Hesap türü</dt>
                      <dd>Özel yönetici</dd>
                    </div>
                  ) : (
                    <>
                      <div>
                        <dt>E-posta</dt>
                        <dd>
                          {account.email_verified
                            ? 'Doğrulandı'
                            : 'Doğrulanmadı'}
                        </dd>
                      </div>
                      <div>
                        <dt>Onay</dt>
                        <dd>{eligibilityLabels[account.eligibility_status]}</dd>
                      </div>
                      <div>
                        <dt>Rol</dt>
                        <dd>
                          {account.role === 'admin' ? 'Yönetici' : 'Öğrenci'}
                        </dd>
                      </div>
                    </>
                  )}
                </dl>
                <div className="account-actions">
                  {!account.username &&
                    account.eligibility_status === 'pending' && (
                      <>
                        <button
                          className="btn btn-primary"
                          onClick={() =>
                            setAction({
                              account,
                              kind: 'review',
                              value: 'approved',
                            })
                          }
                        >
                          Onayla
                        </button>
                        <button
                          className="btn btn-secondary"
                          onClick={() =>
                            setAction({
                              account,
                              kind: 'review',
                              value: 'rejected',
                            })
                          }
                        >
                          Reddet
                        </button>
                      </>
                    )}
                  {!account.username &&
                    account.status === 'active' &&
                    account.email_verified &&
                    account.eligibility_status === 'approved' && (
                      <button
                        className="btn btn-secondary"
                        onClick={() =>
                          setAction({
                            account,
                            kind: 'role',
                            value:
                              account.role === 'admin' ? 'student' : 'admin',
                          })
                        }
                      >
                        {account.role === 'admin'
                          ? 'Öğrenci rolü ver'
                          : 'Yönetici rolü ver'}
                      </button>
                    )}
                </div>
              </article>
            ))}
          </div>
          {accounts.data.length === 0 && <p>Henüz başvuru bulunmuyor.</p>}
        </>
      )}
      {action && (
        <AdminConfirmation
          key={`${action.account.id}-${action.kind}-${action.value}`}
          action={action}
          session={session}
          onClose={() => setAction(null)}
          onComplete={async (message) => {
            setNotice(message);
            setAction(null);
            await client.invalidateQueries({ queryKey: ['accounts'] });
            await refresh();
          }}
        />
      )}
    </section>
  );
}

function AdminConfirmation({
  action,
  session,
  onClose,
  onComplete,
}: {
  action: ReviewAction;
  session: Session;
  onClose: () => void;
  onComplete: (message: string) => Promise<void>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const previousFocus = useRef<HTMLElement | null>(null);
  useEffect(() => {
    previousFocus.current = document.activeElement as HTMLElement;
    const element = dialog.current;
    element?.showModal();
    return () => {
      element?.close();
      previousFocus.current?.focus();
    };
  }, []);
  const description =
    action.kind === 'review'
      ? action.value === 'approved'
        ? 'Başvuruyu onayla'
        : 'Başvuruyu reddet'
      : action.value === 'admin'
        ? 'Yönetici rolü ver'
        : 'Öğrenci rolü ver';
  const mutation = useMutation({
    mutationFn: async (current_password: string) => {
      const options = {
        credentials: 'same-origin' as const,
        params: {
          header: { 'X-CSRF-Token': session.csrf_token },
          path: { id: action.account.id },
        },
      };
      const result =
        action.kind === 'review'
          ? await api.POST('/api/admin/accounts/{id}/review', {
              ...options,
              body: { decision: action.value, current_password },
            })
          : await api.POST('/api/admin/accounts/{id}/role', {
              ...options,
              body: { role: action.value, current_password },
            });
      if (result.error || !result.data) fail(result.error);
      return result.data.message;
    },
    onSuccess: onComplete,
  });
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const values = new FormData(form);
    mutation.mutate(String(values.get('current_password') || ''));
    const input = form.elements.namedItem('current_password');
    if (input instanceof HTMLInputElement) input.value = '';
  }
  return (
    <dialog
      ref={dialog}
      className="account-dialog"
      aria-labelledby="confirm-title"
      aria-describedby="confirm-description"
      onCancel={(event) => {
        if (mutation.isPending) event.preventDefault();
        else onClose();
      }}
    >
      <h2 id="confirm-title">{description}</h2>
      <p id="confirm-description">
        <strong>{action.account.name}</strong>
        <br />
        {action.account.username || action.account.email}
      </p>
      <p>
        {action.kind === 'role'
          ? action.value === 'admin'
            ? 'Bu hesap diğer kullanıcıların başvurularını ve rollerini yönetebilecek.'
            : 'Bu hesabın yönetim alanına erişimi kaldırılacak.'
          : action.value === 'approved'
            ? 'E-posta doğrulaması da tamamlandığında bu hesap giriş yapabilecek.'
            : 'Bu başvuru reddedilecek ve öğrenci giriş yapamayacak.'}
      </p>
      {mutation.isError && <Notice error>{mutation.error.message}</Notice>}
      <form onSubmit={submit}>
        <fieldset disabled={mutation.isPending}>
          <Field
            name="current_password"
            label="Mevcut yönetici parolan"
            type="password"
            autoComplete="current-password"
          />
          <label className="confirmation-check">
            <input type="checkbox" required />
            Bu hesap için belirtilen işlemi onaylıyorum.
          </label>
          <div className="account-actions">
            <button
              className="btn btn-secondary"
              type="button"
              onClick={onClose}
            >
              Vazgeç
            </button>
            <button className="btn btn-primary" type="submit">
              {mutation.isPending ? 'İşleniyor…' : description}
            </button>
          </div>
        </fieldset>
      </form>
    </dialog>
  );
}

function SystemStatus() {
  const readiness = useQuery({
    queryKey: ['readiness'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/ready');
      if (error || data?.status !== 'ok') fail(error);
      return data;
    },
  });
  return (
    <main className="foundation">
      <header className="foundation-header">
        <Brand />
      </header>
      <section className="foundation-card">
        <h1>Sistem durumu</h1>
        <p role="status">
          {readiness.isPending
            ? 'Bağlantı kontrol ediliyor…'
            : readiness.isSuccess
              ? 'API ve PostgreSQL şeması hazır'
              : 'Bağlantı kurulamadı.'}
        </p>
        <div className="account-actions">
          <button
            className="btn btn-secondary"
            disabled={readiness.isFetching}
            onClick={() => void readiness.refetch()}
          >
            Bağlantıyı yeniden kontrol et
          </button>
          <a className="text-link" href="#login/student">
            Girişe dön
          </a>
        </div>
      </section>
    </main>
  );
}

export function AuthApp() {
  const [route, setRoute] = useState(readRoute);
  const [flash, setFlash] = useState('');
  const client = useQueryClient();
  const session = useQuery({
    queryKey: ['session'],
    queryFn: getSession,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
  });
  useEffect(() => {
    client.removeQueries({
      predicate: (query) =>
        ['accounts', 'management-businesses', 'management-benefits'].includes(
          String(query.queryKey[0]),
        ) &&
        (session.data?.user?.role !== 'admin' ||
          query.queryKey[1] !== session.data?.user?.id),
    });
  }, [client, session.data?.user?.id, session.data?.user?.role]);
  useEffect(() => {
    const update = () => setRoute(readRoute());
    window.addEventListener('hashchange', update);
    return () => window.removeEventListener('hashchange', update);
  }, []);
  useEffect(() => {
    document.title = `${titles[route.mode]} · KoçBenefits`;
    document.documentElement.lang = 'tr';
  }, [route.mode]);
  async function refresh() {
    const result = await session.refetch();
    if (!result.data || result.error)
      throw new Error(
        'Oturum yenilenemedi. Sayfayı yeniden yükleyip giriş yap.',
      );
    return result.data;
  }
  if (route.mode === 'system') return <SystemStatus />;
  if (session.isPending || session.isError)
    return (
      <main className="foundation">
        <header className="foundation-header">
          <Brand />
        </header>
        <section className="foundation-card">
          <h1>
            {session.isPending
              ? 'Merhaba, bağlantı kuruluyor…'
              : 'Şu anda bağlantı kurulamıyor.'}
          </h1>
          <p role="status">
            {session.isPending
              ? 'Güvenli oturumun kontrol ediliyor.'
              : 'Hesap hizmetine ulaşılamadı. Biraz sonra yeniden dene.'}
          </p>
          {session.isError && (
            <button
              className="btn btn-primary"
              disabled={session.isFetching}
              onClick={() => void session.refetch()}
            >
              {session.isFetching ? 'Bağlanıyor…' : 'Yeniden dene'}
            </button>
          )}
        </section>
      </main>
    );
  const tokenPage =
    route.mode === 'verify-email' || route.mode === 'reset-password';
  if (session.data.user && !tokenPage)
    return (
      <AccountPage
        session={session.data}
        refresh={refresh}
        flash={flash}
        adminPage={
          route.mode === 'admin' ||
          (route.mode === 'login/admin' && session.data.user.role === 'admin')
        }
      />
    );
  const protectedTarget =
    route.mode === 'admin'
      ? safeNext(
          '/' + location.hash.slice(1).split('?')[0].replace(/^\//, ''),
        ) || '/admin'
      : route.mode === 'account'
        ? '/account'
        : route.next;
  const authRoute: Route = {
    ...route,
    mode:
      route.mode === 'admin'
        ? 'login/admin'
        : route.mode === 'account'
          ? 'login/student'
          : route.mode,
    next: protectedTarget,
  };
  return (
    <main className="auth-shell">
      <button
        className="skip-link"
        onClick={() => document.getElementById('auth-content')?.focus()}
      >
        Giriş formuna geç
      </button>
      <Story />
      <section className="auth-form-area" aria-label="Giriş ve kayıt">
        <div className="auth-content" id="auth-content" tabIndex={-1}>
          <AuthForm
            key={`${authRoute.mode}:${authRoute.token}`}
            route={authRoute}
            session={session.data}
            refresh={refresh}
            flash={flash}
            onFlash={setFlash}
          />
        </div>
        <footer className="auth-bottom">
          KoçBenefits · 3B hesap ve güvenli giriş
        </footer>
      </section>
    </main>
  );
}
