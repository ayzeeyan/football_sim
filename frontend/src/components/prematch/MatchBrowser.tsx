import React, { useMemo, useState } from 'react';
import type { Fixture } from '../../types';
import { fetchFixtureSummaries } from '../../services/api';
import { useAsyncData } from '../../hooks/useAsyncData';
import { ClubCrest, EmptyState, LoadingState } from '../ui/ui';
import { cx } from '../../lib/format';
import { prettyCompetitionName } from '../clubs/PlayerSheet';
import { soundManager } from '../../audio/webAudio';
import { LEAGUES_5 } from '../../lib/constants';

interface MatchBrowserProps {
  careerKey?: number;
  selectedFixtureId?: string;
  onSelect: (fixture: Fixture) => void;
}

const STATUS_FILTERS = ['all', 'scheduled', 'finished'] as const;
type StatusFilter = (typeof STATUS_FILTERS)[number];

export const MatchBrowser: React.FC<MatchBrowserProps> = ({ careerKey = 0, selectedFixtureId, onSelect }) => {
  const { data, loading, error } = useAsyncData(() => fetchFixtureSummaries(), [careerKey]);
  const [status, setStatus] = useState<StatusFilter>('all');
  const [competition, setCompetition] = useState('all');

  const competitions = useMemo(() => {
    const ids = new Set<string>();
    for (const fixture of data?.fixtures ?? []) ids.add(fixture.competition);
    return ['all', ...Array.from(ids)];
  }, [data]);

  const rows = useMemo(() => {
    const fixtures = data?.fixtures ?? [];
    return fixtures.filter((fixture) => {
      if (status !== 'all' && fixture.status !== status) return false;
      if (competition !== 'all' && fixture.competition !== competition) return false;
      return true;
    });
  }, [data, status, competition]);

  const featured = rows.filter((fixture) => fixture.status === 'scheduled');
  const featuredIds = new Set(featured.slice(0, 4).map((fixture) => fixture.id));
  const upcoming = rows.filter((fixture) => fixture.status === 'scheduled' && !featuredIds.has(fixture.id));
  const completed = rows.filter((fixture) => fixture.status === 'finished');

  return (
    <section className="console-card overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/10 px-4 py-3">
        <div>
          <p className="text-[10px] font-bold uppercase tracking-[0.14em] text-brass/70">Match browser</p>
          <h3 className="font-display text-[18px] font-bold text-bone">This matchweek</h3>
        </div>
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <label className="sr-only" htmlFor="match-status-filter">Status</label>
          <select id="match-status-filter" className="field h-9 text-[12px]" value={status} onChange={(e) => setStatus(e.target.value as StatusFilter)}>
            {STATUS_FILTERS.map((item) => <option key={item} value={item}>{item === 'all' ? 'All fixtures' : item}</option>)}
          </select>
          <label className="sr-only" htmlFor="match-comp-filter">Competition</label>
          <select id="match-comp-filter" className="field h-9 max-w-[12rem] text-[12px]" value={competition} onChange={(e) => setCompetition(e.target.value)}>
            {competitions.map((id) => (
              <option key={id} value={id}>{id === 'all' ? 'All competitions' : prettyCompetitionName(id)}</option>
            ))}
          </select>
        </div>
      </div>
      {loading && <LoadingState message="Loading the matchweek slate…" />}
      {error && <EmptyState message={error} />}
      {!loading && !error && rows.length === 0 && <EmptyState message="No fixtures match those filters." />}
      {!loading && rows.length > 0 && (
        <div className="max-h-[70vh] divide-y divide-white/[0.06] overflow-y-auto overscroll-contain">
          {status !== 'finished' && featured.length > 0 && (
            <div className="px-4 py-3">
              <p className="mb-2 text-[10px] font-bold uppercase tracking-[0.14em] text-brass/70">Featured</p>
              <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
                {featured.slice(0, 4).map((fixture) => (
                  <FixturePick key={fixture.id} fixture={fixture} selected={fixture.id === selectedFixtureId} onSelect={onSelect} />
                ))}
              </div>
            </div>
          )}
          {upcoming.length > 0 && (
            <FixtureGroup title={`Upcoming · ${upcoming.length}`} fixtures={upcoming} selectedFixtureId={selectedFixtureId} onSelect={onSelect} />
          )}
          {completed.length > 0 && (
            <FixtureGroup title={`Completed · ${completed.length}`} fixtures={completed} selectedFixtureId={selectedFixtureId} onSelect={onSelect} />
          )}
        </div>
      )}
      <p className="px-4 py-2 text-[11px] text-sage">
        {data ? `Matchweek ${Math.min(data.current_matchweek, data.max_matchweeks)} of ${data.max_matchweeks}` : ''}
        {LEAGUES_5.length ? ' · five leagues plus cups and UEFA' : ''}
      </p>
    </section>
  );
};

function FixtureGroup({ title, fixtures, selectedFixtureId, onSelect }: { title: string; fixtures: Fixture[]; selectedFixtureId?: string; onSelect: (fixture: Fixture) => void }) {
  return (
    <div className="px-4 py-3">
      <p className="mb-2 text-[10px] font-bold uppercase tracking-[0.14em] text-brass/70">{title}</p>
      <div className="space-y-1">
        {fixtures.map((fixture) => (
          <FixturePick key={fixture.id} fixture={fixture} selected={fixture.id === selectedFixtureId} onSelect={onSelect} compact />
        ))}
      </div>
    </div>
  );
}

function FixturePick({ fixture, selected, onSelect, compact = false }: { fixture: Fixture; selected: boolean; onSelect: (fixture: Fixture) => void; compact?: boolean }) {
  return (
    <button
      type="button"
      onClick={() => { soundManager.playClick(); onSelect(fixture); }}
      className={cx(
        'grid w-full min-w-0 grid-cols-[1fr_auto_1fr] items-center gap-2 rounded-md border px-3 py-2 text-left',
        selected ? 'border-brass/50 bg-brass/10' : 'border-white/10 bg-black/15 hover:border-brass/30',
        compact ? 'py-1.5' : 'py-2.5',
      )}
    >
      <span className="flex min-w-0 items-center gap-2">
        <ClubCrest club={fixture.home} size={compact ? 18 : 24} className="!border-0 !bg-transparent" />
        <span className="truncate text-[12px] font-semibold text-bone">{fixture.home.short_name}</span>
      </span>
      <span className="text-center font-mono text-[12px] text-brass">
        {fixture.status === 'finished' ? `${fixture.home_goals}–${fixture.away_goals}` : 'vs'}
      </span>
      <span className="flex min-w-0 items-center justify-end gap-2 text-right">
        <span className="truncate text-[12px] font-semibold text-bone">{fixture.away.short_name}</span>
        <ClubCrest club={fixture.away} size={compact ? 18 : 24} className="!border-0 !bg-transparent" />
      </span>
      <span className="col-span-3 truncate text-[10px] uppercase tracking-[0.12em] text-sage">
        {prettyCompetitionName(fixture.competition)} · MW {fixture.matchweek}
        {fixture.importance ? ` · ${fixture.importance}` : ''}
      </span>
    </button>
  );
}
