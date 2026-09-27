import React, { useEffect, useMemo, useRef, useState } from 'react';
import { ArrowRight, CalendarDays, ChartNoAxesCombined, CircleDot, Zap } from 'lucide-react';
import type { Club, Fixture } from '../../types';
import { fetchFixture, fetchFixtureSummaries, fetchNationsCup, simulateFixture } from '../../services/api';
import { useAsyncData } from '../../hooks/useAsyncData';
import { cx } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';
import { ClubCrest, ErrorState, LoadingState } from '../ui/ui';
import { MatchBrowser } from '../prematch/MatchBrowser';
import { PreMatchModal } from '../prematch/PreMatchModal';
import { PostMatchModal } from '../postmatch/PostMatchBroadcast';
import { NationsMatchModal } from '../competitions/NationsMatchModal';
import type { NationsCupFixture } from '../../types';
import { Globe2 } from 'lucide-react';

interface SimulationCentreTabProps {
  careerKey: number;
  selectedFixtureId: string | null;
  onSelectFixture: (fixtureId: string) => void;
  onWeekAdvanced: () => void;
  onShowToast: (message: string) => void;
  onViewCompetition: () => void;
  onViewClub: (club: Club) => void;
}

function score(fixture: Fixture): string {
  if (fixture.home_goals == null || fixture.away_goals == null) return 'VS';
  return `${fixture.home_goals}–${fixture.away_goals}`;
}

export const SimulationCentreTab: React.FC<SimulationCentreTabProps> = ({
  careerKey,
  selectedFixtureId,
  onSelectFixture,
  onWeekAdvanced,
  onShowToast,
  onViewCompetition,
  onViewClub,
}) => {
  const { data: slate, loading: slateLoading, error: slateError, reload: reloadSlate } = useAsyncData(fetchFixtureSummaries, [careerKey]);
  const fallbackFixtureId = slate?.fixtures.find((fixture) => fixture.status === 'scheduled')?.id ?? slate?.fixtures[0]?.id ?? null;
  const activeFixtureId = selectedFixtureId ?? fallbackFixtureId;
  const { data: selectedFixture, loading: fixtureLoading, error: fixtureError, reload: reloadSelected } = useAsyncData(
    () => activeFixtureId ? fetchFixture(activeFixtureId) : Promise.resolve(null),
    [careerKey, activeFixtureId],
  );
  const [previewFixture, setPreviewFixture] = useState<Fixture | null>(null);
  const { data: nations, reload: reloadNations } = useAsyncData(fetchNationsCup, [careerKey]);
  const [nationsFixture, setNationsFixture] = useState<NationsCupFixture | null>(null);
  const [reportFixture, setReportFixture] = useState<Fixture | null>(null);
  const [busy, setBusy] = useState(false);
  const simulationLock = useRef(false);
  const autoOpenedReport = useRef<string | null>(null);

  useEffect(() => {
    if (activeFixtureId && activeFixtureId !== selectedFixtureId) onSelectFixture(activeFixtureId);
  }, [activeFixtureId, selectedFixtureId, onSelectFixture]);

  useEffect(() => {
    if (selectedFixture?.status !== 'finished' || autoOpenedReport.current === selectedFixture.id) return;
    autoOpenedReport.current = selectedFixture.id;
    setReportFixture(selectedFixture);
  }, [selectedFixture]);

  const weekSummary = useMemo(() => {
    if (!slate) return { played: 0, remaining: 0, competitions: 0 };
    return {
      played: slate.fixtures.filter((fixture) => fixture.status === 'finished').length,
      remaining: slate.fixtures.filter((fixture) => fixture.status === 'scheduled').length,
      competitions: new Set(slate.fixtures.map((fixture) => fixture.competition)).size,
    };
  }, [slate]);

  const selectFixture = (fixture: Fixture) => {
    soundManager.playClick();
    onSelectFixture(fixture.id);
    if (fixture.status === 'finished') {
      setReportFixture(fixture);
      autoOpenedReport.current = fixture.id;
    } else {
      setReportFixture(null);
    }
  };

  const resolveFixture = async (fixture: Fixture) => {
    if (simulationLock.current || fixture.status !== 'scheduled') return;
    simulationLock.current = true;
    setBusy(true);
    soundManager.playWhistle();
    try {
      const result = await simulateFixture(fixture.id);
      if (result.status === 'error') {
        onShowToast(result.message || 'This fixture could not be simulated.');
        return;
      }
      const report = await fetchFixture(fixture.id);
      if (!report || report.status !== 'finished') {
        onShowToast('Result saved, but the full report is not ready yet. Use Retry to reload it.');
        onWeekAdvanced();
        reloadSlate();
        reloadSelected();
        return;
      }
      setPreviewFixture(null);
      setReportFixture(report);
      autoOpenedReport.current = report.id;
      onShowToast(`${report.home.short_name} ${score(report)} ${report.away.short_name}. Full-time report ready.`);
      onWeekAdvanced();
      reloadSlate();
      reloadSelected();
    } catch (error) {
      onShowToast(error instanceof Error ? error.message : 'Simulation failed.');
    } finally {
      simulationLock.current = false;
      setBusy(false);
    }
  };

  const openReport = () => {
    if (!selectedFixture) return;
    autoOpenedReport.current = selectedFixture.id;
    setReportFixture(selectedFixture);
  };

  return (
    <div className="space-y-4 animate-fade-in" data-simulation-centre>
      <section className="console-hero overflow-hidden p-5 sm:p-7">
        <div className="relative z-10 flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
          <div className="max-w-2xl">
            <p className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-[0.18em] text-brass">
              <CircleDot size={13} aria-hidden="true" /> Match Centre · instant simulation
            </p>
            <h2 className="mt-2 font-display text-[30px] font-bold leading-tight tracking-tight text-bone sm:text-[38px]">
              Pick a fixture. Get the full story.
            </h2>
            <p className="mt-2 max-w-xl text-[13px] leading-relaxed text-sage">
              Every club is simulated by the world engine. Choose a match to inspect its setup, resolve the result in one step, then open the complete report.
            </p>
          </div>
          <div className="grid grid-cols-3 gap-2 sm:min-w-[390px]">
            <CentreMetric label="Matchweek" value={slate ? `${Math.min(slate.matchweek, slate.max_matchweeks)} / ${slate.max_matchweeks}` : '—'} />
            <CentreMetric label="Played" value={String(weekSummary.played)} />
            <CentreMetric label="To play" value={String(weekSummary.remaining)} />
          </div>
        </div>
        <div className="relative z-10 mt-5 flex flex-wrap items-center justify-between gap-3 border-t border-white/[0.08] pt-4">
          <p className="inline-flex items-center gap-2 text-[11px] text-sage">
            <CalendarDays size={13} className="text-brass" aria-hidden="true" />
            {slate ? `${slate.season_name.replace('-', '–')} · ${weekSummary.competitions} competitions on this slate` : slateError ? 'Current schedule unavailable' : slateLoading ? 'Loading current slate…' : 'No current slate'}
          </p>
          <button type="button" onClick={onViewCompetition} className="inline-flex min-h-9 items-center gap-1.5 px-3 text-[12px] font-semibold text-brass hover:text-bone">
            Competition boards <ArrowRight size={14} aria-hidden="true" />
          </button>
        </div>
      </section>

      <div className="grid grid-cols-1 gap-4 2xl:grid-cols-[minmax(0,1.08fr)_minmax(370px,.92fr)]">
        <MatchBrowser careerKey={careerKey} selectedFixtureId={activeFixtureId ?? undefined} onSelect={selectFixture} />

        <section className="console-card min-w-0 overflow-hidden">
          <div className="border-b border-white/10 px-4 py-3">
            <p className="text-[10px] font-bold uppercase tracking-[0.14em] text-brass/70">Selected fixture</p>
            <h3 className="font-display text-[18px] font-bold text-bone">Match preview and result</h3>
          </div>
          {fixtureLoading && <LoadingState message="Loading fixture details…" />}
          {fixtureError && <div className="p-4"><ErrorState message={fixtureError} onRetry={reloadSelected} /></div>}
          {!fixtureLoading && !fixtureError && !selectedFixture && (
            <div className="p-8 text-center">
              <ChartNoAxesCombined size={30} className="mx-auto text-brass/70" aria-hidden="true" />
              <p className="mt-3 text-[13px] font-semibold text-bone">No fixture selected</p>
              <p className="mt-1 text-[12px] text-sage">Choose a match from the current slate to review or simulate it.</p>
            </div>
          )}
          {selectedFixture && (
            <div className="p-4 sm:p-5">
              <div className="flex items-center justify-between gap-2 text-[10px] font-semibold uppercase tracking-[0.12em] text-sage">
                <span>{selectedFixture.competition.replace(/-/g, ' ')} · {selectedFixture.stage || `MW ${selectedFixture.matchweek}`}</span>
                <span className={cx('rounded-full border px-2.5 py-1', selectedFixture.status === 'finished' ? 'border-pitchtone/30 bg-pitchtone/10 text-pitchtone' : 'border-brass/30 bg-brass/10 text-brass')}>
                  {selectedFixture.status === 'finished' ? 'Full time' : 'Scheduled'}
                </span>
              </div>
              <div className="mt-6 grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2 sm:gap-4">
                <TeamLockup club={selectedFixture.home} align="right" />
                <div className="min-w-[72px] text-center">
                  <p className="font-display text-[28px] font-bold tabular-nums text-bone sm:text-[34px]">{score(selectedFixture)}</p>
                  <p className="mt-1 text-[9px] uppercase tracking-[0.12em] text-sage">{selectedFixture.status === 'finished' ? 'Final score' : `Matchweek ${selectedFixture.matchweek}`}</p>
                </div>
                <TeamLockup club={selectedFixture.away} align="left" />
              </div>
              <div className="mt-5 rounded-md border border-white/[0.07] bg-black/15 p-3 text-[12px] leading-relaxed text-sage">
                {selectedFixture.preview?.kickoff_note || `${selectedFixture.importance || selectedFixture.stage || 'Scheduled fixture'} in ${selectedFixture.competition.replace(/-/g, ' ')}. The report includes events, lineups, statistics and competition impact.`}
              </div>
              <div className="mt-4 flex flex-wrap gap-2">
                {selectedFixture.status === 'scheduled' ? (
                  <>
                    <button type="button" disabled={busy} onClick={() => setPreviewFixture(selectedFixture)} className="min-h-10 border border-line bg-cardLight px-4 text-[12px] font-semibold text-bone hover:bg-cardHover disabled:opacity-50">
                      Inspect match setup
                    </button>
                    <button type="button" disabled={busy} onClick={() => void resolveFixture(selectedFixture)} className="gold-btn min-h-10">
                      <Zap size={15} aria-hidden="true" /> {busy ? 'Simulating…' : 'Simulate match'}
                    </button>
                  </>
                ) : (
                  <button type="button" onClick={openReport} className="gold-btn min-h-10">Open full report <ArrowRight size={14} aria-hidden="true" /></button>
                )}
              </div>
              <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-[11px]">
                <button type="button" onClick={() => onViewClub(selectedFixture.home)} className="text-sage hover:text-brass">View {selectedFixture.home.short_name}</button>
                <button type="button" onClick={() => onViewClub(selectedFixture.away)} className="text-sage hover:text-brass">View {selectedFixture.away.short_name}</button>
              </div>
              <div className="mt-4 border-t border-white/[0.07] pt-3 text-[10px] text-sage/70">
                {selectedFixture.date_label || `Matchweek ${selectedFixture.matchweek}`}{selectedFixture.home.home_stadium ? ` · ${selectedFixture.home.home_stadium}` : ''}
              </div>
            </div>
          )}
        </section>
      </div>

      {nations && nations.fixtures.length > 0 && (
        <section className="console-card overflow-hidden" data-nations-slate>
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line bg-cardLight/50 px-4 py-3">
            <div className="min-w-0">
              <h3 className="flex items-center gap-2 font-display text-[16px] font-bold text-bone"><Globe2 size={16} className="text-brass" aria-hidden="true" /> European Nations Cup</h3>
              <p className="mt-0.5 text-[11px] text-sage">International fixtures run on their own schedule; results never touch club statistics.</p>
            </div>
            <span className="shrink-0 font-mono text-[10px] uppercase tracking-[0.12em] text-sage">{nations.stage}</span>
          </div>
          <ul className="divide-y divide-line/70">
            {nations.fixtures.map((fixture) => {
              const finished = fixture.status === 'finished';
              const home = fixture.home?.name ?? fixture.home_id;
              const away = fixture.away?.name ?? fixture.away_id;
              const line = finished && fixture.home_goals != null && fixture.away_goals != null ? `${fixture.home_goals}–${fixture.away_goals}` : 'vs';
              const pens = finished && fixture.home_penalties != null && fixture.away_penalties != null ? ` · Pens ${fixture.home_penalties}–${fixture.away_penalties}` : '';
              return (
                <li key={fixture.id} className="flex flex-wrap items-center gap-3 px-4 py-2.5">
                  <span className="min-w-[54px] font-mono text-[10px] text-sage">MW {fixture.matchweek}</span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-[12.5px] font-semibold text-bone">{home} <span className="px-1 font-mono text-brass">{line}</span> {away}</span>
                    <span className="block truncate text-[10px] text-sage">{fixture.stage}{pens}</span>
                  </span>
                  <button type="button" onClick={() => { soundManager.playClick(); setNationsFixture(fixture); }} className={cx('shrink-0 rounded-sm border px-3 py-1.5 text-[11px] font-semibold transition-colors', finished ? 'border-brass/40 bg-brass/10 text-brass hover:bg-brass/20' : 'border-line bg-cardLight text-bone hover:bg-cardHover')}>
                    {finished ? 'Report' : 'Inspect / simulate'}
                  </button>
                </li>
              );
            })}
          </ul>
        </section>
      )}

      <PreMatchModal fixture={previewFixture} onClose={() => setPreviewFixture(null)} busy={busy} onSimulate={resolveFixture} />
      <PostMatchModal fixture={reportFixture} onClose={() => setReportFixture(null)} />
      <NationsMatchModal fixture={nationsFixture} onClose={() => setNationsFixture(null)} onPlayed={() => { reloadNations(); onWeekAdvanced(); }} />
    </div>
  );
};

const CentreMetric: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="border border-white/[0.08] bg-black/20 px-3 py-2.5">
    <p className="text-[9px] font-bold uppercase tracking-[0.12em] text-sage/70">{label}</p>
    <p className="mt-1 font-mono text-[15px] font-bold tabular-nums text-bone sm:text-[17px]">{value}</p>
  </div>
);

const TeamLockup: React.FC<{ club: Club; align: 'left' | 'right' }> = ({ club, align }) => (
  <div className={cx('flex min-w-0 items-center gap-2 sm:gap-3', align === 'right' ? 'justify-end text-right' : 'justify-start')}>
    {align === 'right' && <span className="min-w-0 truncate font-display text-[14px] font-bold text-bone sm:text-[18px]">{club.short_name}</span>}
    <ClubCrest club={club} size={42} className="shrink-0 !border-0 !bg-transparent sm:!h-[54px] sm:!w-[54px]" />
    {align === 'left' && <span className="min-w-0 truncate font-display text-[14px] font-bold text-bone sm:text-[18px]">{club.short_name}</span>}
  </div>
);
