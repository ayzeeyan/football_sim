import React, { useEffect, useState } from 'react';
import { WifiOff } from 'lucide-react';

/**
 * Offline banner (F14): surfaces connectivity loss so the viewer knows the
 * world is being read from the last-good cache, not live state.
 */
export const OfflineBanner: React.FC = () => {
  const [online, setOnline] = useState(true);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    setOnline(window.navigator.onLine);
    const goOnline = () => setOnline(true);
    const goOffline = () => setOnline(false);
    window.addEventListener('online', goOnline);
    window.addEventListener('offline', goOffline);
    return () => {
      window.removeEventListener('online', goOnline);
      window.removeEventListener('offline', goOffline);
    };
  }, []);

  if (online) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className="flex items-center justify-center gap-2 border-b border-brass/30 bg-brass/10 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.1em] text-brass"
    >
      <WifiOff size={13} aria-hidden="true" />
      Offline — showing the last cached snapshot. Simulations resume when the server returns.
    </div>
  );
};
