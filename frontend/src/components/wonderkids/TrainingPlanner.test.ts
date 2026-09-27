import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { TRAINING_FOCUSES } from '../../lib/constants';

const planner = readFileSync(new URL('./TrainingPlanner.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const lab = readFileSync(new URL('./WonderkidLabTab/WonderkidLabTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const prodigiesApi = readFileSync(new URL('../../services/api/prodigies.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('TrainingPlanner (F1)', () => {
  test('offers every backend regimen from TRAINING_FOCUSES', () => {
    // Labels render at runtime from TRAINING_FOCUSES; the source must map
    // over it and key its per-regimen icon/blurb tables by focus value.
    expect(planner).toContain('TRAINING_FOCUSES.map');
    for (const { value } of TRAINING_FOCUSES) {
      expect(planner).toContain(value);
    }
  });

  test('spends energy through the existing train endpoint and shows gains', () => {
    expect(planner).toContain('trainProdigy(');
    expect(planner).toContain('remaining_energy');
    expect(planner).toContain('No energy');
  });

  test('renders the read-only staff projection for non-prodigies', () => {
    expect(planner).toContain('fetchTrainingProjection(');
    expect(planner).toContain('read-only projection');
    expect(planner).toContain('trainable');
  });

  test('the wonderkid lab wires the planner to the selected prodigy', () => {
    expect(lab).toContain('<TrainingPlanner prodigy={selected}');
    expect(lab).toContain("onTrained={() => void loadData()}");
  });

  test('the api client exposes the projection endpoint', () => {
    expect(prodigiesApi).toContain('/training/projection/');
  });
});
