import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const manifest = JSON.parse(readFileSync(new URL('../../../public/manifest.webmanifest', import.meta.url), 'utf8'));
const swSource = readFileSync(new URL('../../../public/sw.js', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const indexHtml = readFileSync(new URL('../../../index.html', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const mainSource = readFileSync(new URL('../../main.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const appSource = readFileSync(new URL('../../App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const bannerSource = readFileSync(new URL('./OfflineBanner.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('PWA shell (F14)', () => {
  test('the web manifest describes an installable standalone shell', () => {
    expect(manifest.name).toBe('Top Five European Football Sim');
    expect(manifest.display).toBe('standalone');
    expect(manifest.start_url).toBe('/');
    expect(manifest.icons.length).toBeGreaterThan(0);
    expect(manifest.icons[0].src).toBe('/icon.svg');
    expect(indexHtml).toContain('<link rel="manifest" href="/manifest.webmanifest" />');
  });

  test('the service worker caches the shell and a read-only data snapshot', () => {
    // App shell precache.
    expect(swSource).toContain("cache.addAll(SHELL_ASSETS)");
    // API reads are network-first with a last-good fallback.
    expect(swSource).toContain("url.pathname.startsWith('/api/')");
    expect(swSource).toContain('X-Offline-Cache');
    // Mutations are never cached or replayed.
    expect(swSource).toContain("request.method !== 'GET'");
    // Old caches are cleaned on activate.
    expect(swSource).toContain('clients.claim()');
  });

  test('the service worker registers only in production builds', () => {
    expect(mainSource).toContain('import.meta.env.PROD');
    expect(mainSource).toContain("navigator.serviceWorker.register('/sw.js')");
  });

  test('an offline banner surfaces connectivity loss', () => {
    expect(appSource).toContain('<OfflineBanner />');
    expect(bannerSource).toContain("window.addEventListener('offline'");
    expect(bannerSource).toContain('last cached snapshot');
  });
});
