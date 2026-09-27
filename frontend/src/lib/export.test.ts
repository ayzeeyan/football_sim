import { describe, expect, test } from 'bun:test';
import { csvEscape, toCSV } from './export';

describe('export helpers (F7)', () => {
  test('csvEscape quotes separators, quotes, and newlines', () => {
    expect(csvEscape('plain')).toBe('plain');
    expect(csvEscape('a,b')).toBe('"a,b"');
    expect(csvEscape('he said "hi"')).toBe('"he said ""hi"""');
    expect(csvEscape('line\nbreak')).toBe('"line\nbreak"');
    expect(csvEscape(null)).toBe('');
    expect(csvEscape(undefined)).toBe('');
    expect(csvEscape(42)).toBe('42');
  });

  test('toCSV emits a header row and one row per record', () => {
    const csv = toCSV([
      { a: 1, b: 'x' },
      { a: 2, b: 'y,z' },
    ]);
    const lines = csv.split('\r\n');
    expect(lines[0]).toBe('a,b');
    expect(lines[1]).toBe('1,x');
    expect(lines[2]).toBe('2,"y,z"');
  });

  test('toCSV honours an explicit column order and empty input', () => {
    const csv = toCSV([{ b: 2, a: 1 }], ['a', 'b']);
    expect(csv.split('\r\n')[0]).toBe('a,b');
    expect(csv.split('\r\n')[1]).toBe('1,2');
    expect(toCSV([])).toBe('');
  });
});
