import { useCallback, useEffect, useState } from 'react';
import type { Club } from '../types';
import { fetchClubs } from '../services/api';

/** Loads the club index for inspection screens without starting a match engine. */
export function useClubs() {
  const [clubs, setClubs] = useState<Club[]>([]);

  const reloadClubs = useCallback(async () => {
    const loaded = await fetchClubs();
    setClubs(loaded);
    return loaded;
  }, []);

  useEffect(() => {
    void reloadClubs();
  }, [reloadClubs]);

  return { clubs, reloadClubs };
}
