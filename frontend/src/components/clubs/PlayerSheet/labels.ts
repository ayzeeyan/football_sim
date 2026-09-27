
export const RESULT_TONE: Record<string, string> = {
  W: 'text-[#A9CDBB]',
  D: 'text-brass',
  L: 'text-[#D89A84]',
};

export const COMPETITION_LABELS: Record<string, string> = {
  'premier-league': 'Premier League',
  'la-liga': 'La Liga',
  'bundesliga': 'Bundesliga',
  'serie-a': 'Serie A',
  'ligue-1': 'Ligue 1',
  'fa-cup': 'FA Cup',
  'efl-cup': 'EFL Cup',
  'copa-del-rey': 'Copa del Rey',
  'dfb-pokal': 'DFB-Pokal',
  'coppa-italia': 'Coppa Italia',
  'coupe-de-france': 'Coupe de France',
  'champions-league': 'UEFA Champions League',
  'europa-league': 'UEFA Europa League',
  'conference-league': 'UEFA Conference League',
  'super-league': 'Super League',
  'ucl': 'Champions Cup',
  'super-cup': 'Super Cup',
};

export function prettyCompetitionName(id: string | null | undefined): string {
  if (!id) return 'League';
  const key = id.trim().toLowerCase();
  if (COMPETITION_LABELS[key]) return COMPETITION_LABELS[key];
  return id
    .replace(/[_]+/g, ' ')
    .replace(/-/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export type PlayerSheetTab = 'overview' | 'stats' | 'growth' | 'career' | 'history';

/** Club ids look like "EPL-ARS"; surface the short suffix instead of the raw id. */
export function clubIdLabel(id?: string | null): string {
  if (!id) return '';
  const parts = id.split('-');
  return parts[parts.length - 1] || id;
}
