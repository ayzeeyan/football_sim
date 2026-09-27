import React, { useEffect, useState } from 'react';
import type { PlayerProfile } from '../../../types';
import { ovrTone, positionTone } from '../../../lib/constants';
import { cx, loyaltyLabel } from '../../../lib/format';
import { soundManager } from '../../../audio/webAudio';
import { ClubCrest, ErrorState, LoadingState, Modal, PlayerPortrait, ProgressBar } from '../../ui/ui';
import { Sparkle, Shield, AlertTriangle, Trophy, TrendingUp, Calendar, Heart, Award, Activity, Briefcase } from 'lucide-react';
import { AttackingRoleMap } from '../AttackingRoleMap';
import { WatchToggle } from '../../layout/WatchlistPanel';
import { RESULT_TONE, clubIdLabel, prettyCompetitionName, type PlayerSheetTab } from './labels';

export const PlayerSheetModal: React.FC<{
  profile: PlayerProfile | null;
  loading: boolean;
  error?: string | null;
  open: boolean;
  onClose: () => void;
  onRetry?: () => void;
}> = ({ profile, loading, error, open, onClose, onRetry }) => {
  const [activeTab, setActiveTab] = useState<PlayerSheetTab>('overview');
  const player = profile?.player;

  // Reset to overview tab when a new player is loaded
  useEffect(() => {
    if (open) setActiveTab('overview');
  }, [open, player?.player_id]);

  return (
    <Modal open={open} onClose={onClose} maxWidth="max-w-6xl">
      {loading && !player ? (
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-12">
          <LoadingState message="Scouting player records…" />
        </div>
      ) : error && !player ? (
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-6">
          <ErrorState message={error} onRetry={onRetry} />
        </div>
      ) : player ? (
        <div className="player-sheet flex min-h-0 flex-1 flex-col">
          {/* Original club-led career profile hero */}
          <div className={cx('player-sheet-hero relative shrink-0 overflow-hidden border-b border-brass/20 bg-gradient-to-br from-[#183a2a] via-[#10281d] to-[#08140e] p-3 sm:p-4', activeTab !== 'overview' && 'player-sheet-hero-compact')}>
            {/* Background Watermark Crest */}
            {profile?.club && (
              <div className="absolute right-4 -bottom-4 opacity-10 pointer-events-none scale-150 transform">
                <ClubCrest club={profile.club} size={140} />
              </div>
            )}

            <div className="flex items-start justify-between gap-4 relative z-10">
              <div className="flex items-center gap-4 min-w-0">
                <div className="relative shrink-0">
                  <PlayerPortrait player={player} size={56} className="player-sheet-portrait" />
                  {profile?.club && (
                    <div className="absolute -bottom-1 -right-1 rounded-full bg-ink/90 p-0.5 border border-line">
                      <ClubCrest club={profile.club} size={24} />
                    </div>
                  )}
                </div>

                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className={cx('px-2 py-0.5 rounded text-[11px] font-mono font-bold uppercase tracking-wider', positionTone(player.category))}>
                      {player.position}
                    </span>
                    {player.secondary_position && (
                      <span className="px-1.5 py-0.5 rounded bg-cardLight text-sage font-mono text-[10.5px] border border-line">
                        / {player.secondary_position}
                      </span>
                    )}
                    {player.is_captain && (
                      <span className="px-1.5 py-0.5 rounded bg-brass/20 text-brass border border-brass/40 font-mono text-[10px] font-bold">
                        CAPTAIN
                      </span>
                    )}
                    {player.is_vice_captain && !player.is_captain && (
                      <span className="px-1.5 py-0.5 rounded bg-cardLight text-sage border border-line font-mono text-[10px]">
                        VC
                      </span>
                    )}
                    {player.universe_wonderkid && (
                      <span className="px-2 py-0.5 rounded bg-brass/15 text-brass border border-brass/40 font-mono text-[10px] font-bold flex items-center gap-1">
                        <Sparkle size={10} /> U-17 PRODIGY
                      </span>
                    )}
                    {player.on_loan && (
                      <span className="px-2 py-0.5 rounded bg-[#8AB4C8]/15 text-[#A9CBDD] border border-[#8AB4C8]/30 font-mono text-[10px] font-bold">
                        LOAN
                      </span>
                    )}
                  </div>

                  <h2 className="font-display text-[22px] font-bold text-bone truncate mt-1 leading-tight flex items-center gap-2">
                    {player.full_name}
                    <WatchToggle entity="player" id={player.player_id} onShowToast={() => undefined} className="!min-h-7 text-[10px]" />
                  </h2>

                  <p className="text-[13px] text-sage flex items-center gap-2 mt-0.5">
                    <span>{profile?.club?.club_name ?? 'Free Agent'}</span>
                    <span>·</span>
                    <span className="font-mono">Age {player.age}</span>
                    <span>·</span>
                    <span>{player.squad_role || 'Squad Player'}</span>
                  </p>
                </div>
              </div>

              {/* Large Glowing OVR Badge */}
              <div className="flex shrink-0 flex-col items-center justify-center rounded-md border border-brass/25 bg-black/25 px-4 py-2 shadow-lg">
                <span className="text-[10px] font-mono uppercase tracking-widest text-sage">OVR</span>
                <span className={cx('player-sheet-ovr score-display mt-0.5 text-[36px] font-bold leading-none', ovrTone(player.ovr))}>
                  {player.ovr}
                </span>
                {profile?.avg_rating != null && (
                  <span className="text-[11px] font-mono text-brass mt-1 font-semibold">
                    {profile.avg_rating.toFixed(2)} ★
                  </span>
                )}
              </div>
            </div>

            {/* Quick Financial & Form Vitals Strip */}
            <div className="player-sheet-vitals mt-3 grid grid-cols-2 gap-1.5 border-t border-line/50 pt-2 font-mono text-[12px] sm:grid-cols-4">
              <div className="rounded-md border border-line/60 bg-ink/40 px-2 py-1.5">
                <span className="block text-[10px] text-sage uppercase tracking-wider">Market Value</span>
                <span className="text-bone font-bold text-[13px]">{player.formatted_value}</span>
              </div>
              <div className="rounded-md border border-line/60 bg-ink/40 px-2 py-1.5">
                <span className="block text-[10px] text-sage uppercase tracking-wider">Weekly Wage</span>
                <span className="text-bone font-bold text-[13px]">{player.formatted_wage}</span>
              </div>
              <div className="rounded-md border border-line/60 bg-ink/40 px-2 py-1.5">
                <span className="block text-[10px] text-sage uppercase tracking-wider">Contract</span>
                <span className={cx('font-bold text-[13px]', player.contract_years <= 1 ? 'text-ember' : 'text-bone')}>
                  {player.contract_years} yr{player.contract_years === 1 ? '' : 's'}
                  {player.contract_years <= 1 && ' (Expiring)'}
                </span>
              </div>
              <div className="rounded-md border border-line/60 bg-ink/40 px-2 py-1.5">
                <span className="block text-[10px] text-sage uppercase tracking-wider">Form & Morale</span>
                <span className="text-bone font-bold text-[13px]">
                  {player.form_band || 'Average'} · {player.morale_band || 'Content'}
                </span>
              </div>
            </div>
          </div>

          {/* Modal Tab Navigation */}
          <div className="section-tabs px-3 sm:px-5" role="group" aria-label="Player profile sections">
            {(
              [
                ['overview', 'Overview', Activity],
                ['stats', 'Season', Trophy],
                ['growth', 'Development', TrendingUp],
                ['career', 'Career', Briefcase],
                ['history', 'History', Calendar],
              ] as Array<[PlayerSheetTab, string, React.ComponentType<{ size?: number }>]>
            ).map(([tabKey, label, Icon]) => (
              <button
                key={tabKey}
                type="button"
                aria-pressed={activeTab === tabKey}
                onClick={() => {
                  soundManager.playClick();
                  setActiveTab(tabKey);
                }}
                className={cx(
                  'section-tab flex items-center gap-1.5 whitespace-nowrap',
                  activeTab === tabKey
                    ? 'section-tab-active'
                    : '',
                )}
              >
                <Icon size={14} />
                <span>{label}</span>
              </button>
            ))}
          </div>

          {/* Tab Contents */}
          <div className="player-sheet-content min-h-0 flex-1 space-y-3 overflow-y-auto p-3 sm:p-4">
            {/* TAB 1: OVERVIEW */}
            {activeTab === 'overview' && (
              <div className="space-y-4">
                {/* Urgent Alerts */}
                {player.transfer_requested && (
                  <div className="border border-ember/40 bg-ember/10 p-3 rounded-xl flex items-center gap-2.5 text-[13px] text-[#D89A84]">
                    <AlertTriangle size={16} className="text-ember shrink-0" />
                    <span>Transfer request submitted by player. Board and club will review incoming market proposals.</span>
                  </div>
                )}
                {player.promise_kind && (
                  <div className="border border-brass/40 bg-brass/[0.08] p-3 rounded-xl flex items-center gap-2.5 text-[13px] text-brass">
                    <Award size={16} className="text-brass shrink-0" />
                    <span>Manager Promise: {player.promise_kind}{player.promise_season ? ` · targeted for ${player.promise_season}` : ''}</span>
                  </div>
                )}

                {player.category === 'FWD' && <AttackingRoleMap player={player} />}

                {/* 3 Console Summary Cards (matching Panel 3) */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3.5">
                  {/* Card 1: Contract */}
                  <div className="p-4 rounded-xl border border-[#214332] bg-[#0E1D16] space-y-2.5">
                    <div className="flex items-center justify-between border-b border-[#214332] pb-2">
                      <span className="eyebrow !text-brass text-[11px] font-bold uppercase tracking-wider">Contract</span>
                      <Briefcase size={13} className="text-brass/70" />
                    </div>
                    <div className="flex items-center gap-2.5 pt-1">
                      <ClubCrest club={profile?.club ?? null} size={28} />
                      <div className="min-w-0">
                        <p className="font-semibold text-bone text-[13px] truncate">{profile?.club?.club_name ?? 'Free Agent'}</p>
                        <p className="font-mono text-[11.5px] text-sage">{player.formatted_wage} / wk</p>
                      </div>
                    </div>
                    <div className="pt-2 border-t border-[#214332]/60 space-y-1 text-[12px] font-mono">
                      <div className="flex justify-between text-sage">
                        <span>Duration</span>
                        <span className={cx('font-semibold', player.contract_years <= 1 ? 'text-ember' : 'text-bone')}>
                          {player.contract_years} yr{player.contract_years === 1 ? '' : 's'} {player.contract_years <= 1 ? '(Expiring)' : 'remaining'}
                        </span>
                      </div>
                      <div className="flex justify-between text-sage">
                        <span>Squad Role</span>
                        <span className="font-semibold text-bone">{player.squad_role || 'Important'}</span>
                      </div>
                    </div>
                  </div>

                  {/* Card 2: Status */}
                  <div className="p-4 rounded-xl border border-[#214332] bg-[#0E1D16] space-y-2.5">
                    <div className="flex items-center justify-between border-b border-[#214332] pb-2">
                      <span className="eyebrow !text-brass text-[11px] font-bold uppercase tracking-wider">Status</span>
                      <Activity size={13} className="text-brass/70" />
                    </div>
                    <div className="space-y-2 pt-1 text-[12px]">
                      <div>
                        <div className="flex justify-between text-sage mb-0.5">
                          <span>Morale</span>
                          <span className="font-semibold text-bone">{player.morale_band || 'Not rated'} ({player.morale ?? '—'})</span>
                        </div>
                        {typeof player.morale === 'number' && <ProgressBar pct={player.morale} toneClass="bg-pitchtone" />}
                      </div>
                      <div>
                        <div className="flex justify-between text-sage mb-0.5">
                          <span>Fitness</span>
                          <span className="font-semibold text-bone">{typeof player.fitness === 'number' ? `${player.fitness}%` : '—'}</span>
                        </div>
                        {typeof player.fitness === 'number' && <ProgressBar pct={player.fitness} toneClass="bg-emerald-500" />}
                      </div>
                      <div>
                        <div className="flex justify-between text-sage mb-0.5">
                          <span>Sharpness</span>
                          <span className="font-semibold text-bone">{typeof player.sharpness === 'number' ? `${player.sharpness}%` : '—'}</span>
                        </div>
                        {typeof player.sharpness === 'number' && <ProgressBar pct={player.sharpness} toneClass="bg-teal-500" />}
                      </div>
                    </div>
                  </div>

                  {/* Card 3: Season Stats */}
                  <div className="p-4 rounded-xl border border-[#214332] bg-[#0E1D16] space-y-2.5">
                    <div className="flex items-center justify-between border-b border-[#214332] pb-2">
                      <span className="eyebrow !text-brass text-[11px] font-bold uppercase tracking-wider">Season Stats</span>
                      <Trophy size={13} className="text-brass/70" />
                    </div>
                    <div className="grid grid-cols-3 gap-2 text-center pt-1 font-mono">
                      <div className="p-1.5 rounded-lg bg-[#142920] border border-[#234A37]">
                        <span className="text-[10px] text-sage block uppercase">Apps</span>
                        <strong className="text-[16px] text-bone block leading-tight mt-0.5">{player.appearances}</strong>
                      </div>
                      <div className="p-1.5 rounded-lg bg-[#142920] border border-[#234A37]">
                        <span className="text-[10px] text-sage block uppercase">Goals</span>
                        <strong className="text-[16px] text-bone block leading-tight mt-0.5">{player.goals}</strong>
                      </div>
                      <div className="p-1.5 rounded-lg bg-[#142920] border border-[#234A37]">
                        <span className="text-[10px] text-sage block uppercase">Assists</span>
                        <strong className="text-[16px] text-bone block leading-tight mt-0.5">{player.assists}</strong>
                      </div>
                    </div>
                    <div className="pt-2 border-t border-[#214332]/60 flex items-center justify-between text-[12px] font-mono">
                      <span className="text-sage">Match Rating</span>
                      <strong className="text-brass text-[13px]">
                        {profile?.avg_rating != null ? `${profile.avg_rating.toFixed(2)} ★` : '—'}
                      </strong>
                    </div>
                  </div>
                </div>

                {/* Medical record (F11): per-player injury history */}
                {(player.injury_history?.length ?? 0) > 0 && (
                  <div className="p-4 rounded-xl border border-line bg-ink/40 space-y-2 text-[13px]">
                    <h4 className="font-display text-[14px] font-semibold text-bone mb-2">Medical record</h4>
                    <ul className="divide-y divide-white/[0.07]">
                      {[...(player.injury_history ?? [])].reverse().map((rec, i) => (
                        <li key={`${rec.season}-${rec.matchweek}-${i}`} className="flex flex-wrap items-center justify-between gap-2 py-1.5 text-[12px]">
                          <span className="text-bone">{rec.kind}</span>
                          <span className="text-sage">
                            {rec.season} week {rec.matchweek} · out {rec.matches_out} {rec.matches_out === 1 ? 'match' : 'matches'} ·{' '}
                            <span className={rec.severity === 'serious' ? 'text-brass' : rec.severity === 'moderate' ? 'text-sage' : 'text-emerald-400'}>{rec.severity}</span>
                          </span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Leadership, Versatility & Identity */}
                <div className="p-4 rounded-xl border border-line bg-ink/40 space-y-2 text-[13px]">
                  <h4 className="font-display text-[14px] font-semibold text-bone mb-2">Tactical Profile & Registration</h4>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-sage">
                    <div>
                      <span>Role & Leadership: </span>
                      <strong className="text-bone">{player.squad_role || 'Squad Player'} · Lead {player.leadership ?? '—'}</strong>
                    </div>
                    <div>
                      <span>Versatility: </span>
                      <strong className="text-bone">{player.versatility ? `${player.versatility}/100` : 'Standard'}</strong>
                    </div>
                    <div>
                      <span>Homegrown Status: </span>
                      <strong className="text-bone">
                        {player.homegrown ? 'Homegrown (HG)' : player.association_trained ? 'Association-trained' : 'Non-homegrown'}
                      </strong>
                    </div>
                    <div>
                      <span>UEFA Registration: </span>
                      <strong className="text-bone">{player.registered_europe ? 'Registered on European List A' : 'Not registered for Europe'}</strong>
                    </div>
                    {player.universe_wonderkid && player.education_label && (
                      <div className="sm:col-span-2 text-brass">
                        <span>Academy & School: </span>
                        <strong>{player.education_label}</strong>
                      </div>
                    )}
                  </div>
                </div>

                {/* Recent Form & Matches */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <h4 className="font-display text-[14px] font-semibold text-bone">Recent Appearances</h4>
                    <span className="text-[11px] font-mono text-sage">{profile?.last_matches?.length ?? 0} logged</span>
                  </div>
                  {(profile?.last_matches?.length ?? 0) === 0 ? (
                    <div className="p-4 rounded-xl border border-line bg-ink/40 text-[13px] text-sage text-center">
                      No competitive appearances logged this season.
                    </div>
                  ) : (
                    <div className="space-y-1.5">
                      {(profile?.last_matches ?? []).map((m) => (
                        <div
                          key={`${m.fixture_id}-${m.matchweek}`}
                          className="flex items-center justify-between gap-3 p-2.5 rounded-xl border border-line bg-ink/40 text-[12.5px]"
                        >
                          <div className="min-w-0">
                            <p className="text-bone font-semibold truncate flex items-center gap-1.5">
                              <span className={cx('font-mono font-bold text-[11px] px-1.5 py-0.2 rounded border', RESULT_TONE[m.result] || 'text-bone')}>
                                {m.result}
                              </span>
                              <span>{m.home ? 'vs' : '@'} {m.opponent}</span>
                              <span className="font-mono text-sage">{m.score}</span>
                            </p>
                            <p className="font-mono text-sage text-[11px] mt-0.5">
                              MW {m.matchweek}
                              {m.competition ? ` · ${prettyCompetitionName(m.competition)}` : ''}
                              {m.motm ? ' · ★ MOTM' : ''} · {m.minutes}'
                              {m.goals ? ` · ${m.goals} G` : ''}
                              {m.assists ? ` · ${m.assists} A` : ''}
                            </p>
                          </div>
                          <span
                            className={cx(
                              'font-mono font-bold text-[14px] px-2 py-0.5 rounded border',
                              (m.rating ?? 0) >= 8
                                ? 'text-brass bg-brass/15 border-brass/40'
                                : (m.rating ?? 0) >= 7
                                  ? 'text-pitchtone bg-pitchtone/10 border-pitchtone/30'
                                  : 'text-bone bg-cardLight border-line',
                            )}
                          >
                            {m.rating != null ? m.rating.toFixed(1) : '—'}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* TAB 2: SEASON STATS */}
            {activeTab === 'stats' && (
              <div className="space-y-4">
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                  <div className="p-3.5 rounded-xl border border-line bg-ink/40 text-center">
                    <span className="eyebrow">Appearances</span>
                    <p className="score-display text-[26px] text-bone mt-1">{player.appearances}</p>
                    <p className="text-[11px] font-mono text-sage">{player.starts ?? 0} starts · {player.minutes ?? 0}'</p>
                  </div>
                  <div className="p-3.5 rounded-xl border border-line bg-ink/40 text-center">
                    <span className="eyebrow">Goals</span>
                    <p className="score-display text-[26px] text-bone mt-1">{player.goals}</p>
                    <p className="text-[11px] font-mono text-sage">{player.appearances ? (player.goals / player.appearances).toFixed(2) : '0.00'} per app</p>
                  </div>
                  <div className="p-3.5 rounded-xl border border-line bg-ink/40 text-center">
                    <span className="eyebrow">Assists</span>
                    <p className="score-display text-[26px] text-bone mt-1">{player.assists}</p>
                    <p className="text-[11px] font-mono text-sage">{player.appearances ? (player.assists / player.appearances).toFixed(2) : '0.00'} per app</p>
                  </div>
                  <div className="p-3.5 rounded-xl border border-line bg-ink/40 text-center">
                    <span className="eyebrow">{player.category === 'GK' || player.category === 'DEF' ? 'Clean Sheets' : 'Avg Rating'}</span>
                    <p className="score-display text-[26px] text-brass mt-1">
                      {player.category === 'GK' || player.category === 'DEF' ? (player.clean_sheets ?? 0) : (profile?.avg_rating ? profile.avg_rating.toFixed(2) : '—')}
                    </p>
                    <p className="text-[11px] font-mono text-sage">{profile?.apps_rated ?? 0} rated matches</p>
                  </div>
                </div>

                {/* Competition breakdown */}
                {player.competition_stats && Object.keys(player.competition_stats).length > 0 ? (
                  <div className="p-4 rounded-xl border border-line bg-ink/40">
                    <h4 className="font-display text-[14px] font-semibold text-bone mb-3 flex items-center gap-1.5">
                      <Trophy size={15} className="text-brass" /> Breakdown by Competition
                    </h4>
                    <div className="divide-y divide-line/60">
                      {Object.values(player.competition_stats).map((row) => (
                        <div key={row.competition_id} className="py-2.5 flex items-center justify-between gap-3 text-[13px]">
                          <span className="font-semibold text-bone truncate">{prettyCompetitionName(row.competition_id)}</span>
                          <div className="font-mono text-sage text-[12px] flex items-center gap-3 shrink-0">
                            <span><strong>{row.appearances}</strong> apps</span>
                            <span><strong>{row.goals}</strong> G</span>
                            <span><strong>{row.assists}</strong> A</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <div className="p-4 rounded-xl border border-line bg-ink/40 text-center text-[13px] text-sage">
                    No multi-competition stats recorded for this campaign.
                  </div>
                )}
              </div>
            )}

            {/* TAB 3: DEVELOPMENT */}
            {activeTab === 'growth' && (
              <div className="space-y-4">
                <div className="p-4 rounded-xl border border-brass/40 bg-brass/[0.06]">
                  <h4 className="font-display text-[15px] font-bold text-brass flex items-center gap-2 mb-2">
                    <TrendingUp size={16} /> Development Status
                  </h4>
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-3">
                    <div className="p-2.5 rounded-lg bg-ink/50 border border-line">
                      <span className="eyebrow">Current OVR</span>
                      <p className="font-mono font-bold text-[18px] text-bone mt-0.5">{player.ovr}</p>
                    </div>
                    <div className="p-2.5 rounded-lg bg-ink/50 border border-line">
                      <span className="eyebrow">Sharpness</span>
                      <p className="font-mono font-bold text-[18px] text-bone mt-0.5">{player.sharpness ?? '—'}</p>
                    </div>
                    <div className="p-2.5 rounded-lg bg-ink/50 border border-line">
                      <span className="eyebrow">Position XP</span>
                      <p className="font-mono font-bold text-[18px] text-bone mt-0.5">{player.position_xp ?? '—'}</p>
                    </div>
                    <div className="p-2.5 rounded-lg bg-ink/50 border border-line">
                      <span className="eyebrow">Mentor</span>
                      <p className="font-mono font-bold text-[14px] text-bone mt-1 truncate">{player.mentor_name || 'None assigned'}</p>
                    </div>
                  </div>
                  {player.grew_note && <p className="mt-3 border-t border-brass/20 pt-3 text-[12px] text-sage">{player.grew_note}</p>}
                </div>

                {/* Wonderkid Puberty & Biometric Trajectory */}
                {player.universe_wonderkid && (
                  <div className="p-4 rounded-xl border border-line bg-ink/40 space-y-3 text-[13px]">
                    <h4 className="font-display text-[15px] font-semibold text-bone flex items-center gap-2">
                      <Sparkle size={16} className="text-brass" /> Prodigy Progression & Academy Track
                    </h4>
                    <p className="text-sage leading-relaxed">
                      Canonical U-17 prodigy undergoing biometric maturation and academy mentorship. Growth curves are driven by match minutes, training XP, and high-school academic exam schedules.
                    </p>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-sage pt-2 border-t border-line/60">
                      <div>
                        <span>Academic Track: </span>
                        <strong className="text-bone">{player.education_label || 'High School Track'}</strong>
                      </div>
                      <div>
                        <span>Development Stage: </span>
                        <strong className="text-brass">Puberty Maturation Wave (U-17)</strong>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* TAB 4: CAREER */}
            {activeTab === 'career' && (
              <div className="space-y-3">
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                  {[
                    ['Career apps', player.career_apps ?? player.appearances],
                    ['Career goals', player.career_goals],
                    ['Career assists', player.career_assists],
                    ['Best season', player.best_season ? `${player.best_goals ?? 0}G · ${player.best_season}` : '—'],
                  ].map(([label, value]) => (
                    <div key={label} className="rounded-md border border-line bg-ink/40 px-3 py-2">
                      <span className="eyebrow">{label}</span>
                      <p className="mt-1 font-mono text-[17px] font-bold text-bone truncate" title={String(value)}>{value}</p>
                    </div>
                  ))}
                </div>
                <div className="space-y-2 rounded-lg border border-line bg-ink/40 p-3">
                  <h4 className="font-display text-[15px] font-semibold text-bone flex items-center gap-2">
                    <Briefcase size={16} className="text-brass" /> Contract Terms & Valuation
                  </h4>
                  <div className="grid grid-cols-1 gap-2 text-[13px] sm:grid-cols-2">
                    <div className="space-y-0.5 rounded-md border border-line bg-card px-3 py-2">
                      <span className="eyebrow">Market Valuation</span>
                      <p className="font-mono text-[16px] font-bold text-[#A9CDBB]">{player.formatted_value}</p>
                      <p className="text-[11px] text-sage">Subject to dynamic corridor clamping</p>
                    </div>
                    <div className="space-y-0.5 rounded-md border border-line bg-card px-3 py-2">
                      <span className="eyebrow">Wage Package</span>
                      <p className="font-mono text-[16px] font-bold text-bone">{player.formatted_wage} / wk</p>
                      <p className="text-[11px] text-sage">Annual impact: €{(((player.wage_eur || 0) * 52) / 1_000_000).toFixed(2)}M / yr</p>
                    </div>
                    <div className="space-y-0.5 rounded-md border border-line bg-card px-3 py-2">
                      <span className="eyebrow">Contract Length</span>
                      <p className={cx('font-mono text-[16px] font-bold', player.contract_years <= 1 ? 'text-ember' : 'text-bone')}>
                        {player.contract_years} year{player.contract_years === 1 ? '' : 's'} remaining
                      </p>
                      <p className="text-[11px] text-sage">
                        {player.contract_years <= 1 ? 'Contract running down - free agency risk' : 'Secure contract tenure'}
                      </p>
                    </div>
                    <div className="space-y-0.5 rounded-md border border-line bg-card px-3 py-2">
                      <span className="eyebrow">Dressing Room Loyalty</span>
                      <p className="font-mono text-[16px] font-bold text-bone flex items-center gap-1.5">
                        <Heart size={14} className={player.loyalty >= 70 ? 'text-brass' : 'text-ember'} fill="currentColor" />
                        <span>{player.loyalty} / 100</span>
                      </p>
                      <p className="text-[11px] text-sage">{loyaltyLabel(player.loyalty)}</p>
                    </div>
                  </div>
                </div>

                {player.on_loan && (
                  <div className="p-4 rounded-xl border border-[#8AB4C8]/40 bg-[#8AB4C8]/10 text-[13px] space-y-1.5">
                    <h5 className="font-semibold text-bone flex items-center gap-2">
                      <Shield size={15} className="text-[#A9CBDD]" /> Loan Agreement
                    </h5>
                    <p className="text-sage">
                      Temporary loan assignment from parent club.
                      {(player.loan_buy_clause_eur ?? 0) > 0 && (
                        <span className="block text-bone font-mono mt-1">
                          Pre-agreed purchase option: <strong>{player.formatted_buy_clause}</strong>
                        </span>
                      )}
                    </p>
                  </div>
                )}
                <div className="p-4 rounded-xl border border-line bg-ink/40">
                  <h4 className="font-display text-[14px] font-semibold text-bone mb-2">Clubs</h4>
                  <p className="text-[13px] text-sage">Current: <strong className="text-bone">{profile?.club?.club_name ?? (player.is_free_agent ? 'Free agent' : '—')}</strong></p>
                  {(profile?.previous_clubs?.length ?? 0) > 0 && (
                    <p className="mt-1 font-mono text-[12px] text-sage">Previous: {profile!.previous_clubs!.join(' · ')}</p>
                  )}
                </div>
                {((profile?.awards_history?.length ?? 0) > 0 || (player.awards_history?.length ?? 0) > 0) && (
                  <div className="p-4 rounded-xl border border-line bg-ink/40">
                    <h4 className="font-display text-[14px] font-semibold text-bone mb-2 flex items-center gap-1.5"><Trophy size={14} className="text-brass" /> Trophies & awards</h4>
                    <div className="space-y-1.5">
                      {(profile?.awards_history ?? player.awards_history ?? []).map((row, idx) => (
                        <p key={`${row.title}-${row.season}-${idx}`} className="text-[13px] text-bone">
                          <span className="font-mono text-sage">{row.season || '—'}</span> · {row.title}
                        </p>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'history' && (
              <div className="space-y-4">
                <div className="p-4 rounded-xl border border-line bg-ink/40">
                  <h4 className="font-display text-[15px] font-semibold text-bone mb-3">Season-by-season</h4>
                  {(profile?.season_history?.length ?? player.season_history?.length ?? 0) === 0 ? (
                    <p className="text-[13px] text-sage">No compact season ledger has been recorded in this save yet. Future seasons will appear here.</p>
                  ) : (
                    <div className="divide-y divide-line/60">
                      {(profile?.season_history ?? player.season_history ?? []).map((row, idx) => (
                        <div key={`${row.season}-${row.competition_id}-${idx}`} className="py-2.5 flex items-center justify-between gap-3">
                          <div>
                            <p className="font-semibold text-bone">{row.season}{row.club_name ? ` · ${row.club_name}` : ''}</p>
                            <p className="font-mono text-[12px] text-sage">
                              {row.competition_id ? prettyCompetitionName(row.competition_id) : 'Club season'}
                              {row.avg_rating ? ` · ${row.avg_rating.toFixed(2)} avg` : ''}
                            </p>
                          </div>
                          <p className="font-mono text-[12px] text-sage shrink-0">{row.appearances} apps · {row.goals} G · {row.assists} A</p>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
                <div className="p-4 rounded-xl border border-line bg-ink/40">
                  <h4 className="font-display text-[15px] font-semibold text-bone mb-3">Transfer history</h4>
                  {(profile?.transfer_history?.length ?? player.transfer_history?.length ?? 0) === 0 ? (
                    <p className="text-[13px] text-sage">No recorded moves in this career yet.</p>
                  ) : (
                    <div className="divide-y divide-line/60">
                      {(profile?.transfer_history ?? player.transfer_history ?? []).map((row, idx) => (
                        <div key={`${row.from_club_id}-${row.to_club_id}-${idx}`} className="py-2.5">
                          <p className="text-[13px] text-bone">
                            <span title={row.from_club_id || undefined}>{clubIdLabel(row.from_club_id) || '—'}</span>
                            {' → '}
                            <span title={row.to_club_id || undefined}>{clubIdLabel(row.to_club_id) || 'Free agent'}</span>
                          </p>
                          <p className="font-mono text-[12px] text-sage">{row.season || `MW ${row.matchweek ?? '—'}`} · {row.type}{row.fee_eur ? ` · €${(row.fee_eur / 1_000_000).toFixed(1)}M` : ''}</p>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
                <div className="p-4 rounded-xl border border-line bg-ink/40">
                  <h4 className="font-display text-[15px] font-semibold text-bone mb-3">Contract history</h4>
                  {(profile?.contract_history?.length ?? player.contract_history?.length ?? 0) === 0 ? (
                    <p className="text-[13px] text-sage">Contract events are recorded going forward from this version.</p>
                  ) : (
                    <div className="divide-y divide-line/60">
                      {(profile?.contract_history ?? player.contract_history ?? []).map((row, idx) => (
                        <div key={`${row.kind}-${row.season}-${idx}`} className="py-2.5 flex justify-between text-[13px]">
                          <span className="text-bone capitalize">{row.kind}</span>
                          <span className="font-mono text-sage">{row.season || '—'}{row.years ? ` · ${row.years} yr` : ''}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="p-12 text-center text-sage text-[13px]">
          Player record not found in active career save.
        </div>
      )}
    </Modal>
  );
};
