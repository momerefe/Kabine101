const currency = new Intl.NumberFormat('tr-TR', {
  style: 'currency',
  currency: 'TRY',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

// Keep decimal amounts exact, including values beyond Number's integer range.
export function formatTRY(value: string): string {
  if (!/^\d+(?:\.\d{1,2})?$/.test(value)) return '—';
  const [whole, fraction = ''] = value.split('.');
  return currency
    .formatToParts(BigInt(whole))
    .map((part) =>
      part.type === 'fraction' ? fraction.padEnd(2, '0') : part.value,
    )
    .join('');
}

export function purchaseDate(value: string): string {
  return new Intl.DateTimeFormat('tr-TR', {
    dateStyle: 'medium',
    timeZone: 'Europe/Istanbul',
  }).format(new Date(`${value}T12:00:00+03:00`));
}
