import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { Club, ClubHistoryResponse, ClubProfile, ClubTransferActivity, Fixture, Player } from '../../../types';
import { fetchClubFixtures, fetchClubHistory, fetchClubProfile, fetchClubSquad, fetchClubTransfers, fetchClubXi } from '../../../services/api';
import { Search, ArrowUpDown, Shield, Sparkle, Eye, LayoutGrid, List, Columns, AlertTriangle, Flame, Clock, UserX } from 'lucide-react';
import { soundManager } from '../../../audio/webAudio';
import { LEAGUES_5 } from '../../../lib/constants';
import { sortBy } from '../../../lib/sort';
import { ovrTone, positionTone } from '../../../lib/constants';
import { Card, ClubCrest, EmptyState, ErrorState, FormPips, LoadingState, PanelHeader } from '../../ui/ui';
import { cx, formatEUR, formatMillions, loyaltyLabel } from '../../../lib/format';
import { usePlayerSheet } from '../PlayerSheet';
import { FormationPitch } from '../../matches/FormationPitch';

import { CLUB_PROFILE_TABS, FIXTURE_COMP_FILTERS, formatWageBill, isSquadFatigued, isSquadInForm, isSquadUnhappy, passesSquadStatus, squadStatusCounts, type ClubProfileTab, type SquadStatusFilter, type SquadTabProps, type SquadViewMode } from './status';
import { ClubFinancesPanel, ClubFixturesPanel, ClubHistoryPanel, ClubOverviewPanel, ClubTransfersPanel } from './panels';
import { RecruitmentPanel } from './RecruitmentPanel';
import { SetPiecesPanel } from './SetPiecesPanel';
import { MedicalPanel } from './MedicalPanel';
import { LineupEditorPanel } from './LineupEditorPanel';

export const SquadTab: React.FC<SquadTabProps> = ({ clubs, initialClubId, onWatchClub, onWatchFixture, onShowToast }) => {
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
    const seed = linked ?? clubs[0] ?? null;
    setSelectedClub(seed);
    if (linked) setSelectedLeague(linked.league);
  }, [clubs, selectedClub]);

  const selectLeague = (league: string) => {
    soundManager.playClick();
    setSelectedLeague(league);
    const list = league === 'All clubs' ? clubs : clubs.filter((c) => c.league === league);
    if (list.length > 0) setSelectedClub(list[0] ?? null);
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

      {selectedClub && profileTab === 'recruitment' && (
        <RecruitmentPanel club={selectedClub} onOpenPlayer={(id) => openPlayer(id)} />
      )}

      {selectedClub && profileTab === 'set-pieces' && (
        <SetPiecesPanel club={selectedClub} />
      )}

      {selectedClub && profileTab === 'medical' && (
        <MedicalPanel club={selectedClub} onOpenPlayer={(id) => openPlayer(id)} />
      )}

      {selectedClub && profileTab === 'lineup' && (
        <LineupEditorPanel club={selectedClub} squad={squad} onToast={(m) => onShowToast?.(m)} onOpenPlayer={(id) => openPlayer(id)} />
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
