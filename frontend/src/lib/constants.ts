import type { Club, Player } from '../types';

/** Central app constants — single source of truth for tabs, leagues, speeds. */

export const LEAGUES_5 = ['Premier League', 'La Liga', 'Serie A', 'Bundesliga', 'Ligue 1'] as const;

export const LEAGUE_FILTER_ALL = ['All', ...LEAGUES_5] as const;

export const MATCH_SPEEDS = [
  { value: 1, label: '1x' },
  { value: 2, label: '2x' },
  { value: 5, label: '5x' },
  { value: 999, label: 'Instant' },
] as const;

export const TRAINING_FOCUSES = [
  { value: 'hypertrophy', label: 'Hypertrophy' },
  { value: 'technical', label: 'Technical' },
  { value: 'tactical', label: 'Tactical' },
] as const;

export type TrainingFocus = (typeof TRAINING_FOCUSES)[number]['value'];

export const TABS = [
  { id: 0, slug: 'match', label: 'Match' },
  { id: 1, slug: 'lab', label: 'Wonderkids' },
  { id: 2, slug: 'league', label: 'League' },
  { id: 6, slug: 'inbox', label: 'Inbox' },
  { id: 5, slug: 'history', label: 'History' },
  { id: 3, slug: 'squads', label: 'Squads' },
  { id: 4, slug: 'transfers', label: 'Transfers' },
] as const;

export type TabId = (typeof TABS)[number]['id'];
export type TabSlug = (typeof TABS)[number]['slug'];

export function tabFromSlug(slug: string | null | undefined): TabId {
  if (slug === 'cup') return 2;
  const found = TABS.find((t) => t.slug === slug);
  return (found?.id ?? 2) as TabId;
}

export function slugFromTab(id: TabId): TabSlug {
  return TABS.find((t) => t.id === id)?.slug ?? 'league';
}

export function clubById(clubs: Club[], id: string | null | undefined): Club | null {
  if (!id) return null;
  return clubs.find((c) => c.club_id === id) ?? null;
}

export function positionTone(category: Player['category']): string {
  switch (category) {
    case 'GK':
      return 'bg-brass/15 text-brass border border-brass/40';
    case 'DEF':
      return 'bg-[#5B7FA6]/15 text-[#9DBBDC] border border-[#5B7FA6]/30';
    case 'MID':
      return 'bg-[#8E86C8]/15 text-[#B9B3E6] border border-[#8E86C8]/30';
    case 'FWD':
    default:
      return 'bg-pitchtone/15 text-[#A9CDBB] border border-pitchtone/30';
  }
}

export function ovrTone(ovr: number): string {
  if (ovr >= 85) return 'text-brass';
  if (ovr >= 78) return 'text-pitchtone';
  return 'text-bone/80';
}
