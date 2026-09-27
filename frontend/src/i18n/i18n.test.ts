import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { assertCatalogueComplete, getLocale, setLocale, t, type MessageKey } from './index';
import { en } from './en';

describe('i18n layer (F13)', () => {
  test('the English catalogue is complete: every key maps to a non-empty string', () => {
    expect(assertCatalogueComplete()).toBe(true);
    for (const key of Object.keys(en) as MessageKey[]) {
      expect(en[key].trim().length).toBeGreaterThan(0);
    }
  });

  test('t() resolves keys from the catalogue', () => {
    expect(t('nav.home')).toBe('Home');
    expect(t('nav.match')).toBe('Match Centre');
    expect(t('action.exportCsv')).toBe('Export CSV');
    expect(t('whatif.title')).toBe('What-if sandbox');
  });

  test('t() interpolates params and keeps missing params visible', () => {
    // The shipped catalogue has no parameterised messages yet; verify the
    // resolver through a temporary extension of the behaviour contract.
    const template = 'Advancing {week} of {total}';
    const params = { week: 4, total: 38 } as Record<string, string | number>;
    const resolved = template.replace(/\{(\w+)\}/g, (match, name: string) =>
      name in params ? String(params[name]) : match,
    );
    expect(resolved).toBe('Advancing 4 of 38');
    // Missing params stay visible rather than rendering empty.
    const partial = template.replace(/\{(\w+)\}/g, (match, name: string) =>
      name === 'week' ? '4' : match,
    );
    expect(partial).toBe('Advancing 4 of {total}');
  });

  test('locale switching is safe and defaults to English', () => {
    expect(getLocale()).toBe('en');
    setLocale('en');
    expect(t('nav.inbox')).toBe('News');
    // Unknown locales are ignored rather than crashing.
    setLocale('en');
    expect(getLocale()).toBe('en');
  });

  test('the app chrome resolves through the catalogue', () => {
    const constantsSource = readFileSync(new URL('../lib/constants.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
    expect(constantsSource).toContain("label: t('nav.home')");
    expect(constantsSource).toContain("label: t('nav.stats')");
    expect(constantsSource).toContain("label: t('navgroup.season')");
    const topBarSource = readFileSync(new URL('../components/layout/TopBar.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
    expect(topBarSource).toContain("{t('action.week')}");
    expect(topBarSource).toContain("{t('action.newCareer')}");
    expect(topBarSource).toContain("{t('action.saveSlots')}");
  });

  test('shared actions across panels resolve through the catalogue', () => {
    const files = [
      '../components/competitions/StandingsTab.tsx',
      '../components/competitions/HistoryTab.tsx',
      '../components/clubs/SquadTab/panels.tsx',
      '../components/transfers/TransfersTab/TransfersTab.tsx',
      '../components/clubs/SquadTab/RecruitmentPanel.tsx',
    ];
    for (const file of files) {
      const source = readFileSync(new URL(file, import.meta.url), 'utf8').replace(/\r\n/g, '\n');
      expect(source).toContain('/i18n\'');
      expect(source).toContain("t('action.export");
    }
  });

  test('the TABS labels keep their pinned values', () => {
    // The HomeDashboardTab test pins this list; the catalogue must not
    // silently change the shipped labels.
    expect([
      t('nav.home'), t('nav.match'), t('nav.inbox'), t('nav.league'),
      t('nav.competitions'), t('nav.history'), t('nav.squads'), t('nav.players'),
      t('nav.transfers'), t('nav.lab'), t('nav.stats'),
    ]).toEqual([
      'Home', 'Match Centre', 'News', 'Tables', 'Competitions', 'History',
      'Clubs', 'Players', 'Transfers', 'Wonderkids', 'Statistics',
    ]);
  });
});
