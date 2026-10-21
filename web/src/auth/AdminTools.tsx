import { useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { components } from '../api/schema.gen';

type Settings = components['schemas']['PlatformSettings'];
type SettingsResponse = components['schemas']['PlatformSettingsResponse'];

export const auditActionLabels: Record<string, string> = {
  'account.registered': 'Hesap başvurusu oluşturuldu',
  'account.admin_bootstrapped': 'Yönetici hesabı oluşturuldu',
  'account.operator_bootstrapped': 'Özel yönetici hesabı oluşturuldu',
  'account.login': 'Hesaba giriş yapıldı',
  'account.logout': 'Hesaptan çıkış yapıldı',
  'account.verify_email_requested': 'E-posta doğrulaması istendi',
  'account.verify_email_completed': 'E-posta doğrulandı',
  'account.reset_password_requested': 'Parola yenileme istendi',
  'account.reset_password_completed': 'Parola yenilendi',
  'business.save': 'İşletme kaydedildi',
  'benefit.save': 'Avantaj kaydedildi',
  'benefit.publish': 'Avantaj yayımlandı',
  'benefit.suspend': 'Avantaj askıya alındı',
  'account.review.approved': 'Öğrenci başvurusu onaylandı',
  'account.review.rejected': 'Öğrenci başvurusu reddedildi',
  'account.role.admin': 'Yönetici rolü verildi',
  'account.role.student': 'Öğrenci rolü verildi',
  'account.status.active': 'Hesap etkinleştirildi',
  'account.status.disabled': 'Hesap devre dışı bırakıldı',
  'platform.settings': 'Platform ayarları kaydedildi',
};
const entityLabels: Record<string, string> = {
  business: 'İşletme',
  benefit: 'Avantaj',
  account: 'Hesap',
  platform: 'Platform',
};
export function adminData<T>(result: {
  data?: T;
  error?: unknown;
  response: Response;
}): T {
  if (result.error || !result.data) {
    const message =
      result.response.status === 409
        ? 'Ayarlar başka bir yönetici tarafından değiştirildi. Girdiğin bilgiler korundu. Güncel ayarları yeniden yükleyerek karşılaştır.'
        : result.error &&
            typeof result.error === 'object' &&
            'message' in result.error
          ? String(result.error.message)
          : 'Veriler alınamadı. Oturumunu ve bağlantını kontrol edip yeniden dene.';
    throw new Error(message);
  }
  return result.data;
}
function time(value: string) {
  return new Intl.DateTimeFormat('tr-TR', {
    dateStyle: 'medium',
    ...(value.includes('T') ? { timeStyle: 'short' as const } : {}),
    timeZone: 'Europe/Istanbul',
  }).format(new Date(value.includes('T') ? value : `${value}T12:00:00+03:00`));
}
function LoadError({ error, retry }: { error: Error; retry: () => void }) {
  return (
    <div className="form-error" role="alert">
      <p>{error.message}</p>
      <button className="btn btn-secondary" onClick={retry}>
        Yeniden dene
      </button>
    </div>
  );
}
export function useAdminSettings(accountId: string) {
  return useQuery({
    queryKey: ['management-settings', accountId],
    queryFn: async ({ signal }) =>
      adminData(await api.GET('/api/admin/settings', { signal })),
  });
}
export function Audit({ accountId }: { accountId: string }) {
  const [q, setQ] = useState('');
  const [action, setAction] = useState('');
  const [from, setFrom] = useState('');
  const [until, setUntil] = useState('');
  const [page, setPage] = useState(1);
  const invalidRange = !!from && !!until && from > until;
  const events = useQuery({
    queryKey: ['management-audit', accountId, q, action, from, until, page],
    enabled: !invalidRange,
    queryFn: async ({ signal }) =>
      adminData(
        await api.GET('/api/admin/audit', {
          signal,
          params: {
            query: {
              q,
              action,
              from: from || undefined,
              until: until || undefined,
              page,
            },
          },
        }),
      ),
  });
  function change(update: () => void) {
    update();
    setPage(1);
  }
  return (
    <div className="management-audit">
      <p>
        Hesap ve yönetim işlemleri, en yeniden eskiye. Tarih filtreleri İstanbul
        saatine göre iki günü de kapsar.
      </p>
      <div className="admin-toolbar">
        <label className="field">
          Günlükte ara
          <input
            type="search"
            maxLength={100}
            value={q}
            placeholder="İşlemi yapan veya kayıt adı"
            onChange={(e) => change(() => setQ(e.target.value))}
          />
        </label>
        <label className="field">
          İşlem türü
          <select
            value={action}
            onChange={(e) => change(() => setAction(e.target.value))}
          >
            <option value="">Tüm işlemler</option>
            {Object.entries(auditActionLabels).map(([id, label]) => (
              <option key={id} value={id}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          Başlangıç tarihi
          <input
            type="date"
            value={from}
            max={until || undefined}
            onChange={(e) => change(() => setFrom(e.target.value))}
          />
        </label>
        <label className="field">
          Bitiş tarihi
          <input
            type="date"
            value={until}
            min={from || undefined}
            onChange={(e) => change(() => setUntil(e.target.value))}
          />
        </label>
      </div>
      {invalidRange ? (
        <p className="form-error" role="alert">
          Bitiş tarihi başlangıçtan önce olamaz.
        </p>
      ) : events.isPending ? (
        <p role="status">İşlem günlüğü yükleniyor…</p>
      ) : events.isError ? (
        <LoadError error={events.error} retry={() => void events.refetch()} />
      ) : (
        <>
          <p role="status">
            {events.data.total.toLocaleString('tr-TR')} işlem · Sayfa{' '}
            {events.data.page} /{' '}
            {Math.max(1, Math.ceil(events.data.total / events.data.page_size))}
          </p>
          {events.data.events.length ? (
            <div
              className="table-scroll"
              tabIndex={0}
              role="region"
              aria-label="İşlem günlüğü"
            >
              <table className="admin-table">
                <thead>
                  <tr>
                    <th scope="col">Tarih · İstanbul</th>
                    <th scope="col">İşlemi yapan</th>
                    <th scope="col">İşlem</th>
                    <th scope="col">Kayıt</th>
                  </tr>
                </thead>
                <tbody>
                  {events.data.events.map((event) => (
                    <tr key={event.id}>
                      <td>
                        <time dateTime={event.created_at}>
                          {time(event.created_at)}
                        </time>
                      </td>
                      <td>{event.actor_name}</td>
                      <td>{auditActionLabels[event.action] || event.action}</td>
                      <td>
                        {event.entity_label ||
                          (event.entity_type === 'platform'
                            ? 'Platform'
                            : 'Kayıt artık mevcut değil')}
                        <small>
                          {entityLabels[event.entity_type] || 'Diğer'}
                        </small>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <section className="admin-empty">
              <h2>Eşleşen işlem bulunamadı.</h2>
              <p>Arama veya tarih filtrelerini değiştirebilirsin.</p>
            </section>
          )}
          <div
            className="account-actions audit-pagination"
            aria-label="Günlük sayfaları"
          >
            <button
              className="btn btn-secondary"
              disabled={page <= 1 || events.isFetching}
              onClick={() => setPage((value) => value - 1)}
            >
              Önceki sayfa
            </button>
            <button
              className="btn btn-secondary"
              disabled={
                page * events.data.page_size >= events.data.total ||
                events.isFetching
              }
              onClick={() => setPage((value) => value + 1)}
            >
              Sonraki sayfa
            </button>
          </div>
        </>
      )}
    </div>
  );
}
export function PlatformSettings({
  accountId,
  csrf,
}: {
  accountId: string;
  csrf: string;
}) {
  const settings = useAdminSettings(accountId);
  if (settings.isPending)
    return <p role="status">Platform ayarları yükleniyor…</p>;
  if (settings.isError && !settings.data)
    return (
      <LoadError error={settings.error} retry={() => void settings.refetch()} />
    );
  return (
    <SettingsForm
      accountId={accountId}
      csrf={csrf}
      initial={settings.data!}
      reload={async () => {
        const result = await settings.refetch();
        if (result.error || !result.data)
          throw result.error || new Error('Ayarlar alınamadı.');
        return result.data;
      }}
    />
  );
}
function SettingsForm({
  accountId,
  csrf,
  initial,
  reload,
}: {
  accountId: string;
  csrf: string;
  initial: SettingsResponse;
  reload: () => Promise<SettingsResponse>;
}) {
  const client = useQueryClient();
  const [form, setForm] = useState<Settings>(() => ({ ...initial.settings }));
  const [notice, setNotice] = useState('');
  const [reloadError, setReloadError] = useState('');
  const [reloading, setReloading] = useState(false);
  const [reloaded, setReloaded] = useState<SettingsResponse | null>(null);
  const save = useMutation({
    mutationFn: async () =>
      adminData(
        await api.PUT('/api/admin/settings', {
          params: { header: { 'X-CSRF-Token': csrf } },
          body: {
            ...form,
            platform_name: form.platform_name.trim(),
            support_contact: form.support_contact.trim(),
          },
        }),
      ),
    onSuccess: async (data) => {
      setForm({ ...data.settings });
      setNotice('Platform ayarları kaydedildi.');
      setReloaded(null);
      client.setQueryData(['management-settings', accountId], data);
      await Promise.all([
        client.invalidateQueries({
          queryKey: ['management-settings', accountId],
        }),
        client.invalidateQueries({ queryKey: ['student-settings'] }),
        client.invalidateQueries({ queryKey: ['management-audit', accountId] }),
        client.invalidateQueries({
          queryKey: ['management-overview', accountId],
        }),
      ]);
    },
  });
  async function loadLatest() {
    setReloading(true);
    setReloadError('');
    try {
      setReloaded(await reload());
    } catch (error) {
      setReloadError(
        error instanceof Error ? error.message : 'Ayarlar alınamadı.',
      );
    } finally {
      setReloading(false);
    }
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNotice('');
    setReloadError('');
    const contact = form.support_contact.trim();
    if (!form.platform_name.trim()) {
      setReloadError('Platform adı boş olamaz.');
      return;
    }
    const digits = contact.replace(/\D/g, '').length;
    if (
      contact &&
      !(
        /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(contact) ||
        /^https:\/\/\S+$/i.test(contact) ||
        (/^\+?\d[\d ()-]{5,30}$/.test(contact) && digits >= 6 && digits <= 20)
      )
    ) {
      setReloadError(
        'Destek iletişimi için e-posta, https bağlantısı veya telefon numarası gir.',
      );
      return;
    }
    save.mutate();
  }
  return (
    <section className="surface management-intro">
      <p>
        Platform adını, destek iletişimini ve öğrenci genel bakışındaki öne
        çıkan önerileri yönet.
      </p>
      {notice && (
        <div className="auth-notice" role="status">
          {notice}
        </div>
      )}
      {save.isError && (
        <div className="form-error" role="alert">
          {save.error.message}
        </div>
      )}
      {reloadError && (
        <div className="form-error" role="alert">
          {reloadError}
        </div>
      )}
      <form onSubmit={submit}>
        <fieldset
          className="settings-fields"
          disabled={save.isPending || reloading}
        >
          <div className="management-grid">
            <label className="field">
              Platform adı
              <input
                name="platform_name"
                required
                minLength={1}
                maxLength={80}
                value={form.platform_name}
                onChange={(e) =>
                  setForm({ ...form, platform_name: e.target.value })
                }
              />
            </label>
            <label className="field">
              Destek iletişimi
              <input
                name="support_contact"
                maxLength={254}
                value={form.support_contact}
                onChange={(e) =>
                  setForm({ ...form, support_contact: e.target.value })
                }
              />
              <small>
                E-posta, https bağlantısı veya telefon; boş bırakılabilir.
              </small>
            </label>
            <label className="field">
              Okul e-posta alan adı
              <input
                value={initial.school_domain}
                readOnly
                aria-describedby="domain-help"
              />
              <small id="domain-help">
                Sunucu yapılandırmasından gelir. Bu ekrandan değiştirilemez.
              </small>
            </label>
            <label className="confirmation-check">
              <input
                name="show_featured"
                type="checkbox"
                checked={form.show_featured}
                onChange={(e) =>
                  setForm({ ...form, show_featured: e.target.checked })
                }
              />
              Öne çıkan önerileri göster
            </label>
          </div>
          <div className="account-actions settings-actions">
            <button className="btn btn-primary" type="submit">
              {save.isPending ? 'Kaydediliyor…' : 'Ayarları kaydet'}
            </button>
            <button
              className="btn btn-secondary"
              type="button"
              onClick={() => void loadLatest()}
            >
              {reloading ? 'Yükleniyor…' : 'Güncel ayarları yeniden yükle'}
            </button>
          </div>
        </fieldset>
      </form>
      {reloaded && (
        <div className="auth-notice" role="status">
          <strong>Sunucudaki güncel ayarlar</strong>
          <dl className="account-details">
            <div>
              <dt>Platform adı</dt>
              <dd>{reloaded.settings.platform_name}</dd>
            </div>
            <div>
              <dt>Destek iletişimi</dt>
              <dd>{reloaded.settings.support_contact || 'Belirtilmedi'}</dd>
            </div>
            <div>
              <dt>Öne çıkan öneriler</dt>
              <dd>
                {reloaded.settings.show_featured ? 'Gösteriliyor' : 'Gizli'}
              </dd>
            </div>
          </dl>
          <p>
            Formundaki bilgiler korundu. Güncel değerlerle devam etmek istersen
            aşağıdaki düğmeyi kullan.
          </p>
          <button
            className="btn btn-secondary"
            disabled={save.isPending}
            onClick={() => {
              setForm({ ...reloaded.settings });
              setReloaded(null);
              save.reset();
              setNotice('Güncel ayarlar forma yüklendi.');
            }}
          >
            Güncel değerleri forma uygula
          </button>
        </div>
      )}
    </section>
  );
}
export function Analytics({ accountId }: { accountId: string }) {
  const analytics = useQuery({
    queryKey: ['management-analytics', accountId],
    queryFn: async ({ signal }) =>
      adminData(await api.GET('/api/admin/analytics', { signal })),
  });
  const data = analytics.data;
  const maxViews = Math.max(
    1,
    ...(data?.benefits.map((benefit) => benefit.view_count) || []),
  );
  const status: Record<string, string> = {
    active: 'Aktif',
    scheduled: 'Yakında',
    expired: 'Süresi doldu',
    draft: 'Taslak',
    suspended: 'Askıda',
    unavailable: 'İşletme etkin değil',
    published: 'Yayımlanmış',
  };
  return (
    <div className="management-analytics">
      <div className="auth-notice">
        <strong>Görüntüleme ve favori istatistikleri</strong>
        <p>
          Bu veriler doğrulanmış avantaj kullanımını göstermez. Bir öğrencinin
          bir avantajı aynı İstanbul gününde tekrar açması tek görüntüleme
          sayılır. Favoriler mevcut toplamı gösterir; tarih aralığına bağlı
          değildir.
        </p>
      </div>
      <button
        className="btn btn-secondary"
        disabled={analytics.isFetching}
        onClick={() => void analytics.refetch()}
      >
        İstatistikleri yenile
      </button>
      {analytics.isPending ? (
        <p role="status">İstatistikler yükleniyor…</p>
      ) : analytics.isError ? (
        <LoadError
          error={analytics.error}
          retry={() => void analytics.refetch()}
        />
      ) : (
        data && (
          <>
            <p>
              Son 30 gün: {time(data.period_start)} – {time(data.period_end)} ·
              İstanbul
            </p>
            <p className="analytics-generated">
              Son güncelleme: {time(data.generated_at)}
            </p>
            <dl className="analytics-totals">
              <div className="surface">
                <dt>Görüntüleme · son 30 gün</dt>
                <dd>{data.totals.views.toLocaleString('tr-TR')}</dd>
              </div>
              <div className="surface">
                <dt>Mevcut favori</dt>
                <dd>{data.totals.favorites.toLocaleString('tr-TR')}</dd>
              </div>
              <div className="surface">
                <dt>Toplam avantaj</dt>
                <dd>{data.totals.benefits.toLocaleString('tr-TR')}</dd>
              </div>
            </dl>
            <h2>En çok görüntülenen {data.limit} avantaj</h2>
            {data.benefits.length ? (
              <div
                className="table-scroll"
                tabIndex={0}
                role="region"
                aria-label="Avantaj istatistikleri"
              >
                <table className="admin-table">
                  <thead>
                    <tr>
                      <th scope="col">Avantaj / işletme</th>
                      <th scope="col">Durum</th>
                      <th scope="col">Görüntüleme · 30 gün</th>
                      <th scope="col">Mevcut favori</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.benefits.map((benefit) => (
                      <tr key={benefit.id}>
                        <td>
                          <strong>{benefit.title}</strong>
                          <small>{benefit.business_name}</small>
                        </td>
                        <td>{status[benefit.status] || benefit.status}</td>
                        <td>
                          <span>
                            {benefit.view_count.toLocaleString('tr-TR')}
                          </span>
                          <div className="analytics-bar" aria-hidden="true">
                            <span
                              style={{
                                width: `${(benefit.view_count / maxViews) * 100}%`,
                              }}
                            />
                          </div>
                        </td>
                        <td>
                          {benefit.favorite_count.toLocaleString('tr-TR')}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <section className="admin-empty">
                <h2>Henüz avantaj bulunmuyor.</h2>
                <p>
                  Avantajlar eklendiğinde görüntüleme ve favori sayıları burada
                  gösterilecek.
                </p>
              </section>
            )}
          </>
        )
      )}
    </div>
  );
}
