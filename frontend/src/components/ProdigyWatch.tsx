import React from 'react';
import type { ProdigyWatchRow } from '../types';
import { Card } from './ui/ui';

interface ProdigyWatchProps {
  watchRows: ProdigyWatchRow[];
  onSelectProdigy: (playerId: string) => void;
}

export const ProdigyWatch: React.FC<ProdigyWatchProps> = ({ watchRows, onSelectProdigy }) => {
  return (
    <Card className="overflow-hidden !p-0">
      <div className="px-5 py-4 border-b border-line flex items-center justify-between">
        <div>
          <p className="eyebrow !text-brass">Commissioner board</p>
          <h3 className="font-display text-xl font-semibold text-bone">Prodigy Watch · Golden Boy race</h3>
        </div>
        <span className="font-mono text-[11px] text-sage">{watchRows.length} franchise prodigies</span>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-[12px]">
          <thead className="text-sage font-mono uppercase text-[10px] border-b border-line">
            <tr>
              <th className="text-left px-4 py-2">#</th>
              <th className="text-left px-3 py-2">Player</th>
              <th className="text-left px-3 py-2">Club</th>
              <th className="text-right px-3 py-2">Age</th>
              <th className="text-right px-3 py-2">OVR</th>
              <th className="text-right px-3 py-2">Growth</th>
              <th className="text-right px-3 py-2">Apps</th>
              <th className="text-right px-3 py-2">G</th>
              <th className="text-right px-3 py-2">A</th>
              <th className="text-right px-4 py-2">GB score</th>
            </tr>
          </thead>
          <tbody>
            {watchRows.map((row) => (
              <tr key={row.player_id} className="border-b border-line/60 hover:bg-cardLight/50">
                <td className="px-4 py-3 font-mono font-bold text-brass">{row.rank}</td>
                <td className="px-3 py-3">
                  <button
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
                <td className="px-4 py-3 text-right font-mono font-bold text-brass">{row.golden_boy_score.toFixed(1)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
};
