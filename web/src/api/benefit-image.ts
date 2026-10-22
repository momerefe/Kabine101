// Only same-origin, authenticated image endpoints are accepted from API records.
export function benefitImageURL(path: string): string | undefined {
  return /^\/api\/(?:admin|student)\/benefits\/[0-9a-f-]{36}\/image(?:\?v=\d+)?$/i.test(
    path,
  )
    ? path
    : undefined;
}
