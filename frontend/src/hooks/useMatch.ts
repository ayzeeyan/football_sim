import { useCallback, useEffect, useRef, useState } from 'react';
import type { Club, MatchTickPayload } from '../types';
import { fetchClubs } from '../services/api';
import { matchWs, type ConnectionStatus } from '../services/matchSocket';
import { soundManager } from '../audio/webAudio';

/** Owns club list + home/away selection and keeps the backend match engine in sync. */
export function useClubs() {
  const [clubs, setClubs] = useState<Club[]>([]);
  const [homeClub, setHomeClub] = useState<Club | null>(null);
  const [awayClub, setAwayClub] = useState<Club | null>(null);

  const reloadClubs = useCallback(() => {
    return fetchClubs().then((loaded) => {
      setClubs(loaded);
      if (loaded.length === 0) return loaded;
      const rma = loaded.find((c) => c.club_id === 'LAL-RMA') ?? loaded[0];
      const ars = loaded.find((c) => c.club_id === 'EPL-ARS') ?? loaded[1 % loaded.length];
      setHomeClub(rma);
      setAwayClub(ars);
      matchWs.sendCommand('set_clubs', { home_id: rma.club_id, away_id: ars.club_id });
      return loaded;
    });
  }, []);

  useEffect(() => {
    void reloadClubs();
  }, [reloadClubs]);

  const applyClubs = useCallback((home: Club | null, away: Club | null) => {
    setHomeClub(home);
    setAwayClub(away);
    if (home && away) matchWs.sendCommand('set_clubs', { home_id: home.club_id, away_id: away.club_id });
  }, []);

  const setHome = useCallback(
    (c: Club) => {
      setHomeClub(c);
      if (awayClub) matchWs.sendCommand('set_clubs', { home_id: c.club_id, away_id: awayClub.club_id });
    },
    [awayClub],
  );

  const setAway = useCallback(
    (c: Club) => {
      setAwayClub(c);
      if (homeClub) matchWs.sendCommand('set_clubs', { home_id: homeClub.club_id, away_id: c.club_id });
    },
    [homeClub],
  );

  const swap = useCallback(() => {
    if (!homeClub || !awayClub) return;
    applyClubs(awayClub, homeClub);
  }, [homeClub, awayClub, applyClubs]);

  return { clubs, homeClub, awayClub, setHome, setAway, swap, applyClubs, reloadClubs };
}

/** Owns the WS lifecycle + latest tick + command helpers. Goal roar handled once here. */
export function useMatchEngine() {
  const [matchData, setMatchData] = useState<MatchTickPayload | null>(null);
  const [status, setStatus] = useState<ConnectionStatus>('CONNECTING');
  const lastBanner = useRef<string | null>(null);

  useEffect(() => {
    const unsub = matchWs.addStatusListener(setStatus);
    matchWs.connect((tick) => {
      setMatchData(tick);
      if (tick.goal_banner && tick.goal_banner !== lastBanner.current) {
        lastBanner.current = tick.goal_banner;
        soundManager.playGoalRoar();
      }
      if (!tick.goal_banner) lastBanner.current = null;
    });
    return () => {
      unsub();
      matchWs.disconnect();
    };
  }, []);

  const kickoff = useCallback(() => {
    soundManager.playWhistle();
    matchWs.sendCommand('kickoff');
  }, []);

  const pause = useCallback(() => {
    soundManager.playClick();
    matchWs.sendCommand('pause');
  }, []);

  const setSpeed = useCallback((speed: number) => matchWs.sendCommand('set_speed', { speed }), []);

  const reset = useCallback(() => matchWs.sendCommand('reset'), []);

  const seek70 = useCallback(() => matchWs.sendCommand('seek_70'), []);

  const seekChance = useCallback(() => matchWs.sendCommand('seek_chance'), []);

  return { matchData, status, kickoff, pause, setSpeed, reset, seek70, seekChance };
}
