import React from 'react';
import { Calendar, Landmark, Trophy } from 'lucide-react';
import type { Club, ClubHistoryResponse, ClubProfile, ClubTransferActivity, Fixture, Player } from '../../../types';
import { Card } from '../../ui/ui';
import { cx, formatEUR } from '../../../lib/format';
import { prettyCompetitionName } from '../PlayerSheet';
import { FIXTURE_COMP_FILTERS, formatWageBill, matchesFixtureFilter, type SquadTabProps } from './status';
import { ClubIdentityPanel } from '../ClubIdentityPanel';

export const ClubOverviewPanel: React.FC<{
  club: Club;
  profile: ClubProfile | null;
  squad: Player[];
  onOpenPlayer: (id: string) => void;
  onWatchFixture?: SquadTabProps['onWatchFixture'];
}> = ({ club, profile, squad, onOpenPlayer, onWatchFixture }) => {
  const avgAge = squad.length ? (squad.reduce((s, p) => s + p.age, 0) / squad.length).toFixed(1) : '—';
  const avgMorale = squad.length ? Math.round(squad.reduce((s, p) => s + (p.morale ?? 0), 0) / squad.length) : 0;
  return (
    <div className="space-y-4">
      {(profile?.storylines?.length ?? 0) > 0 && (
        <Card>
          <p className="eyebrow">Current storylines</p>
          <ul className="mt-2 space-y-1.5">
            {profile!.storylines.map((line) => (
              <li key={line} className="text-[13px] text-bone">{line}</li>
            ))}
          </ul>
        </Card>
      )}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          ['League position', profile?.league_position ? `#${profile.league_position}` : '—'],
          ['Points', String(profile?.points ?? club.pts ?? 0)],
          ['Squad OVR', String(profile?.squad_avg_ovr ?? 0)],
          ['Average age', profile?.average_age ? profile.average_age.toFixed(1) : avgAge],
          ['Squad morale', String(profile?.squad_morale ?? avgMorale)],
          ['Formation', profile?.formation || club.manager?.formation || '—'],
          ['Record', `${club.w ?? 0}W ${club.d ?? 0}D ${club.l ?? 0}L`],
          ['Board', club.board_objective || '—'],
        ].map(([label, value]) => (
          <Card key={label}>
            <p className="eyebrow">{label}</p>
            <p className="mt-1 font-mono text-[16px] font-bold text-bone">{value}</p>
          </Card>
        ))}
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <p className="eyebrow">Previous result</p>
          <FixtureMini fixture={profile?.previous_result ?? null} onWatchFixture={onWatchFixture} />
        </Card>
        <Card>
          <p className="eyebrow">Next fixture</p>
          <FixtureMini fixture={profile?.next_fixture ?? null} onWatchFixture={onWatchFixture} />
        </Card>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <PlayerStatCard title="Top scorer" player={profile?.top_scorer} stat={profile?.top_scorer ? `${profile.top_scorer.goals} G` : '—'} onOpen={onOpenPlayer} />
        <PlayerStatCard title="Top assister" player={profile?.top_assister} stat={profile?.top_assister ? `${profile.top_assister.assists} A` : '—'} onOpen={onOpenPlayer} />
        <PlayerStatCard title="Best recent performer" player={profile?.best_recent} stat={profile?.best_recent?.form_band || '—'} onOpen={onOpenPlayer} />
      </div>
      {(profile?.injuries?.length ?? 0) > 0 && (
        <Card>
          <p className="eyebrow">Major injuries</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {profile!.injuries.map((p) => (
              <button key={p.player_id} type="button" className="px-2 py-1 rounded-md border border-ember/35 text-[12px] text-[#D89A84]" onClick={() => onOpenPlayer(p.player_id)}>
                {p.full_name} · {p.injured_matches} MW
              </button>
            ))}
          </div>
        </Card>
      )}
      <ClubIdentityPanel club={club} />
    </div>
  );
};

const PlayerStatCard: React.FC<{ title: string; player: Player | null | undefined; stat: string; onOpen: (id: string) => void }> = ({ title, player, stat, onOpen }) => (
  <Card>
    <p className="eyebrow">{title}</p>
    {player ? (
      <button type="button" className="mt-2 text-left" onClick={() => onOpen(player.player_id)}>
        <p className="font-semibold text-bone">{player.full_name}</p>
        <p className="font-mono text-[12px] text-sage">{stat}</p>
      </button>
    ) : (
      <p className="mt-2 text-sage text-[13px]">No recorded leader yet.</p>
    )}
  </Card>
);

const FixtureMini: React.FC<{
  fixture: Fixture | null;
  onWatchFixture?: SquadTabProps['onWatchFixture'];
}> = ({ fixture, onWatchFixture }) => {
  if (!fixture) return <p className="mt-2 text-[13px] text-sage">No fixture logged.</p>;
  const home = fixture.home?.short_name || fixture.home_id;
  const away = fixture.away?.short_name || fixture.away_id;
  const score = fixture.status === 'finished' && fixture.home_goals != null ? `${fixture.home_goals}–${fixture.away_goals}` : 'vs';
  return (
    <button
      type="button"
      className="mt-2 w-full text-left"
      onClick={() => onWatchFixture?.(fixture, fixture.status === 'finished' ? 'result' : 'watch')}
    >
      <p className="font-semibold text-bone">{home} {score} {away}</p>
      <p className="font-mono text-[12px] text-sage">MW {fixture.matchweek} · {prettyCompetitionName(fixture.competition)}</p>
    </button>
  );
};

export const ClubFixturesPanel: React.FC<{
  club: Club;
  fixtures: Fixture[];
  filter: (typeof FIXTURE_COMP_FILTERS)[number]['id'];
  section: 'previous' | 'upcoming' | 'all';
  onFilter: (id: (typeof FIXTURE_COMP_FILTERS)[number]['id']) => void;
  onSection: (section: 'previous' | 'upcoming' | 'all') => void;
  onWatchFixture?: SquadTabProps['onWatchFixture'];
}> = ({ club, fixtures, filter, section, onFilter, onSection, onWatchFixture }) => {
  const filtered = fixtures.filter((f) => matchesFixtureFilter(f, filter, club.league));
  const shown = filtered.filter((f) => {
    if (section === 'previous') return f.status === 'finished';
    if (section === 'upcoming') return f.status !== 'finished';
    return true;
  });
  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="eyebrow flex items-center gap-1.5"><Calendar size={13} /> Club schedule</p>
        <div className="flex flex-wrap gap-1">
          {(['previous', 'upcoming', 'all'] as const).map((s) => (
            <button key={s} type="button" className={cx('px-2 py-1 rounded-md text-[11px] font-semibold uppercase border', section === s ? 'bg-brass text-ink border-brass' : 'border-line text-sage')} onClick={() => onSection(s)}>
              {s === 'all' ? 'All fixtures' : s}
            </button>
          ))}
        </div>
      </div>
      <div className="mt-3 flex flex-wrap gap-1">
        {FIXTURE_COMP_FILTERS.map((item) => (
          <button key={item.id} type="button" className={cx('px-2 py-1 rounded-md text-[11px] font-semibold border', filter === item.id ? 'bg-cardLight text-bone border-brass/40' : 'border-line text-sage')} onClick={() => onFilter(item.id)}>
            {item.label}
          </button>
        ))}
      </div>
      <div className="mt-3 divide-y divide-line/60">
        {shown.length === 0 ? (
          <p className="py-6 text-center text-[13px] text-sage">No fixtures in this view.</p>
        ) : shown.map((fixture) => {
          const opponent = fixture.home_id === club.club_id ? fixture.away : fixture.home;
          const venue = fixture.home_id === club.club_id ? 'Home' : 'Away';
          const score = fixture.status === 'finished' && fixture.home_goals != null ? `${fixture.home_goals}–${fixture.away_goals}` : fixture.status;
          return (
            <button
              key={fixture.id || fixture.fixture_id}
              type="button"
              className="w-full flex items-center justify-between gap-3 py-2.5 text-left hover:bg-cardHover"
              onClick={() => onWatchFixture?.(fixture, fixture.status === 'finished' ? 'result' : 'watch')}
            >
              <div className="min-w-0">
                <p className="text-[13px] font-semibold text-bone truncate">{prettyCompetitionName(fixture.competition)} · {opponent?.club_name || opponent?.short_name || 'Opponent'}</p>
                <p className="font-mono text-[11px] text-sage">MW {fixture.matchweek} · {venue}</p>
              </div>
              <span className="font-mono text-[13px] text-brass shrink-0">{score}</span>
            </button>
          );
        })}
      </div>
    </Card>
  );
};

export const ClubTransfersPanel: React.FC<{ activity: ClubTransferActivity | null; onOpenPlayer: (id: string) => void }> = ({ activity, onOpenPlayer }) => {
  const rows = [
    ['Arrivals', activity?.arrivals ?? []],
    ['Departures', activity?.departures ?? []],
  ] as const;
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-3">
        <Card><p className="eyebrow">Spent</p><p className="mt-1 font-mono text-[18px] font-bold text-bone">{formatEUR(activity?.spent)}</p></Card>
        <Card><p className="eyebrow">Received</p><p className="mt-1 font-mono text-[18px] font-bold text-bone">{formatEUR(activity?.received)}</p></Card>
        <Card><p className="eyebrow">Net spend</p><p className="mt-1 font-mono text-[18px] font-bold text-bone">{formatEUR(activity?.net_spend)}</p></Card>
      </div>
      {rows.map(([title, deals]) => (
        <Card key={title}>
          <p className="eyebrow">{title}</p>
          {deals.length === 0 ? (
            <p className="mt-2 text-[13px] text-sage">No recorded {title.toLowerCase()}.</p>
          ) : (
            <div className="mt-2 divide-y divide-line/60">
              {deals.map((deal) => (
                <button key={`${deal.player_id}-${deal.buyer_id}-${deal.seller_id}`} type="button" className="w-full flex justify-between py-2 text-left" onClick={() => onOpenPlayer(deal.player_id)}>
                  <span className="text-[13px] text-bone">{deal.player_name}</span>
                  <span className="font-mono text-[12px] text-sage">{deal.seller_short} → {deal.buyer_short} · {deal.formatted_fee}</span>
                </button>
              ))}
            </div>
          )}
        </Card>
      ))}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <p className="eyebrow">Loans in</p>
          {(activity?.loans_in?.length ?? 0) === 0 ? <p className="mt-2 text-[13px] text-sage">None.</p> : activity!.loans_in.map((row) => (
            <button key={row.player_id} type="button" className="block mt-2 text-[13px] text-bone" onClick={() => onOpenPlayer(row.player_id)}>{row.full_name}</button>
          ))}
        </Card>
        <Card>
          <p className="eyebrow">Loans out</p>
          {(activity?.loans_out?.length ?? 0) === 0 ? <p className="mt-2 text-[13px] text-sage">None.</p> : activity!.loans_out.map((row) => (
            <button key={row.player_id} type="button" className="block mt-2 text-[13px] text-bone" onClick={() => onOpenPlayer(row.player_id)}>{row.full_name} → {row.to_club_name}</button>
          ))}
        </Card>
      </div>
    </div>
  );
};

export const ClubFinancesPanel: React.FC<{ club: Club; squad: Player[] }> = ({ club, squad }) => (
  <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
    <Card><p className="eyebrow flex items-center gap-1"><Landmark size={12} /> Balance</p><p className="mt-1 font-mono text-[20px] font-bold text-bone">{formatEUR(club.finances?.balance)}</p></Card>
    <Card><p className="eyebrow">Transfer budget</p><p className="mt-1 font-mono text-[20px] font-bold text-brass">{formatEUR(club.finances?.transfer_budget ?? club.transfer_warchest_eur)}</p></Card>
    <Card><p className="eyebrow">Wage capacity</p><p className="mt-1 font-mono text-[20px] font-bold text-bone">{formatEUR(club.finances?.wage_cap)}</p></Card>
    <Card><p className="eyebrow">Committed wage bill</p><p className="mt-1 font-mono text-[20px] font-bold text-bone">{club.finances?.wage_bill ? formatEUR(club.finances.wage_bill) : formatWageBill(squad)}</p></Card>
    <Card><p className="eyebrow">European revenue</p><p className="mt-1 font-mono text-[20px] font-bold text-bone">{formatEUR(club.finances?.european_revenue)}</p></Card>
  </div>
);

export const ClubHistoryPanel: React.FC<{ history: ClubHistoryResponse | null }> = ({ history }) => (
  <Card>
    <p className="eyebrow flex items-center gap-1.5"><Trophy size={13} /> Club history</p>
    {(history?.history?.length ?? 0) === 0 ? (
      <p className="mt-3 text-[13px] text-sage">No completed seasons recorded in this save yet.</p>
    ) : (
      <div className="mt-3 divide-y divide-line/60">
        {history!.history.map((row, idx) => (
          <div key={`${row.season_name}-${idx}`} className="py-2.5 flex items-center justify-between gap-3">
            <div>
              <p className="font-semibold text-bone">{row.season_name}</p>
              <p className="font-mono text-[12px] text-sage">#{row.position} · {row.pts} pts · {row.w}W {row.d}D {row.l}L</p>
            </div>
            <p className="text-[12px] text-brass text-right">{Array.isArray(row.trophies) && row.trophies.length ? row.trophies.join(' · ') : ''}</p>
          </div>
        ))}
      </div>
    )}
  </Card>
);
