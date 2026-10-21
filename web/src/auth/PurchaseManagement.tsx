import { useEffect, useId, useRef, useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { newRecordId } from '../api/record-id';
import { formatTRY, purchaseDate } from '../api/purchase-format';
import type { components } from '../api/schema.gen';

type Purchase = components['schemas']['Purchase'];
type Draft = components['schemas']['PurchaseCreate'];
type Decision = components['schemas']['PurchaseReview']['decision'];
const statuses = {
  pending: 'Onay bekliyor',
  approved: 'Onaylandı',
  rejected: 'Reddedildi',
  revoked: 'Onay geri alındı',
};
const decisions = {
  approved: 'Alışverişi onayla',
  rejected: 'Alışverişi reddet',
  revoked: 'Onayı geri al',
};
const blank = (): Draft => ({
  id: newRecordId(),
  student_email: '',
  benefit_id: '',
  purchase_date: '',
  original_amount: '',
  paid_amount: '',
  reference: '',
});
function checked<T>(result: {
  data?: T;
  error?: unknown;
  response: Response;
}): T {
  if (result.error || !result.data) {
    const detail =
      result.error &&
      typeof result.error === 'object' &&
      'message' in result.error
        ? String(result.error.message)
        : '';
    throw new Error(
      result.response.status === 409
        ? 'Kayıt değişmiş veya aynı fiş referansı daha önce kaydedilmiş. Listeyi yenileyip kontrol et.'
        : detail ||
            'İşlem tamamlanamadı. Oturumunu ve bağlantını kontrol edip yeniden dene.',
    );
  }
  return result.data;
}

export function PurchaseManagement({
  accountId,
  csrf,
}: {
  accountId: string;
  csrf: string;
}) {
  const client = useQueryClient();
  const [q, setQ] = useState('');
  const [status, setStatus] = useState('');
  const [draft, setDraft] = useState<Draft | null>(null);
  const [notice, setNotice] = useState('');
  const [review, setReview] = useState<{
    purchase: Purchase;
    decision: Decision;
  } | null>(null);
  const purchases = useQuery({
    queryKey: ['management-purchases', accountId, q, status],
    queryFn: async ({ signal }) =>
      checked(
        await api.GET('/api/admin/purchases', {
          params: { query: { q, status: status || undefined } },
          signal,
        }),
      ),
    refetchInterval: 60000,
    refetchOnWindowFocus: true,
  });
  const benefits = useQuery({
    queryKey: ['management-benefits', accountId, 'purchase-options'],
    queryFn: async ({ signal }) =>
      checked(await api.GET('/api/admin/benefits', { signal })),
    enabled: draft !== null,
  });
  const availableBenefits =
    benefits.data?.benefits.filter(
      (benefit) => benefit.publication_status === 'published',
    ) || [];
  const today = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Europe/Istanbul',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date());
  function refreshed(studentId: string) {
    void client.invalidateQueries({
      queryKey: ['management-purchases', accountId],
    });
    void client.invalidateQueries({
      queryKey: ['management-overview', accountId],
    });
    void client.invalidateQueries({ queryKey: ['student-savings', studentId] });
    void client.invalidateQueries({
      queryKey: ['student-purchases', studentId],
    });
  }
  const create = useMutation({
    mutationFn: async (body: Draft) =>
      checked(
        await api.POST('/api/admin/purchases', {
          params: { header: { 'X-CSRF-Token': csrf } },
          body,
        }),
      ),
    onSuccess: ({ purchase }) => {
      setDraft(null);
      setNotice(
        'Alışveriş onay bekleyen kayıt olarak eklendi. Fişi ve sponsorluğu kontrol edip ayrıca onayla.',
      );
      refreshed(purchase.student_id);
    },
  });
  return (
    <>
      <div className="surface management-intro">
        <p>
          Gerçek alışverişin fişini ve uygulanan sponsorluğu incele. Yalnızca
          yönetici onaylı kayıtlar öğrencinin tasarrufuna yansır. Kod veya QR
          görüntülemek onay değildir.
        </p>
        <p>
          Tutarlar TRY, alışveriş tarihi İstanbul saatine göredir. Tutarlar
          sonradan değiştirilemez; hatalı kaydı reddet veya onayını geri alıp
          doğru bilgilerle yeni kayıt oluştur.
        </p>
      </div>
      <div className="heading-actions">
        <button
          className="btn btn-primary"
          disabled={draft !== null}
          onClick={() => {
            create.reset();
            setNotice('');
            setDraft(blank());
          }}
        >
          Alışveriş ekle +
        </button>
      </div>
      {notice && (
        <div className="auth-notice" role="status">
          {notice}
        </div>
      )}
      {draft && (
        <section
          className="surface form-group purchase-create"
          aria-labelledby="purchase-create-title"
        >
          <h2 id="purchase-create-title">Yeni alışveriş</h2>
          <p>
            Öğrenci hesabına bağlı e-posta adresini, fiş tutarlarını ve
            işletmede benzersiz fiş / şube referansını gir.
          </p>
          {create.isError && (
            <p className="form-error" role="alert">
              {create.error.message} Bilgilerin korundu; yeniden denediğinde
              aynı kayıt kimliği kullanılır.
            </p>
          )}
          {benefits.isPending ? (
            <p role="status">Avantajlar yükleniyor…</p>
          ) : benefits.isError ? (
            <div role="alert">
              <p>Avantaj listesi alınamadı.</p>
              <button
                className="btn btn-secondary"
                onClick={() => void benefits.refetch()}
              >
                Avantajları yeniden yükle
              </button>
            </div>
          ) : !availableBenefits.length ? (
            <p>Alışveriş eklemek için yayımlanmış bir avantaj gerekli.</p>
          ) : null}
          {benefits.data?.truncated && (
            <p className="auth-notice">
              Avantaj listesinin bir bölümü gösteriliyor. Aradığın avantaj
              listede yoksa avantaj yönetiminden kontrol et.
            </p>
          )}
          <form
            onSubmit={(event: FormEvent) => {
              event.preventDefault();
              create.mutate({
                ...draft,
                student_email: draft.student_email.trim(),
                original_amount: draft.original_amount.replace(',', '.'),
                paid_amount: draft.paid_amount.replace(',', '.'),
                reference: draft.reference.trim(),
              });
            }}
          >
            <fieldset disabled={create.isPending}>
              <div className="management-grid">
                <label className="field">
                  Öğrenci e-postası
                  <input
                    name="student_email"
                    type="email"
                    required
                    maxLength={254}
                    value={draft.student_email}
                    onChange={(e) =>
                      setDraft({ ...draft, student_email: e.target.value })
                    }
                  />
                </label>
                <label className="field">
                  Avantaj
                  <select
                    name="benefit_id"
                    required
                    value={draft.benefit_id}
                    disabled={benefits.isPending || benefits.isError}
                    onChange={(e) =>
                      setDraft({ ...draft, benefit_id: e.target.value })
                    }
                  >
                    <option value="">Avantaj seç</option>
                    {availableBenefits.map((b) => (
                      <option key={b.id} value={b.id}>
                        {b.business_name} · {b.title}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field">
                  Alışveriş tarihi
                  <input
                    name="purchase_date"
                    type="date"
                    max={today}
                    required
                    value={draft.purchase_date}
                    onChange={(e) =>
                      setDraft({ ...draft, purchase_date: e.target.value })
                    }
                  />
                </label>
                <label className="field">
                  Fiş / şube referansı
                  <input
                    name="reference"
                    required
                    maxLength={160}
                    value={draft.reference}
                    onChange={(e) =>
                      setDraft({ ...draft, reference: e.target.value })
                    }
                  />
                </label>
                <label className="field">
                  İndirim öncesi tutar (TRY)
                  <input
                    name="original_amount"
                    inputMode="decimal"
                    required
                    pattern="[0-9]{1,7}([.,][0-9]{1,2})?"
                    maxLength={10}
                    title="En fazla 9999999,99 TRY; en çok iki ondalık basamak gir."
                    placeholder="100,00"
                    value={draft.original_amount}
                    onChange={(e) =>
                      setDraft({ ...draft, original_amount: e.target.value })
                    }
                  />
                </label>
                <label className="field">
                  Ödenen tutar (TRY)
                  <input
                    name="paid_amount"
                    inputMode="decimal"
                    required
                    pattern="[0-9]{1,7}([.,][0-9]{1,2})?"
                    maxLength={10}
                    title="En fazla indirim öncesi tutar; en çok iki ondalık basamak gir."
                    placeholder="80,00"
                    value={draft.paid_amount}
                    onChange={(e) =>
                      setDraft({ ...draft, paid_amount: e.target.value })
                    }
                  />
                </label>
              </div>
              <div className="account-actions">
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={
                    benefits.isPending ||
                    benefits.isError ||
                    !availableBenefits.length
                  }
                >
                  {create.isPending
                    ? 'Kaydediliyor…'
                    : 'Onay bekleyen kayıt oluştur'}
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => {
                    setDraft(null);
                    create.reset();
                  }}
                >
                  Vazgeç
                </button>
              </div>
            </fieldset>
          </form>
        </section>
      )}
      <div className="admin-toolbar">
        <label className="field">
          Alışverişlerde ara
          <input
            type="search"
            maxLength={100}
            value={q}
            placeholder="Öğrenci, işletme veya fiş referansı"
            onChange={(e) => setQ(e.target.value)}
          />
        </label>
        <label className="field">
          Alışveriş durumu
          <select value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">Tüm durumlar</option>
            {Object.entries(statuses).map(([key, label]) => (
              <option key={key} value={key}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <button
          className="btn btn-secondary"
          disabled={purchases.isFetching}
          onClick={() => void purchases.refetch()}
        >
          Listeyi yenile
        </button>
      </div>
      {purchases.isPending ? (
        <p role="status">Alışverişler yükleniyor…</p>
      ) : purchases.isError ? (
        <div className="form-error" role="alert">
          <p>{purchases.error.message}</p>
          <button
            className="btn btn-secondary"
            disabled={purchases.isFetching}
            onClick={() => void purchases.refetch()}
          >
            Yeniden dene
          </button>
        </div>
      ) : purchases.data.purchases.length ? (
        <div
          className="table-scroll"
          tabIndex={0}
          role="region"
          aria-label="Alışveriş kayıtları"
        >
          <table className="admin-table purchase-table">
            <thead>
              <tr>
                <th scope="col">Öğrenci / işletme</th>
                <th scope="col">Alışveriş</th>
                <th scope="col">Tutarlar</th>
                <th scope="col">Durum / işlemler</th>
              </tr>
            </thead>
            <tbody>
              {purchases.data.purchases.map((purchase) => (
                <tr key={purchase.id}>
                  <td>
                    <strong>{purchase.student_name}</strong>
                    <small>{purchase.student_email}</small>
                    <small>{purchase.business_name}</small>
                    <small>{purchase.benefit_title}</small>
                  </td>
                  <td>
                    <time dateTime={purchase.purchase_date}>
                      {purchaseDate(purchase.purchase_date)}
                    </time>
                    <small>Referans: {purchase.reference}</small>
                  </td>
                  <td>
                    <small>
                      İndirim öncesi: {formatTRY(purchase.original_amount)}
                    </small>
                    <small>Ödenen: {formatTRY(purchase.paid_amount)}</small>
                    <small>
                      Tasarruf: {formatTRY(purchase.savings_amount)}
                    </small>
                  </td>
                  <td>
                    <span className="status-badge">
                      {statuses[purchase.status]}
                    </span>
                    {purchase.review_reason && (
                      <small>Neden: {purchase.review_reason}</small>
                    )}
                    <div className="account-actions">
                      {(purchase.status === 'pending'
                        ? (['approved', 'rejected'] as const)
                        : purchase.status === 'approved'
                          ? (['revoked'] as const)
                          : []
                      ).map((decision) => (
                        <button
                          key={decision}
                          className="btn btn-secondary"
                          onClick={() => setReview({ purchase, decision })}
                        >
                          {decisions[decision]}
                        </button>
                      ))}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="surface management-intro">
          <h2>
            {q || status
              ? 'Filtrelere uygun alışveriş yok.'
              : 'Henüz alışveriş kaydı yok.'}
          </h2>
          <p>Kaydedilen alışverişler burada incelenir ve onaylanır.</p>
        </div>
      )}
      {!purchases.isError && purchases.data?.truncated && (
        <p className="auth-notice" role="status">
          Sonuçların bir bölümü gösteriliyor. Arama veya durum filtresiyle
          listeyi daralt.
        </p>
      )}
      {review && (
        <PurchaseReview
          key={`${review.purchase.id}:${review.decision}`}
          {...review}
          csrf={csrf}
          onClose={() => setReview(null)}
          onSuccess={() => {
            setNotice(`${decisions[review.decision]} işlemi tamamlandı.`);
            setReview(null);
            refreshed(review.purchase.student_id);
          }}
        />
      )}
    </>
  );
}

function PurchaseReview({
  purchase,
  decision,
  csrf,
  onClose,
  onSuccess,
}: {
  purchase: Purchase;
  decision: Decision;
  csrf: string;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  const [reason, setReason] = useState('');
  const [confirmed, setConfirmed] = useState(false);
  useEffect(() => {
    const previous = document.activeElement as HTMLElement;
    const element = dialog.current;
    element?.showModal();
    return () => {
      element?.close();
      previous?.focus();
    };
  }, []);
  const mutation = useMutation({
    mutationFn: async () =>
      checked(
        await api.POST('/api/admin/purchases/{id}/review', {
          params: {
            path: { id: purchase.id },
            header: { 'X-CSRF-Token': csrf },
          },
          body: { decision, reason: reason.trim() },
        }),
      ),
    onSuccess,
  });
  return (
    <dialog
      className="account-dialog purchase-review"
      ref={dialog}
      aria-labelledby={titleId}
      onCancel={(e) => {
        if (mutation.isPending) e.preventDefault();
        else onClose();
      }}
    >
      <h2 id={titleId}>{decisions[decision]}</h2>
      <p>
        <strong>{purchase.student_name}</strong>
        <br />
        {purchase.student_email}
        <br />
        {purchase.business_name} · {purchase.benefit_title}
        <br />
        {purchaseDate(purchase.purchase_date)} · Referans: {purchase.reference}
      </p>
      <p>
        İndirim öncesi: {formatTRY(purchase.original_amount)}
        <br />
        Ödenen: {formatTRY(purchase.paid_amount)}
        <br />
        Tasarruf: {formatTRY(purchase.savings_amount)}
      </p>
      <p>
        {decision === 'approved'
          ? 'Onayladığında bu alışveriş öğrencinin tasarrufuna ve geçmişine yansır.'
          : decision === 'revoked'
            ? 'Hata veya iade nedeniyle bu alışverişin onayı geri alınacak; öğrencinin tasarrufundan ve onaylı geçmişinden çıkarılacak.'
            : 'Bu alışveriş reddedilecek ve öğrencinin tasarrufuna eklenmeyecek.'}
      </p>
      {mutation.isError && (
        <p className="form-error" role="alert">
          {mutation.error.message}
        </p>
      )}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (confirmed && (decision === 'approved' || reason.trim()))
            mutation.mutate();
        }}
      >
        <fieldset disabled={mutation.isPending}>
          {decision !== 'approved' && (
            <label className="field">
              {decision === 'rejected' ? 'Ret nedeni' : 'Geri alma nedeni'}
              <textarea
                name="reason"
                required
                maxLength={2000}
                rows={3}
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              />
            </label>
          )}
          <label className="confirmation-check">
            <input
              type="checkbox"
              required
              checked={confirmed}
              onChange={(e) => setConfirmed(e.target.checked)}
            />
            {decision === 'approved'
              ? 'Gerçek fişi inceledim ve sponsorluğun bu alışverişe uygulandığını doğruladım.'
              : 'Bu alışveriş için belirtilen işlemi ve nedenini onaylıyorum.'}
          </label>
          <div className="account-actions">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onClose}
            >
              Vazgeç
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={
                !confirmed || (decision !== 'approved' && !reason.trim())
              }
            >
              {mutation.isPending ? 'İşleniyor…' : decisions[decision]}
            </button>
          </div>
        </fieldset>
      </form>
    </dialog>
  );
}
