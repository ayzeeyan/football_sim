import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const appSource = readFileSync(new URL('./App.tsx', import.meta.url), 'utf8');

function macroHandlerSource(): string {
  const start = appSource.indexOf("const handleMacroSim = useCallback(async (mode: 'week' | 'month' | 'season') => {");
  const end = appSource.indexOf('\n\n  const watchClub', start);
  if (start < 0 || end < 0) throw new Error('handleMacroSim source not found');
  return appSource.slice(start, end);
}

describe('macro simulation execution lock', () => {
  test('acquires synchronously before asynchronous work and releases in finally', () => {
    const source = macroHandlerSource();
    const guard = source.indexOf('if (simLockRef.current) return;');
    const acquire = source.indexOf('simLockRef.current = true;');
    const visualState = source.indexOf('setSimulating(true);');
    const firstAwait = source.indexOf('await ');
    const finallyBlock = source.indexOf('finally {');
    const release = source.lastIndexOf('simLockRef.current = false;');

    expect(guard).toBeGreaterThanOrEqual(0);
    expect(acquire).toBeGreaterThan(guard);
    expect(visualState).toBeGreaterThan(acquire);
    expect(firstAwait).toBeGreaterThan(acquire);
    expect(finallyBlock).toBeGreaterThan(firstAwait);
    expect(release).toBeGreaterThan(finallyBlock);
  });

  test('buttons and keyboard shortcuts share the same guarded handler', () => {
    expect(appSource).toContain("onSimWeek={() => void handleMacroSim('week')}");
    expect(appSource).toContain("onSimMonth={() => void handleMacroSim('month')}");
    expect(appSource).toContain("onSimSeason={() => void handleMacroSim('season')}");

    expect(appSource).toContain("void handleMacroSim('week');");
    expect(appSource).toContain("void handleMacroSim('month');");
    expect(appSource).toContain("void handleMacroSim('season');");
  });
});
