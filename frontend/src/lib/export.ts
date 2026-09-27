/**
 * User-initiated data export (Phase 3 F7). Converts already-fetched state
 * to CSV or JSON and triggers a browser download. Nothing is written to
 * the repository or the runtime save — exports live only in the user's
 * downloads folder.
 */

/** Escapes and quotes one CSV field when required. */
export function csvEscape(value: unknown): string {
  const text = value == null ? '' : String(value);
  if (/[",\r\n]/.test(text)) {
    return `"${text.split('"').join('""')}"`;
  }
  return text;
}

/** Serializes rows of key-value records to RFC-4180-ish CSV. */
export function toCSV(rows: Array<Record<string, unknown>>, columns?: string[]): string {
  if (rows.length === 0) return '';
  const keys = columns ?? Object.keys(rows[0]!);
  const lines = [keys.join(',')];
  for (const row of rows) {
    lines.push(keys.map((key) => csvEscape(row[key])).join(','));
  }
  return lines.join('\r\n');
}

function download(filename: string, mime: string, data: string): void {
  const blob = new Blob([data], { type: mime });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  URL.revokeObjectURL(url);
}

/** Downloads rows as a CSV file. */
export function downloadCSV(filename: string, rows: Array<Record<string, unknown>>, columns?: string[]): void {
  download(filename.endsWith('.csv') ? filename : `${filename}.csv`, 'text/csv;charset=utf-8', toCSV(rows, columns));
}

/** Downloads any JSON-serializable payload. */
export function downloadJSON(filename: string, data: unknown): void {
  download(filename.endsWith('.json') ? filename : `${filename}.json`, 'application/json', JSON.stringify(data, null, 2));
}
