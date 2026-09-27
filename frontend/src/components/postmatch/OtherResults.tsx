import React from 'react';
import type { Fixture } from '../../types';
import { ClubCrest } from '../ui/ui';

export function OtherResults({
  results,
  onOpenFixture,
}: {
  results: Fixture[];
  onOpenFixture?: (id: string) => void;
}) {
  if (!results.length) {
    return <p className="mt-3 text-[12px] text-sage">No other completed fixtures were returned for this slate.</p>;
  }
  return (
    <div className="mt-2 divide-y divide-white/[0.07]">
      {results.slice(0, 10).map((result) => {
        const body = (
          <>
            <span className="flex min-w-0 items-center gap-2"><ClubCrest club={result.home} size={20} className="!border-0 !bg-transparent" /><span className="truncate text-bone">{result.home.short_name}</span></span>
            <span className="text-center"><strong className="block rounded bg-black/25 px-2 py-1 font-mono tabular-nums text-brass">{result.home_goals}–{result.away_goals}</strong><small className="mt-0.5 block text-[8px] uppercase tracking-[0.12em] text-sage">FT</small></span>
            <span className="flex min-w-0 items-center justify-end gap-2"><span className="truncate text-right text-bone">{result.away.short_name}</span><ClubCrest club={result.away} size={20} className="!border-0 !bg-transparent" /></span>
          </>
        );
        const layout = 'grid w-full grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2 py-2.5 text-left text-[12px]';
        if (onOpenFixture && result.status === 'finished') {
          return (
            <button key={result.id} type="button" onClick={() => onOpenFixture(result.id)} className={`${layout} hover:bg-white/[0.03]`}>
              {body}
            </button>
          );
        }
        return <div key={result.id} className={layout}>{body}</div>;
      })}
    </div>
  );
}
