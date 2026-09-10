import React, { useState } from 'react';
import type { Club, Fixture } from '../types';
import { fetchUCL, fetchUclFixtures, fetchSuperLeague, simulateFixture } from '../services/api';
import { Trophy, Shield, Crown, LayoutGrid, Swords } from 'lucide-react';
import { soundManager } from '../audio/webAudio';
import { useAsyncData } from '../hooks/useAsyncData';
import { formatGd, cx } from '../lib/format';
import { Card, ClubDot, LoadingState, PanelHeader } from './ui/ui';
import { MatchCard } from './MatchCard';
import { MatchDetailModal } from './MatchDetailModal';
import { PreMatchModal } from './PreMatchModal';

interface UclTournamentTabProps {
  onWatchFixture: (fixture: Fixture) => void;
  onShowToast: (msg: string) => void;
  /** Tables and bracket only — fixtures live on the league slate. */
  boardOnly?: boolean;
}

const GroupTable: React.FC<{ title: string; clubs: Club[] }> = ({ title, clubs }) => (
  <div className="panel-tight overflow-hidden">
    <div className="px-4 py-3 bg-cardLight/60 border-b border-line flex items-center justify-between">
      <h3 className="font-semibold text-[14px] text-bone flex items-center gap-2">
        <Shield size={15} className="text-sage" /> {title}
      </h3>
      <span className="text-[11px] font-mono text-sage">Top four advance</span>
    </div>
    <table className="w-full text-left text-[13px]">
      <thead className="table-head">
        <tr>
          <th className="py-2.5 px-3">#</th>
          <th className="py-2.5 px-3">Club</th>
          {['P', 'W', 'D', 'L', 'GD'].map((h) => (
            <th key={h} className="py-2.5 px-2">{h}</th>
          ))}
          <th className="py-2.5 px-3">PTS</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-line/70">
        {clubs.map((c, idx) => {
          const status = c.cup_status;
          const through = status === 'qualified';
          const out = status === 'eliminated';
          const must = status === 'must_win';
          return (
            <tr
              key={c.club_id}
              className={cx(
                'hover:bg-cardLight/50 transition-colors',
                through && 'bg-brass/[0.05]',
                out && 'opacity-55',
              )}
            >
              <td className="py-2.5 px-3 font-bold font-mono">
                <span className={through ? 'text-brass' : out ? 'text-sage/60' : 'text-bone/85'}>{idx + 1}</span>
              </td>
              <td className="py-2.5 px-3 font-semibold text-bone">
                <span className="flex items-center gap-2">
                  <ClubDot club={c} />
                  <span>{c.club_name}</span>
                  {through && <span className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-brass/12 text-brass border border-brass/35">Through</span>}
                  {out && <span className="font-mono text-[10px] px-1.5 py-0.5 rounded border border-line text-sage">Out</span>}
                  {must && <span className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-ember/12 text-ember border border-ember/35">Must-win</span>}
                </span>
              </td>
              <td className="py-2.5 px-2 text-sage font-mono">{c.p}</td>
              <td className="py-2.5 px-2 text-bone font-semibold font-mono">{c.w}</td>
              <td className="py-2.5 px-2 text-sage font-mono">{c.d}</td>
              <td className="py-2.5 px-2 text-sage font-mono">{c.l}</td>
              <td className="py-2.5 px-2 font-mono text-bone/75">{formatGd(c.gd)}</td>
              <td className="py-2.5 px-3 font-bold text-bone font-mono text-[14px]">{c.pts}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  </div>
);

export const UclTournamentTab: React.FC<UclTournamentTabProps> = ({ onWatchFixture, onShowToast, boardOnly = false }) => {
  const { data: tournament, loading, reload } = useAsyncData(fetchUCL);
  const { data: uclFixtures, reload: reloadFixtures } = useAsyncData(fetchUclFixtures);
  const { data: league } = useAsyncData(fetchSuperLeague);
  const [viewMode, setViewMode] = useState<'groups' | 'knockout'>('groups');
  const [busyId, setBusyId] = useState<string | null>(null);
  const [openFixture, setOpenFixture] = useState<Fixture | null>(null);
  const [previewFixture, setPreviewFixture] = useState<Fixture | null>(null);

  const currentMw = league?.current_matchweek ?? 1;
  const slate = (uclFixtures ?? [])
    .filter((f) => f.status === 'scheduled' && f.matchweek <= currentMw)
    .sort((a, b) => a.matchweek - b.matchweek || (a.id < b.id ? -1 : 1));

  const handleSimulateUcl = async (f: Fixture) => {
    soundManager.playWhistle();
    setBusyId(f.id);
    try {
      const res = await simulateFixture(f.id);
      if (res.status === 'error') {
        onShowToast(res.message || 'That result already stands and cannot be replayed.');
      } else {
        onShowToast(`${f.home.short_name} ${res.home_goals} – ${res.away_goals} ${f.away.short_name}. Result stands.`);
        if (res.ucl_event) onShowToast(res.ucl_event);
      }
      reload();
      reloadFixtures();
    } finally {
      setBusyId(null);
    }
  };

  const uclEyebrow = (f: Fixture) =>
    `Champions Cup · ${f.stage}${f.leg ? ` · Leg ${f.leg}` : ''} · MW ${f.matchweek}`;

  if (loading || !tournament) return <Card><LoadingState message="Loading the Champions Cup…" /></Card>;

  const semi1 = tournament.semi_finals?.semi_1;
  const semi2 = tournament.semi_finals?.semi_2;

  return (
    <div className="space-y-5">
      <Card>
        <PanelHeader
          kicker={`Stage · ${(tournament.stage || 'GROUP_STAGE').replace(/_/g, ' ').toLowerCase()}`}
          title="Champions Cup"
          subtitle="Two groups of six across the 44-week calendar. Top four reach two-legged quarter-finals, then semis and a final. Ties sit on the same fixture list."
        />
      </Card>

      {!boardOnly && slate.length > 0 && (
        <Card>
          <PanelHeader
            kicker={`${slate.length} tie${slate.length === 1 ? '' : 's'} on the slate`}
            title="Playable cup ties"
            subtitle="Watch live or simulate. Super League weeks move on without these."
          />
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 mt-5">
            {slate.map((f) => (
              <MatchCard
                key={f.id}
                fixture={f}
                eyebrow={uclEyebrow(f)}
                busyId={busyId}
                onWatch={(fx) => {
                  soundManager.playClick();
                  onWatchFixture(fx);
                }}
                onSimulate={handleSimulateUcl}
                onOpen={(fx) => {
                  if (fx.status === 'finished') setOpenFixture(fx);
                  else setPreviewFixture(fx);
                }}
              />
            ))}
          </div>
        </Card>
      )}

      <div className="flex gap-1.5 border-b border-line pb-3">
        {(
          [
            ['groups', 'Group stage'],
            ['knockout', 'Knockout bracket'],
          ] as const
        ).map(([mode, label]) => (
          <button
            key={mode}
            onClick={() => {
              soundManager.playClick();
              setViewMode(mode);
            }}
            aria-pressed={viewMode === mode}
            className={cx(
              'px-4 py-2 rounded-lg text-[13px] font-semibold transition-colors flex items-center gap-1.5 border',
              viewMode === mode
                ? 'bg-brass text-ink border-brass'
                : 'bg-transparent text-sage hover:text-bone border-line',
            )}
          >
            {mode === 'groups' ? <LayoutGrid size={13} /> : <Swords size={13} />}
            {label}
          </button>
        ))}
      </div>

      {viewMode === 'groups' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          <GroupTable title="Group A" clubs={tournament.group_a || []} />
          <GroupTable title="Group B" clubs={tournament.group_b || []} />
        </div>
      )}

      {viewMode === 'knockout' && (
        <div className="space-y-5">
          <Card>
            <p className="eyebrow mb-3">Quarter-finals · two legs</p>
            {['qf_1', 'qf_2', 'qf_3', 'qf_4'].every((k) => !tournament.quarter_finals?.[k as keyof typeof tournament.quarter_finals]?.home) ? (
              <div className="p-4 bg-ink/40 border border-line rounded-xl text-center text-[13px] text-sage">
                Decided once the group stage ends at matchweek 16.
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {(['qf_1', 'qf_2', 'qf_3', 'qf_4'] as const).map((key, i) => {
                  const tie = tournament.quarter_finals?.[key];
                  if (!tie?.home) return null;
                  return (
                    <div key={key} className="bg-ink/40 border border-line rounded-xl p-4 space-y-2">
                      <p className="eyebrow">QF {i + 1}</p>
                      <div className="flex justify-between items-center text-sm font-bold text-white">
                        <span>{tie.home.club_name}</span>
                        <span className="font-mono text-sage">{tie.leg1 ? `${tie.leg1[0]} (L1)` : '–'}</span>
                      </div>
                      <div className="flex justify-between items-center text-sm font-bold text-white">
                        <span>{tie.away?.club_name}</span>
                        <span className="font-mono text-sage">{tie.leg1 ? `${tie.leg1[1]} (L1)` : '–'}</span>
                      </div>
                      {tie.leg2 && (
                        <div className="text-[11px] font-mono text-sage">
                          Leg 2: {tie.leg2[0]} – {tie.leg2[1]}
                          {tie.leg1 && ` · Agg ${tie.leg1[0] + tie.leg2[1]}–${tie.leg1[1] + tie.leg2[0]}`}
                          {tie.decided_by === 'extra_time' ? ' after extra time' : ''}
                          {tie.decided_by === 'penalties' && tie.penalties ? ` (${tie.penalties[0]}–${tie.penalties[1]} pens)` : ''}
                        </div>
                      )}
                      {tie.winner && (
                        <div className="pt-2 border-t border-line text-[11px] font-bold text-brass">Through: {tie.winner.club_name}</div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
          <Card className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 items-start">
            <div className="space-y-3">
              <p className="eyebrow">Semi-final 1 · Two legs</p>
              {semi1?.home ? (
                <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-2">
                  <div className="flex justify-between items-center text-[14px] font-semibold text-bone">
                    <span>{semi1.home.club_name}</span>
                    <span className="font-mono text-sage">{semi1.leg1 ? `${semi1.leg1[0]} (L1)` : '–'}</span>
                  </div>
                  <div className="flex justify-between items-center text-[14px] font-semibold text-bone">
                    <span>{semi1.away?.club_name}</span>
                    <span className="font-mono text-sage">{semi1.leg1 ? `${semi1.leg1[1]} (L1)` : '–'}</span>
                  </div>
                  {semi1.leg2 && (
                    <div className="text-[12px] font-mono text-sage">
                      Leg 2: {semi1.leg2[0]} – {semi1.leg2[1]}
                      {semi1.leg1 && ` · Agg ${semi1.leg1[0] + semi1.leg2[1]}–${semi1.leg1[1] + semi1.leg2[0]}`}
                      {semi1.decided_by === 'penalties' && semi1.penalties ? ` (${semi1.penalties[0]}–${semi1.penalties[1]} pens)` : ''}
                    </div>
                  )}
                  {semi1.winner && (
                    <div className="pt-2 border-t border-line text-[12px] font-semibold text-brass">Through: {semi1.winner.club_name}</div>
                  )}
                </div>
              ) : (
                <div className="p-4 bg-ink/40 border border-line rounded-xl text-center text-[13px] text-sage">
                  Decided once the group stage ends at matchweek 16.
                </div>
              )}
            </div>

            <div className="space-y-3 text-center">
              <div className="w-12 h-12 rounded-full bg-brass mx-auto flex items-center justify-center text-ink">
                <Crown size={24} />
              </div>
              <p className="eyebrow !text-brass text-center">Grand final · Neutral ground</p>
              {tournament.final?.team1 ? (
                <div className="bg-ink/40 border border-brass/45 rounded-xl p-5 space-y-2">
                  <div className="font-semibold text-bone text-[15px]">{tournament.final.team1.club_name}</div>
                  <div className="score-display text-[30px] text-brass">
                    {tournament.final.score ? `${tournament.final.score[0]} – ${tournament.final.score[1]}` : 'vs'}
                  </div>
                  <div className="font-semibold text-bone text-[15px]">{tournament.final.team2?.club_name}</div>
                  {tournament.final.decided_by === 'penalties' && tournament.final.penalties && (
                    <div className="text-[12px] font-mono text-sage">{tournament.final.penalties[0]}–{tournament.final.penalties[1]} on penalties</div>
                  )}
                  {tournament.final.decided_by === 'extra_time' && (
                    <div className="text-[12px] font-mono text-sage">After extra time</div>
                  )}
                  {tournament.final.winner && (
                    <div className="pt-2 border-t border-line text-[12px] font-bold text-brass uppercase tracking-[0.1em]">
                      Champions: {tournament.final.winner.club_name}
                    </div>
                  )}
                </div>
              ) : (
                <div className="p-6 bg-ink/40 border border-line rounded-xl text-[13px] text-sage">Set once both semi-finals conclude.</div>
              )}
            </div>

            <div className="space-y-3">
              <p className="eyebrow">Semi-final 2 · Two legs</p>
              {semi2?.home ? (
                <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-2">
                  <div className="flex justify-between items-center text-[14px] font-semibold text-bone">
                    <span>{semi2.home.club_name}</span>
                    <span className="font-mono text-sage">{semi2.leg1 ? `${semi2.leg1[0]} (L1)` : '–'}</span>
                  </div>
                  <div className="flex justify-between items-center text-[14px] font-semibold text-bone">
                    <span>{semi2.away?.club_name}</span>
                    <span className="font-mono text-sage">{semi2.leg1 ? `${semi2.leg1[1]} (L1)` : '–'}</span>
                  </div>
                  {semi2.leg2 && (
                    <div className="text-[12px] font-mono text-sage">
                      Leg 2: {semi2.leg2[0]} – {semi2.leg2[1]}
                      {semi2.leg1 && ` · Agg ${semi2.leg1[0] + semi2.leg2[1]}–${semi2.leg1[1] + semi2.leg2[0]}`}
                      {semi2.decided_by === 'penalties' && semi2.penalties ? ` (${semi2.penalties[0]}–${semi2.penalties[1]} pens)` : ''}
                    </div>
                  )}
                  {semi2.winner && (
                    <div className="pt-2 border-t border-line text-[12px] font-semibold text-brass">Through: {semi2.winner.club_name}</div>
                  )}
                </div>
              ) : (
                <div className="p-4 bg-ink/40 border border-line rounded-xl text-center text-[13px] text-sage">
                  Decided once the group stage ends at matchweek 16.
                </div>
              )}
            </div>
          </div>
        </Card>
        </div>
      )}

      <p className="flex items-center gap-2 text-[12px] text-sage font-mono">
        <Trophy size={13} /> The programme cover goes to the cup winners each season.
      </p>

      <MatchDetailModal fixture={openFixture} onClose={() => setOpenFixture(null)} />
      <PreMatchModal
        fixture={previewFixture}
        onClose={() => setPreviewFixture(null)}
        busy={busyId === previewFixture?.id}
        onWatch={(fx) => {
          setPreviewFixture(null);
          onWatchFixture(fx);
        }}
        onSimulate={async (fx) => {
          await handleSimulateUcl(fx);
          setPreviewFixture(null);
        }}
      />
    </div>
  );
};
