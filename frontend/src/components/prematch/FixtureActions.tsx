import React from 'react';
import { Zap } from 'lucide-react';
import { cx } from '../../lib/format';

type ActionFixture = {
  status: string;
  home: { club_name?: string | null } | null;
  away: { club_name?: string | null } | null;
};

export interface FixtureActionsProps<T extends ActionFixture = ActionFixture> {
  fixture: T;
  onSimulate?: (fixture: T) => void;
  busy?: boolean;
  className?: string;
}

/** A single simulation action keeps the match flow quick and consistent. */
export function FixtureActions<T extends ActionFixture>({ fixture, onSimulate, busy = false, className }: FixtureActionsProps<T>) {
  return (
    <div role="group" aria-label={`Fixture simulation: ${fixture.home?.club_name || 'Home'} vs ${fixture.away?.club_name || 'Away'}`} aria-busy={busy} className={cx('match-action-dock shrink-0', className)}>
      <p className="min-w-0 flex-1 text-[12px] leading-relaxed text-sage">
        Resolve the fixture instantly and open its full report.
      </p>
      <button
        type="button"
        disabled={busy || fixture.status !== 'scheduled' || !onSimulate}
        onClick={() => { if (!busy && fixture.status === 'scheduled') onSimulate?.(fixture); }}
        className="match-action-primary"
      >
        <Zap size={15} aria-hidden="true" />
        <span>{busy ? 'Simulating…' : 'Simulate match'}</span>
      </button>
    </div>
  );
}
