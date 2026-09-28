import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const panelSource = readFileSync(new URL('./ManagerCareerPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const homeSource = readFileSync(new URL('../competitions/HomeDashboardTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const careerApiSource = readFileSync(new URL('../../services/api/career.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('ManagerCareerPanel (Tier B5 viewer manager career)', () => {
  test('is surfaced on the Home dashboard', () => {
    expect(homeSource).toContain("import { ManagerCareerPanel } from '../career/ManagerCareerPanel'");
    expect(homeSource).toContain('<ManagerCareerPanel');
  });

  test('renders the active job with security, ledger stats, and trophies', () => {
    expect(panelSource).toContain('job_security');
    expect(panelSource).toContain('Sackings');
    expect(panelSource).toContain('Trophies');
    expect(panelSource).toContain('(manager.trophies ?? []).map');
  });

  test('offers every dugout and guards the resign action behind a confirm', () => {
    expect(panelSource).toContain('career.clubs.map');
    expect(panelSource).toContain('Take charge');
    expect(panelSource).toContain('confirmResign');
    expect(panelSource).toContain('Confirm');
  });

  test('resigning hands the club back to the AI world', () => {
    expect(panelSource).toContain('The club appoint a successor');
  });

  test('the api client targets the career manager endpoints', () => {
    expect(careerApiSource).toContain("'/career/manager'");
    expect(careerApiSource).toContain('/career/manager/job');
    expect(careerApiSource).toContain('/career/manager/resign');
    expect(careerApiSource).toContain('export function fetchViewerCareer');
    expect(careerApiSource).toContain('export async function acceptViewerJob');
    expect(careerApiSource).toContain('export async function resignViewerJob');
  });
});
