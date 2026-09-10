import React, { useState } from 'react';
import type { MatchTickPayload } from '../types';
import { matchWs } from '../services/matchSocket';
import { Card, PanelHeader } from './ui/ui';

export function HalfTimeDugout({ match, home, away }: { match: MatchTickPayload; home: string; away: string }) {
  const [side, setSide] = useState<'home' | 'away'>('home');
  const [stance, setStance] = useState('');
  const [outID, setOutID] = useState('');
  const [inID, setInID] = useState('');
  const events = (match.match_events ?? []).filter(e => e.side === side);
  const used = new Set(events.flatMap(e => [e.player_in?.player_id, e.player_out?.player_id, e.type === 'red' ? e.player?.player_id : undefined]));
  const starters = (side === 'home' ? match.home_coords : match.away_coords).filter(p => !p.sent_off);
  const active = new Set(starters.map(p => p.player.player_id));
  const bench = (side === 'home' ? match.home_bench : match.away_bench) ?? [];
  const available = bench.filter(p => p.status === 'bench' && !used.has(p.player.player_id) && !active.has(p.player.player_id));
  const canSub = events.filter(e => e.type === 'sub').length < 5 && available.length > 0;
  const valid = (!outID && !inID) || (canSub && starters.some(p => p.player.player_id === outID) && available.some(p => p.player.player_id === inID));
  const selectClass = 'block w-full mt-1 p-2 bg-ink text-bone border border-line';
  return <Card>
    <PanelHeader title="Half-time dugout" />
    <p className="text-sage text-sm my-3">45' · Choose a stance and up to one substitution, then resume.</p>
    <div className="grid sm:grid-cols-2 gap-3 text-sm">
      <label>Manage team<select className={selectClass} value={side} onChange={e => { setSide(e.target.value as 'home' | 'away'); setStance(''); setOutID(''); setInID(''); }}>
        <option value="home">{home}</option><option value="away">{away}</option>
      </select></label>
      <label>Second-half stance<select className={selectClass} value={stance} onChange={e => setStance(e.target.value)}>
        <option value="">Keep stance</option><option value="OVERLOAD">Overload</option><option value="PARK_BUS">Park the bus</option>
      </select></label>
      <label>Player off<select className={selectClass} disabled={!canSub} value={outID} onChange={e => { setOutID(e.target.value); if (!e.target.value) setInID(''); }}>
        <option value="">No substitution</option>{starters.map(p => <option key={p.player.player_id} value={p.player.player_id}>{p.player.full_name} · {p.player.position}</option>)}
      </select></label>
      <label>Player on<select className={selectClass} disabled={!canSub || !outID} value={inID} onChange={e => setInID(e.target.value)}>
        <option value="">Choose substitute</option>{available.map(p => <option key={p.player.player_id} value={p.player.player_id}>{p.player.full_name} · {p.player.position}</option>)}
      </select></label>
    </div>
    <button className="mt-4 px-5 py-2 bg-bone text-ink font-semibold disabled:opacity-40" disabled={!valid} onClick={() => matchWs.sendCommand('halftime_resume', { side, stance, out_id: outID, in_id: inID })}>Resume second half</button>
  </Card>;
}
