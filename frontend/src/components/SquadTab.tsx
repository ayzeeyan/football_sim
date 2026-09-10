import React, { useEffect, useMemo, useState } from 'react';
import type { Club, Player } from '../types';
import { fetchClubSquad, fetchClubXi } from '../services/api';
import { Search, ArrowUpDown, Shield, Sparkle, Eye } from 'lucide-react';
import { soundManager } from '../audio/webAudio';
import { LEAGUES_5 } from '../lib/constants';
import { sortBy } from '../lib/sort';
import { ovrTone, positionTone } from '../lib/constants';
import { Card, ClubCrest, EmptyState, LoadingState, PanelHeader } from './ui/ui';
import { cx, formatMillions, loyaltyLabel } from '../lib/format';
import { usePlayerSheet } from './PlayerSheet';

function formatWageBill(squad: Player[]): string {
  const annual = squad.reduce((sum, p) => sum + (p.wage_eur || 0) * 52, 0);
  if (annual >= 1_000_000_000) return `€${(annual / 1_000_000_000).toFixed(2)}B`;
  return `€${(annual / 1_000_000).toFixed(1)}M`;
}

interface SquadTabProps {
  clubs: Club[];
  onWatchClub: (club: Club) => void;
}

export const SquadTab: React.FC<SquadTabProps> = ({ clubs, onWatchClub }) => {
  const { openPlayer } = usePlayerSheet();
  const [selectedLeague, setSelectedLeague] = useState<string>('All twelve');
  const [selectedClub, setSelectedClub] = useState<Club | null>(null);
  const [squad, setSquad] = useState<Player[]>([]);
  const [xi, setXi] = useState<Player[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortCol, setSortCol] = useState<keyof Player>('ovr');
  const [sortAsc, setSortAsc] = useState(false);
  const [loading, setLoading] = useState(false);


  const clubsInLeague = useMemo(
    () => (selectedLeague === 'All twelve' ? clubs : clubs.filter((c) => c.league === selectedLeague)),
    [clubs, selectedLeague],
  );

  useEffect(() => {
    if (clubsInLeague.length > 0) setSelectedClub(clubsInLeague[0]);
    else if (clubs.length > 0) setSelectedClub(clubs[0]);
  }, [clubsInLeague, clubs]);

  useEffect(() => {
    if (!selectedClub) return;
    let cancelled = false;
    setLoading(true);
    Promise.all([fetchClubSquad(selectedClub.club_id), fetchClubXi(selectedClub.club_id)])
      .then(([data, eleven]) => {
        if (!cancelled) {
          setSquad(data);
          setXi(eleven);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [selectedClub]);

  const handleSort = (col: keyof Player) => {
    soundManager.playClick();
    if (sortCol === col) setSortAsc((v) => !v);
    else {
      setSortCol(col);
      setSortAsc(false);
    }
  };

  const filteredSquad = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    const filtered =
      q === ''
        ? squad
        : squad.filter((p) => p.full_name.toLowerCase().includes(q) || p.position.toLowerCase().includes(q));
    return sortBy(filtered, sortCol, sortAsc);
  }, [squad, searchQuery, sortCol, sortAsc]);

  const sortIndicator = (col: keyof Player) => (sortCol === col ? (sortAsc ? ' ↑' : ' ↓') : '');

  const avgOvr = squad.length ? Math.round(squad.reduce((s, p) => s + p.ovr, 0) / squad.length) : 0;
  const avgAge = squad.length ? (squad.reduce((s, p) => s + p.age, 0) / squad.length).toFixed(1) : '—';
  const totalValue = squad.reduce((s, p) => s + (p.market_value_eur || 0), 0);
  const byLine = {
    GK: squad.filter((p) => p.category === 'GK').length,
    DEF: squad.filter((p) => p.category === 'DEF').length,
    MID: squad.filter((p) => p.category === 'MID').length,
    FWD: squad.filter((p) => p.category === 'FWD').length,
  };

  return (
    <div className="space-y-4">
      <Card>
        <PanelHeader
          kicker="Squad explorer · Spectator view"
          title="Squads"
          subtitle="Filter by competition and pick a club to inspect. Transfers are negotiated by the clubs themselves — this stand is for watching."
        />
        <div className="flex flex-wrap items-center justify-between gap-3 mt-4">
          <div className="flex gap-1.5 overflow-x-auto" role="tablist" aria-label="League filter">
            {['All twelve', ...LEAGUES_5].map((lg) => (
              <button
                key={lg}
                onClick={() => {
                  soundManager.playClick();
                  setSelectedLeague(lg);
                }}
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
              {clubsInLeague.map((c) => (
                <option key={c.club_id} value={c.club_id}>
                  {c.club_name} ({c.short_name})
                </option>
              ))}
            </select>

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
        </div>
      </Card>

      {selectedClub && (
        <Card className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3.5">
            <ClubCrest club={selectedClub} size={60} />
            <div>
              <div className="font-display text-[20px] font-semibold text-bone flex items-center gap-2">
                <span>{selectedClub.club_name}</span>
                <span className="text-[11px] px-2 py-0.5 rounded-md bg-cardLight border border-line text-sage font-mono font-semibold">
                  {selectedClub.short_name}
                </span>
              </div>
              <div className="text-[13px] text-sage font-normal mt-0.5">
                {selectedClub.league} · {selectedClub.home_stadium} · Team rating{' '}
                <strong className="text-bone font-mono">{selectedClub.overall_team_rating}</strong> ·{' '}
                <strong className="text-bone font-mono">{selectedClub.squad_size} players</strong>
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-2.5">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-brass/[0.08] border border-brass/40" title="Every weekly wage × 52, added up across the squad">
                  <span className="eyebrow !text-brass">Wage bill</span>
                  <strong className="text-bone font-mono text-[14px]">{formatWageBill(squad)}/yr</strong>
                </div>

                {(() => {
                  const morale = typeof selectedClub.morale === 'number' ? selectedClub.morale : 75;
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
                    {' · '}<strong className="text-brass font-mono">{selectedClub.manager.formatted_budget}</strong> warchest
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
            <Eye size={15} /> Watch this club live
          </button>
        </Card>
      )}

      {xi.length > 0 && (
        <Card>
          <p className="eyebrow mb-3">Likely starting XI · 4-3-3</p>
          <div className="flex gap-2 overflow-x-auto pb-1">
            {xi.map((p) => (
              <button
                key={p.player_id}
                type="button"
                onClick={() => {
                  openPlayer(p.player_id);
                }}
                className="shrink-0 w-[92px] p-2 rounded-xl border border-line bg-ink/40 text-center hover:border-sage/50"
              >
                <span className={cx('text-[10px] font-mono font-semibold px-1.5 py-0.5 rounded', positionTone(p.category))}>{p.position}</span>
                <p className="text-[12px] font-semibold text-bone truncate mt-1.5">{p.full_name.split(' ').slice(-1)[0]}</p>
                <p className={cx('font-mono text-[13px] font-bold mt-0.5', ovrTone(p.ovr))}>{p.ovr}</p>
              </button>
            ))}
          </div>
        </Card>
      )}

      <div className="panel-tight overflow-hidden">
        {loading ? (
          <LoadingState message="Loading the 24-player roster…" />
        ) : filteredSquad.length === 0 ? (
          <EmptyState message="No players match that search. Clear the search to see the full squad." />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-[13px] min-w-[980px]">
              <thead className="table-head">
                <tr>
                  <th className="py-3 px-4 font-semibold">Pos</th>
                  {(
                    [
                      ['full_name', 'Player'],
                      ['ovr', 'OVR'],
                      ['age', 'Age'],
                      ['market_value_eur', 'Valuation'],
                    ] as Array<[keyof Player, string]>
                  ).map(([col, label]) => (
                    <th key={col} className="py-3 px-3 font-semibold cursor-pointer hover:text-bone" onClick={() => handleSort(col)}>
                      <span className="flex items-center gap-1">
                        <span>
                          {label}
                          {sortIndicator(col)}
                        </span>
                        <ArrowUpDown size={12} />
                      </span>
                    </th>
                  ))}
                  <th className="py-3 px-3 font-semibold">Apps</th>
                  <th className="py-3 px-3 font-semibold">G</th>
                  <th className="py-3 px-3 font-semibold">A</th>
                  <th className="py-3 px-3 font-semibold whitespace-nowrap">All-time</th>
                  <th
                    className="py-3 px-3 font-semibold cursor-pointer hover:text-bone whitespace-nowrap"
                    onClick={() => handleSort('wage_eur')}
                    title="Sort by weekly wage"
                  >
                    <span className="flex items-center gap-1">
                      <span>
                        Wage/wk
                        {sortIndicator('wage_eur')}
                      </span>
                      <ArrowUpDown size={12} />
                    </span>
                  </th>
                  <th className="py-3 px-4 font-semibold whitespace-nowrap">Contract</th>
                  <th className="py-3 px-4 font-semibold">Status</th>
                  <th className="py-3 px-4 font-semibold">Avail.</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line/70">
                {filteredSquad.map((player) => {
                  const isWk = player.universe_wonderkid;
                  const isHeadline = player.player_source === 'headline';
                  return (
                    <tr
                      key={player.player_id}
                      className={cx('hover:bg-cardLight/60 transition-colors cursor-pointer', isWk && 'bg-brass/[0.06]')}
                      onClick={() => {
                        openPlayer(player.player_id);
                      }}
                    >
                      <td className="py-3 px-4 font-mono font-semibold">
                        <span className={cx('px-2 py-0.5 rounded-md text-[11px] font-semibold', positionTone(player.category))}>{player.position}</span>
                      </td>
                      <td className="py-3 px-3 font-semibold text-bone">{player.full_name}</td>
                      <td className="py-3 px-3 font-bold font-mono text-[14px]">
                        <span className={ovrTone(player.ovr)}>{player.ovr}</span>
                      </td>
                      <td className="py-3 px-3 text-sage font-mono">{player.age}</td>
                      <td className="py-3 px-3 font-mono text-[#A9CDBB] font-semibold whitespace-nowrap">{player.formatted_value}</td>
                      <td className="py-3 px-3 text-sage font-mono">{player.appearances}</td>
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
                        {isWk ? (
                          <span className="px-2 py-0.5 rounded-md bg-brass/10 text-brass border border-brass/40 font-semibold text-[11px] flex items-center gap-1 w-fit uppercase tracking-[0.06em]">
                            <Sparkle size={11} /> U-14 prodigy
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

    </div>
  );
};
