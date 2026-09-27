import React from 'react';
import {
  Trophy,
  AlertTriangle,
  ArrowRight,
  Coins,
  Newspaper,
  Calendar,
  MapPin,
  TrendingUp,
  Clock,
} from 'lucide-react';
import type { Club, CompetitionClub, CompetitionFixtureRow, WorldDashboard } from '../../types';
import { fetchWorldDashboard } from '../../services/api';
import { useAsyncData } from '../../hooks/useAsyncData';
import { usePlayerSheet } from '../clubs/PlayerSheet';
import {
  Card,
  ClubCrest,
  ErrorState,
  LoadingState,
  ProgressBar,
} from '../ui/ui';
import { cx, formatMillions } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';

interface HomeDashboardTabProps {
  careerKey: number;
  onOpenFixture: (fixture: CompetitionFixtureRow) => void;
  onViewSquad: (clubId: string) => void;
  onOpenInbox: () => void;
  onOpenTransfers: () => void;
  onOpenLeague: () => void;
  onOpenCompetitions: () => void;
}

function asClub(club: CompetitionClub | null | undefined): Club | null {
  if (!club) return null;
  return club as unknown as Club;
}

function importanceBadge(importance?: string | null): { label: string; tone: string } {
  switch (importance) {
    case 'Final':
    case 'Cup Final':
      return { label: importance, tone: 'bg-gradient-to-r from-[#D4AF37] to-[#AA7C11] text-black font-bold border-amber-300' };
    case 'Semi-Final':
      return { label: 'Semi-Final', tone: 'bg-gradient-to-r from-[#D4AF37] to-[#AA7C11] text-black font-bold border-amber-300' };
    case 'Derby':
      return { label: 'Derby', tone: 'bg-gradient-to-r from-[#D84A38] to-[#992617] text-white font-bold border-red-500' };
    case 'European Decider':
    case 'Big Match':
      return { label: importance, tone: 'bg-gradient-to-r from-[#3B82F6] to-[#1D4ED8] text-white font-semibold border-blue-400' };
    case 'Title Race':
    case 'Relegation Battle':
    case 'Six-Pointer':
      return { label: importance, tone: 'bg-[#C98A4B] text-black font-bold border-amber-400' };
    case 'Cup Tie':
    case 'Important':
    case 'Notable':
      return { label: importance, tone: 'bg-[#2A4333] text-[#A9CDBB] border-[#3E654E]' };
    default:
      return { label: importance || 'Scheduled Fixture', tone: 'bg-[#18241C] text-sage/80 border-[#28382C]' };
  }
}

export const HomeDashboardTab: React.FC<HomeDashboardTabProps> = ({
  careerKey,
  onOpenFixture,
  onViewSquad,
  onOpenInbox,
  onOpenTransfers,
  onOpenLeague,
  onOpenCompetitions,
}) => {
  const { openPlayer } = usePlayerSheet();
  const { data, loading, error, reload } = useAsyncData(fetchWorldDashboard, [careerKey]);

  if (loading) {
    return <Card><LoadingState message="Connecting to European Headquarters…" /></Card>;
  }

  if (error || !data) {
    return <Card><ErrorState message={error || 'The career dashboard returned no data.'} onRetry={reload} /></Card>;
  }

  const next = data.next_fixture;
  const storylines = data.storylines || [];

  const windowLabel = data.transfer_window.open
    ? `${data.transfer_window.type === 'WINTER' ? 'Winter' : 'Summer'} Window · Week ${data.transfer_window.week}/${data.transfer_window.weeks}`
    : data.season_phase === 'transfer_window'
      ? 'Transfer Window Closing'
      : `Matchweek ${Math.min(data.current_matchweek, data.max_matchweeks)} of ${data.max_matchweeks}`;

  return (
    <div className="space-y-4 animate-fade-in">
      {/* Neutral world-save header */}
      <div className="console-hero overflow-hidden p-5 sm:px-7 sm:py-6">
        <div className="relative z-10 flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
          <div className="min-w-0">
            <div>
              <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span className="text-[10px] font-bold uppercase tracking-[0.2em] text-brass">
                  European football world
                </span>
                <span className="h-1 w-1 rounded-full bg-sage/60" aria-hidden="true" />
                <span className="font-mono text-[11px] text-sage/80">{data.season_name.replace('-', '–')}</span>
              </div>
              <h2 className="mt-1 font-display text-[30px] font-bold leading-[0.98] tracking-tight text-bone sm:text-[38px]">
                The season, at a glance.
              </h2>
              <p className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-[13px] text-sage">
                <span>96 clubs across Europe</span>
                <span className="text-sage/40" aria-hidden="true">/</span>
                <span className="font-medium text-bone">{windowLabel}</span>
              </p>
            </div>
          </div>

          <div className="flex shrink-0 items-center gap-2.5 self-start sm:self-auto">
            <button
              type="button"
              onClick={onOpenTransfers}
              className="inline-flex min-h-10 items-center gap-2 border border-brass/25 bg-black/20 px-4 text-[12px] font-semibold text-bone transition-colors hover:border-brass/50 hover:bg-white/[0.05]"
            >
              <Coins size={15} className="text-brass" aria-hidden="true" />
              <span>Transfers</span>
              <ArrowRight size={13} className="text-sage/70" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>

      {/* 2-COLUMN HERO SPLIT: Left (Col 1-8 Next Match Hero) | Right (Col 9-12 Club & Manager Pulse) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4">
        {/* ================= LEFT (Col 1-8): Large Next Match Hero ================= */}
        <div className="lg:col-span-8 space-y-4">
          {next && next.status === 'scheduled' ? (
            <div className="console-hero p-5 sm:p-6 min-h-[360px] flex flex-col">
              {/* Stadium light glow */}
              <div className="absolute top-0 left-1/2 -translate-x-1/2 w-3/4 h-32 bg-gradient-to-b from-[#2E6B47]/20 to-transparent blur-3xl pointer-events-none" />

              {/* Match Header Bar */}
              <div className="relative z-10 flex items-center justify-between gap-3 pb-4 border-b border-[#214332]">
                <div className="flex items-center gap-2.5">
                  <span className={cx('px-2.5 py-0.5 rounded-full text-[10px] uppercase tracking-wider font-bold border', importanceBadge(next.importance).tone)}>
                    {importanceBadge(next.importance).label}
                  </span>
                  <span className="text-[13px] font-semibold text-bone">
                    {next.stage.trim().toLowerCase() !== 'league' ? next.stage : `Matchweek ${next.matchweek}`}
                  </span>
                </div>
                <span className="text-[12px] font-mono text-sage uppercase tracking-wider font-semibold">
                  {next.competition.replace('-', ' ')}
                </span>
              </div>

              {/* Clubs Face-off */}
              <div className="relative z-10 py-7 flex flex-1 items-center justify-around gap-4">
                {/* Home Club */}
                <div className="flex-1 flex flex-col items-center text-center">
                  <div className="p-2 rounded-2xl bg-[#0D1C15] border border-[#244738] shadow-xl transition-transform hover:scale-105">
                    <ClubCrest club={asClub(next.home)} size={76} className="drop-shadow-lg" />
                  </div>
                  <h4 className="font-display text-[20px] sm:text-[24px] font-bold text-bone mt-3 tracking-tight truncate max-w-[200px]">
                    {next.home?.short_name || 'Home'}
                  </h4>
                  <span className="text-[11px] font-mono uppercase text-sage tracking-wider">Home</span>
                </div>

                {/* Match Center / Kickoff */}
                <div className="flex flex-col items-center justify-center px-4 shrink-0">
                  <div className="w-10 h-10 rounded-full bg-[#183325] border border-[#2E5946] flex items-center justify-center font-display font-bold text-brass text-[13px] shadow-md">
                    VS
                  </div>
                  <div className="mt-2.5 flex items-center gap-1.5 text-[11.5px] font-mono text-sage">
                    <Calendar size={12} className="text-brass/70" />
                    <span>MW {next.matchweek}</span>
                  </div>
                  <div className="flex items-center gap-1.5 text-[11.5px] text-sage/80 mt-0.5">
                    <MapPin size={12} className="text-sage/60" />
                    <span className="truncate max-w-[140px]">{next.home?.short_name ?? 'Home side'} at home</span>
                  </div>
                </div>

                {/* Away Club */}
                <div className="flex-1 flex flex-col items-center text-center">
                  <div className="p-2 rounded-2xl bg-[#0D1C15] border border-[#244738] shadow-xl transition-transform hover:scale-105">
                    <ClubCrest club={asClub(next.away)} size={76} className="drop-shadow-lg" />
                  </div>
                  <h4 className="font-display text-[20px] sm:text-[24px] font-bold text-bone mt-3 tracking-tight truncate max-w-[200px]">
                    {next.away?.short_name || 'Away'}
                  </h4>
                  <span className="text-[11px] font-mono uppercase text-sage tracking-wider">Away</span>
                </div>
              </div>

              <div className="relative z-10 pt-5 border-t border-[#214332]">
                <p className="mb-3 text-[12px] text-sage">{next.importance ? next.importance : 'Featured fixture for this matchweek'}</p>
                <button
                  type="button"
                  onClick={() => onOpenFixture(next)}
                  className="inline-flex min-h-11 w-full items-center justify-center gap-2 bg-bone px-4 text-[13px] font-bold text-ink transition-colors hover:bg-[#fff6dc]"
                >
                  <span>Open Match Centre</span><ArrowRight size={15} aria-hidden="true" />
                </button>
              </div>
            </div>
          ) : (
            <div className="console-card p-8 text-center space-y-3">
              <Trophy size={36} className="mx-auto text-brass" />
              <h3 className="font-display text-[20px] font-bold text-bone">No Scheduled Match This Matchweek</h3>
              <p className="text-[13px] text-sage max-w-md mx-auto">
                No featured fixture is currently scheduled. Explore competitions, clubs, and the transfer market while the world calendar advances.
              </p>
            </div>
          )}

          {/* Quick League Leaders Table Preview */}
          <div className="console-card p-5">
            <div className="flex items-center justify-between pb-3 border-b border-[#214332]">
              <div className="flex items-center gap-2">
                <Trophy size={16} className="text-brass" />
                <h3 className="font-display text-[16px] font-bold text-bone tracking-tight">League Standings Preview</h3>
              </div>
              <button
                type="button"
                onClick={onOpenLeague}
                className="text-[12px] font-semibold text-brass hover:text-[#FFE082] flex items-center gap-1 transition-colors"
              >
                Domestic tables <ArrowRight size={13} />
              </button>
            </div>

            <div className="mt-3 grid grid-cols-1 sm:grid-cols-2 gap-2">
              {data.league_leaders && data.league_leaders.length > 0 ? (
                data.league_leaders.slice(0, 6).map((c, idx) => (
                  <div
                    key={c.club_id}
                    role="button"
                    tabIndex={0}
                    aria-label={`View ${c.short_name} club profile`}
                    onClick={() => onViewSquad(c.club_id)}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault();
                        onViewSquad(c.club_id);
                      }
                    }}
                    className={cx(
                      'flex items-center justify-between border border-white/[0.08] bg-black/15 p-2.5 text-[12.5px] text-sage transition-colors hover:border-brass/25 hover:bg-white/[0.04] focus-visible:outline focus-visible:outline-2 focus-visible:outline-brass',
                    )}
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className="font-mono font-bold text-[12px] text-sage/70 w-4 text-center">{idx + 1}</span>
                      <ClubCrest club={asClub(c)} size={22} />
                      <span className="truncate text-bone">{c.short_name}</span>
                    </div>
                    <div className="flex items-center gap-2 font-mono text-[11px] shrink-0">
                      <span className="text-sage">{c.pts_gap != null ? `+${c.pts_gap}` : ''}</span>
                      <span className="font-bold text-bone">{c.pts ?? '–'} pts</span>
                    </div>
                  </div>
                ))
              ) : (
                <p className="text-[12px] text-sage/60 py-4 text-center col-span-2">Season commencing…</p>
              )}
            </div>
          </div>
        </div>

        {/* ================= RIGHT (Col 9-12): World context ================= */}
        <div className="lg:col-span-4">
          <section className="console-card h-full overflow-hidden p-5 sm:p-6" aria-label="Season briefing">
            <div className="flex items-center justify-between gap-3">
              <p className="text-[10px] font-bold uppercase tracking-[0.18em] text-brass/80">Season briefing</p>
              <span className="inline-flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-[0.12em] text-sage">
                <span className="h-1.5 w-1.5 rounded-full bg-pitchtone" aria-hidden="true" />
                World in progress
              </span>
            </div>

            <div className="mt-6 flex items-end justify-between gap-3">
              <div>
                <div className="font-display text-[48px] font-bold leading-none tracking-tight text-bone sm:text-[56px]">
                  {String(Math.min(data.current_matchweek, data.max_matchweeks)).padStart(2, '0')}
                  <span className="ml-1 text-[16px] font-semibold text-sage">/ {data.max_matchweeks}</span>
                </div>
                <p className="mt-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-sage">Season progress</p>
              </div>
              <Calendar size={24} className="mb-1 text-brass/75" aria-hidden="true" />
            </div>

            <ProgressBar
              pct={(Math.min(data.current_matchweek, data.max_matchweeks) / Math.max(1, data.max_matchweeks)) * 100}
              toneClass="bg-gradient-to-r from-pitchtone to-brass"
              className="mt-4 h-2"
            />

            <div className="mt-5 grid grid-cols-2 gap-2 border-y border-white/[0.08] py-4">
              <div>
                <strong className="block font-display text-[26px] font-bold leading-none text-bone">{data.upcoming_fixtures?.length ?? 0}</strong>
                <span className="mt-1 block text-[10px] uppercase tracking-[0.12em] text-sage">Featured fixtures</span>
              </div>
              <div className="border-l border-white/[0.08] pl-4">
                <strong className="block font-display text-[26px] font-bold leading-none text-bone">{data.league_leaders?.length ?? 0}</strong>
                <span className="mt-1 block text-[10px] uppercase tracking-[0.12em] text-sage">Domestic leagues</span>
              </div>
            </div>

            <p className="mt-4 text-[12px] leading-relaxed text-sage">
              Results, squad development and the transfer market move together as the world calendar advances.
            </p>

            <button
              type="button"
              onClick={onOpenCompetitions}
              className="mt-5 flex w-full items-center justify-between gap-3 border-t border-white/[0.08] pt-4 text-left transition-colors hover:text-bone"
            >
              <span className="min-w-0">
                <span className="block text-[10px] font-bold uppercase tracking-[0.14em] text-brass/80">European competitions</span>
                <strong className="mt-1 block truncate text-[13px] font-semibold text-bone">{data.europe?.name || 'UEFA competitions'}</strong>
                <span className="mt-0.5 block truncate text-[11px] text-sage">{data.europe?.stage || 'Competition overview'}</span>
              </span>
              <ArrowRight size={16} className="shrink-0 text-brass" aria-hidden="true" />
            </button>
          </section>
        </div>
      </div>

      {/* Live career intelligence — every value is supplied by the simulation. */}
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3">
        <div className="console-card p-4 min-h-[132px]">
          <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-[0.14em] text-brass/70">
            <span>Top performer</span><TrendingUp size={14} />
          </div>
          {data.top_scorer ? (
            <button type="button" onClick={() => openPlayer(data.top_scorer!.player_id)} className="mt-4 w-full text-left group">
              <p className="font-display text-[20px] font-bold text-bone group-hover:text-brass truncate">{data.top_scorer.full_name}</p>
              <p className="mt-1 text-[12px] text-sage">{data.top_scorer.club_short || data.top_scorer.club_name} · {data.top_scorer.goals} goals · {data.top_scorer.assists} assists</p>
            </button>
          ) : <p className="mt-4 text-[12px] text-sage">No scoring leader has emerged yet.</p>}
        </div>

        <button type="button" onClick={onOpenCompetitions} className="console-card p-4 min-h-[132px] text-left group">
          <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-[0.14em] text-brass/70">
            <span>European campaign</span><Trophy size={14} />
          </div>
          <p className="mt-4 font-display text-[20px] font-bold text-bone group-hover:text-brass truncate">{data.europe.name}</p>
          <p className="mt-1 text-[12px] text-sage">{data.europe.stage}{data.europe.champion ? ` · ${data.europe.champion.short_name} champions` : ''}</p>
        </button>

        <div className="console-card p-4 min-h-[132px]">
          <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-[0.14em] text-brass/70">
            <span>Medical report</span><AlertTriangle size={14} />
          </div>
          {data.injuries.length > 0 ? (
            <button type="button" onClick={() => openPlayer(data.injuries[0].player_id)} className="mt-4 w-full text-left group">
              <p className="font-display text-[20px] font-bold text-bone group-hover:text-brass truncate">{data.injuries[0].full_name}</p>
              <p className="mt-1 text-[12px] text-sage">{data.injuries[0].club_short} · {data.injuries[0].injury} · {data.injuries[0].injured_matches} matches</p>
            </button>
          ) : <p className="mt-4 text-[12px] text-sage">No injuries are currently reported.</p>}
        </div>

        <button type="button" onClick={onOpenTransfers} className="console-card p-4 min-h-[132px] text-left group">
          <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-[0.14em] text-brass/70">
            <span>Latest major deal</span><Coins size={14} />
          </div>
          {data.biggest_transfers.length > 0 ? (
            <>
              <p className="mt-4 font-display text-[20px] font-bold text-bone group-hover:text-brass truncate">{data.biggest_transfers[0].player_name}</p>
              <p className="mt-1 text-[12px] text-sage">{data.biggest_transfers[0].seller_short} → {data.biggest_transfers[0].buyer_short} · {data.biggest_transfers[0].formatted_fee}</p>
            </>
          ) : <p className="mt-4 text-[12px] text-sage">No completed major transfers yet.</p>}
        </button>
      </div>

      {((data.power_rankings?.length ?? 0) > 0 || (data.wonderkids?.length ?? 0) > 0 || (data.sackings?.length ?? 0) > 0) && (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
          <div className="console-card p-5">
            <h3 className="font-display text-[16px] font-bold text-bone">World power rankings</h3>
            <p className="mt-1 text-[12px] text-sage">Live squad rating, table points and UEFA coefficient.</p>
            <ol className="mt-3 space-y-1.5">
              {(data.power_rankings ?? []).slice(0, 8).map((club, idx) => (
                <li key={club.club_id}>
                  <button type="button" onClick={() => onViewSquad(club.club_id)} className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left hover:bg-white/[0.04]">
                    <span className="w-5 font-mono text-[11px] text-sage">{club.rank || idx + 1}</span>
                    <ClubCrest club={asClub(club)} size={20} />
                    <span className="min-w-0 flex-1 truncate text-[12px] font-semibold text-bone">{club.short_name}</span>
                    <span className="font-mono text-[11px] text-sage">{club.ovr} OVR</span>
                  </button>
                </li>
              ))}
            </ol>
          </div>
          <div className="console-card p-5">
            <h3 className="font-display text-[16px] font-bold text-bone">Wonderkid watch</h3>
            <ol className="mt-3 space-y-1.5">
              {(data.wonderkids ?? []).slice(0, 6).map((player) => (
                <li key={player.player_id}>
                  <button type="button" onClick={() => openPlayer(player.player_id)} className="flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left hover:bg-white/[0.04]">
                    <span className="min-w-0 truncate text-[12px] font-semibold text-bone">{player.full_name}</span>
                    <span className="shrink-0 font-mono text-[11px] text-sage">{player.club_short || player.club_name} · {player.ovr}</span>
                  </button>
                </li>
              ))}
              {(data.wonderkids ?? []).length === 0 && <li className="text-[12px] text-sage">No wonderkid minutes logged yet.</li>}
            </ol>
          </div>
          <div className="console-card p-5">
            <h3 className="font-display text-[16px] font-bold text-bone">Manager changes</h3>
            <ul className="mt-3 space-y-2">
              {(data.sackings ?? []).slice(0, 5).map((row, idx) => (
                <li key={`${row.club_id}-${idx}`} className="text-[12px]">
                  <button type="button" onClick={() => onViewSquad(row.club_id)} className="w-full text-left hover:text-brass">
                    <span className="font-semibold text-bone">{row.club_name}</span>
                    <span className="block text-sage">{row.old_manager} → {row.new_manager}</span>
                  </button>
                </li>
              ))}
              {(data.sackings ?? []).length === 0 && <li className="text-[12px] text-sage">No managerial changes this season.</li>}
            </ul>
          </div>
        </div>
      )}

      {(data.upcoming_fixtures?.length ?? 0) > 1 && (
        <div className="console-card p-5">
          <div className="flex items-center justify-between pb-3 border-b border-[#214332]">
            <h3 className="font-display text-[16px] font-bold text-bone">Important fixtures</h3>
            <button type="button" onClick={() => onOpenFixture(data.upcoming_fixtures[1])} className="text-[12px] font-semibold text-brass">Open Matches</button>
          </div>
          <div className="mt-3 grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-2">
            {data.upcoming_fixtures.slice(0, 6).map((fixture) => (
              <button
                key={fixture.id || fixture.fixture_id}
                type="button"
                onClick={() => onOpenFixture(fixture)}
                className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 rounded-md border border-white/10 bg-black/15 px-3 py-2 text-left hover:border-brass/40"
              >
                <span className="truncate text-[12px] font-semibold text-bone">{fixture.home?.short_name}</span>
                <span className="font-mono text-[11px] text-brass">vs</span>
                <span className="truncate text-right text-[12px] font-semibold text-bone">{fixture.away?.short_name}</span>
                <span className="col-span-3 text-[10px] uppercase tracking-[0.12em] text-sage">{fixture.competition.replace(/-/g, ' ')}{fixture.importance ? ` · ${fixture.importance}` : ''}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* BOTTOM ROW: 3-COLUMN BROADCAST STORYLINES & NEWS FEED */}
      <div className="console-card p-6">
        <div className="flex items-center justify-between pb-4 border-b border-[#214332]">
          <div className="flex items-center gap-2.5">
            <Newspaper size={18} className="text-brass" />
            <h3 className="font-display text-[18px] font-bold text-bone tracking-tight">European Press & Storylines</h3>
          </div>
          <button
            type="button"
            onClick={onOpenInbox}
            className="text-[12px] font-semibold text-brass hover:text-[#FFE082] flex items-center gap-1 transition-colors"
          >
            Open News <ArrowRight size={13} />
          </button>
        </div>

        <div className="mt-5 grid grid-cols-1 md:grid-cols-3 gap-5">
          {storylines.length > 0 ? (
            storylines.slice(0, 3).map((story, i) => {
              const kicker = i === 0 ? 'Top Story' : i === 1 ? 'Market Wire' : 'Dugout Report';
              return (
                <div
                  key={i}
                  className="p-4 rounded-xl bg-[#0D1C15] border border-[#214332] hover:border-[#386B53] transition-colors flex flex-col justify-between"
                >
                  <div>
                    <span className="eyebrow !text-brass text-[11px] font-bold uppercase tracking-wider block mb-1">
                      {kicker}
                    </span>
                    <p className="font-display text-[15px] font-semibold text-bone leading-snug">
                      {story}
                    </p>
                  </div>
                  <div className="mt-4 flex items-center justify-between text-[11px] text-sage/70 font-mono">
                    <span className="flex items-center gap-1"><Clock size={11} /> MW {data.current_matchweek}</span>
                    <span className="text-brass/80">Exclusive</span>
                  </div>
                </div>
              );
            })
          ) : (
            <div className="col-span-3 text-center py-6 text-sage text-[13px]">
              No breaking storylines this matchweek.
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
