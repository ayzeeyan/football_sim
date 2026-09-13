import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import type { PlayerProfile } from '../types';
import { fetchPlayerProfile } from '../services/api';
import { ovrTone } from '../lib/constants';
import { cx, loyaltyLabel } from '../lib/format';
import { soundManager } from '../audio/webAudio';
import { ClubCrest, LoadingState, Modal, ModalHeader, PlayerPortrait } from './ui/ui';

interface PlayerSheetApi {
  openPlayer: (playerId: string) => void;
}

const PlayerSheetContext = createContext<PlayerSheetApi>({ openPlayer: () => undefined });

export function usePlayerSheet(): PlayerSheetApi {
  return useContext(PlayerSheetContext);
}

export const PlayerSheetProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [playerId, setPlayerId] = useState<string | null>(null);
  const [profile, setProfile] = useState<PlayerProfile | null>(null);
  const [loading, setLoading] = useState(false);

  const openPlayer = useCallback((id: string) => {
    if (!id) return;
    soundManager.playClick();
    setPlayerId(id);
  }, []);

  useEffect(() => {
    if (!playerId) {
      setProfile(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    fetchPlayerProfile(playerId)
      .then((data) => {
        if (!cancelled) setProfile(data);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [playerId]);

  return (
    <PlayerSheetContext.Provider value={{ openPlayer }}>
      {children}
      <PlayerSheetModal
        profile={profile}
        loading={loading}
        open={!!playerId}
        onClose={() => setPlayerId(null)}
      />
    </PlayerSheetContext.Provider>
  );
};

const RESULT_TONE: Record<string, string> = {
  W: 'text-[#A9CDBB]',
  D: 'text-brass',
  L: 'text-[#D89A84]',
};

const COMPETITION_LABELS: Record<string, string> = {
  'premier-league': 'Premier League',
  'la-liga': 'La Liga',
  'bundesliga': 'Bundesliga',
  'serie-a': 'Serie A',
  'ligue-1': 'Ligue 1',
  'fa-cup': 'FA Cup',
  'efl-cup': 'EFL Cup',
  'copa-del-rey': 'Copa del Rey',
  'dfb-pokal': 'DFB-Pokal',
  'coppa-italia': 'Coppa Italia',
  'coupe-de-france': 'Coupe de France',
  'champions-league': 'UEFA Champions League',
  'europa-league': 'UEFA Europa League',
  'conference-league': 'UEFA Conference League',
  'super-league': 'Super League',
  'ucl': 'Champions Cup',
  'super-cup': 'Super Cup',
};

export function prettyCompetitionName(id: string | null | undefined): string {
  if (!id) return 'League';
  const key = id.trim().toLowerCase();
  if (COMPETITION_LABELS[key]) return COMPETITION_LABELS[key];
  return id
    .replace(/[_]+/g, ' ')
    .replace(/-/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export const PlayerSheetModal: React.FC<{
  profile: PlayerProfile | null;
  loading: boolean;
  open: boolean;
  onClose: () => void;
}> = ({ profile, loading, open, onClose }) => {
  const player = profile?.player;
  return (
    <Modal open={open} onClose={onClose} maxWidth="max-w-lg">
      {loading && !player ? (
        <div className="p-8">
          <LoadingState message="Loading player…" />
        </div>
      ) : player ? (
        <>
          <ModalHeader
            title={player.full_name}
            subtitle={`${player.position} · ${profile?.club?.club_name ?? ''} · ${
              player.universe_wonderkid
                ? 'U-17 prodigy'
                : player.on_loan
                  ? 'On loan'
                  : player.player_source === 'academy'
                    ? 'Academy'
                    : player.player_source === 'headline'
                      ? 'Headline'
                      : player.squad_role || 'Squad'
            }`}
            onClose={onClose}
          />
          <div className="p-6 space-y-4 overflow-y-auto">
            <div className="flex items-end justify-between gap-3">
              <div className="flex items-center gap-3 min-w-0">
                <PlayerPortrait player={player} size={48} />
                {profile?.club && <ClubCrest club={profile.club} size={48} />}
                <div>
                  <p className="eyebrow">Overall</p>
                  <p className={cx('score-display text-[42px] leading-none', ovrTone(player.ovr))}>{player.ovr}</p>
                </div>
              </div>
              <div className="text-right font-mono text-[13px] text-sage space-y-1">
                <p>Age {player.age}</p>
                <p>{player.formatted_value}</p>
                <p>{player.formatted_wage}</p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div className="border border-line bg-ink/40 p-3">
                <p className="eyebrow">This season</p>
                <p className="font-mono font-bold text-bone mt-1">{player.appearances} apps</p>
                <p className="font-mono text-[13px] text-sage">
                  {player.goals} G · {player.assists} A
                </p>
                {profile?.avg_rating != null && (
                  <p className="font-mono text-[12px] text-brass mt-1">{profile.avg_rating.toFixed(2)} avg rating</p>
                )}
              </div>
              <div className="border border-brass/40 bg-brass/[0.07] p-3">
                <p className="eyebrow !text-brass">All-time</p>
                <p className="font-mono font-bold text-bone mt-1">{player.career_apps ?? player.appearances} apps</p>
                <p className="font-mono text-[13px] text-sage">
                  {player.career_goals ?? 0} G · {player.career_assists ?? 0} A
                </p>
              </div>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
              <div className="border border-line bg-ink/40 p-3">
                <p className="eyebrow">Role</p>
                <p className="font-semibold text-bone mt-1">{player.squad_role || 'Squad'}</p>
              </div>
              <div className="border border-line bg-ink/40 p-3">
                <p className="eyebrow">Morale</p>
                <p className="font-semibold text-bone mt-1">{player.morale_band || 'Content'}</p>
                <p className="font-mono text-[12px] text-sage">{player.morale ?? 70}</p>
              </div>
              <div className="border border-line bg-ink/40 p-3">
                <p className="eyebrow">Fitness</p>
                <p className="font-mono font-bold text-bone mt-1">{player.fitness ?? 80}</p>
              </div>
              <div className="border border-line bg-ink/40 p-3">
                <p className="eyebrow">Sharpness</p>
                <p className="font-mono font-bold text-bone mt-1">{player.sharpness ?? 65}</p>
                <p className="text-[12px] text-sage">{player.form_band || 'Average'} form</p>
              </div>
            </div>

            {player.competition_stats && Object.keys(player.competition_stats).length > 0 && (
              <div className="border border-line bg-ink/40 p-3">
                <p className="eyebrow mb-2">By competition</p>
                <ul className="space-y-1.5">
                  {Object.values(player.competition_stats).map((row) => (
                    <li key={row.competition_id} className="flex items-center justify-between gap-2 text-[12.5px]">
                      <span className="text-bone truncate">{prettyCompetitionName(row.competition_id)}</span>
                      <span className="font-mono text-sage shrink-0">
                        {row.appearances} apps · {row.goals} G · {row.assists} A
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            <div className="border border-line bg-ink/40 p-3 text-[13px] space-y-1.5">
              <p>
                <span className="text-sage">Contract</span>
                {' · '}
                <span className="text-bone font-semibold">
                  {player.contract_years} yr{player.contract_years === 1 ? '' : 's'} left
                </span>
              </p>
              <p>
                <span className="text-sage">Loyalty</span>
                {' · '}
                <span className="text-bone font-semibold">
                  {player.loyalty} · {loyaltyLabel(player.loyalty)}
                </span>
              </p>
              <p>
                <span className="text-sage">Best season</span>
                {' · '}
                <span className="text-bone font-semibold">
                  {player.best_season ? `${player.best_goals} G · ${player.best_season}` : 'Yet to come'}
                </span>
              </p>
              <p>
                <span className="text-sage">Availability</span>
                {' · '}
                <span className={cx('font-semibold', (player.injured_matches ?? 0) > 0 || (player.suspended_matches ?? 0) > 0 ? 'text-ember' : 'text-bone')}>
                  {player.availability ?? ((player.injured_matches ?? 0) > 0 || (player.suspended_matches ?? 0) > 0 ? 'Out' : 'Available')}
                </span>
              </p>
              {player.universe_wonderkid && player.education_label && (
                <p>
                  <span className="text-sage">School</span>
                  {' · '}
                  <span className="text-bone font-semibold">{player.education_label}</span>
                </p>
              )}
              {player.secondary_position && (
                <p>
                  <span className="text-sage">Positions</span>
                  {' · '}
                  <span className="text-bone font-semibold">{player.position} / {player.secondary_position}</span>
                </p>
              )}
            </div>

            <div>
              <p className="eyebrow mb-2">Last matches</p>
              {(profile?.last_matches?.length ?? 0) === 0 ? (
                <p className="text-[13px] text-sage">No rated appearances this season.</p>
              ) : (
                <div className="space-y-1">
                  {(profile?.last_matches ?? []).map((m) => (
                    <div
                      key={`${m.fixture_id}-${m.matchweek}`}
                      className="flex items-center justify-between gap-2 py-1.5 border-b border-line/60 last:border-0 text-[12.5px]"
                    >
                      <div className="min-w-0">
                        <p className="text-bone font-semibold truncate">
                          <span className={cx('font-mono mr-1.5', RESULT_TONE[m.result])}>{m.result}</span>
                          {m.home ? 'vs' : '@'} {m.opponent} {m.score}
                        </p>
                        <p className="font-mono text-sage text-[11px]">
                          MW {m.matchweek}
                          {m.competition ? ` · ${prettyCompetitionName(m.competition)}` : ''}
                          {m.motm ? ' · MOTM' : ''} · {m.minutes}'
                          {m.goals ? ` · ${m.goals}G` : ''}
                          {m.assists ? ` · ${m.assists}A` : ''}
                        </p>
                      </div>
                      <span
                        className={cx(
                          'font-mono font-bold shrink-0',
                          (m.rating ?? 0) >= 8 ? 'text-brass' : 'text-bone',
                        )}
                      >
                        {m.rating != null ? m.rating.toFixed(1) : '—'}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </>
      ) : (
        <div className="p-8 text-center text-sage text-[13px]">That player is not in this career.</div>
      )}
    </Modal>
  );
};

export const PlayerNameButton: React.FC<{
  playerId?: string | null;
  children: React.ReactNode;
  className?: string;
}> = ({ playerId, children, className }) => {
  const { openPlayer } = usePlayerSheet();
  if (!playerId) {
    return <span className={className}>{children}</span>;
  }
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        openPlayer(playerId);
      }}
      className={cx('text-left hover:text-brass transition-colors', className)}
    >
      {children}
    </button>
  );
};

