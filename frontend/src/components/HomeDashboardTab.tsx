import React from 'react';
import { Crown, Newspaper, Sparkle } from 'lucide-react';
import type { Club, CompetitionClub, CompetitionFixtureRow, WorldDashboard } from '../types';
import { fetchWorldDashboard } from '../services/api';
import { useAsyncData } from '../hooks/useAsyncData';
import { usePlayerSheet } from './PlayerSheet';
import { Card, ClubCrest, LoadingState, PanelHeader } from './ui/ui';
import { cx } from '../lib/format';
import { soundManager } from '../audio/webAudio';

interface HomeDashboardTabProps {
  careerKey: number;
  onWatchFixture: (fixture: CompetitionFixtureRow) => void;
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

function scoreLine(f: CompetitionFixtureRow): string {
  if (f.status !== 'finished' || f.home_goals == null || f.away_goals == null) return 'v';
  return `${f.home_goals}–${f.away_goals}`;
}

export const HomeDashboardTab: React.FC<HomeDashboardTabProps> = ({
  careerKey,
  onWatchFixture,
  onViewSquad,
  onOpenInbox,
  onOpenTransfers,
  onOpenLeague,
  onOpenCompetitions,
}) => {
  const { openPlayer } = usePlayerSheet();
  const { data, loading } = useAsyncData(fetchWorldDashboard, [careerKey]);

  if (loading || !data) {
    return <Card><LoadingState message="Loading the European week…" /></Card>;
  }

  if (!data.world) {
    return (
      <div className="space-y-4">
        <PanelHeader
          kicker="Career home"
          title="Super League"
          subtitle="This save is the original 12-club Super League. Start a new career to open the Top Five European world."
        />
        <Card>
          <p className="text-[14px] text-sage leading-relaxed">
            Use League, Match and Transfers as before. A new European career unlocks five leagues, domestic cups and UEFA competitions on one calendar.
          </p>
        </Card>
      </div>
    );
  }

  const windowLabel = data.transfer_window.open
    ? `${data.transfer_window.type === 'WINTER' ? 'Winter' : 'Summer'} window · week ${data.transfer_window.week}/${data.transfer_window.weeks}`
    : data.season_phase === 'transfer_window'
      ? 'Transfer window closing'
      : `Matchweek ${Math.min(data.current_matchweek, data.max_matchweeks)} of ${data.max_matchweeks}`;

  return (
    <div className="space-y-5">
      <PanelHeader
        kicker="Career home"
        title="European week"
        subtitle={`${data.season_name.replace('-', '–')} · ${windowLabel}`}
        right={
          data.favourite_club ? (
            <button
              type="button"
              onClick={() => onViewSquad(data.favourite_club!.club_id)}
              className="flex items-center gap-2 border border-line bg-cardLight/60 px-3 py-2 hover:bg-cardHover"
            >
              <ClubCrest club={asClub(data.favourite_club)} size={28} />
              <span className="text-[13px] font-semibold text-bone">{data.favourite_club.short_name}</span>
            </button>
          ) : null
        }
      />

      {data.next_fixture && data.next_fixture.status === 'scheduled' && (
        <button
          type="button"
          onClick={() => {
            soundManager.playClick();
            onWatchFixture(data.next_fixture!);
          }}
          className="w-full text-left panel-pad border-brass/40 bg-brass/[0.07] hover:bg-brass/[0.12] transition-colors"
        >
          <p className="eyebrow !text-brass">Next watched fixture</p>
          <div className="mt-2 flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3 min-w-0">
              <ClubCrest club={asClub(data.next_fixture.home)} size={36} />
              <span className="font-display text-[22px] font-semibold text-bone truncate">
                {data.next_fixture.home?.short_name} v {data.next_fixture.away?.short_name}
              </span>
              <ClubCrest club={asClub(data.next_fixture.away)} size={36} />
            </div>
            <span className="font-mono text-[12px] text-sage">
              MW {data.next_fixture.matchweek} · {data.next_fixture.competition}
            </span>
          </div>
        </button>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-5 gap-3">
        {data.league_leaders.map((row) => (
          <button
            key={row.competition_id}
            type="button"
            onClick={() => {
              soundManager.playClick();
              onOpenLeague();
              onViewSquad(row.club_id);
            }}
            className="panel-pad text-left hover:bg-cardHover transition-colors"
          >
            <p className="eyebrow">{row.competition_name}</p>
            <div className="mt-2 flex items-center gap-2 min-w-0">
              <ClubCrest club={asClub(row)} size={28} />
              <div className="min-w-0">
                <p className="font-semibold text-bone truncate">{row.short_name}</p>
                <p className="font-mono text-[12px] text-sage">
                  {row.pts ?? 0} pts{typeof row.pts_gap === 'number' ? ` · +${row.pts_gap}` : ''}
                </p>
              </div>
            </div>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <Card>
          <div className="flex items-start justify-between gap-2">
            <div>
              <p className="eyebrow">Champions League</p>
              <p className="font-display text-[22px] font-semibold text-bone mt-1">{data.europe.stage || 'League phase'}</p>
            </div>
            <button type="button" onClick={onOpenCompetitions} className="text-[12px] text-brass hover:text-bone">Open</button>
          </div>
          {data.europe.champion ? (
            <div className="mt-3 flex items-center gap-2">
              <Crown size={14} className="text-brass" />
              <ClubCrest club={asClub(data.europe.champion)} size={24} />
              <span className="text-[13px] text-bone">{data.europe.champion.club_name}</span>
            </div>
          ) : data.europe.leader ? (
            <button type="button" onClick={() => onViewSquad(data.europe.leader!.club_id)} className="mt-3 flex items-center gap-2">
              <ClubCrest club={asClub(data.europe.leader)} size={24} />
              <span className="text-[13px] text-bone">{data.europe.leader.club_name}</span>
              <span className="font-mono text-[12px] text-sage">{data.europe.leader.pts ?? 0} pts</span>
            </button>
          ) : (
            <p className="text-[13px] text-sage mt-3">League phase underway.</p>
          )}
        </Card>

        <button
          type="button"
          className="text-left"
          onClick={() => data.top_scorer?.player_id && openPlayer(data.top_scorer.player_id)}
        >
          <Card>
            <p className="eyebrow">Europe’s top scorer</p>
            {data.top_scorer ? (
              <>
                <p className="font-display text-[22px] font-semibold text-bone mt-2 truncate">{data.top_scorer.full_name}</p>
                <p className="font-mono text-[13px] text-sage mt-1">
                  {data.top_scorer.club_short} · {data.top_scorer.goals} G · {data.top_scorer.assists} A
                </p>
              </>
            ) : (
              <p className="text-[13px] text-sage mt-2">No goals yet.</p>
            )}
          </Card>
        </button>

        <Card>
          <div className="flex items-start justify-between gap-2">
            <p className="eyebrow">Transfer window</p>
            <button type="button" onClick={onOpenTransfers} className="text-[12px] text-brass hover:text-bone">Market</button>
          </div>
          <p className="font-display text-[22px] font-semibold text-bone mt-2">
            {data.transfer_window.open ? `${data.transfer_window.week}/${data.transfer_window.weeks}` : 'Closed'}
          </p>
          <p className="text-[13px] text-sage mt-1">
            {data.biggest_transfers[0]
              ? `${data.biggest_transfers[0].player_name} · ${data.biggest_transfers[0].formatted_fee}`
              : 'No completed deals yet.'}
          </p>
        </Card>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3 border-b border-line bg-cardLight/50 flex items-center justify-between">
            <h3 className="text-[14px] font-semibold text-bone">Upcoming big fixtures</h3>
          </div>
          <ul className="divide-y divide-line/70">
            {data.upcoming_fixtures.length === 0 && (
              <li className="px-4 py-6 text-[13px] text-sage">No scheduled fixtures this week.</li>
            )}
            {data.upcoming_fixtures.map((f) => (
              <li key={f.fixture_id || f.id}>
                <button
                  type="button"
                  onClick={() => {
                    soundManager.playClick();
                    onWatchFixture(f);
                  }}
                  className="w-full px-4 py-2.5 flex items-center gap-3 hover:bg-cardLight/60 text-left"
                >
                  <span className="font-mono text-[11px] text-sage w-10 shrink-0">MW{f.matchweek}</span>
                  <ClubCrest club={asClub(f.home)} size={20} />
                  <span className="flex-1 text-[13px] text-bone truncate">
                    {f.home?.short_name} {scoreLine(f)} {f.away?.short_name}
                  </span>
                  <ClubCrest club={asClub(f.away)} size={20} />
                </button>
              </li>
            ))}
          </ul>
        </div>

        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3 border-b border-line bg-cardLight/50 flex items-center justify-between">
            <h3 className="text-[14px] font-semibold text-bone">Headlines</h3>
            <button type="button" onClick={onOpenInbox} className="text-[12px] text-brass hover:text-bone inline-flex items-center gap-1">
              <Newspaper size={12} /> Inbox{data.unread_inbox > 0 ? ` · ${data.unread_inbox}` : ''}
            </button>
          </div>
          <ul className="divide-y divide-line/70">
            {data.headlines.length === 0 && (
              <li className="px-4 py-6 text-[13px] text-sage">The wire is quiet.</li>
            )}
            {data.headlines.map((h) => (
              <li key={h.id} className="px-4 py-2.5">
                <p className={cx('text-[13px] font-semibold', h.unread ? 'text-bone' : 'text-sage')}>{h.headline}</p>
                <p className="font-mono text-[11px] text-sage mt-0.5">MW {h.matchweek} · {h.category}</p>
              </li>
            ))}
          </ul>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3 border-b border-line bg-cardLight/50">
            <p className="text-[14px] font-semibold text-bone">Wonderkid watch</p>
          </div>
          <ul className="divide-y divide-line/70">
            {data.wonderkids.slice(0, 5).map((w) => (
              <li key={w.player_id}>
                <button
                  type="button"
                  onClick={() => openPlayer(w.player_id)}
                  className="w-full px-4 py-2 flex items-center justify-between gap-2 hover:bg-cardLight/60 text-left"
                >
                  <span className="flex items-center gap-2 min-w-0">
                    <Sparkle size={12} className="text-brass shrink-0" />
                    <span className="text-[13px] text-bone truncate">{w.full_name}</span>
                  </span>
                  <span className="font-mono text-[11px] text-sage shrink-0">{w.ovr} · {w.club_short}</span>
                </button>
              </li>
            ))}
          </ul>
        </div>

        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3 border-b border-line bg-cardLight/50">
            <p className="text-[14px] font-semibold text-bone">Injuries</p>
          </div>
          <ul className="divide-y divide-line/70">
            {data.injuries.length === 0 && <li className="px-4 py-6 text-[13px] text-sage">No senior injuries listed.</li>}
            {data.injuries.map((p) => (
              <li key={p.player_id}>
                <button type="button" onClick={() => openPlayer(p.player_id)} className="w-full px-4 py-2 text-left hover:bg-cardLight/60">
                  <p className="text-[13px] text-bone truncate">{p.full_name}</p>
                  <p className="font-mono text-[11px] text-ember">{p.club_short} · {p.injury || 'Injured'} · {p.injured_matches}</p>
                </button>
              </li>
            ))}
          </ul>
        </div>

        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3 border-b border-line bg-cardLight/50 flex items-center justify-between">
            <p className="text-[14px] font-semibold text-bone">Biggest transfers</p>
            <button type="button" onClick={onOpenTransfers} className="text-[12px] text-brass">Market</button>
          </div>
          <ul className="divide-y divide-line/70">
            {data.biggest_transfers.length === 0 && <li className="px-4 py-6 text-[13px] text-sage">No completed deals.</li>}
            {data.biggest_transfers.map((tr) => (
              <li key={`${tr.player_id}-${tr.matchweek}-${tr.formatted_fee}`} className="px-4 py-2">
                <p className="text-[13px] text-bone truncate">{tr.player_name}</p>
                <p className="font-mono text-[11px] text-sage">{tr.seller_short} → {tr.buyer_short} · {tr.formatted_fee}</p>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {(data.power_rankings?.length ?? 0) > 0 && (
        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3 border-b border-line bg-cardLight/50">
            <p className="text-[14px] font-semibold text-bone">Club power rankings</p>
          </div>
          <ol className="divide-y divide-line/70">
            {data.power_rankings!.map((row) => (
              <li key={row.club_id}>
                <button type="button" onClick={() => onViewSquad(row.club_id)} className="w-full px-4 py-2 flex items-center justify-between gap-2 hover:bg-cardLight/60 text-left">
                  <span className="flex items-center gap-3 min-w-0">
                    <span className="font-mono text-[12px] text-brass w-6">{row.rank}</span>
                    <span className="text-[13px] text-bone truncate">{row.club_name}</span>
                    <span className="font-mono text-[11px] text-sage">{row.league}</span>
                  </span>
                  <span className="font-mono text-[11px] text-sage shrink-0">{row.ovr} OVR · chem {row.chemistry}</span>
                </button>
              </li>
            ))}
          </ol>
        </div>
      )}

      {(data.loan_watch?.length ?? 0) > 0 && (
        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3 border-b border-line bg-cardLight/50">
            <p className="text-[14px] font-semibold text-bone">Loan watch</p>
          </div>
          <ul className="divide-y divide-line/70">
            {data.loan_watch!.map((row) => (
              <li key={row.player_id}>
                <button type="button" onClick={() => openPlayer(row.player_id)} className="w-full px-4 py-2 text-left hover:bg-cardLight/60">
                  <p className="text-[13px] text-bone truncate">{row.full_name}</p>
                  <p className="font-mono text-[11px] text-sage">{row.club_short} · {row.appearances} apps · {row.goals} G · {row.form_band}</p>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {data.sackings.length > 0 && (
        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3 border-b border-line bg-cardLight/50">
            <p className="text-[14px] font-semibold text-bone">Manager changes</p>
          </div>
          <ul className="divide-y divide-line/70">
            {data.sackings.map((s, i) => (
              <li key={`${s.club_id}-${s.matchweek}-${i}`} className="px-4 py-2.5 flex flex-wrap items-center justify-between gap-2">
                <span className="text-[13px] text-bone">{s.club_name}</span>
                <span className="text-[12px] text-sage">
                  {s.old_manager} → {s.new_manager}
                  {s.reason ? ` · ${s.reason}` : ''}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};
