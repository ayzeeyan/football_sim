import React, { useEffect, useId, useMemo, useState } from 'react';
import {
  BarChart3, Clock3, CloudSun, Gauge, MapPin, ShieldAlert, Swords, Users,
} from 'lucide-react';
import type { Fixture, Player } from '../../types';
import { cx } from '../../lib/format';
import { FixtureActions } from './FixtureActions';
import { ClubCrest, FormPips, Modal } from '../ui/ui';
import { prettyCompetitionName, usePlayerSheet } from '../clubs/PlayerSheet';
import { getWeatherDetails } from '../matches/MatchCard';
import { normalizeFormation } from '../../lib/tactics';
import { FormationPitch } from '../matches/FormationPitch';
import { FormGuide, HeadToHeadSummary, MatchPulseCard, PlayersToWatch } from './MatchCentreInsights';
import { HeadToHeadPanel } from './HeadToHeadPanel';
import { LeagueContextCard } from './LeagueContextCard';

type PreMatchTab = 'overview' | 'lineups' | 'tactics' | 'news' | 'h2h' | 'context';

interface PreMatchModalProps {
  fixture: Fixture | null;
  onClose: () => void;
  onSimulate?: (f: Fixture) => void;
  busy?: boolean;
}

const TABS: Array<{ id: PreMatchTab; label: string }> = [
  { id: 'overview', label: 'Overview' },
  { id: 'lineups', label: 'Lineups' },
  { id: 'tactics', label: 'Tactics' },
  { id: 'news', label: 'Team News' },
  { id: 'h2h', label: 'Head to Head' },
  { id: 'context', label: 'Competition Context' },
];

function ordinal(position?: number | null): string {
  if (!position) return 'Position unavailable';
  const mod = position % 100;
  const suffix = mod >= 11 && mod <= 13 ? 'th' : position % 10 === 1 ? 'st' : position % 10 === 2 ? 'nd' : position % 10 === 3 ? 'rd' : 'th';
  return `${position}${suffix}`;
}

function importance(fixture: Fixture): string {
  if (fixture.importance) return fixture.importance;
  if (fixture.stage?.toLowerCase() === 'final') return 'Final';
  if (fixture.is_derby || fixture.derby_name) return 'Derby';
  return fixture.stage || 'Scheduled fixture';
}

export function handleFixtureTabKey(event: React.KeyboardEvent<HTMLDivElement>) {
  const tabs = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('[role="tab"]'));
  const index = tabs.indexOf(event.target as HTMLButtonElement);
  if (index < 0 || !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
  event.preventDefault();
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (index + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length;
  tabs[next]?.focus();
  tabs[next]?.click();
}

function AbsenceList({ players }: { players: Player[] }) {
  if (!players.length) return <p className="text-[12px] text-sage">No absences reported.</p>;
  return (
    <ul className="divide-y divide-white/[0.07]">
      {players.map((player) => (
        <li key={player.player_id} className="flex items-center justify-between gap-3 py-2 text-[12px]">
          <span className="font-semibold text-bone">{player.full_name}</span>
          <span className="text-ember">{player.availability || player.injury || ((player.suspended_matches ?? 0) > 0 ? 'Suspended' : 'Unavailable')}</span>
        </li>
      ))}
    </ul>
  );
}

export const PreMatchModal: React.FC<PreMatchModalProps> = ({
  fixture, onClose, onSimulate, busy,
}) => {
  const { openPlayer } = usePlayerSheet();
  const [tab, setTab] = useState<PreMatchTab>('overview');
  const tabsId = useId().replace(/:/g, '');
  useEffect(() => { if (fixture) setTab('overview'); }, [fixture?.id]);

  const preview = fixture?.preview;
  const weather = getWeatherDetails(fixture?.weather);
  const homeXi = preview?.home_xi ?? [];
  const awayXi = preview?.away_xi ?? [];
  const homeConcerns = useMemo(() => homeXi.filter((p) => (typeof p.fitness === 'number' && p.fitness < 65) || (typeof p.morale === 'number' && p.morale < 55)), [homeXi]);
  const awayConcerns = useMemo(() => awayXi.filter((p) => (typeof p.fitness === 'number' && p.fitness < 65) || (typeof p.morale === 'number' && p.morale < 55)), [awayXi]);

  return (
    <Modal open={!!fixture} onClose={onClose} maxWidth="max-w-7xl" fillViewport>
      {fixture && (
        <div className="flex min-h-0 flex-1 flex-col">
          <div className={cx('prematch-hero relative shrink-0 overflow-hidden border-b border-brass/20 bg-gradient-to-br from-[#1c4937] via-[#0d271b] to-[#06110c] px-4 py-4 sm:px-7 sm:py-5', tab !== 'overview' && 'prematch-hero-compact')}>
            <div className="pointer-events-none absolute inset-0 opacity-60 [background-image:linear-gradient(rgba(255,255,255,.025)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.025)_1px,transparent_1px)] [background-size:32px_32px]" />
            <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_120%,rgba(243,230,196,0.16),transparent_50%)]" />
            <button type="button" onClick={onClose} aria-label="Close pre-match hub" className="absolute right-3 top-3 z-10 grid h-8 w-8 place-items-center rounded-sm border border-white/10 bg-black/20 text-sage hover:text-bone">×</button>
            <div className="relative flex flex-wrap items-center justify-between gap-3 pr-10 text-[10px] font-semibold uppercase tracking-[0.14em] text-sage">
              <span>{prettyCompetitionName(fixture.competition)} · {fixture.stage || `Matchweek ${fixture.matchweek}`}</span>
              <span className="rounded-full border border-brass/30 bg-brass/10 px-3 py-1 text-brass">{importance(fixture)}</span>
            </div>
            <div className={cx('prematch-compact-fixture relative mt-2 items-center justify-center gap-4', tab === 'overview' ? 'hidden' : 'flex')}>
              <span className="flex min-w-0 items-center justify-end gap-2 text-right"><span className="min-w-0"><strong className="block truncate font-display text-[16px] text-bone">{fixture.home.short_name}</strong><small className="block text-[9px] text-sage">{normalizeFormation(preview?.home_formation ?? fixture.home_formation)}</small></span><ClubCrest club={fixture.home} size={30} className="!border-0 !bg-transparent" /></span>
              <span className="rounded-full border border-white/10 bg-black/20 px-2.5 py-1 text-[9px] font-bold text-brass">VS</span>
              <span className="flex min-w-0 items-center gap-2"><ClubCrest club={fixture.away} size={30} className="!border-0 !bg-transparent" /><span className="min-w-0"><strong className="block truncate font-display text-[16px] text-bone">{fixture.away.short_name}</strong><small className="block text-[9px] text-sage">{normalizeFormation(preview?.away_formation ?? fixture.away_formation)}</small></span></span>
            </div>
            <div className="prematch-score-lockup relative mx-auto mt-4 grid max-w-4xl grid-cols-[1fr_auto_1fr] items-center gap-3 sm:gap-10">
              <div className="flex min-w-0 flex-col items-center text-center">
                <ClubCrest club={fixture.home} size={76} className="prematch-crest !border-0 !bg-transparent" />
                <h2 className="prematch-team-name mt-2 max-w-full truncate font-display text-[21px] font-bold text-bone sm:text-[28px]">{fixture.home.club_name}</h2>
                <div className="prematch-team-meta"><p className="text-[11px] text-sage">{ordinal(preview?.home_pos)}{preview ? ` · ${preview.home_pts} pts` : ''}</p><div className="mt-2"><FormPips form={preview?.home_form ?? fixture.home.form ?? []} size="sm" /></div></div>
              </div>
              <div className="text-center">
                <p className="rounded-full border border-white/10 bg-black/25 px-3 py-1 text-[9px] font-bold uppercase tracking-[0.16em] text-sage">Upcoming</p>
                <div className="mt-2 font-display text-[26px] font-bold text-brass">VS</div>
                <p className="mt-1 whitespace-nowrap text-[10px] uppercase tracking-[0.15em] text-sage">MW {fixture.matchweek}</p>
              </div>
              <div className="flex min-w-0 flex-col items-center text-center">
                <ClubCrest club={fixture.away} size={76} className="prematch-crest !border-0 !bg-transparent" />
                <h2 className="prematch-team-name mt-2 max-w-full truncate font-display text-[21px] font-bold text-bone sm:text-[28px]">{fixture.away.club_name}</h2>
                <div className="prematch-team-meta"><p className="text-[11px] text-sage">{ordinal(preview?.away_pos)}{preview ? ` · ${preview.away_pts} pts` : ''}</p><div className="mt-2"><FormPips form={preview?.away_form ?? fixture.away.form ?? []} size="sm" /></div></div>
              </div>
            </div>
            <div className="prematch-meta-strip relative mx-auto mt-4 flex w-fit max-w-full flex-wrap justify-center gap-x-5 gap-y-2 rounded-full border border-white/[0.08] bg-black/20 px-4 py-2 text-[11px] text-sage">
              <span className="inline-flex items-center gap-1.5"><MapPin size={12} /> <span className="max-w-[220px] truncate">{preview?.venue || fixture.home.home_stadium}</span></span>
              {preview?.capacity ? <span className="hidden items-center gap-1.5 sm:inline-flex"><Users size={12} /> {preview.capacity.toLocaleString()}</span> : null}
              {weather ? <span className="inline-flex items-center gap-1.5"><CloudSun size={12} /> {weather.label}</span> : null}
              <span className="inline-flex items-center gap-1.5"><Clock3 size={12} /> {fixture.date_label || `Matchweek ${fixture.matchweek}`}</span>
            </div>
          </div>

          <div className="section-tabs shrink-0 px-2 sm:px-4" role="tablist" aria-label="Pre-match information" onKeyDown={handleFixtureTabKey}>
            {TABS.map((item) => (
              <button key={item.id} id={`${tabsId}-tab-${item.id}`} type="button" role="tab" aria-selected={tab === item.id} aria-controls={`${tabsId}-panel`} tabIndex={tab === item.id ? 0 : -1} onClick={() => setTab(item.id)} className={cx('section-tab', tab === item.id && 'section-tab-active')}>{item.label}</button>
            ))}
          </div>

          <div id={`${tabsId}-panel`} role="tabpanel" aria-labelledby={`${tabsId}-tab-${tab}`} tabIndex={0} className="min-h-0 flex-1 overflow-y-auto p-4 sm:p-5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-brass">
            {tab === 'overview' && (
              <div className="grid grid-cols-1 gap-3 lg:grid-cols-12">
                <FormGuide fixture={fixture} />
                <MatchPulseCard fixture={fixture} />
                <section className="console-card p-4 lg:col-span-7">
                  <h3 className="match-section-title"><Swords size={14} /> Match story</h3>
                  <p className="mt-3 text-[14px] leading-relaxed text-bone/90">{preview?.kickoff_note || `${fixture.home.club_name} face ${fixture.away.club_name} in ${prettyCompetitionName(fixture.competition)}.`}</p>
                  <div className="mt-4 grid grid-cols-2 gap-2 text-[11px]">
                    <div className="rounded-md bg-black/15 p-3"><span className="block text-sage">Home setup</span><strong className="mt-1 block text-bone">{normalizeFormation(preview?.home_formation ?? fixture.home_formation)} · {fixture.home.manager?.style || 'Balanced'}</strong></div>
                    <div className="rounded-md bg-black/15 p-3 text-right"><span className="block text-sage">Away setup</span><strong className="mt-1 block text-bone">{normalizeFormation(preview?.away_formation ?? fixture.away_formation)} · {fixture.away.manager?.style || 'Balanced'}</strong></div>
                  </div>
                </section>
                <section className="console-card p-4 lg:col-span-5">
                  <h3 className="match-section-title"><ShieldAlert size={14} /> Availability</h3>
                  <div className="mt-3 grid grid-cols-2 gap-4"><div><p className="mb-1 text-[11px] font-bold text-bone">{fixture.home.short_name}</p><AbsenceList players={preview?.home_missing ?? []} /></div><div><p className="mb-1 text-[11px] font-bold text-bone">{fixture.away.short_name}</p><AbsenceList players={preview?.away_missing ?? []} /></div></div>
                </section>
                <PlayersToWatch fixture={fixture} onOpen={openPlayer} />
                <HeadToHeadSummary fixture={fixture} />
              </div>
            )}

            {tab === 'lineups' && (
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                {[{ club: fixture.home, xi: homeXi, formation: preview?.home_formation ?? fixture.home_formation, average: preview?.home_xi_avg }, { club: fixture.away, xi: awayXi, formation: preview?.away_formation ?? fixture.away_formation, average: preview?.away_xi_avg }].map(({ club, xi, formation, average }) => (
                  <div key={club.club_id} className="min-w-0">
                    <div className="mb-2 flex items-center justify-between gap-3 rounded-md border border-white/[0.08] bg-black/15 px-3 py-2">
                      <span className="flex min-w-0 items-center gap-2"><ClubCrest club={club} size={28} className="!border-0 !bg-transparent" /><span className="min-w-0"><strong className="block truncate text-[13px] text-bone">{club.club_name}</strong><small className="block text-[9px] uppercase tracking-[0.12em] text-sage">{club.manager?.name || 'Manager unavailable'}</small></span></span>
                      <span className="flex items-center gap-2 text-right"><span><strong className="block text-[12px] text-bone">{normalizeFormation(formation)}</strong><small className="block text-[9px] text-sage">Expected</small></span>{average ? <strong className="rounded bg-brass px-2 py-1 font-mono text-[10px] text-ink">{average.toFixed(1)}</strong> : null}</span>
                    </div>
                    {xi.length ? <FormationPitch players={xi} formation={formation} compact onPlayerClick={(p) => openPlayer(p.player_id)} /> : <p className="grid h-[390px] place-items-center rounded-lg border border-line bg-ink/30 text-center text-[12px] text-sage sm:h-[430px]">Expected lineup unavailable.</p>}
                  </div>
                ))}
              </div>
            )}

            {tab === 'tactics' && (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                {[fixture.home, fixture.away].map((club) => <section key={club.club_id} className="console-card p-4"><div className="flex items-center gap-3"><ClubCrest club={club} size={38} /><div><h3 className="font-display text-[18px] font-bold text-bone">{club.club_name}</h3><p className="text-[11px] text-sage">{club.manager?.name || 'Manager unavailable'}</p></div></div><dl className="mt-4 grid grid-cols-2 gap-2 text-[12px]">{[['Formation', normalizeFormation(club === fixture.home ? preview?.home_formation ?? fixture.home_formation : preview?.away_formation ?? fixture.away_formation)], ['Tactic', club.manager?.tactic || 'Not supplied'], ['Style', club.manager?.style || 'Not supplied'], ['Focus', club.manager?.focus || 'Not supplied']].map(([label, value]) => <div key={label} className="bg-black/15 p-3"><dt className="text-sage">{label}</dt><dd className="mt-1 font-semibold text-bone">{value}</dd></div>)}</dl></section>)}
              </div>
            )}

            {tab === 'news' && <div className="grid grid-cols-1 gap-4 md:grid-cols-2">{[{ club: fixture.home, missing: preview?.home_missing ?? [], concerns: homeConcerns }, { club: fixture.away, missing: preview?.away_missing ?? [], concerns: awayConcerns }].map(({ club, missing, concerns }) => <section key={club.club_id} className="console-card p-4"><h3 className="font-display text-[18px] font-bold text-bone">{club.club_name}</h3><p className="mt-4 text-[10px] font-semibold uppercase tracking-[0.14em] text-brass/70">Unavailable</p><AbsenceList players={missing} /><p className="mt-4 text-[10px] font-semibold uppercase tracking-[0.14em] text-brass/70">Fitness and morale concerns</p>{concerns.length ? <ul className="mt-1 divide-y divide-white/[0.07]">{concerns.map((p) => <li key={p.player_id} className="flex justify-between py-2 text-[12px]"><span className="text-bone">{p.full_name}</span><span className="text-sage">Fit {p.fitness ?? '—'} · Morale {p.morale ?? '—'}</span></li>)}</ul> : <p className="mt-2 text-[12px] text-sage">No lineup concerns reported.</p>}</section>)}</div>}

            {tab === 'h2h' && <HeadToHeadPanel fixture={fixture} />}

            {tab === 'context' && <div className="grid grid-cols-1 gap-4 lg:grid-cols-2"><LeagueContextCard fixture={fixture} /><section className="console-card p-4"><h3 className="match-section-title"><BarChart3 size={14} /> Table context</h3><div className="mt-4 space-y-3">{[{ club: fixture.home, pos: preview?.home_pos, pts: preview?.home_pts }, { club: fixture.away, pos: preview?.away_pos, pts: preview?.away_pts }].map(({ club, pos, pts }) => <div key={club.club_id} className="flex items-center gap-3"><ClubCrest club={club} size={30} /><span className="min-w-0 flex-1 truncate text-[12px] font-semibold text-bone">{club.short_name}</span><strong className="text-[12px] text-brass">{ordinal(pos)}{preview ? ` · ${pts} pts` : ''}</strong></div>)}</div></section><section className="console-card p-4 lg:col-span-2"><h3 className="match-section-title"><Gauge size={14} /> Match context</h3><p className="mt-4 text-[13px] leading-relaxed text-sage">{preview?.kickoff_note || `${importance(fixture)} fixture in ${prettyCompetitionName(fixture.competition)}. No additional competition narrative was supplied by the save.`}</p></section></div>}
          </div>

          {fixture.status !== 'finished' && (
            <FixtureActions
              fixture={fixture}
              busy={busy}
              onSimulate={onSimulate}
            />
          )}
        </div>
      )}
    </Modal>
  );
};
