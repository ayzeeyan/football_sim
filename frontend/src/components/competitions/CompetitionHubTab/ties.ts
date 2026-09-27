import type { CompetitionFixtureRow } from '../../../types';

export function tieGroups(ties: CompetitionFixtureRow[]): CompetitionFixtureRow[][] {
  const order: string[] = [];
  const byId = new Map<string, CompetitionFixtureRow[]>();
  for (const t of ties) {
    const k = t.tie_id ?? t.id;
    const arr = byId.get(k);
    if (arr) arr.push(t);
    else {
      byId.set(k, [t]);
      order.push(k);
    }
  }
  return order.map((k) => {
    const arr = byId.get(k) ?? [];
    return [...arr].sort((a, b) => (a.leg ?? 0) - (b.leg ?? 0) || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
  });
}

// aggregateLine renders a finished two-legged tie's real aggregate (summed
// per club, since venues swap). Unfinished or degenerate groups yield null
// so no manufactured scoreline is ever shown.
export function aggregateLine(group: CompetitionFixtureRow[]): string | null {
  if (group.length !== 2) return null;
  const l1 = group[0]!;
  const l2 = group[1]!;
  if (l1.status !== 'finished' || l2.status !== 'finished') return null;
  if (l1.home_goals == null || l1.away_goals == null || l2.home_goals == null || l2.away_goals == null) return null;
  const totals = new Map<string, { goals: number; short: string }>();
  const add = (id: string, short: string | undefined, g: number) => {
    const cur = totals.get(id) ?? { goals: 0, short: short ?? id };
    cur.goals += g;
    totals.set(id, cur);
  };
  add(l1.home_id, l1.home?.short_name, l1.home_goals);
  add(l1.away_id, l1.away?.short_name, l1.away_goals);
  add(l2.home_id, l2.home?.short_name, l2.home_goals);
  add(l2.away_id, l2.away?.short_name, l2.away_goals);
  if (totals.size !== 2) return null;
  const [x, y] = [...totals.entries()].sort((a, b) => (a[0] < b[0] ? -1 : 1));
  if (!x || !y) return null;
  return `AGG ${x[1].short} ${x[1].goals}–${y[1].goals} ${y[1].short}`;
}

// tieWinnerId resolves the side that won a finished tie: aggregate goals
// across legs, then penalties from the deciding leg. Unfinished ties and
// unresolvable shootouts yield null so no winner is ever invented.
export function tieWinnerId(group: CompetitionFixtureRow[]): string | null {
  if (group.length === 0) return null;
  const totals = new Map<string, number>();
  let pensWinner: string | null = null;
  let allFinished = true;
  for (const f of group) {
    if (f.status !== 'finished' || f.home_goals == null || f.away_goals == null) {
      allFinished = false;
      continue;
    }
    totals.set(f.home_id, (totals.get(f.home_id) ?? 0) + f.home_goals);
    totals.set(f.away_id, (totals.get(f.away_id) ?? 0) + f.away_goals);
    if (f.decided_by === 'penalties' && f.penalties && f.penalties.length >= 2) {
      pensWinner = (f.penalties[0] ?? 0) > (f.penalties[1] ?? 0) ? f.home_id : f.away_id;
    }
  }
  if (!allFinished) return null;
  const [a, b] = [...totals.entries()];
  if (!a) return null;
  if (!b) return a[0];
  if (a[1] !== b[1]) return a[1] > b[1] ? a[0] : b[0];
  return pensWinner;
}
