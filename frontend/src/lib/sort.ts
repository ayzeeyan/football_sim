/** Generic sorting / filtering helpers shared by Squad + Wonderkid tables. */

export function sortBy<T, K extends keyof T>(rows: T[], col: K, asc: boolean): T[] {
  return [...rows].sort((a, b) => {
    const va = a[col];
    const vb = b[col];
    if (typeof va === 'number' && typeof vb === 'number') {
      return asc ? va - vb : vb - va;
    }
    return asc
      ? String(va).localeCompare(String(vb))
      : String(vb).localeCompare(String(va));
  });
}

export function matchesQuery(haystack: string, query: string): boolean {
  return haystack.toLowerCase().includes(query.trim().toLowerCase());
}
