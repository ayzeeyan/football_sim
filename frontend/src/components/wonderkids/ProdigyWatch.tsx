import React, { useMemo, useState } from 'react';
import type { ProdigyWatchRow } from '../../types';
import { soundManager } from '../../audio/webAudio';
import { Card } from '../ui/ui';

type RaceMode = 'golden_boy' | 'goals' | 'assists' | 'contributions' | 'ovr';

const RACE_MODES: Array<{ id: RaceMode; label: string; title: string; metric: string }> = [
  { id: 'golden_boy', label: 'Golden Boy', title: 'Golden Boy race', metric: 'weighted score with featured-player bonuses' },
  { id: 'goals', label: 'Golden Boot', title: 'Golden Boot race', metric: 'raw season goals' },
  { id: 'assists', label: 'Playmaker', title: 'Playmaker race', metric: 'raw season assists' },
  { id: 'contributions', label: 'G + A', title: 'Goal contributions', metric: 'raw season goals + assists' },
  { id: 'ovr', label: 'Best OVR', title: 'Highest rated', metric: 'overall rating' },
];

function compareRows(a: ProdigyWatchRow, b: ProdigyWatchRow, mode: RaceMode): number {
  const score = (row: ProdigyWatchRow): number => {
    switch (mode) {
      case 'goals': return row.goals;
      case 'assists': return row.assists;
      case 'contributions': return row.goals + row.assists;
      case 'ovr': return row.ovr;
      default: return row.golden_boy_score;
    }
  };
  const primary = score(b) - score(a);
  if (primary !== 0) return primary;
  return a.full_name.localeCompare(b.full_name);
}

interface ProdigyWatchProps {
  watchRows: ProdigyWatchRow[];
  onSelectProdigy: (playerId: string) => void;
}

export const ProdigyWatch: React.FC<ProdigyWatchProps> = ({ watchRows, onSelectProdigy }) => {
  const [raceMode, setRaceMode] = useState<RaceMode>('golden_boy');
  const activeRace = RACE_MODES.find((mode) => mode.id === raceMode) ?? RACE_MODES[0]!;
  const rankedRows = useMemo(() => [...watchRows].sort((a, b) => compareRows(a, b, raceMode)), [watchRows, raceMode]);

  return (
    <Card className="overflow-hidden !p-0">
      <div className="px-5 py-4 border-b border-line flex items-center justify-between flex-wrap gap-3">
        <div>
          <p className="eyebrow !text-brass">Commissioner board</p>
          <h3 className="font-display text-xl font-semibold text-bone">Prodigy Watch · {activeRace.title}</h3>
          <p className="mt-1 text-[11px] text-sage">Ranked by {activeRace.metric}. Other races sort by the selected raw stat.</p>
        </div>
        <span className="font-mono text-[11px] text-sage">{watchRows.length} franchise prodigies</span>
      </div>
      <div className="flex gap-1.5 overflow-x-auto border-b border-line bg-ink/35 px-4 py-3" role="group" aria-label="Choose the prodigy race">
        {RACE_MODES.map((mode) => (
          <button
            key={mode.id}
            type="button"
            onClick={() => { soundManager.playClick(); setRaceMode(mode.id); }}
            aria-pressed={raceMode === mode.id}
            className={`shrink-0 rounded-lg border px-3 py-1.5 text-[11px] font-semibold transition-colors ${raceMode === mode.id ? 'border-brass/50 bg-brass/10 text-brass' : 'border-line bg-cardBg text-sage hover:text-bone'}`}
          >
            {mode.label}
          </button>
        ))}
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-[12px]">
          <caption className="sr-only">
            {activeRace.title}, ranked by {activeRace.metric}. Golden Boy includes featured-player bonuses; other races use raw statistics.
          </caption>
          <thead className="text-sage font-mono uppercase text-[10px] border-b border-line">
            <tr>
              <th scope="col" className="text-left px-4 py-2">#</th>
              <th scope="col" className="text-left px-3 py-2">Player</th>
              <th scope="col" className="text-left px-3 py-2">Club</th>
              <th scope="col" className="text-right px-3 py-2">Age</th>
              <th scope="col" className="text-right px-3 py-2">OVR</th>
              <th scope="col" className="text-right px-3 py-2">Growth</th>
              <th scope="col" className="text-right px-3 py-2">Apps</th>
              <th scope="col" className="text-right px-3 py-2">G</th>
              <th scope="col" className="text-right px-3 py-2">A</th>
              <th scope="col" className="text-right px-3 py-2">G + A</th>
              <th scope="col" className="text-right px-4 py-2">GB score</th>
            </tr>
          </thead>
          <tbody>
            {rankedRows.map((row, index) => (
              <tr key={row.player_id} className="border-b border-line/60 hover:bg-cardLight/50">
                <td className="px-4 py-3 font-mono font-bold text-brass">{index + 1}</td>
                <td className="px-3 py-3">
                  <button
                    type="button"
                    onClick={() => onSelectProdigy(row.player_id)}
                    className="font-semibold text-bone hover:text-brass text-left"
                  >
                    {row.full_name}
                  </button>
                  <div className="font-mono text-[10px] text-sage">{row.position} · {row.puberty_stage || 'developing'}</div>
                </td>
                <td className="px-3 py-3 text-sage">{row.club_short || row.club_name || row.club_id}</td>
                <td className="px-3 py-3 text-right font-mono text-sage">{row.age}</td>
                <td className="px-3 py-3 text-right font-mono font-bold text-bone">{row.ovr}<span className="text-sage font-normal">/{row.potential ?? '—'}</span></td>
                <td className="px-3 py-3 text-right font-mono text-sage">+{(row.height_gain_cm ?? 0).toFixed(1)}cm</td>
                <td className="px-3 py-3 text-right font-mono text-sage">{row.appearances}</td>
                <td className="px-3 py-3 text-right font-mono text-bone">{row.goals}</td>
                <td className="px-3 py-3 text-right font-mono text-bone">{row.assists}</td>
                <td className="px-3 py-3 text-right font-mono font-bold text-bone">{row.goals + row.assists}</td>
                <td className="px-4 py-3 text-right font-mono font-bold text-brass">{row.golden_boy_score.toFixed(1)}</td>
              </tr>
            ))}
            {rankedRows.length === 0 && (
              <tr><td colSpan={11} className="px-4 py-8 text-center text-sage">The season’s first appearances will start the race.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </Card>
  );
};
