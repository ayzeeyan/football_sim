import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import type { PlayerProfile } from '../../../types';
import { fetchPlayerProfile } from '../../../services/api';
import { soundManager } from '../../../audio/webAudio';
import { PlayerSheetModal } from './PlayerSheetModal';


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
  const [error, setError] = useState<string | null>(null);
  const [reloadNonce, setReloadNonce] = useState(0);

  const openPlayer = useCallback((id: string) => {
    if (!id) return;
    soundManager.playClick();
    setPlayerId(id);
  }, []);

  useEffect(() => {
    if (!playerId) {
      setProfile(null);
      setError(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    setProfile(null);
    fetchPlayerProfile(playerId)
      .then((data) => {
        if (!cancelled) setProfile(data);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Could not load this player.');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [playerId, reloadNonce]);

  return (
    <PlayerSheetContext.Provider value={{ openPlayer }}>
      {children}
      <PlayerSheetModal
        profile={profile}
        loading={loading}
        error={error}
        open={!!playerId}
        onClose={() => setPlayerId(null)}
        onRetry={() => setReloadNonce((n) => n + 1)}
      />
    </PlayerSheetContext.Provider>
  );
};