
export const API_BASE = '/api';
/** Short-lived GET cache: enough to dedupe remounts/tab switches without going stale after a sim. */
const API_CACHE_TTL_MS = 12_000;

type ApiCacheEntry = { value: unknown; expires: number };
const apiGetCache = new Map<string, ApiCacheEntry>();
const apiInflightGet = new Map<string, Promise<unknown>>();
let apiCacheGeneration = 0;

/** Drop cached GET responses (all, or those whose path starts with the prefix). */
export function invalidateApiCache(pathPrefix?: string): void {
  // Requests already in flight may finish after invalidation. Mark that
  // generation stale so those responses cannot repopulate the cache.
  apiCacheGeneration += 1;
  if (!pathPrefix) {
    apiGetCache.clear();
    apiInflightGet.clear();
    return;
  }
  for (const key of [...apiGetCache.keys()]) {
    if (key.startsWith(pathPrefix)) apiGetCache.delete(key);
  }
  for (const key of [...apiInflightGet.keys()]) {
    if (key.startsWith(pathPrefix)) apiInflightGet.delete(key);
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit, fallback?: T): Promise<T> {
  const method = (init?.method ?? 'GET').toUpperCase();
  const cacheable = method === 'GET' && init?.body == null;

  if (cacheable) {
    const cached = apiGetCache.get(path);
    if (cached && cached.expires > Date.now()) return cached.value as T;
    const pending = apiInflightGet.get(path);
    if (pending) return pending as Promise<T>;
  }

  const requestGeneration = apiCacheGeneration;
  const run = async (): Promise<T> => {
    try {
      const res = await fetch(`${API_BASE}${path}`, init);
      if (!res.ok) {
        let errMsg = `HTTP ${res.status} for ${path}`;
        try {
          const body = await res.json();
          if (body.detail) errMsg = body.detail;
          else if (body.message) errMsg = body.message;
        } catch {
          // ignore json parse error
        }
        throw new Error(errMsg);
      }
      const data = (await res.json()) as T;
      if (cacheable && requestGeneration === apiCacheGeneration) {
        apiGetCache.set(path, { value: data, expires: Date.now() + API_CACHE_TTL_MS });
      } else if (method !== 'GET') {
        // Mutations change world state; never serve stale GETs afterward.
        invalidateApiCache();
      }
      return data;
    } catch (err) {
      console.warn(`[API] ${path} failed`, err);
      if (fallback !== undefined && (!(err instanceof Error) || !err.message || err.message.startsWith('HTTP'))) {
        return fallback;
      }
      throw err;
    }
  };

  if (!cacheable) return run();

  let pending: Promise<T>;
  pending = run().finally(() => {
    if (apiInflightGet.get(path) === pending) apiInflightGet.delete(path);
  });
  apiInflightGet.set(path, pending);
  return pending;
}