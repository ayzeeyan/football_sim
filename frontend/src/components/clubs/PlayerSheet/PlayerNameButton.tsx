import React from 'react';
import { cx } from '../../../lib/format';
import { usePlayerSheet } from './context';

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
