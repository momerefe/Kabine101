import { useEffect, useRef, useState } from 'react';
import { benefitImageURL } from '../api/benefit-image';

export type ImageChange = {
  data: string;
  preview: string;
  name: string;
} | null;
const maxBytes = 5 * 1024 * 1024;

export function BenefitImageField({
  savedPath,
  change,
  onChange,
  onBusy,
}: {
  savedPath: string;
  change: ImageChange | undefined;
  onChange: (value: ImageChange) => void;
  onBusy: (busy: boolean) => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  const request = useRef(0);
  const [error, setError] = useState('');
  const [reading, setReading] = useState(false);
  const preview =
    change === undefined ? benefitImageURL(savedPath) : change?.preview;
  useEffect(
    () => () => {
      request.current++;
    },
    [],
  );

  async function selectFile(file: File | undefined) {
    if (!file) return;
    const id = ++request.current;
    setError('');
    if (!['image/png', 'image/jpeg'].includes(file.type)) {
      setError('JPG veya PNG biçiminde bir görsel seç.');
      if (input.current) input.current.value = '';
      return;
    }
    if (!file.size || file.size > maxBytes) {
      setError('Görsel boş olmamalı ve en fazla 5 MB olmalı.');
      if (input.current) input.current.value = '';
      return;
    }
    setReading(true);
    onBusy(true);
    try {
      let preview = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result));
        reader.onerror = () =>
          reject(new Error('Görsel okunamadı. Dosyayı yeniden seç.'));
        reader.readAsDataURL(file);
      });
      const image = new Image();
      image.src = preview;
      try {
        await image.decode();
      } catch {
        throw new Error(
          'Görsel açılamadı. Geçerli bir JPG veya PNG dosyası seç.',
        );
      }
      if (
        image.naturalWidth > 8192 ||
        image.naturalHeight > 8192 ||
        image.naturalWidth * image.naturalHeight > 16000000
      ) {
        throw new Error(
          'Görsel en fazla 16 megapiksel ve her kenarı en fazla 8192 piksel olmalı. Daha küçük bir görsel seç.',
        );
      }
      // Bake the browser's JPEG orientation into pixels before the server strips metadata.
      if (file.type === 'image/jpeg') {
        const canvas = document.createElement('canvas');
        canvas.width = image.naturalWidth;
        canvas.height = image.naturalHeight;
        const context = canvas.getContext('2d');
        if (!context) throw new Error('Görsel hazırlanamadı. Yeniden dene.');
        context.drawImage(image, 0, 0);
        preview = canvas.toDataURL('image/jpeg', 0.92);
        if (
          preview.length - preview.indexOf(',') - 1 >
          4 * Math.ceil(maxBytes / 3)
        ) {
          throw new Error(
            'Hazırlanan görsel 5 MB sınırını aşıyor. Daha küçük bir görsel seç.',
          );
        }
      }
      if (id === request.current)
        onChange({
          data: preview.slice(preview.indexOf(',') + 1),
          preview,
          name: file.name,
        });
    } catch (e) {
      if (id === request.current)
        setError(
          e instanceof Error ? e.message : 'Görsel okunamadı. Yeniden dene.',
        );
      if (input.current) input.current.value = '';
    } finally {
      if (id === request.current) {
        setReading(false);
        onBusy(false);
      }
    }
  }

  return (
    <div className="benefit-image-editor">
      <label className="field" htmlFor="edit-image_file">
        Avantaj görseli (isteğe bağlı)
      </label>
      <input
        ref={input}
        id="edit-image_file"
        type="file"
        accept="image/jpeg,image/png"
        aria-describedby="benefit-image-help"
        onChange={(e) => void selectFile(e.target.files?.[0])}
      />
      <p id="benefit-image-help">
        JPG veya PNG, en fazla 5 MB ve 16 megapiksel. Görsel, avantajı
        kaydettiğinde yüklenir.
      </p>
      {reading && <p role="status">Görsel hazırlanıyor…</p>}
      {error && (
        <p className="field-validation" role="alert">
          {error}
        </p>
      )}
      {preview ? (
        <>
          <img
            src={preview}
            alt="Avantaj görseli önizlemesi"
            onError={() =>
              setError(
                'Görsel yüklenemedi. Bağlantını kontrol et veya görseli yeniden seç.',
              )
            }
          />
          {change && <p>{change.name} — kaydetmeye hazır.</p>}
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => {
              onChange(null);
              setError('');
              if (input.current) input.current.value = '';
            }}
          >
            Görseli kaldır
          </button>
        </>
      ) : (
        <p>
          {change === null ? 'Görsel kaydettiğinde kaldırılacak. ' : ''}Görsel
          yüklenmezse işletmenin görseli kullanılır.
        </p>
      )}
    </div>
  );
}
