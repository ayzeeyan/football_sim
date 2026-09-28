import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const standingsSource = readFileSync(new URL('./StandingsTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const homeSource = readFileSync(new URL('./HomeDashboardTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const appSource = readFileSync(new URL('../../App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const careerApiSource = readFileSync(new URL('../../services/api/career.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('League-change surfacing (closed-pyramid promotion/relegation)', () => {
  test('the tables badge promoted and relegated clubs', () => {
    expect(standingsSource).toContain('fetchLeagueChanges');
    expect(standingsSource).toContain("leagueChanges[c.club_id]");
    expect(standingsSource).toContain('Promoted from');
    expect(standingsSource).toContain('Relegated from');
    expect(standingsSource).toContain("'Up' : 'Down'");
  });

  test('the legend explains the badges only when moves exist', () => {
    expect(standingsSource).toContain('Promoted this season');
    expect(standingsSource).toContain('Relegated this season');
    expect(standingsSource).toContain('Object.keys(leagueChanges).length > 0');
  });

  test('the season reset reports the boundary swaps', () => {
    expect(standingsSource).toContain('res.league_changes');
    expect(standingsSource).toContain('crossed league boundaries');
  });

  test('the api client targets the league-changes endpoint', () => {
    expect(careerApiSource).toContain("'/season/league-changes'");
    expect(careerApiSource).toContain('export function fetchLeagueChanges');
  });
});

describe('Toast plumbing regression (silent failures)', () => {
  test('the Home dashboard forwards the app toast channel to its panels', () => {
    expect(homeSource).toContain('onShowToast: (msg: string) => void;');
    expect(homeSource).toContain('onShowToast={onShowToast}');
    expect(homeSource).not.toContain('onShowToast={() => undefined}');
  });

  test('App passes showToast to the Home dashboard', () => {
    expect(appSource).toContain('onShowToast={showToast}');
  });
});
