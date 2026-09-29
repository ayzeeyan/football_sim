import type { FormationName, PositionFit } from '../types';

export const FORMATION_SLOTS: Record<FormationName, readonly string[]> = {
  '4-3-3': ['GK', 'LB', 'LCB', 'RCB', 'RB', 'LCM', 'CM', 'RCM', 'LW', 'ST', 'RW'],
  '4-3-3 Attack': ['GK', 'LB', 'LCB', 'RCB', 'RB', 'LCM', 'RCM', 'CAM', 'LW', 'ST', 'RW'],
  '4-2-3-1': ['GK', 'LB', 'LCB', 'RCB', 'RB', 'LDM', 'RDM', 'LW', 'CAM', 'RW', 'ST'],
  '4-4-2': ['GK', 'LB', 'LCB', 'RCB', 'RB', 'LM', 'LCM', 'RCM', 'RM', 'LST', 'RST'],
  '3-4-3': ['GK', 'LCB', 'CB', 'RCB', 'LM', 'LCM', 'RCM', 'RM', 'LW', 'ST', 'RW'],
  '3-5-2': ['GK', 'LCB', 'CB', 'RCB', 'LWB', 'CDM', 'LCM', 'RCM', 'RWB', 'LST', 'RST'],
  '4-1-4-1': ['GK', 'LB', 'LCB', 'RCB', 'RB', 'CDM', 'LM', 'LCM', 'RCM', 'RM', 'ST'],
  '5-3-2': ['GK', 'LWB', 'LCB', 'CB', 'RCB', 'RWB', 'LCM', 'CDM', 'RCM', 'LST', 'RST'],
};

export const FORMATION_PITCH_COORDS: Record<string, [number, number]> = {
  GK: [50, 92],
  LB: [14, 73], LWB: [12, 68], LCB: [36, 78], CB: [50, 80], RCB: [64, 78], RB: [86, 73], RWB: [88, 68],
  LDM: [35, 62], CDM: [50, 62], RDM: [65, 62],
  LM: [14, 49], LCM: [32, 51], CM: [50, 57], RCM: [68, 51], RM: [86, 49],
  LAM: [32, 37], CAM: [50, 37], RAM: [68, 37],
  LW: [17, 21], LF: [35, 24], CF: [50, 22], RF: [65, 24], RW: [83, 21],
  LST: [37, 17], ST: [50, 15], RST: [63, 17],
};

// Match-report cards use a portrait pitch with their own goal at the top.
export const REPORT_PITCH_COORDS: Record<string, [number, number]> = {
  GK: [50, 17],
  LB: [13, 44], LWB: [11, 53], LCB: [37.5, 44], CB: [50, 44], RCB: [62.5, 44], RB: [87, 44], RWB: [89, 53],
  LDM: [36, 65], CDM: [50, 65], RDM: [64, 65],
  LM: [13, 76], LCM: [29, 75], CM: [50, 75], RCM: [71, 75], RM: [87, 76],
  LAM: [30, 91], CAM: [50, 91], RAM: [70, 91],
  LW: [16, 108], LF: [33, 106], CF: [50, 106], RF: [67, 106], RW: [84, 108],
  LST: [38, 112], ST: [50, 113], RST: [62, 112],
};

// Formation-specific layout overrides. The base table places the interior
// roles for a central trio; shapes with a flat bank of four or a back
// three/five need their own spreads so adjacent name chips never collide
// (same-row tokens keep >= 22% horizontal separation, or >= 6% vertical).
export const FORMATION_COORD_OVERRIDES: Record<FormationName, Record<string, [number, number]>> = {
  '4-3-3': {},
  '4-3-3 Attack': {},
  '4-2-3-1': {},
  '4-4-2': {
    LM: [14, 49], LCM: [38, 52], RCM: [62, 52], RM: [86, 49],
  },
  '3-4-3': {
    LCB: [26, 78], CB: [50, 80], RCB: [74, 78],
    LM: [14, 50], LCM: [38, 53], RCM: [62, 53], RM: [86, 50],
  },
  '3-5-2': {
    LCB: [26, 78], CB: [50, 80], RCB: [74, 78],
    LWB: [12, 64], RWB: [88, 64],
    CDM: [50, 58], LCM: [32, 48], RCM: [68, 48],
    LST: [37, 17], RST: [63, 17],
  },
  '4-1-4-1': {
    CDM: [50, 60],
    LM: [14, 48], LCM: [38, 50], RCM: [62, 50], RM: [86, 48],
  },
  '5-3-2': {
    LWB: [12, 66], LCB: [28, 78], CB: [50, 80], RCB: [72, 78], RWB: [88, 66],
    CDM: [50, 57], LCM: [33, 47], RCM: [67, 47],
    LST: [37, 17], RST: [63, 17],
  },
};

// pitchCoords resolves a slot's pitch position for a specific formation:
// the formation's override first, then the shared base table.
export function pitchCoords(formation: string | undefined, slot: string): [number, number] | undefined {
  const overrides = FORMATION_COORD_OVERRIDES[normalizeFormation(formation)];
  const overridden = overrides?.[slot];
  if (overridden) return overridden;
  return FORMATION_PITCH_COORDS[slot];
}

// Unit tones: the formation reads as banks at a glance when each line of
// the shape carries its own colour. GK stays amber, defenders sky,
// midfielders emerald, forwards rose. The slot label always renders as
// text next to the ring, so colour is never the only signal.
export type UnitTone = 'gk' | 'def' | 'mid' | 'fwd';

export function unitForSlot(slot: string): UnitTone {
  const normalized = normalizedPosition(slot);
  if (normalized === 'GK') return 'gk';
  if (['LB', 'LCB', 'CB', 'RCB', 'RB', 'LWB', 'RWB'].includes(normalized)) return 'def';
  if (['LDM', 'CDM', 'RDM', 'DM', 'LM', 'LCM', 'CM', 'RCM', 'RM', 'LAM', 'AM', 'CAM', 'RAM'].includes(normalized)) return 'mid';
  return 'fwd';
}

export const UNIT_TONE_CLASSES: Record<UnitTone, { ring: string; pill: string }> = {
  gk: { ring: 'border-amber-300/80', pill: 'bg-amber-300/15 text-amber-200' },
  def: { ring: 'border-sky-300/80', pill: 'bg-sky-300/15 text-sky-200' },
  mid: { ring: 'border-emerald-300/80', pill: 'bg-emerald-300/15 text-emerald-200' },
  fwd: { ring: 'border-rose-300/80', pill: 'bg-rose-300/15 text-rose-200' },
};

// inferFormationFromSlots recovers the formation a lineup actually played
// from the tactical slots the backend assigned: the shape whose slot set
// exactly matches the observed slots. Returns undefined when the payload
// carries no slot information (legacy saves) or matches no known shape.
export function inferFormationFromSlots<T extends TacticalPlayerLike>(players: T[]): FormationName | undefined {
  const slots = new Set<string>();
  for (const player of players) {
    const slot = normalizedPosition(player.tactical_slot || player.starting_slot);
    if (slot) slots.add(slot);
  }
  if (slots.size === 0) return undefined;
  const names = Object.keys(FORMATION_SLOTS) as FormationName[];
  for (const name of names) {
    const defined = FORMATION_SLOTS[name] ?? [];
    if (defined.length !== slots.size) continue;
    let matches = true;
    for (const slot of slots) {
      if (!defined.includes(slot)) {
        matches = false;
        break;
      }
    }
    if (matches) return name;
  }
  return undefined;
}

function sameSlotSet(a: FormationName, b: FormationName): boolean {
  const left = FORMATION_SLOTS[a] ?? [];
  const right = FORMATION_SLOTS[b] ?? [];
  if (left.length !== right.length) return false;
  const seen = new Set(right);
  return left.every((slot) => seen.has(slot));
}

// resolveLineupFormation picks the shape a lineup should render as.
// The backend-assigned tactical slots are ground truth: a stale declared
// formation loses to the observed slots. A declared formation that is
// compatible with the observed slots wins, though — 3-5-2 and 5-3-2 field
// the same eleven slots, and only the declaration can tell them apart.
// Payloads with no slot information keep the declared (or default) shape.
export function resolveLineupFormation<T extends TacticalPlayerLike>(formation: string | undefined, players: T[]): FormationName {
  const inferred = inferFormationFromSlots(players);
  if (!inferred) return normalizeFormation(formation);
  const declared = normalizeFormation(formation);
  if (sameSlotSet(declared, inferred)) return declared;
  return inferred;
}

interface TacticalPlayerLike {
  player_id: string;
  full_name?: string;
  position: string;
  secondary_position?: string;
  category?: string;
  ovr?: number;
  universe_wonderkid?: boolean;
  is_wk?: boolean;
  tactical_slot?: string;
  starting_slot?: string;
  natural_position?: string;
  position_fit?: PositionFit;
}

export interface TacticalAssignment<T extends TacticalPlayerLike> {
  player: T;
  naturalPosition: string;
  tacticalSlot: string;
  positionFit: PositionFit;
}

export function normalizedPosition(position?: string): string {
  return (position ?? '').trim().toUpperCase();
}

export function normalizeFormation(formation?: string): FormationName {
  switch ((formation ?? '').trim().toLowerCase()) {
    case '4-2-3-1': case '4231': return '4-2-3-1';
    case '4-4-2': case '442': return '4-4-2';
    case '3-4-3': case '343': return '3-4-3';
    case '3-5-2': case '352': return '3-5-2';
    case '4-1-4-1': case '4141': return '4-1-4-1';
    case '5-3-2': case '532': return '5-3-2';
    case '4-3-3 attack': case '4-3-3-attack': case '433 attack': case '433a': return '4-3-3 Attack';
    default: return '4-3-3';
  }
}

const FITS: Record<string, Record<string, PositionFit>> = {
  GK: { GK: 'Natural' },
  LB: { LB: 'Natural', LWB: 'Good', LCB: 'Acceptable', CB: 'Acceptable' },
  LCB: { LCB: 'Natural', CB: 'Natural', RCB: 'Good', LB: 'Emergency', LWB: 'Emergency', RB: 'Emergency' },
  RCB: { RCB: 'Natural', CB: 'Natural', LCB: 'Good', RB: 'Emergency', RWB: 'Emergency', LB: 'Emergency' },
  RB: { RB: 'Natural', RWB: 'Good', RCB: 'Acceptable', CB: 'Acceptable' },
  CB: { CB: 'Natural', LCB: 'Natural', RCB: 'Natural', LB: 'Emergency', RB: 'Emergency', LWB: 'Emergency', RWB: 'Emergency' },
  LWB: { LWB: 'Natural', LB: 'Good', LM: 'Good', LCB: 'Acceptable', CB: 'Acceptable' },
  RWB: { RWB: 'Natural', RB: 'Good', RM: 'Good', RCB: 'Acceptable', CB: 'Acceptable' },
  LDM: { LDM: 'Natural', CDM: 'Natural', DM: 'Natural', CM: 'Acceptable', LCM: 'Acceptable', RCM: 'Acceptable' },
  CDM: { CDM: 'Natural', DM: 'Natural', LDM: 'Good', RDM: 'Good', CM: 'Acceptable', LCM: 'Acceptable', RCM: 'Acceptable' },
  RDM: { RDM: 'Natural', CDM: 'Natural', DM: 'Natural', CM: 'Acceptable', LCM: 'Acceptable', RCM: 'Acceptable' },
  LM: { LM: 'Natural', LW: 'Good', LWB: 'Good', LCM: 'Acceptable', CM: 'Acceptable', CAM: 'Acceptable', RM: 'Emergency', RW: 'Emergency' },
  LCM: { LCM: 'Natural', CM: 'Natural', RCM: 'Good', LM: 'Good', CDM: 'Acceptable', DM: 'Acceptable', CAM: 'Acceptable', RM: 'Acceptable' },
  CM: { CM: 'Natural', LCM: 'Good', RCM: 'Good', CDM: 'Good', DM: 'Good', CAM: 'Acceptable', LM: 'Acceptable', RM: 'Acceptable' },
  RCM: { RCM: 'Natural', CM: 'Natural', LCM: 'Good', RM: 'Good', CDM: 'Acceptable', DM: 'Acceptable', CAM: 'Acceptable', LM: 'Acceptable' },
  RM: { RM: 'Natural', RW: 'Good', RWB: 'Good', RCM: 'Acceptable', CM: 'Acceptable', CAM: 'Acceptable', LM: 'Emergency', LW: 'Emergency' },
  LAM: { LAM: 'Natural', CAM: 'Good', AM: 'Good', LW: 'Good', LM: 'Acceptable', CM: 'Acceptable', RAM: 'Emergency' },
  CAM: { CAM: 'Natural', AM: 'Natural', LAM: 'Good', RAM: 'Good', CM: 'Acceptable', LCM: 'Acceptable', RCM: 'Acceptable', CF: 'Acceptable' },
  RAM: { RAM: 'Natural', CAM: 'Good', AM: 'Good', RW: 'Good', RM: 'Acceptable', CM: 'Acceptable', LAM: 'Emergency' },
  LW: { LW: 'Natural', LM: 'Good', LF: 'Good', RW: 'Acceptable', RM: 'Acceptable' },
  LF: { LF: 'Natural', LW: 'Good', CF: 'Good', ST: 'Acceptable', CAM: 'Acceptable', RF: 'Emergency' },
  CF: { CF: 'Natural', ST: 'Good', CAM: 'Acceptable', LF: 'Acceptable', RF: 'Acceptable', LW: 'Emergency', RW: 'Emergency' },
  RF: { RF: 'Natural', RW: 'Good', CF: 'Good', ST: 'Acceptable', CAM: 'Acceptable', LF: 'Emergency' },
  RW: { RW: 'Natural', RM: 'Good', RF: 'Good', LW: 'Acceptable', LM: 'Acceptable' },
  ST: { ST: 'Natural', CF: 'Good', LF: 'Acceptable', RF: 'Acceptable', LW: 'Emergency', RW: 'Emergency' },
  LST: { LST: 'Natural', ST: 'Natural', CF: 'Good', LF: 'Good', RF: 'Acceptable', LW: 'Acceptable', RW: 'Emergency' },
  RST: { RST: 'Natural', ST: 'Natural', CF: 'Good', RF: 'Good', LF: 'Acceptable', RW: 'Acceptable', LW: 'Emergency' },
};

const FIT_RANK: Record<PositionFit, number> = { Natural: 4, Good: 3, Acceptable: 2, Emergency: 1 };
const FIT_BONUS: Record<PositionFit, number> = { Natural: 3000, Good: 2400, Acceptable: 1600, Emergency: 0 };
export const ATTACKING_ROLE_GROUPS = [
  { label: 'Attacking midfield', slots: ['LAM', 'AM', 'CAM', 'RAM'] },
  { label: 'Forward roles', slots: ['LW', 'LF', 'CF', 'RF', 'RW'] },
  { label: 'Striker roles', slots: ['LST', 'ST', 'RST'] },
] as const;

function secondaryFit(fit: PositionFit): PositionFit {
  if (fit === 'Natural') return 'Good';
  if (fit === 'Good') return 'Acceptable';
  return fit;
}

export function positionFit(player: TacticalPlayerLike, tacticalSlot: string): PositionFit {
  const slot = normalizedPosition(tacticalSlot);
  const lookup = FITS[slot] ?? {};
  const primary = lookup[normalizedPosition(player.position)] ?? 'Emergency';
  const secondary = player.secondary_position
    ? secondaryFit(lookup[normalizedPosition(player.secondary_position)] ?? 'Emergency')
    : 'Emergency';
  return (FIT_RANK[secondary] ?? 0) > (FIT_RANK[primary] ?? 0) ? secondary : primary;
}

function slotCategory(slot: string): string {
  if (slot === 'GK') return 'GK';
  if (['LB', 'LCB', 'CB', 'RCB', 'RB', 'LWB', 'RWB'].includes(slot)) return 'DEF';
  if (['LDM', 'CDM', 'RDM', 'DM', 'LM', 'LCM', 'CM', 'RCM', 'RM'].includes(slot)) return 'MID';
  return 'FWD';
}

function inferredCategory(player: TacticalPlayerLike): string {
  if (player.category) return player.category;
  const position = normalizedPosition(player.position);
  if (position === 'GK') return 'GK';
  if (['LB', 'LCB', 'CB', 'RCB', 'RB', 'LWB', 'RWB'].includes(position)) return 'DEF';
  if (['LDM', 'CDM', 'RDM', 'DM', 'LM', 'LCM', 'CM', 'RCM', 'RM'].includes(position)) return 'MID';
  return 'FWD';
}

function assignmentScore(player: TacticalPlayerLike, slot: string): number {
  const fit = positionFit(player, slot);
  let score = (FIT_BONUS[fit] ?? 0) + (player.ovr ?? 0) * 10 + ((player.universe_wonderkid || player.is_wk) ? 200 : 0) + 1;
  if (fit === 'Emergency') {
    const category = inferredCategory(player);
    if (category === slotCategory(slot)) score += 300;
    else if (slot === 'GK') score += category === 'DEF' ? 250 : category === 'MID' ? 120 : category === 'FWD' ? 60 : 0;
  }
  return score;
}

function stablePlayerKey(player: TacticalPlayerLike): string {
  return player.player_id || `${(player.full_name ?? '').trim().toLowerCase()}|${normalizedPosition(player.position)}`;
}

function minimumCostAssignment(weights: number[][]): number[] {
  const n = weights.length;
  if (!n) return [];
  const m = weights[0]?.length ?? 0;
  const u = Array<number>(n + 1).fill(0);
  const v = Array<number>(m + 1).fill(0);
  const p = Array<number>(m + 1).fill(0);
  const way = Array<number>(m + 1).fill(0);
  for (let i = 1; i <= n; i += 1) {
    p[0] = i;
    let j0 = 0;
    const minv = Array<number>(m + 1).fill(Number.MAX_SAFE_INTEGER);
    const used = Array<boolean>(m + 1).fill(false);
    do {
      used[j0] = true;
      const i0 = p[j0]!;
      let delta = Number.MAX_SAFE_INTEGER;
      let j1 = 0;
      for (let j = 1; j <= m; j += 1) {
        if (used[j]) continue;
        const cur = -(weights[i0 - 1]?.[j - 1] ?? -1_000_000) - (u[i0] ?? 0) - (v[j] ?? 0);
        if (cur < (minv[j] ?? Number.MAX_SAFE_INTEGER)) {
          minv[j] = cur;
          way[j] = j0;
        }
        if ((minv[j] ?? Number.MAX_SAFE_INTEGER) < delta || (minv[j] === delta && (j1 === 0 || j < j1))) {
          delta = minv[j]!;
          j1 = j;
        }
      }
      for (let j = 0; j <= m; j += 1) {
        if (used[j]) {
          u[p[j]!] = (u[p[j]!] ?? 0) + delta;
          v[j] = (v[j] ?? 0) - delta;
        } else if (j > 0) minv[j]! -= delta;
      }
      j0 = j1;
    } while (p[j0] !== 0);
    do {
      const j1 = way[j0]!;
      p[j0] = p[j1]!;
      j0 = j1;
    } while (j0 !== 0);
  }
  const result = Array<number>(n).fill(-1);
  for (let j = 1; j <= m; j += 1) {
    const row = p[j]!;
    if (row > 0 && row <= n) result[row - 1] = j - 1;
  }
  return result;
}

// resolveTacticalAssignments honours valid backend slots first. Only missing
// legacy fields are inferred from the declared formation and natural positions.
export function resolveTacticalAssignments<T extends TacticalPlayerLike>(players: T[], formation?: string): TacticalAssignment<T>[] {
  const canonicalFormation = normalizeFormation(formation);
  const formationSlots = [...(FORMATION_SLOTS[canonicalFormation] ?? [])];
  const indexed = players.map((player, index) => ({ player, index })).sort((a, b) => {
    const keyOrder = stablePlayerKey(a.player).localeCompare(stablePlayerKey(b.player));
    return keyOrder || a.index - b.index;
  });
  const resolved = new Map<number, TacticalAssignment<T>>();
  const usedSlots = new Set<string>();

  for (const { player, index } of indexed) {
    const explicit = normalizedPosition(player.tactical_slot || player.starting_slot);
    if (!explicit || !formationSlots.includes(explicit) || usedSlots.has(explicit)) continue;
    usedSlots.add(explicit);
    resolved.set(index, {
      player,
      naturalPosition: normalizedPosition(player.natural_position || player.position),
      tacticalSlot: explicit,
      positionFit: player.position_fit ?? positionFit(player, explicit),
    });
  }

  const remaining = indexed.filter(({ index }) => !resolved.has(index));
  const openSlots = formationSlots.filter((slot) => !usedSlots.has(slot));
  if (remaining.length && openSlots.length) {
    const columns = Math.max(remaining.length, openSlots.length);
    const weights = openSlots.map((slot) => Array.from({ length: columns }, (_, column) => (
      column < remaining.length ? assignmentScore(remaining[column]!.player, slot) : -1_000_000
    )));
    const assignment = minimumCostAssignment(weights);
    assignment.forEach((column, slotIndex) => {
      if (column < 0 || column >= remaining.length) return;
      const row = remaining[column];
      if (!row) return;
      const { player, index } = row;
      const tacticalSlot = openSlots[slotIndex];
      if (!tacticalSlot) return;
      resolved.set(index, {
        player,
        naturalPosition: normalizedPosition(player.natural_position || player.position),
        tacticalSlot,
        positionFit: positionFit(player, tacticalSlot),
      });
    });
  }

  // This only applies to malformed payloads with more actors than formation
  // slots. It remains deterministic and never changes player data.
  for (const { player, index } of indexed) {
    if (resolved.has(index)) continue;
    const tacticalSlot = normalizedPosition(player.position) || inferredCategory(player);
    resolved.set(index, {
      player,
      naturalPosition: normalizedPosition(player.natural_position || player.position),
      tacticalSlot,
      positionFit: positionFit(player, tacticalSlot),
    });
  }

  return players.map((_, index) => resolved.get(index)!).filter(Boolean);
}

export function isOutOfPosition(fit?: PositionFit): boolean {
  return fit === 'Acceptable' || fit === 'Emergency';
}
