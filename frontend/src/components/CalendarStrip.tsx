import React from 'react';
import type { CalendarState } from '../services/api';
import { cx } from '../lib/format';

export const CalendarStrip: React.FC<{ calendar: CalendarState; onPickWeek?: (mw: number) => void; viewMw: number }> = ({
  calendar,
  onPickWeek,
  viewMw,
}) => {
  if (!calendar.weeks.length) return null;
  const thisWeek = calendar.this_week ?? calendar.current_matchweek;
  const nextCup = calendar.next_cup_night ?? null;
  return (
    <div className="panel-pad space-y-3">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <p className="eyebrow">{calendar.month} · {calendar.phase}</p>
          <h3 className="font-display text-[20px] font-semibold text-bone mt-1">
            {calendar.chapter ?? `${calendar.max_matchweeks}-week calendar`}
          </h3>
        </div>
        <div className="flex gap-3 text-[11px] font-mono text-sage">
          <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-bone" /> League</span>
          <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-brass" /> Champions Cup</span>
          <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-pitchtone" /> Super Cup</span>
        </div>
      </div>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => onPickWeek?.(thisWeek)}
          className="px-3 py-1.5 text-[12px] font-semibold border border-line bg-cardLight hover:bg-cardHover text-bone"
        >
          This week
        </button>
        {nextCup != null && (
          <button
            type="button"
            onClick={() => onPickWeek?.(nextCup)}
            className="px-3 py-1.5 text-[12px] font-semibold border border-brass/40 bg-brass/10 hover:bg-brass/20 text-brass"
          >
            Next cup night{nextCup !== thisWeek ? ` · MW ${nextCup}` : ''}
          </button>
        )}
      </div>
      <div className="flex flex-wrap gap-1">
        {calendar.weeks.map((w) => {
          const isView = w.matchweek === viewMw;
          const isNow = w.current;
          return (
            <button
              key={w.matchweek}
              type="button"
              title={`MW ${w.matchweek} · ${w.month} · ${w.phase}`}
              onClick={() => onPickWeek?.(w.matchweek)}
              className={cx(
                'w-7 h-8 rounded-md border flex flex-col items-center justify-center gap-0.5',
                isView ? 'border-brass bg-brass/15' : isNow ? 'border-bone/50 bg-ink/60' : 'border-line bg-ink/40 hover:border-sage/50',
              )}
            >
              <span className="text-[9px] font-mono text-sage leading-none">{w.matchweek}</span>
              <span className="flex gap-0.5">
                {w.league > 0 && <span className="w-1 h-1 rounded-full bg-bone" />}
                {w.ucl > 0 && <span className="w-1 h-1 rounded-full bg-brass" />}
                {w.super_cup > 0 && <span className="w-1 h-1 rounded-full bg-pitchtone" />}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
};
