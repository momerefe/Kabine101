import type { components } from '../api/schema.gen';
export type Benefit = components['schemas']['StudentBenefit'];
export type Catalog = components['schemas']['StudentBenefitList'];
export type Redemption = components['schemas']['StudentRedemption'];
export async function studentRequest<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch('/api/student/' + path, {
    credentials: 'same-origin',
    cache: 'no-store',
    ...init,
  });
  if (!response.ok)
    throw new Error(
      response.status === 404
        ? 'Bu avantaj bulunamadı veya artık erişilebilir değil.'
        : response.status === 401 || response.status === 403
          ? 'Bu içeriğe erişilemiyor. Oturumunu ve hesap durumunu kontrol et.'
          : 'İşlem tamamlanamadı. Lütfen yeniden dene.',
    );
  return response.json() as Promise<T>;
}
export const categories = [
  ['food', 'Yeme & içme'],
  ['fashion', 'Giyim & moda'],
  ['tech', 'Teknoloji'],
  ['sport', 'Spor & fitness'],
  ['culture', 'Eğlence & etkinlik'],
  ['other', 'Diğer avantajlar'],
];
export const methodLabels: Record<string, string> = {
  code: 'İndirim kodu',
  link: 'Avantaj bağlantısı',
  qr: 'QR kodu',
  membership: 'Üyelik kartı',
};
export const statusLabels: Record<string, string> = {
  active: 'Aktif',
  scheduled: 'Yakında',
  expired: 'Süresi doldu',
  unavailable: 'Artık erişilemiyor',
};
export function safeURL(value: string | undefined) {
  try {
    const url = new URL(value || '');
    return ['http:', 'https:'].includes(url.protocol) ? url.href : undefined;
  } catch {
    return undefined;
  }
}
export async function copyText(value: string): Promise<boolean> {
  try {
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(value);
      return true;
    }
  } catch {
    /* HTTP LAN uses the selection fallback. */
  }
  const area = document.createElement('textarea');
  area.value = value;
  area.style.position = 'fixed';
  area.style.opacity = '0';
  document.body.append(area);
  area.select();
  let copied = false;
  try {
    copied = document.execCommand('copy');
  } catch {
    /* Manual copy remains available. */
  }
  area.remove();
  return copied;
}
