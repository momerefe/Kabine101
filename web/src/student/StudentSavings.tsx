import { useQuery } from '@tanstack/react-query';
import type { components } from '../api/schema.gen';
import { formatTRY, purchaseDate } from '../api/purchase-format';
import { studentRequest } from './student-api';

export function MonthlySavings({ accountId }: { accountId: string }) {
  const savings = useQuery({
    queryKey: ['student-savings', accountId],
    queryFn: ({ signal }) =>
      studentRequest<components['schemas']['StudentSavings']>('savings', {
        signal,
      }),
    refetchInterval: 60000,
    refetchOnWindowFocus: true,
  });
  const data = savings.isError ? undefined : savings.data;
  const month = new Intl.DateTimeFormat('tr-TR', {
    month: 'long',
    year: 'numeric',
    timeZone: 'Europe/Istanbul',
  }).format(data ? new Date(`${data.month}-01T12:00:00+03:00`) : new Date());
  return (
    <section
      className="surface student-savings"
      aria-labelledby="student-savings-title"
    >
      <div className="student-savings-summary">
        <div className="student-savings-period">{month}</div>
        <h2 id="student-savings-title">Bu ayki tasarrufun</h2>
        <strong
          className="student-savings-total"
          aria-label={
            data
              ? undefined
              : savings.isPending
                ? 'Tutar yükleniyor'
                : 'Tutar yüklenemedi'
          }
        >
          {data ? formatTRY(data.total_savings) : '—'}
        </strong>
        <span className="student-savings-status">
          Yönetici onaylı{data ? ` · ${data.purchase_count} alışveriş` : ''}
        </span>
      </div>
      <div className="student-savings-breakdown">
        {savings.isPending ? (
          <p role="status">Tasarrufların yükleniyor…</p>
        ) : savings.isError ? (
          <div role="alert">
            <p>Tasarruf bilgileri alınamadı. {savings.error.message}</p>
            <button
              className="btn btn-secondary"
              disabled={savings.isFetching}
              onClick={() => void savings.refetch()}
            >
              Yeniden dene
            </button>
          </div>
        ) : (
          data && (
            <table aria-describedby="student-savings-note">
              <caption className="sr-only">
                {month} işletme bazında tasarruf
              </caption>
              <thead>
                <tr>
                  <th scope="col">İşletme</th>
                  <th scope="col">Tasarruf</th>
                </tr>
              </thead>
              <tbody>
                {data.businesses.length ? (
                  data.businesses.slice(0, 3).map((business) => (
                    <tr key={business.business_id}>
                      <td>
                        {business.business_name}
                        <small>{business.purchase_count} alışveriş</small>
                      </td>
                      <td>{formatTRY(business.savings_amount)}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={2}>
                      Bu ay henüz yönetici onaylı alışverişin yok.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          )
        )}
        {data && data.businesses.length > 3 && (
          <p>Diğer {data.businesses.length - 3} işletme de toplama dahildir.</p>
        )}
        <p id="student-savings-note">
          Alışveriş tarihine göre hesaplanır.{' '}
          <a className="text-link" href="#history">
            Geçmişi gör →
          </a>
        </p>
      </div>
    </section>
  );
}

export function PurchaseHistory({ accountId }: { accountId: string }) {
  const history = useQuery({
    queryKey: ['student-purchases', accountId],
    queryFn: ({ signal }) =>
      studentRequest<components['schemas']['StudentPurchaseList']>(
        'purchases',
        { signal },
      ),
    refetchInterval: 60000,
    refetchOnWindowFocus: true,
  });
  if (history.isPending) return <p role="status">Alışverişlerin yükleniyor…</p>;
  if (history.isError)
    return (
      <div className="notice" role="alert">
        <p>Alışveriş geçmişi alınamadı. {history.error.message}</p>
        <button
          className="btn btn-secondary"
          disabled={history.isFetching}
          onClick={() => void history.refetch()}
        >
          Yeniden dene
        </button>
      </div>
    );
  if (!history.data.purchases.length)
    return (
      <div className="empty-state">
        <h2>Henüz yönetici onaylı alışverişin yok.</h2>
        <p>
          Fişi ve uygulanan sponsorluğu doğrulanan alışverişlerin burada
          görünür. Kod veya QR görüntülemek alışveriş kaydı oluşturmaz.
        </p>
      </div>
    );
  return (
    <section
      className="student-purchase-history"
      aria-label="Yönetici onaylı alışverişler"
    >
      {history.data.truncated && (
        <p className="notice">
          Son 1000 onaylı alışveriş gösteriliyor. Aylık tasarruf toplamına
          ilgili aydaki tüm onaylı alışverişler dahildir.
        </p>
      )}
      <div className="student-purchase-list">
        {history.data.purchases.map((purchase) => (
          <article className="surface student-purchase" key={purchase.id}>
            <div className="student-purchase-heading">
              <div>
                <h2>{purchase.business_name}</h2>
                <p>{purchase.benefit_title}</p>
                <time dateTime={purchase.purchase_date}>
                  {purchaseDate(purchase.purchase_date)}
                </time>
              </div>
              <span className="student-savings-status">Yönetici onaylı</span>
            </div>
            <dl>
              <div>
                <dt>İndirim öncesi</dt>
                <dd>{formatTRY(purchase.original_amount)}</dd>
              </div>
              <div>
                <dt>Ödenen</dt>
                <dd>{formatTRY(purchase.paid_amount)}</dd>
              </div>
              <div>
                <dt>Tasarruf</dt>
                <dd className="purchase-savings">
                  {formatTRY(purchase.savings_amount)}
                </dd>
              </div>
            </dl>
          </article>
        ))}
      </div>
    </section>
  );
}
