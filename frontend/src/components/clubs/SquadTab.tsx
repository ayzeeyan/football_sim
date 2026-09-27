import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { Club, ClubHistoryResponse, ClubProfile, ClubTransferActivity, Fixture, Player } from '../../types';
import { fetchClubFixtures, fetchClubHistory, fetchClubProfile, fetchClubSquad, fetchClubTransfers, fetchClubXi } from '../../services/api';
import { Search, ArrowUpDown, Shield, Sparkle, Eye, LayoutGrid, List, Columns, AlertTriangle, Flame, Clock, UserX, Landmark, Calendar, Trophy } from 'lucide-react';
import { soundManager } from '../../audio/webAudio';
import { LEAGUES_5 } from '../../lib/constants';
import { sortBy } from '../../lib/sort';
import { ovrTone, positionTone } from '../../lib/constants';
import { Card, ClubCrest, EmptyState, ErrorState, FormPips, LoadingState, PanelHeader } from '../ui/ui';
import { cx, formatEUR, formatMillions, loyaltyLabel } from '../../lib/format';
import { prettyCompetitionName, usePlayerSheet } from '../clubs/PlayerSheet';
import { FormationPitch } from '../matches/FormationPitch';
import { ClubIdentityPanel } from './ClubIdentityPanel';

function formatWageBill(squad: Player[]): string {
  const annual = squad.reduce((sum, p) => sum + (p.wage_eur || 0) * 52, 0);
  if (annual >= 1_000_000_000) return `€${(annual / 1_000_000_000).toFixed(2)}B`;
  return `€${(annual / 1_000_000).toFixed(1)}M`;
}

const CLUB_PROFILE_TABS = ['overview', 'squad', 'fixtures', 'transfers', 'finances', 'history'] as const;
type ClubProfileTab = (typeof CLUB_PROFILE_TABS)[number];

/** Single fatigue cutoff shared by status counts, filters, and row styling. */
export const SQUAD_FATIGUE_FITNESS = 60;
/** Morale below this counts as unhappy in filters and row styling. */
export const SQUAD_UNHAPPY_MORALE = 55;

export function isSquadFatigued(player: Player): boolean {
  return typeof player.fitness === 'number' && player.fitness < SQUAD_FATIGUE_FITNESS;
}

export function isSquadInForm(player: Player): boolean {
  return (player.form ?? 0) >= 4 || player.form_band === 'Excellent';
}

export function isSquadUnhappy(player: Player): boolean {
  return typeof player.morale === 'number' && player.morale < SQUAD_UNHAPPY_MORALE;
}

export function isSquadExpiring(player: Player): boolean {
  return player.contract_years <= 1;
}

export function isSquadUnavailable(player: Player): boolean {
  return (player.injured_matches ?? 0) > 0 || (player.suspended_matches ?? 0) > 0;
}

export type SquadStatusCounts = Record<SquadStatusFilter, number>;

export function squadStatusCounts(squad: Player[]): SquadStatusCounts {
  return {
    all: squad.length,
    in_form: squad.filter(isSquadInForm).length,
    fatigued: squad.filter(isSquadFatigued).length,
    unhappy: squad.filter(isSquadUnhappy).length,
    expiring: squad.filter(isSquadExpiring).length,
    unavailable: squad.filter(isSquadUnavailable).length,
  };
}

export function passesSquadStatus(player: Player, filter: SquadStatusFilter): boolean {
  switch (filter) {
    case 'all':
      return true;
    case 'in_form':
      return isSquadInForm(player);
    case 'fatigued':
      return isSquadFatigued(player);
    case 'unhappy':
      return isSquadUnhappy(player);
    case 'expiring':
      return isSquadExpiring(player);
    case 'unavailable':
      return isSquadUnavailable(player);
    default:
      return true;
  }
}

const FIXTURE_COMP_FILTERS = [
  { id: 'all', label: 'All' },
  { id: 'league', label: 'League' },
  { id: 'cup', label: 'Domestic Cup' },
  { id: 'champions-league', label: 'UCL' },
  { id: 'europa-league', label: 'UEL' },
  { id: 'conference-league', label: 'UECL' },
] as const;

const LEAGUE_COMP_IDS = new Set(['premier-league', 'la-liga', 'bundesliga', 'serie-a', 'ligue-1', 'super-league']);
const UEFA_COMP_IDS = new Set(['champions-league', 'ucl', 'europa-league', 'conference-league', 'super-cup']);

/** Fixture competition filter; cup = anything that is not a domestic league or UEFA tie. */
export function matchesFixtureFilter(fixture: Fixture, filter: string, league: string): boolean {
  const comp = (fixture.competition || '').toLowerCase();
  if (filter === 'all') return true;
  if (filter === 'champions-league') return comp === 'champions-league' || comp === 'ucl';
  if (filter === 'europa-league') return comp === 'europa-league';
  if (filter === 'conference-league') return comp === 'conference-league';
  if (filter === 'cup') return !LEAGUE_COMP_IDS.has(comp) && !UEFA_COMP_IDS.has(comp) && prettyCompetitionName(comp) !== league;
  if (filter === 'league') return LEAGUE_COMP_IDS.has(comp) || prettyCompetitionName(comp) === league;
  return true;
}

interface SquadTabProps {
  clubs: Club[];
  initialClubId?: string;
  onWatchClub: (club: Club) => void;
  onWatchFixture?: (fixture: Fixture, intent?: 'watch' | 'visual' | 'quick' | 'result') => void;
}

export type SquadViewMode = 'pitch' | 'table' | 'split';
export type SquadStatusFilter = 'all' | 'in_form' | 'fatigued' | 'unhappy' | 'expiring' | 'unavailable';

export const SquadTab: React.FC<SquadTabProps> = ({ clubs, initialClubId, onWatchClub, onWatchFixture }) => {
  const { openPlayer } = usePlayerSheet();
  const [selectedLeague, setSelectedLeague] = useState<string>('All clubs');
  const [selectedClub, setSelectedClub] = useState<Club | null>(null);
  const initialClubRef = useRef(initialClubId);

  const [squad, setSquad] = useState<Player[]>([]);
  const [xi, setXi] = useState<Player[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [posGroup, setPosGroup] = useState<'All' | 'GK' | 'DEF' | 'MID' | 'FWD'>('All');
  const [viewMode, setViewMode] = useState<SquadViewMode>('split');
  const [statusFilter, setStatusFilter] = useState<SquadStatusFilter>('all');
  const [sortCol, setSortCol] = useState<keyof Player>('ovr');
  const [sortAsc, setSortAsc] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [reloadNonce, setReloadNonce] = useState(0);
  const [profileTab, setProfileTab] = useState<ClubProfileTab>('overview');
  const [profile, setProfile] = useState<ClubProfile | null>(null);
  const [clubFixtures, setClubFixtures] = useState<Fixture[]>([]);
  const [clubTransfers, setClubTransfers] = useState<ClubTransferActivity | null>(null);
  const [clubHistory, setClubHistory] = useState<ClubHistoryResponse | null>(null);
  const [fixtureFilter, setFixtureFilter] = useState<(typeof FIXTURE_COMP_FILTERS)[number]['id']>('all');
  const [fixtureSection, setFixtureSection] = useState<'previous' | 'upcoming' | 'all'>('upcoming');

  const clubsInLeague = useMemo(
    () => (selectedLeague === 'All clubs' ? clubs : clubs.filter((c) => c.league === selectedLeague)),
    [clubs, selectedLeague],
  );

  // Seed selection once: honour deep-linked club first, else first available club.
  // Never clobber a club the user (or deep link) has already chosen.
  useEffect(() => {
    if (selectedClub || clubs.length === 0) return;
    const wanted = initialClubRef.current;
    const linked = wanted ? clubs.find((c) => c.club_id === wanted) : undefined;
    const seed = linked ?? clubs[0];
    setSelectedClub(seed);
    if (linked) setSelectedLeague(linked.league);
  }, [clubs, selectedClub]);

  const selectLeague = (league: string) => {
    soundManager.playClick();
    setSelectedLeague(league);
    const list = league === 'All clubs' ? clubs : clubs.filter((c) => c.league === league);
    if (list.length > 0) setSelectedClub(list[0]);
  };

  useEffect(() => {
    if (!selectedClub) return;
    let cancelled = false;
    setLoading(true);
    setLoadError(null);
    Promise.all([
      fetchClubSquad(selectedClub.club_id),
      fetchClubXi(selectedClub.club_id),
      fetchClubProfile(selectedClub.club_id),
      fetchClubFixtures(selectedClub.club_id),
      fetchClubTransfers(selectedClub.club_id),
      fetchClubHistory(selectedClub.club_id),
    ])
      .then(([data, eleven, clubProfile, fixtures, transfers, history]) => {
        if (!cancelled) {
          setSquad(data);
          setXi(eleven);
          setProfile(clubProfile);
          setClubFixtures(fixtures);
          setClubTransfers(transfers);
          setClubHistory(history);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : 'Could not load this club’s data.');
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [selectedClub, reloadNonce]);

  const handleSort = (col: keyof Player) => {
    soundManager.playClick();
    if (sortCol === col) setSortAsc((v) => !v);
    else {
      setSortCol(col);
      setSortAsc(false);
    }
  };

  const statusCounts = useMemo(() => squadStatusCounts(squad), [squad]);

  const aggregates = useMemo(() => {
    const avgOvr = squad.length ? Math.round(squad.reduce((s, p) => s + p.ovr, 0) / squad.length) : 0;
    const avgAge = squad.length ? (squad.reduce((s, p) => s + p.age, 0) / squad.length).toFixed(1) : '—';
    const totalValue = squad.reduce((s, p) => s + (p.market_value_eur || 0), 0);
    const byLine = {
      GK: squad.filter((p) => p.category === 'GK').length,
      DEF: squad.filter((p) => p.category === 'DEF').length,
      MID: squad.filter((p) => p.category === 'MID').length,
      FWD: squad.filter((p) => p.category === 'FWD').length,
    };
    return { avgOvr, avgAge, totalValue, byLine };
  }, [squad]);
  const { avgOvr, avgAge, totalValue, byLine } = aggregates;

  const filteredSquad = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    const filtered = squad.filter((p) => {
      if (posGroup !== 'All' && p.category !== posGroup) return false;
      if (!passesSquadStatus(p, statusFilter)) return false;
      if (q === '') return true;
      return (
        p.full_name.toLowerCase().includes(q) ||
        p.position.toLowerCase().includes(q) ||
        (p.squad_role || '').toLowerCase().includes(q)
      );
    });
    return sortBy(filtered, sortCol, sortAsc);
  }, [squad, searchQuery, posGroup, statusFilter, sortCol, sortAsc]);

  const bench = useMemo(() => {
    const xiIds = new Set(xi.map((p) => p.player_id));
    return filteredSquad.filter((p) => !xiIds.has(p.player_id));
  }, [filteredSquad, xi]);

  const sortIndicator = (col: keyof Player) => (sortCol === col ? (sortAsc ? ' ↑' : ' ↓') : '');

  const renderBenchCard = (
    <Card>
      <div className="flex items-center justify-between gap-2 pb-3 border-b border-line mb-3">
        <div>
          <h3 className="font-display text-[16px] font-semibold text-bone">Bench & Reserves</h3>
          <p className="text-[12px] text-sage">Squad depth ready for tactical rotation</p>
        </div>
        <span className="font-mono text-[12px] text-sage">{bench.length} players</span>
      </div>
      {bench.length === 0 ? (
        <EmptyState message="No bench players match the active filter." className="py-6" />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5 max-h-[520px] overflow-y-auto pr-1">
          {bench.map((player) => {
            const isWk = player.universe_wonderkid;
            const injured = (player.injured_matches ?? 0) > 0;
            const suspended = (player.suspended_matches ?? 0) > 0;
            const unhappy = isSquadUnhappy(player);
            const inForm = isSquadInForm(player);
            const fatigued = isSquadFatigued(player);
            return (
              <button
                key={player.player_id}
                type="button"
                onClick={() => {
                  soundManager.playClick();
                  openPlayer(player.player_id);
                }}
                className={cx(
                  'p-2.5 rounded-xl border text-left flex items-center justify-between gap-2.5 transition-colors hover:bg-cardHover',
                  injured || suspended
                    ? 'bg-ember/[0.08] border-ember/30'
                    : inForm
                      ? 'bg-brass/[0.08] border-brass/40'
                      : unhappy
                        ? 'bg-[#D89A84]/[0.08] border-[#D89A84]/30'
                        : 'bg-ink/40 border-line',
                )}
              >
                <div className="flex items-center gap-2 min-w-0">
                  <span className={cx('px-1.5 py-0.5 rounded text-[10px] font-mono font-bold shrink-0', positionTone(player.category))}>
                    {player.position}
                  </span>
                  <div className="min-w-0">
                    <div className="text-[13px] font-semibold text-bone truncate flex items-center gap-1">
                      <span className="truncate">{player.full_name}</span>
                      {isWk && <Sparkle size={11} className="text-brass shrink-0" />}
                    </div>
                    <div className="text-[11px] text-sage font-mono flex items-center gap-1 mt-0.5">
                      <span>{player.squad_role || 'Depth'}</span>
                      <span>·</span>
                      <span className={fatigued ? 'text-amber-400 font-semibold' : 'text-sage'}>Fit {player.fitness ?? '—'}</span>
                      <span>·</span>
                      <span className={unhappy ? 'text-ember font-semibold' : 'text-sage'}>Mor {player.morale ?? '—'}</span>
                    </div>
                  </div>
                </div>
                <div className="text-right shrink-0">
                  <span className={cx('font-display font-bold text-[14px]', ovrTone(player.ovr))}>{player.ovr}</span>
                  <span className="block font-mono text-[10.5px] text-sage mt-0.5">{player.formatted_value}</span>
                </div>
              </button>
            );
          })}
        </div>
      )}
    </Card>
  );

  const renderPitchCard = (
    <Card>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="eyebrow">Likely starting XI</p>
          <p className="mt-1 text-[12px] text-sage">
            Stored tactical positions drive the pitch. Generic central roles are separated without changing player data.
          </p>
        </div>
        <span className="rounded-md border border-pitchtone/35 bg-pitchtone/10 px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-[0.08em] text-pitchtone">
          {selectedClub?.manager?.formation ?? xi[0]?.formation ?? 'Formation-aware'}
        </span>
      </div>
      <div className="mt-4">
        <FormationPitch players={xi} formation={selectedClub?.manager?.formation ?? xi[0]?.formation} onPlayerClick={(player) => openPlayer(player.player_id)} />
      </div>
    </Card>
  );

  const sortableColumns: Array<[keyof Player, string]> = [
    ['category', 'Pos'],
    ['full_name', 'Player'],
    ['ovr', 'OVR'],
    ['age', 'Age'],
    ['squad_role', 'Role'],
    ['morale', 'Morale'],
    ['form', 'Form'],
    ['fitness', 'Fit'],
    ['sharpness', 'Sharp'],
    ['market_value_eur', 'Valuation'],
    ['starts', 'Starts'],
    ['appearances', 'Apps'],
    ['minutes', 'Min'],
    ['goals', 'G'],
    ['assists', 'A'],
  ];

  const renderSortHeader = (col: keyof Player, label: string, extraClass = '') => (
    <th
      className={cx('py-3 px-3 font-semibold cursor-pointer hover:text-bone whitespace-nowrap sticky top-0 z-10', extraClass)}
      onClick={() => handleSort(col)}
      aria-sort={sortCol === col ? (sortAsc ? 'ascending' : 'descending') : 'none'}
    >
      <span className="flex items-center gap-1">
        <span>
          {label}
          {sortIndicator(col)}
        </span>
        <ArrowUpDown size={12} aria-hidden="true" />
        {sortCol === col && (
          <span className="sr-only">{sortAsc ? 'sorted ascending' : 'sorted descending'}</span>
        )}
      </span>
    </th>
  );

  const renderTableCard = (
    <div className="panel-tight overflow-hidden">
      {loading ? (
        <LoadingState message={`Loading ${selectedClub?.short_name ?? 'club'} roster…`} />
      ) : filteredSquad.length === 0 ? (
        <EmptyState message="No players match that search. Clear the search to see the full squad." />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-[13px] min-w-[1180px]">
            <thead className="table-head">
              <tr>
                {sortableColumns.map(([col, label]) => renderSortHeader(col, label))}
                <th className="py-3 px-3 font-semibold whitespace-nowrap sticky top-0 z-10">All-time</th>
                {renderSortHeader('wage_eur', 'Wage/wk')}
                {renderSortHeader('contract_years', 'Contract')}
                <th className="py-3 px-4 font-semibold sticky top-0 z-10">Status</th>
                <th className="py-3 px-4 font-semibold sticky top-0 z-10">Avail.</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line/70">
              {filteredSquad.map((player) => {
                const isWk = player.universe_wonderkid;
                const isHeadline = player.player_source === 'headline';
                const injured = (player.injured_matches ?? 0) > 0;
                const suspended = (player.suspended_matches ?? 0) > 0;
                const unhappy = isSquadUnhappy(player);
                const excellentForm = isSquadInForm(player);
                const fatigued = isSquadFatigued(player);
                return (
                  <tr
                    key={player.player_id}
                    className={cx(
                      'hover:bg-cardLight/60 transition-colors cursor-pointer',
                      injured && 'bg-ember/[0.08]',
                      !injured && suspended && 'bg-ember/[0.05]',
                      !injured && !suspended && unhappy && 'bg-[#D89A84]/[0.06]',
                      !injured && excellentForm && 'bg-brass/[0.05]',
                      fatigued && 'opacity-80',
                      isWk && !injured && 'bg-brass/[0.04]',
                    )}
                    onClick={() => {
                      openPlayer(player.player_id);
                    }}
                  >
                    <td className="py-3 px-4 font-mono font-semibold">
                      <span className={cx('px-2 py-0.5 rounded-md text-[11px] font-semibold', positionTone(player.category))}>{player.position}</span>
                    </td>
                    <td className="py-3 px-3 font-semibold text-bone">
                      <span className="inline-flex items-center gap-1.5">
                        {player.full_name}
                        {player.is_captain && <span className="text-[10px] font-mono uppercase tracking-[0.08em] text-brass">C</span>}
                        {player.is_vice_captain && !player.is_captain && <span className="text-[10px] font-mono uppercase tracking-[0.08em] text-sage">VC</span>}
                        {player.homegrown && <span className="text-[10px] font-mono uppercase tracking-[0.08em] text-pitchtone">HG</span>}
                      </span>
                    </td>
                    <td className="py-3 px-3 font-bold font-mono text-[14px]">
                      <span className={ovrTone(player.ovr)}>{player.ovr}</span>
                    </td>
                    <td className="py-3 px-3 text-sage font-mono">{player.age}</td>
                    <td className="py-3 px-3 text-sage text-[12px]">{player.squad_role || '—'}</td>
                    <td className={cx('py-3 px-3 font-mono text-[12px]', unhappy ? 'text-ember' : 'text-bone/85')}>{player.morale ?? '—'}</td>
                    <td className={cx('py-3 px-3 font-mono text-[12px]', excellentForm ? 'text-brass font-semibold' : 'text-sage')}>{player.form_band || '—'}</td>
                    <td className={cx('py-3 px-3 font-mono text-[12px]', fatigued ? 'text-ember' : 'text-sage')}>{player.fitness ?? '—'}</td>
                    <td className="py-3 px-3 font-mono text-[12px] text-sage">{player.sharpness ?? '—'}</td>
                    <td className="py-3 px-3 font-mono text-[#A9CDBB] font-semibold whitespace-nowrap">{player.formatted_value}</td>
                    <td className="py-3 px-3 text-sage font-mono">{player.starts ?? '—'}</td>
                    <td className="py-3 px-3 text-sage font-mono">{player.appearances}</td>
                    <td className="py-3 px-3 text-sage font-mono">{player.minutes ?? '—'}</td>
                    <td className="py-3 px-3 text-bone font-semibold font-mono">{player.goals}</td>
                    <td className="py-3 px-3 text-bone/70 font-mono">{player.assists}</td>
                    <td className="py-3 px-3 font-mono text-[12px] whitespace-nowrap">
                      <span className="text-bone font-semibold">{player.career_goals ?? 0} G</span>
                      <span className="text-sage"> · {player.career_assists ?? 0} A</span>
                    </td>
                    <td className="py-3 px-3 text-bone/85 font-mono text-[12.5px] font-semibold whitespace-nowrap">{player.formatted_wage}</td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      <span className="block font-mono font-bold text-[13px] text-bone">
                        {player.contract_years} yr{player.contract_years === 1 ? '' : 's'} left
                        {player.contract_years <= 1 && (
                          <span className="ml-1.5 text-ember uppercase tracking-[0.06em] text-[10px]">final</span>
                        )}
                      </span>
                      <span
                        className="mt-0.5 flex items-center gap-1 font-mono text-[11px] text-sage"
                        title={`Loyalty ${player.loyalty}/100 — ${loyaltyLabel(player.loyalty)}`}
                      >
                        <span
                          className={cx(
                            'w-1.5 h-1.5 rounded-full inline-block',
                            player.loyalty >= 75 ? 'bg-brass' : player.loyalty >= 55 ? 'bg-sage' : 'bg-ember',
                          )}
                        />
                        Loyalty {player.loyalty} · {loyaltyLabel(player.loyalty)}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      {player.on_loan ? (
                        <span className="flex flex-col gap-1">
                          <span className="px-2 py-0.5 rounded-md bg-[#8AB4C8]/10 text-[#A9CBDD] border border-[#8AB4C8]/30 font-semibold text-[11px] w-fit uppercase tracking-[0.06em]">
                            Loan
                          </span>
                          {(player.loan_buy_clause_eur ?? 0) > 0 && (
                            <span className="font-mono text-[10px] text-sage" title="Permanent-transfer fee agreed with the parent club">
                              Buy {player.formatted_buy_clause ?? 'clause agreed'}
                            </span>
                          )}
                        </span>
                      ) : isWk ? (
                        <span className="px-2 py-0.5 rounded-md bg-brass/10 text-brass border border-brass/40 font-semibold text-[11px] flex items-center gap-1 w-fit uppercase tracking-[0.06em]">
                          <Sparkle size={11} /> U-17 prodigy
                        </span>
                      ) : player.player_source === 'academy' ? (
                        <span className="px-2 py-0.5 rounded-md bg-pitchtone/10 text-[#A9CDBB] border border-pitchtone/30 font-semibold text-[11px] w-fit uppercase tracking-[0.06em]">
                          Academy
                        </span>
                      ) : isHeadline ? (
                        <span className="px-2 py-0.5 rounded-md bg-[#8AB4C8]/10 text-[#A9CBDD] border border-[#8AB4C8]/30 font-semibold text-[11px] flex items-center gap-1 w-fit uppercase tracking-[0.06em]">
                          <Shield size={11} /> Headline
                        </span>
                      ) : (
                        <span className="px-2 py-0.5 rounded-md bg-cardLight text-sage border border-line font-mono text-[11px]">
                          Depth
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-4 font-mono text-[11px]">
                      {(player.injured_matches ?? 0) > 0 || (player.suspended_matches ?? 0) > 0 ? (
                        <span className="text-ember font-semibold">{player.availability ?? `Out ${player.injured_matches || player.suspended_matches}`}</span>
                      ) : (
                        <span className="text-sage">Available</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );

  return (
    <div className="space-y-4">
      <Card>
        <PanelHeader
          kicker="Club directory · Spectator view"
          title="Clubs"
          subtitle="Inspect an AI-controlled club. Formation, finances, and transfers are club decisions — this stand is for watching."
        />
        <div className="flex flex-wrap items-center justify-between gap-3 mt-4">
          <div className="flex gap-1.5 overflow-x-auto" role="tablist" aria-label="League filter">
            {['All clubs', ...LEAGUES_5].map((lg) => (
              <button
                key={lg}
                onClick={() => selectLeague(lg)}
                aria-pressed={selectedLeague === lg}
                className={cx(
                  'px-3.5 py-2 rounded-lg text-[13px] font-semibold transition-colors whitespace-nowrap border',
                  selectedLeague === lg
                    ? 'bg-brass text-ink border-brass'
                    : 'bg-cardLight text-sage hover:text-bone border-line',
                )}
              >
                {lg}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-3">
            <select
              value={selectedClub?.club_id || ''}
              onChange={(e) => {
                soundManager.playClick();
                const found = clubs.find((c) => c.club_id === e.target.value);
                if (found) setSelectedClub(found);
              }}
              aria-label="Select club"
              className="field px-3.5 py-2 font-semibold"
            >
              {selectedLeague === 'All clubs'
                ? LEAGUES_5.map((lg) => {
                    const group = clubs.filter((c) => c.league === lg);
                    if (group.length === 0) return null;
                    return (
                      <optgroup key={lg} label={lg}>
                        {group.map((c) => (
                          <option key={c.club_id} value={c.club_id}>
                            {c.club_name} ({c.short_name})
                          </option>
                        ))}
                      </optgroup>
                    );
                  })
                : clubsInLeague.map((c) => (
                    <option key={c.club_id} value={c.club_id}>
                      {c.club_name} ({c.short_name})
                    </option>
                  ))}
            </select>
          </div>
        </div>

        {profileTab === 'squad' && (
          <>
            <div className="flex flex-wrap items-center gap-3 mt-4">
              <div className="flex gap-1" role="group" aria-label="Position group">
                {(['All', 'GK', 'DEF', 'MID', 'FWD'] as const).map((g) => (
                  <button
                    key={g}
                    type="button"
                    onClick={() => {
                      soundManager.playClick();
                      setPosGroup(g);
                    }}
                    aria-pressed={posGroup === g}
                    className={cx(
                      'px-2.5 py-1.5 text-[12px] font-semibold border',
                      posGroup === g ? 'bg-bone text-ink border-bone' : 'bg-cardLight text-sage border-line hover:text-bone',
                    )}
                  >
                    {g}
                  </button>
                ))}
              </div>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-sage" size={14} aria-hidden="true" />
                <label className="sr-only" htmlFor="squad-search">
                  Search name or position
                </label>
                <input
                  id="squad-search"
                  name="squad-search"
                  type="search"
                  autoComplete="off"
                  spellCheck={false}
                  placeholder="Search name or position…"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="field pl-9 pr-3.5 py-2 placeholder-sage/70 w-52"
                />
              </div>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-3 mt-4 pt-3.5 border-t border-line/60">
          <div className="flex items-center gap-1.5 p-1 bg-ink/60 rounded-xl border border-line" role="group" aria-label="View mode">
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setViewMode('pitch');
              }}
              aria-pressed={viewMode === 'pitch'}
              className={cx(
                'px-3 py-1.5 rounded-lg text-[12px] font-semibold flex items-center gap-1.5 transition-colors',
                viewMode === 'pitch' ? 'bg-brass text-ink shadow-sm' : 'text-sage hover:text-bone',
              )}
            >
              <LayoutGrid size={13} /> Pitch
            </button>
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setViewMode('table');
              }}
              aria-pressed={viewMode === 'table'}
              className={cx(
                'px-3 py-1.5 rounded-lg text-[12px] font-semibold flex items-center gap-1.5 transition-colors',
                viewMode === 'table' ? 'bg-brass text-ink shadow-sm' : 'text-sage hover:text-bone',
              )}
            >
              <List size={13} /> Table
            </button>
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setViewMode('split');
              }}
              aria-pressed={viewMode === 'split'}
              className={cx(
                'px-3 py-1.5 rounded-lg text-[12px] font-semibold flex items-center gap-1.5 transition-colors',
                viewMode === 'split' ? 'bg-brass text-ink shadow-sm' : 'text-sage hover:text-bone',
              )}
            >
              <Columns size={13} /> Split Screen
            </button>
          </div>

          <div className="flex flex-wrap items-center gap-1.5" role="group" aria-label="Status filter">
            <span className="text-[11px] font-mono text-sage uppercase tracking-wider mr-1">Status:</span>
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setStatusFilter('all');
              }}
              aria-pressed={statusFilter === 'all'}
              className={cx(
                'px-2.5 py-1 rounded-lg text-[12px] font-mono font-semibold border transition-colors',
                statusFilter === 'all' ? 'bg-bone text-ink border-bone' : 'bg-cardLight text-sage border-line hover:text-bone',
              )}
            >
              All ({statusCounts.all})
            </button>
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setStatusFilter('in_form');
              }}
              aria-pressed={statusFilter === 'in_form'}
              className={cx(
                'px-2.5 py-1 rounded-lg text-[12px] font-mono font-semibold border transition-colors flex items-center gap-1',
                statusFilter === 'in_form' ? 'bg-brass text-ink border-brass' : 'bg-cardLight text-brass border-line hover:border-brass/50',
              )}
            >
              <Flame size={11} /> In Form ({statusCounts.in_form})
            </button>
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setStatusFilter('fatigued');
              }}
              aria-pressed={statusFilter === 'fatigued'}
              className={cx(
                'px-2.5 py-1 rounded-lg text-[12px] font-mono font-semibold border transition-colors flex items-center gap-1',
                statusFilter === 'fatigued' ? 'bg-amber-500 text-ink border-amber-500' : 'bg-cardLight text-amber-400 border-line hover:border-amber-500/50',
              )}
            >
              <AlertTriangle size={11} /> Fatigued ({statusCounts.fatigued})
            </button>
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setStatusFilter('unhappy');
              }}
              aria-pressed={statusFilter === 'unhappy'}
              className={cx(
                'px-2.5 py-1 rounded-lg text-[12px] font-mono font-semibold border transition-colors flex items-center gap-1',
                statusFilter === 'unhappy' ? 'bg-ember text-ink border-ember' : 'bg-cardLight text-ember border-line hover:border-ember/50',
              )}
            >
              <UserX size={11} /> Unhappy ({statusCounts.unhappy})
            </button>
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setStatusFilter('expiring');
              }}
              aria-pressed={statusFilter === 'expiring'}
              className={cx(
                'px-2.5 py-1 rounded-lg text-[12px] font-mono font-semibold border transition-colors flex items-center gap-1',
                statusFilter === 'expiring' ? 'bg-[#D89A84] text-ink border-[#D89A84]' : 'bg-cardLight text-[#D89A84] border-line hover:border-[#D89A84]/50',
              )}
            >
              <Clock size={11} /> Expiring ({statusCounts.expiring})
            </button>
            <button
              type="button"
              onClick={() => {
                soundManager.playClick();
                setStatusFilter('unavailable');
              }}
              aria-pressed={statusFilter === 'unavailable'}
              className={cx(
                'px-2.5 py-1 rounded-lg text-[12px] font-mono font-semibold border transition-colors flex items-center gap-1',
                statusFilter === 'unavailable' ? 'bg-ember text-bone border-ember' : 'bg-cardLight text-sage border-line hover:text-ember',
              )}
            >
              Out ({statusCounts.unavailable})
            </button>
          </div>
        </div>
          </>
        )}
      </Card>

      {selectedClub && (
        <Card className="overflow-hidden p-0">
          <div className="flex flex-wrap items-center justify-between gap-4 p-5">
          <div className="flex items-center gap-3.5">
            <ClubCrest club={selectedClub} size={72} />
            <div>
              <div className="font-display text-[26px] font-semibold text-bone tracking-tight uppercase">
                {selectedClub.club_name}
              </div>
              <div className="text-[13px] text-sage font-normal mt-0.5">
                {selectedClub.league}
              </div>
              <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 font-mono text-[12px] text-sage">
                <span>Manager: <strong className="text-bone">{selectedClub.manager?.name ?? '—'}</strong></span>
                <span>League position: <strong className="text-bone">{profile?.league_position ? `#${profile.league_position}` : '—'}</strong></span>
                <span className="flex items-center gap-2">Form: <FormPips form={profile?.form ?? selectedClub.form ?? []} size="sm" /></span>
                <span>Reputation: <strong className="text-bone">{selectedClub.identity?.reputation ?? selectedClub.reputation ?? '—'}</strong></span>
                <span>Squad rating: <strong className="text-bone">{profile?.squad_avg_ovr ?? avgOvr}</strong></span>
              </div>
              {(profile?.competitions?.length ?? 0) > 0 && (
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {profile!.competitions.map((comp) => (
                    <span key={comp.id} className="px-2 py-0.5 rounded-md border border-brass/35 bg-brass/[0.08] text-[11px] font-semibold text-brass">
                      {comp.name}
                    </span>
                  ))}
                </div>
              )}
              <div className="mt-2 flex flex-wrap items-center gap-2.5">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-brass/[0.08] border border-brass/40" title="Every weekly wage × 52, added up across the squad">
                  <span className="eyebrow !text-brass">Wage bill</span>
                  <strong className="text-bone font-mono text-[14px]">{formatWageBill(squad)}/yr</strong>
                </div>

                {(() => {
                  const morale = selectedClub.morale;
                  if (typeof morale !== 'number') return null;
                  const moraleLabel =
                    morale >= 85 ? 'Euphoric' :
                    morale >= 75 ? 'High' :
                    morale >= 60 ? 'Stable' :
                    morale >= 45 ? 'Fragile' : 'Crisis';
                  const moraleTone =
                    morale >= 85 ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/35' :
                    morale >= 75 ? 'text-pitchtone bg-pitchtone/10 border-pitchtone/35' :
                    morale >= 60 ? 'text-brass bg-brass/10 border-brass/35' :
                    morale >= 45 ? 'text-amber-400 bg-amber-500/10 border-amber-500/35' :
                    'text-ember bg-ember/10 border-ember/35';
                  return (
                    <div className={cx('inline-flex items-center gap-2 px-3 py-1.5 rounded-lg border', moraleTone)} title="Club dressing room atmosphere influenced by match streaks">
                      <span className="eyebrow !text-current">Morale</span>
                      <strong className="font-mono text-[14px]">{morale}/100 · {moraleLabel}</strong>
                      <div className="w-16 h-1.5 rounded-full bg-ink/60 border border-line overflow-hidden ml-1">
                        <div
                          className={cx('h-full transition-all duration-300', morale >= 75 ? 'bg-pitchtone' : morale >= 50 ? 'bg-brass' : 'bg-ember')}
                          style={{ width: `${morale}%` }}
                        />
                      </div>
                    </div>
                  );
                })()}
              </div>
              {selectedClub.manager && (
                <div className="mt-2 space-y-1">
                  <div className="text-[13px] text-sage font-normal">
                    Managed by <strong className="text-bone">{selectedClub.manager.name}</strong>
                    {' · '}{selectedClub.manager.tactic}
                    {' · '}{selectedClub.manager.focus}
                    {' · '}<strong className="text-brass font-mono">{selectedClub.formatted_transfer_warchest ?? selectedClub.manager.formatted_budget}</strong> warchest
                    {typeof selectedClub.finances?.balance === 'number' && (
                      <span className="text-sage"> · <strong className="text-bone font-mono">{formatEUR(selectedClub.finances.balance)}</strong> balance</span>
                    )}
                  </div>
                  <div className="flex flex-wrap items-center gap-2 text-[11px] font-mono">
                    <span className={cx(
                      'px-2 py-0.5 border',
                      selectedClub.manager.job_security === 'Hot Seat'
                        ? 'border-ember/45 text-ember bg-ember/10'
                        : selectedClub.manager.job_security === 'Under Pressure'
                          ? 'border-brass/45 text-brass bg-brass/10'
                          : 'border-pitchtone/35 text-pitchtone bg-pitchtone/10',
                    )}>
                      {selectedClub.manager.job_security || 'Safe'}
                    </span>
                    {selectedClub.manager.appointed_season && (
                      <span className="text-sage">
                        appointed {selectedClub.manager.appointed_season}
                        {selectedClub.manager.appointed_matchweek ? ` · MW ${selectedClub.manager.appointed_matchweek}` : ''}
                      </span>
                    )}
                    {(selectedClub.manager.history?.length ?? 0) > 0 && (
                      <span className="text-sage">{selectedClub.manager.history!.length} previous manager spell{selectedClub.manager.history!.length === 1 ? '' : 's'} recorded</span>
                    )}
                  </div>
                </div>
              )}
              <div className="flex flex-wrap gap-2 mt-2 font-mono text-[11px] text-sage">
                <span className="px-2 py-1 rounded-md border border-line bg-ink/40">Avg {avgOvr} OVR</span>
                <span className="px-2 py-1 rounded-md border border-line bg-ink/40">Avg age {avgAge}</span>
                <span className="px-2 py-1 rounded-md border border-line bg-ink/40">{formatMillions(totalValue)}</span>
                <span className="px-2 py-1 rounded-md border border-line bg-ink/40">
                  {byLine.GK} GK · {byLine.DEF} DEF · {byLine.MID} MID · {byLine.FWD} FWD
                </span>
              </div>
            </div>
          </div>

          <button
            onClick={() => {
              soundManager.playClick();
              onWatchClub(selectedClub);
            }}
            className="px-5 py-2.5 bg-brass hover:bg-[#D4AF4D] text-ink rounded-xl text-[13px] font-bold transition-colors inline-flex items-center gap-2"
          >
            <Eye size={15} /> Open next fixture in Match Centre
          </button>
          </div>
          <div className="section-tabs px-2" role="tablist" aria-label="Club profile sections">
            {CLUB_PROFILE_TABS.map((tab) => (
              <button
                key={tab}
                type="button"
                role="tab"
                aria-selected={profileTab === tab}
                onClick={() => {
                  soundManager.playClick();
                  setProfileTab(tab);
                }}
                className={cx('section-tab uppercase tracking-[0.08em]', profileTab === tab ? 'section-tab-active' : '')}
              >
                {tab}
              </button>
            ))}
          </div>
        </Card>
      )}

      {loadError && !loading && (
        <Card>
          <ErrorState
            message={loadError}
            onRetry={() => {
              soundManager.playClick();
              setReloadNonce((n) => n + 1);
            }}
          />
        </Card>
      )}

      {selectedClub && profileTab === 'overview' && (
        <ClubOverviewPanel
          club={selectedClub}
          profile={profile}
          squad={squad}
          onOpenPlayer={(id) => openPlayer(id)}
          onWatchFixture={onWatchFixture}
        />
      )}

      {selectedClub && profileTab === 'fixtures' && (
        <ClubFixturesPanel
          club={selectedClub}
          fixtures={clubFixtures}
          filter={fixtureFilter}
          section={fixtureSection}
          onFilter={setFixtureFilter}
          onSection={setFixtureSection}
          onWatchFixture={onWatchFixture}
        />
      )}

      {selectedClub && profileTab === 'transfers' && (
        <ClubTransfersPanel activity={clubTransfers} onOpenPlayer={(id) => openPlayer(id)} />
      )}

      {selectedClub && profileTab === 'finances' && (
        <ClubFinancesPanel club={selectedClub} squad={squad} />
      )}

      {selectedClub && profileTab === 'history' && (
        <ClubHistoryPanel history={clubHistory} />
      )}

      {/* Dynamic View Mode Content */}
      {profileTab === 'squad' && viewMode === 'pitch' && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 items-start">
          <div className="lg:col-span-8">
            {xi.length > 0 && renderPitchCard}
          </div>
          <div className="lg:col-span-4">
            {renderBenchCard}
          </div>
        </div>
      )}

      {profileTab === 'squad' && viewMode === 'table' && renderTableCard}

      {profileTab === 'squad' && viewMode === 'split' && (
        <div className="grid grid-cols-1 xl:grid-cols-12 gap-5 items-start">
          <div className="xl:col-span-6 space-y-4">
            {xi.length > 0 && renderPitchCard}
          </div>
          <div className="xl:col-span-6 space-y-4">
            {renderBenchCard}
            <div className="space-y-2">
              <div className="flex items-center justify-between px-1">
                <span className="eyebrow">Full squad roster</span>
                <span className="text-[11px] font-mono text-sage">{filteredSquad.length} players shown</span>
              </div>
              {renderTableCard}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const ClubOverviewPanel: React.FC<{
  club: Club;
  profile: ClubProfile | null;
  squad: Player[];
  onOpenPlayer: (id: string) => void;
  onWatchFixture?: SquadTabProps['onWatchFixture'];
}> = ({ club, profile, squad, onOpenPlayer, onWatchFixture }) => {
  const avgAge = squad.length ? (squad.reduce((s, p) => s + p.age, 0) / squad.length).toFixed(1) : '—';
  const avgMorale = squad.length ? Math.round(squad.reduce((s, p) => s + (p.morale ?? 0), 0) / squad.length) : 0;
  return (
    <div className="space-y-4">
      {(profile?.storylines?.length ?? 0) > 0 && (
        <Card>
          <p className="eyebrow">Current storylines</p>
          <ul className="mt-2 space-y-1.5">
            {profile!.storylines.map((line) => (
              <li key={line} className="text-[13px] text-bone">{line}</li>
            ))}
          </ul>
        </Card>
      )}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          ['League position', profile?.league_position ? `#${profile.league_position}` : '—'],
          ['Points', String(profile?.points ?? club.pts ?? 0)],
          ['Squad OVR', String(profile?.squad_avg_ovr ?? 0)],
          ['Average age', profile?.average_age ? profile.average_age.toFixed(1) : avgAge],
          ['Squad morale', String(profile?.squad_morale ?? avgMorale)],
          ['Formation', profile?.formation || club.manager?.formation || '—'],
          ['Record', `${club.w ?? 0}W ${club.d ?? 0}D ${club.l ?? 0}L`],
          ['Board', club.board_objective || '—'],
        ].map(([label, value]) => (
          <Card key={label}>
            <p className="eyebrow">{label}</p>
            <p className="mt-1 font-mono text-[16px] font-bold text-bone">{value}</p>
          </Card>
        ))}
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <p className="eyebrow">Previous result</p>
          <FixtureMini fixture={profile?.previous_result ?? null} club={club} onWatchFixture={onWatchFixture} />
        </Card>
        <Card>
          <p className="eyebrow">Next fixture</p>
          <FixtureMini fixture={profile?.next_fixture ?? null} club={club} onWatchFixture={onWatchFixture} />
        </Card>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <PlayerStatCard title="Top scorer" player={profile?.top_scorer} stat={profile?.top_scorer ? `${profile.top_scorer.goals} G` : '—'} onOpen={onOpenPlayer} />
        <PlayerStatCard title="Top assister" player={profile?.top_assister} stat={profile?.top_assister ? `${profile.top_assister.assists} A` : '—'} onOpen={onOpenPlayer} />
        <PlayerStatCard title="Best recent performer" player={profile?.best_recent} stat={profile?.best_recent?.form_band || '—'} onOpen={onOpenPlayer} />
      </div>
      {(profile?.injuries?.length ?? 0) > 0 && (
        <Card>
          <p className="eyebrow">Major injuries</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {profile!.injuries.map((p) => (
              <button key={p.player_id} type="button" className="px-2 py-1 rounded-md border border-ember/35 text-[12px] text-[#D89A84]" onClick={() => onOpenPlayer(p.player_id)}>
                {p.full_name} · {p.injured_matches} MW
              </button>
            ))}
          </div>
        </Card>
      )}
      <ClubIdentityPanel club={club} />
    </div>
  );
};

const PlayerStatCard: React.FC<{ title: string; player: Player | null | undefined; stat: string; onOpen: (id: string) => void }> = ({ title, player, stat, onOpen }) => (
  <Card>
    <p className="eyebrow">{title}</p>
    {player ? (
      <button type="button" className="mt-2 text-left" onClick={() => onOpen(player.player_id)}>
        <p className="font-semibold text-bone">{player.full_name}</p>
        <p className="font-mono text-[12px] text-sage">{stat}</p>
      </button>
    ) : (
      <p className="mt-2 text-sage text-[13px]">No recorded leader yet.</p>
    )}
  </Card>
);

const FixtureMini: React.FC<{
  fixture: Fixture | null;
  club: Club;
  onWatchFixture?: SquadTabProps['onWatchFixture'];
}> = ({ fixture, club, onWatchFixture }) => {
  if (!fixture) return <p className="mt-2 text-[13px] text-sage">No fixture logged.</p>;
  const home = fixture.home?.short_name || fixture.home_id;
  const away = fixture.away?.short_name || fixture.away_id;
  const score = fixture.status === 'finished' && fixture.home_goals != null ? `${fixture.home_goals}–${fixture.away_goals}` : 'vs';
  return (
    <button
      type="button"
      className="mt-2 w-full text-left"
      onClick={() => onWatchFixture?.(fixture, fixture.status === 'finished' ? 'result' : 'watch')}
    >
      <p className="font-semibold text-bone">{home} {score} {away}</p>
      <p className="font-mono text-[12px] text-sage">MW {fixture.matchweek} · {prettyCompetitionName(fixture.competition)}</p>
    </button>
  );
};

const ClubFixturesPanel: React.FC<{
  club: Club;
  fixtures: Fixture[];
  filter: (typeof FIXTURE_COMP_FILTERS)[number]['id'];
  section: 'previous' | 'upcoming' | 'all';
  onFilter: (id: (typeof FIXTURE_COMP_FILTERS)[number]['id']) => void;
  onSection: (section: 'previous' | 'upcoming' | 'all') => void;
  onWatchFixture?: SquadTabProps['onWatchFixture'];
}> = ({ club, fixtures, filter, section, onFilter, onSection, onWatchFixture }) => {
  const filtered = fixtures.filter((f) => matchesFixtureFilter(f, filter, club.league));
  const shown = filtered.filter((f) => {
    if (section === 'previous') return f.status === 'finished';
    if (section === 'upcoming') return f.status !== 'finished';
    return true;
  });
  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="eyebrow flex items-center gap-1.5"><Calendar size={13} /> Club schedule</p>
        <div className="flex flex-wrap gap-1">
          {(['previous', 'upcoming', 'all'] as const).map((s) => (
            <button key={s} type="button" className={cx('px-2 py-1 rounded-md text-[11px] font-semibold uppercase border', section === s ? 'bg-brass text-ink border-brass' : 'border-line text-sage')} onClick={() => onSection(s)}>
              {s === 'all' ? 'All fixtures' : s}
            </button>
          ))}
        </div>
      </div>
      <div className="mt-3 flex flex-wrap gap-1">
        {FIXTURE_COMP_FILTERS.map((item) => (
          <button key={item.id} type="button" className={cx('px-2 py-1 rounded-md text-[11px] font-semibold border', filter === item.id ? 'bg-cardLight text-bone border-brass/40' : 'border-line text-sage')} onClick={() => onFilter(item.id)}>
            {item.label}
          </button>
        ))}
      </div>
      <div className="mt-3 divide-y divide-line/60">
        {shown.length === 0 ? (
          <p className="py-6 text-center text-[13px] text-sage">No fixtures in this view.</p>
        ) : shown.map((fixture) => {
          const opponent = fixture.home_id === club.club_id ? fixture.away : fixture.home;
          const venue = fixture.home_id === club.club_id ? 'Home' : 'Away';
          const score = fixture.status === 'finished' && fixture.home_goals != null ? `${fixture.home_goals}–${fixture.away_goals}` : fixture.status;
          return (
            <button
              key={fixture.id || fixture.fixture_id}
              type="button"
              className="w-full flex items-center justify-between gap-3 py-2.5 text-left hover:bg-cardHover"
              onClick={() => onWatchFixture?.(fixture, fixture.status === 'finished' ? 'result' : 'watch')}
            >
              <div className="min-w-0">
                <p className="text-[13px] font-semibold text-bone truncate">{prettyCompetitionName(fixture.competition)} · {opponent?.club_name || opponent?.short_name || 'Opponent'}</p>
                <p className="font-mono text-[11px] text-sage">MW {fixture.matchweek} · {venue}</p>
              </div>
              <span className="font-mono text-[13px] text-brass shrink-0">{score}</span>
            </button>
          );
        })}
      </div>
    </Card>
  );
};

const ClubTransfersPanel: React.FC<{ activity: ClubTransferActivity | null; onOpenPlayer: (id: string) => void }> = ({ activity, onOpenPlayer }) => {
  const rows = [
    ['Arrivals', activity?.arrivals ?? []],
    ['Departures', activity?.departures ?? []],
  ] as const;
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-3">
        <Card><p className="eyebrow">Spent</p><p className="mt-1 font-mono text-[18px] font-bold text-bone">{formatEUR(activity?.spent)}</p></Card>
        <Card><p className="eyebrow">Received</p><p className="mt-1 font-mono text-[18px] font-bold text-bone">{formatEUR(activity?.received)}</p></Card>
        <Card><p className="eyebrow">Net spend</p><p className="mt-1 font-mono text-[18px] font-bold text-bone">{formatEUR(activity?.net_spend)}</p></Card>
      </div>
      {rows.map(([title, deals]) => (
        <Card key={title}>
          <p className="eyebrow">{title}</p>
          {deals.length === 0 ? (
            <p className="mt-2 text-[13px] text-sage">No recorded {title.toLowerCase()}.</p>
          ) : (
            <div className="mt-2 divide-y divide-line/60">
              {deals.map((deal) => (
                <button key={`${deal.player_id}-${deal.buyer_id}-${deal.seller_id}`} type="button" className="w-full flex justify-between py-2 text-left" onClick={() => onOpenPlayer(deal.player_id)}>
                  <span className="text-[13px] text-bone">{deal.player_name}</span>
                  <span className="font-mono text-[12px] text-sage">{deal.seller_short} → {deal.buyer_short} · {deal.formatted_fee}</span>
                </button>
              ))}
            </div>
          )}
        </Card>
      ))}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <p className="eyebrow">Loans in</p>
          {(activity?.loans_in?.length ?? 0) === 0 ? <p className="mt-2 text-[13px] text-sage">None.</p> : activity!.loans_in.map((row) => (
            <button key={row.player_id} type="button" className="block mt-2 text-[13px] text-bone" onClick={() => onOpenPlayer(row.player_id)}>{row.full_name}</button>
          ))}
        </Card>
        <Card>
          <p className="eyebrow">Loans out</p>
          {(activity?.loans_out?.length ?? 0) === 0 ? <p className="mt-2 text-[13px] text-sage">None.</p> : activity!.loans_out.map((row) => (
            <button key={row.player_id} type="button" className="block mt-2 text-[13px] text-bone" onClick={() => onOpenPlayer(row.player_id)}>{row.full_name} → {row.to_club_name}</button>
          ))}
        </Card>
      </div>
    </div>
  );
};

const ClubFinancesPanel: React.FC<{ club: Club; squad: Player[] }> = ({ club, squad }) => (
  <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
    <Card><p className="eyebrow flex items-center gap-1"><Landmark size={12} /> Balance</p><p className="mt-1 font-mono text-[20px] font-bold text-bone">{formatEUR(club.finances?.balance)}</p></Card>
    <Card><p className="eyebrow">Transfer budget</p><p className="mt-1 font-mono text-[20px] font-bold text-brass">{formatEUR(club.finances?.transfer_budget ?? club.transfer_warchest_eur)}</p></Card>
    <Card><p className="eyebrow">Wage capacity</p><p className="mt-1 font-mono text-[20px] font-bold text-bone">{formatEUR(club.finances?.wage_cap)}</p></Card>
    <Card><p className="eyebrow">Committed wage bill</p><p className="mt-1 font-mono text-[20px] font-bold text-bone">{club.finances?.wage_bill ? formatEUR(club.finances.wage_bill) : formatWageBill(squad)}</p></Card>
    <Card><p className="eyebrow">European revenue</p><p className="mt-1 font-mono text-[20px] font-bold text-bone">{formatEUR(club.finances?.european_revenue)}</p></Card>
  </div>
);

const ClubHistoryPanel: React.FC<{ history: ClubHistoryResponse | null }> = ({ history }) => (
  <Card>
    <p className="eyebrow flex items-center gap-1.5"><Trophy size={13} /> Club history</p>
    {(history?.history?.length ?? 0) === 0 ? (
      <p className="mt-3 text-[13px] text-sage">No completed seasons recorded in this save yet.</p>
    ) : (
      <div className="mt-3 divide-y divide-line/60">
        {history!.history.map((row, idx) => (
          <div key={`${row.season_name}-${idx}`} className="py-2.5 flex items-center justify-between gap-3">
            <div>
              <p className="font-semibold text-bone">{row.season_name}</p>
              <p className="font-mono text-[12px] text-sage">#{row.position} · {row.pts} pts · {row.w}W {row.d}D {row.l}L</p>
            </div>
            <p className="text-[12px] text-brass text-right">{Array.isArray(row.trophies) && row.trophies.length ? row.trophies.join(' · ') : ''}</p>
          </div>
        ))}
      </div>
    )}
  </Card>
);
