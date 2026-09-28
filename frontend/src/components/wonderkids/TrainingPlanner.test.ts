import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { TRAINING_FOCUSES } from '../../lib/constants';

const planner = readFileSync(new URL('./TrainingPlanner.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const lab = readFileSync(new URL('./WonderkidLabTab/WonderkidLabTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const prodigiesApi = readFileSync(new URL('../../services/api/prodigies.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('TrainingPlanner (F1, read-only staff plan)', () => {
  test('shows every backend regimen from TRAINING_FOCUSES as a chip', () => {
    expect(planner).toContain('TRAINING_FOCUSES.map');
    for (const { value } of TRAINING_FOCUSES) {
      expect(planner).toContain(value);
    }
  });

  test('is purely observational: no training action, no energy spend', () => {
    expect(planner).toContain('fetchTrainingProjection(');
    expect(planner).toContain('read-only projection');
    expect(planner).not.toContain('trainProdigy');
    expect(planner).not.toContain('onClick');
  });

  test('the wonderkid lab wires the planner to the selected prodigy', () => {
    expect(lab).toContain('<TrainingPlanner playerId={selected.player_id} />');
  });

  test('the api client exposes the projection endpoint and no training mutation', () => {
    expect(prodigiesApi).toContain('/training/projection/');
    expect(prodigiesApi).not.toContain('export function trainProdigy');
    expect(prodigiesApi).not.toContain('position-path');
    expect(prodigiesApi).not.toContain('school-track');
  });
});
