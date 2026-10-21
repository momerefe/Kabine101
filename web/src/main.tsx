import React from 'react';
import ReactDOM from 'react-dom/client';
import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from '@tanstack/react-query';
import { api } from './api/client';
import '../../styles.css';
import './foundation.css';

const queryClient = new QueryClient();

function Foundation() {
  const readiness = useQuery({
    queryKey: ['readiness'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/ready');
      if (error || data?.status !== 'ok')
        throw new Error('API veya veritabanı hazır değil.');
      return data;
    },
    retry: false,
    refetchOnWindowFocus: false,
  });

  return (
    <main className="foundation">
      <header className="foundation-header">
        <a
          className="foundation-brand"
          href="/"
          aria-label="KoçBenefits ana sayfa"
        >
          Koç<span>Benefits</span>
        </a>
        <span className="demo-tag">YEREL GELİŞTİRME</span>
      </header>
      <section className="foundation-card">
        <p className="eyebrow">3A · PROJE VE VERİ TEMELİ</p>
        <h1>KoçBenefits’in temeli hazır.</h1>
        <p>
          Öğrenci ve yönetici ekranlarının tasarım referansı mevcut HTML
          prototipidir. Bu başlangıç ekranı yeni uygulamanın API ve veritabanı
          bağlantısını kontrol eder.
        </p>
        <div className="foundation-status" role="status" aria-live="polite">
          <span
            className={`status-dot ${readiness.isSuccess ? 'is-ready' : ''}`}
            aria-hidden="true"
          />
          {readiness.isPending
            ? 'Bağlantı kontrol ediliyor…'
            : readiness.isSuccess
              ? 'API ve PostgreSQL şeması hazır'
              : 'Bağlantı kurulamadı. API’yi başlatıp veritabanı şemasını kontrol et.'}
        </div>
        <button
          className="btn btn-primary"
          onClick={() => void readiness.refetch()}
          disabled={readiness.isFetching}
        >
          {readiness.isFetching
            ? 'Kontrol ediliyor…'
            : 'Bağlantıyı yeniden kontrol et'}
        </button>
        <hr />
        <h2>Mevcut tasarımı incele</h2>
        <p>
          Proje kökündeki <strong>index.html</strong> dosyasını açarak altı
          öğrenci ve 12 yönetici bölümünü demo verilerle inceleyebilirsin.
        </p>
        <p className="foundation-note">
          Gerçek kayıt, giriş ve yönetim API’leri henüz uygulanmadı. Bu ekran
          öğrenci hizmeti veya güvenli oturum sağlamaz.
        </p>
      </section>
      <footer>Öğrencilerden, öğrencilere.</footer>
    </main>
  );
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <Foundation />
    </QueryClientProvider>
  </React.StrictMode>,
);
