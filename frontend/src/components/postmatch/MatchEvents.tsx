import React from 'react';
import type { Club, Fixture, MatchEventItem } from '../../types';
import {
  clubForEvent,
  eventClock,
  eventIcon,
  eventPlayerId,
  eventPlayerLabel,
  eventSecondaryLine,
  isKeyMoment,
} from '../../lib/eventIdentity';

export function EventRow({
  event,
  fixture,
  home,
  away,
  onOpenPlayer,
  showClub,
}: {
  event: MatchEventItem;
  fixture?: Fixture | null;
  home: Club | null;
  away: Club | null;
  onOpenPlayer: (id: string) => void;
  showClub?: boolean;
}) {
  const playerId = event.type === 'sub'
    ? (event.player_in?.player_id || eventPlayerId(event))
    : eventPlayerId(event);
  const name = event.type === 'sub'
    ? (event.player_in?.full_name || eventPlayerLabel(event, fixture))
    : eventPlayerLabel(event, fixture);
  const secondary = eventSecondaryLine(event);
  const club = showClub ? clubForEvent(event, home?.short_name, away?.short_name) : '';

  return (
    <li className="event-row">
      <span className="event-clock">{eventClock(event)}</span>
      <span className="event-icon" aria-hidden="true">{eventIcon(event)}</span>
      <div className="min-w-0">
        {playerId ? (
          <button type="button" onClick={() => onOpenPlayer(playerId)} className="event-row-name text-left hover:text-brass">
            {name}
          </button>
        ) : (
          <p className="event-row-name">{name}</p>
        )}
        <p className="event-row-meta">{secondary}</p>
        {club ? <p className="event-row-meta">{club}</p> : null}
      </div>
    </li>
  );
}

export function KeyMoments({
  events,
  fixture,
  home,
  away,
  onOpenPlayer,
}: {
  events: MatchEventItem[];
  fixture?: Fixture | null;
  home: Club | null;
  away: Club | null;
  onOpenPlayer: (id: string) => void;
}) {
  const moments = events.filter(isKeyMoment).slice(0, 10);
  if (!moments.length) {
    return <p className="mt-3 text-[12px] text-sage">No major events recorded.</p>;
  }
  return (
    <ol className="mt-1 divide-y divide-white/[0.07]">
      {moments.map((event, index) => (
        <EventRow key={`${event.seq}-${index}`} event={event} fixture={fixture} home={home} away={away} onOpenPlayer={onOpenPlayer} />
      ))}
    </ol>
  );
}

function periodMarker(label: string, clock: string) {
  return (
    <li key={label} className="event-row py-2 opacity-80">
      <span className="event-clock">{clock}</span>
      <span className="event-icon" aria-hidden="true">●</span>
      <p className="event-row-name uppercase tracking-[0.12em] text-sage">{label}</p>
    </li>
  );
}

export function MatchTimeline({
  events,
  fixture,
  home,
  away,
  onOpenPlayer,
}: {
  events: MatchEventItem[];
  fixture?: Fixture | null;
  home: Club | null;
  away: Club | null;
  onOpenPlayer: (id: string) => void;
}) {
  if (!events.length) {
    return <p className="p-10 text-center text-[12px] text-sage">No match events recorded.</p>;
  }
  const kickOff = periodMarker('Kick off', "0'");
  const halfTime = periodMarker('Half time', "45'");
  const fullTime = periodMarker('Full time', "90'");
  const firstHalf: React.ReactElement[] = [];
  const secondHalf: React.ReactElement[] = [];
  for (const event of events) {
    const isFirstHalf = event.minute < 45;
    const periodEvents = isFirstHalf ? firstHalf : secondHalf;
    const index = periodEvents.length;
    periodEvents.push(
      <EventRow key={`${isFirstHalf ? 'h1' : 'h2'}-${event.seq}-${index}`} event={event} fixture={fixture} home={home} away={away} onOpenPlayer={onOpenPlayer} showClub />,
    );
  }
  const timelineRows = [
    fullTime,
    ...secondHalf.reverse(),
    halfTime,
    ...firstHalf.reverse(),
    kickOff,
  ];
  return <ol className="divide-y divide-white/[0.07]">{timelineRows}</ol>;
}
