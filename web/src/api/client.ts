import createClient from 'openapi-fetch';
import type { paths } from './schema.gen';

// Same-origin API: Vite proxies /api locally; production uses one reverse proxy.
export const api = createClient<paths>({ baseUrl: '' });
