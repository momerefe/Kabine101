import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { components } from '../api/schema.gen';
import '../overview.css';

type OverviewData = components['schemas']['OverviewResponse'];
class OverviewAccessError extends Error {}
const number = (value: number) => value.toLocaleString('tr-TR');
function timestamp(value: string, dateOnly = false) {
  return new Intl.DateTimeFormat('tr-TR', {
    dateStyle: 'medium',
    ...(!dateOnly ? { timeStyle: 'short' as const } : {}),
    timeZone: 'Europe/Istanbul',
  }).format(new Date(dateOnly ? `${value}T12:00:00+03:00` : value));
}
const actionLabels: Record<string, string> = {
  'purchase.create': 'Alışveriş kaydedildi',
  'purchase.approved': 'Alışveriş onaylandı',
  'purchase.rejected': 'Alışveriş reddedildi',
  'purchase.revoked': 'Alışveriş onayı geri alındı',
  'business.save': 'İşletme kaydedildi',
  'benefit.save': 'Avantaj kaydedildi',
  'benefit.publish': 'Avantaj yayımlandı',
  'benefit.suspend': 'Avantaj askıya alındı',
  'account.review.approved': 'Öğrenci başvurusu onaylandı',
  'account.review.rejected': 'Öğrenci başvurusu reddedildi',
  'account.role.admin': 'Yönetici rolü verildi',
  'account.role.student': 'Öğrenci rolü verildi',
};

export function Overview({ accountId }: { accountId: string }) {
  const client = useQueryClient();
  const [blockedSnapshot, setBlockedSnapshot] = useState<{
    accountId: string;
    updatedAt: number;
  } | null>(null);
  const overview = useQuery<OverviewData>({
    queryKey: ['management-overview', accountId],
    queryFn: async ({ signal }) => {
      const { data, error, response } = await api.GET('/api/admin/overview', {
        signal,
      });
      if (response.status === 401 || response.status === 403)
        throw new OverviewAccessError(
          'Yönetici erişimin doğrulanamadı. Oturumunu kontrol et.',
        );
      if (error || !data)
        throw new Error(
          'Genel bakış alınamadı. Bağlantını kontrol edip yeniden dene.',
        );
      return data;
    },
    staleTime: 30_000,
    refetchInterval: (query) =>
      query.state.error instanceof OverviewAccessError ||
      (blockedSnapshot?.accountId === accountId &&
        query.state.dataUpdatedAt <= blockedSnapshot.updatedAt)
        ? false
        : 60_000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
  });
  const permissionError = overview.error instanceof OverviewAccessError;
  const accessDenied =
    permissionError ||
    (blockedSnapshot?.accountId === accountId &&
      overview.dataUpdatedAt <= blockedSnapshot.updatedAt);
  useEffect(() => {
    if (permissionError) {
      setBlockedSnapshot({ accountId, updatedAt: overview.dataUpdatedAt });
      void client.invalidateQueries({ queryKey: ['session'] });
    }
  }, [permissionError, accountId, overview.dataUpdatedAt, client]);
  const data = accessDenied ? undefined : overview.data;
  return (
    <div className="overview">
      <div className="overview-intro">
        <div>
          <p>İş ortaklıkları, avantajlar ve öğrenci başvuruları tek bakışta.</p>
          {data && (
            <p className="overview-updated">
              Son güncelleme:{' '}
              <time dateTime={data.generated_at}>
                {timestamp(data.generated_at)}
              </time>{' '}
              · İstanbul
            </p>
          )}
        </div>
        <button
          className="btn btn-secondary"
          disabled={overview.isFetching}
          onClick={() => void overview.refetch()}
        >
          {overview.isFetching ? 'Yenileniyor…' : '↻ Verileri yenile'}
        </button>
      </div>
      {overview.isError && (
        <div className="overview-warning" role="alert">
          {accessDenied
            ? 'Yönetici erişimin doğrulanamadı. Oturumunu kontrol edip yeniden dene.'
            : data
              ? 'Veriler yenilenemedi. Son alınan bilgiler gösteriliyor.'
              : 'Genel bakış alınamadı. Bağlantını kontrol edip yeniden dene.'}
          {!data && (
            <button
              className="text-link"
              onClick={() => void overview.refetch()}
            >
              Yeniden dene →
            </button>
          )}
        </div>
      )}
      {!data && overview.isPending && (
        <div className="overview-loading surface" role="status">
          <span className="overview-loading-mark" aria-hidden="true">
            ▦
          </span>
          <h2>Genel bakış hazırlanıyor</h2>
          <p>Güncel kayıtlar yükleniyor…</p>
        </div>
      )}
      {data && <OverviewContent data={data} />}
    </div>
  );
}

function OverviewContent({ data: d }: { data: OverviewData }) {
  const metrics = [
    {
      label: 'Toplam işletme',
      value: d.businesses.total,
      caption: `${number(d.businesses.active)} aktif iş ortaklığı`,
      href: '#admin/businesses',
      icon: '▤',
    },
    {
      label: 'Aktif avantaj',
      value: d.benefits.active,
      caption: `${number(d.benefits.total)} avantaj kaydı içinde`,
      href: '#admin/benefits?status=active',
      icon: '◇',
    },
    {
      label: 'Aktif öğrenci',
      value: d.students.active,
      caption: `${number(d.students.total)} öğrenci hesabı içinde`,
      href: '#admin/students',
      icon: '♙',
    },
    {
      label: 'Onay bekleyen başvuru',
      value: d.students.pending_review,
      caption: `${number(d.students.review_ready)} başvurunun e-postası doğrulandı`,
      href: '#admin/students',
      icon: '✓',
    },
  ];
  const statuses = [
    ['Aktif', d.benefits.active, 'active'],
    ['Yakında başlayacak', d.benefits.scheduled, 'scheduled'],
    ['Taslak', d.benefits.draft, 'draft'],
    ['Süresi dolmuş', d.benefits.expired, 'expired'],
    ['Askıda', d.benefits.suspended, 'suspended'],
  ] as const;
  return (
    <>
      <div className="overview-metrics">
        {metrics.map((m) => (
          <a className="overview-metric surface" href={m.href} key={m.label}>
            <span className="overview-metric-top">
              {m.label}
              <span className="overview-metric-icon" aria-hidden="true">
                {m.icon}
              </span>
            </span>
            <strong>{number(m.value)}</strong>
            <span className="overview-metric-caption">{m.caption}</span>
            <span className="overview-metric-arrow" aria-hidden="true">
              ↗
            </span>
          </a>
        ))}
      </div>
      <div className="overview-columns">
        <section className="overview-panel surface">
          <div className="overview-panel-heading">
            <div>
              <p className="overview-eyebrow">SIRADAKİ ADIMLAR</p>
              <h2>İlgi bekleyenler</h2>
            </div>
            <span className="overview-panel-symbol" aria-hidden="true">
              ↗
            </span>
          </div>
          <a className="overview-action" href="#admin/students">
            <span className="overview-action-icon" aria-hidden="true">
              ✓
            </span>
            <span>
              <strong>Öğrenci başvurularını incele</strong>
              <small>
                {number(d.students.review_ready)} başvurunun e-postası
                doğrulandı.
              </small>
            </span>
            <b>{number(d.students.pending_review)}</b>
            <span aria-hidden="true">→</span>
          </a>
          <a className="overview-action" href="#admin/benefits?status=draft">
            <span className="overview-action-icon" aria-hidden="true">
              ◇
            </span>
            <span>
              <strong>Avantaj taslaklarını tamamla</strong>
              <small>Koşulları gözden geçir, yayına hazırla.</small>
            </span>
            <b>{number(d.benefits.draft)}</b>
            <span aria-hidden="true">→</span>
          </a>
          <a
            className="overview-action"
            href="#admin/overview"
            onClick={(event) => {
              event.preventDefault();
              document
                .getElementById('overview-expiring')
                ?.scrollIntoView({ behavior: 'instant', block: 'start' });
              document.getElementById('overview-expiring')?.focus();
            }}
          >
            <span className="overview-action-icon" aria-hidden="true">
              ◷
            </span>
            <span>
              <strong>Yaklaşan bitişleri kontrol et</strong>
              <small>Bugün dahil 7 gün içinde bitecek aktif avantajlar.</small>
            </span>
            <b>{number(d.benefits.ending_soon)}</b>
            <span aria-hidden="true">↓</span>
          </a>
          <p className="overview-footnote">
            {number(d.students.pending_email)} öğrenci hesabında e-posta
            doğrulaması bekleniyor. Bu hesaplar onay bekleyen başvurularla
            örtüşebilir.
          </p>
        </section>
        <section className="overview-panel surface">
          <div className="overview-panel-heading">
            <div>
              <p className="overview-eyebrow">AVANTAJLAR</p>
              <h2>Durum dağılımı</h2>
            </div>
            <a className="text-link" href="#admin/benefits">
              Tümü →
            </a>
          </div>
          {d.benefits.total === 0 ? (
            <p className="overview-empty">
              Henüz avantaj kaydı yok. İlk avantajını ekleyerek başla.
            </p>
          ) : (
            <div className="overview-bars">
              {statuses.map(([label, value, key]) => (
                <a
                  href={`#admin/benefits?status=${key}`}
                  className={`overview-bar-row overview-status-${key}`}
                  key={key}
                >
                  <span>
                    {label}
                    <b>{number(value)}</b>
                  </span>
                  <span className="overview-bar-track">
                    <span
                      style={{
                        width: `${d.benefits.total ? Math.min(100, (value / d.benefits.total) * 100) : 0}%`,
                      }}
                    />
                  </span>
                </a>
              ))}
            </div>
          )}
          <p className="overview-footnote">
            {number(d.benefits.published)} yayımlanmış avantaj · Aktif, yakında
            başlayacak ve süresi dolmuş kayıtları kapsar.
          </p>
        </section>
      </div>
      <section className="overview-partnership surface">
        <div>
          <p className="overview-eyebrow">İŞ ORTAKLIKLARI</p>
          <h2>Birlikte büyüyen ağ</h2>
        </div>
        <dl>
          <div>
            <dt>Aktif</dt>
            <dd>{number(d.businesses.active)}</dd>
          </div>
          <div>
            <dt>Görüşme sürecinde</dt>
            <dd>{number(d.businesses.in_progress)}</dd>
          </div>
          <div>
            <dt>Sonlandırılmış</dt>
            <dd>{number(d.businesses.ended)}</dd>
          </div>
        </dl>
        <a className="text-link" href="#admin/partners">
          İş ortaklıklarına git →
        </a>
      </section>
      <div className="overview-columns overview-bottom">
        <section
          className="overview-panel surface"
          id="overview-expiring"
          tabIndex={-1}
        >
          <div className="overview-panel-heading">
            <div>
              <p className="overview-eyebrow">ÖNÜMÜZDEKİ 7 GÜN</p>
              <h2>Yakında sona erecekler</h2>
            </div>
            <span className="overview-count">
              {number(d.benefits.ending_soon)}
            </span>
          </div>
          {d.expiring_benefits.length ? (
            <ul className="overview-list">
              {d.expiring_benefits.map((b) => (
                <li key={b.id}>
                  <span className="overview-list-icon" aria-hidden="true">
                    ◇
                  </span>
                  <div>
                    <strong>{b.title}</strong>
                    <small>{b.business_name}</small>
                  </div>
                  <time dateTime={b.valid_until}>
                    {timestamp(b.valid_until, true)}
                  </time>
                </li>
              ))}
            </ul>
          ) : (
            <p className="overview-empty">
              Önümüzdeki 7 gün içinde sona erecek aktif avantaj bulunmuyor.
            </p>
          )}
          {d.benefits.ending_soon > d.expiring_benefits.length && (
            <p className="overview-footnote">
              Bitiş tarihine göre ilk 5 kayıt gösteriliyor.
            </p>
          )}
          <a
            className="text-link overview-list-footer"
            href="#admin/benefits?status=active"
          >
            Aktif avantajları yönet →
          </a>
        </section>
        <section className="overview-panel surface">
          <div className="overview-panel-heading">
            <div>
              <p className="overview-eyebrow">SON HAREKETLER</p>
              <h2>Yönetimden güncellemeler</h2>
            </div>
            <span className="overview-panel-symbol" aria-hidden="true">
              ≡
            </span>
          </div>
          {d.recent_activity.length ? (
            <ol className="overview-activity">
              {d.recent_activity.map((a) => (
                <li key={a.id}>
                  <span className="overview-activity-dot" aria-hidden="true" />
                  <div>
                    <strong>
                      {actionLabels[a.action] || 'Yönetim kaydı güncellendi'}
                    </strong>
                    <p>{a.entity_label || 'Kayıt artık mevcut değil'}</p>
                    <small>
                      {a.actor_name} ·{' '}
                      <time dateTime={a.created_at}>
                        {timestamp(a.created_at)}
                      </time>
                    </small>
                  </div>
                </li>
              ))}
            </ol>
          ) : (
            <p className="overview-empty">
              Henüz yönetim hareketi yok. Yeni işlemler burada görünecek.
            </p>
          )}
        </section>
      </div>
    </>
  );
}
