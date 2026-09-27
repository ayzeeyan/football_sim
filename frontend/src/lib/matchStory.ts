import type { Club, Fixture, MatchEventItem, StoryFact, TeamMatchStats } from '../types';
import { eventClock, eventPlayerId, eventPlayerLabel, isScoringEvent } from './eventIdentity';

export function normalizeTeamStats(raw?: Partial<TeamMatchStats> & {
  shots_on_target?: number;
  yellow_cards?: number;
  red_cards?: number;
} | null): TeamMatchStats {
  return {
    shots: raw?.shots ?? 0,
    on_target: raw?.on_target ?? raw?.shots_on_target ?? 0,
    possession: raw?.possession ?? 50,
    corners: raw?.corners ?? 0,
    passes: raw?.passes ?? 0,
    pass_accuracy: raw?.pass_accuracy ?? 0,
    fouls: raw?.fouls ?? 0,
    yellows: raw?.yellows ?? raw?.yellow_cards ?? 0,
    reds: raw?.reds ?? raw?.red_cards ?? 0,
    xg: raw?.xg,
    saves: raw?.saves,
    big_chances: raw?.big_chances,
  };
}

export function scoringEvents(events: MatchEventItem[]): MatchEventItem[] {
  return events.filter(isScoringEvent);
}

export function scoringSide(event: MatchEventItem): 'home' | 'away' {
  if (event.type === 'own_goal') {
    if (event.beneficiary === 'home' || event.beneficiary === 'away') return event.beneficiary;
    return event.side === 'home' ? 'away' : 'home';
  }
  return event.side === 'away' ? 'away' : 'home';
}

function ordinalMinute(minute?: number): string {
  if (!minute || minute < 1) return '';
  return `in the ${minute}th minute`;
}

function nameFor(side: string | undefined, homeName: string, awayName: string, fallback: string): string {
  if (side === 'home') return homeName;
  if (side === 'away') return awayName;
  return fallback;
}

/** 2–4 factual sentences. Never claims that one event caused another. */
export function composeMatchStory(
  home: Club | null,
  away: Club | null,
  homeScore: number,
  awayScore: number,
  facts: StoryFact[] = [],
  events: MatchEventItem[] = [],
): string {
  const homeName = home?.club_name || 'The home side';
  const awayName = away?.club_name || 'the visitors';
  const club = (side?: string, fallback = homeName) => nameFor(side, homeName, awayName, fallback);
  const late = facts.find((f) => f.kind === 'late_winner');
  const winner = facts.find((f) => f.kind === 'winner') ?? late;
  const opening = facts.find((f) => f.kind === 'opening_goal');
  const comeback = facts.find((f) => f.kind === 'two_goal_comeback' || f.kind === 'comeback');
  const hat = facts.find((f) => f.kind === 'hat_trick');
  const red = facts.find((f) => f.kind === 'red_card');
  const extra = facts.find((f) => f.kind === 'extra_time');
  const pens = facts.find((f) => f.kind === 'penalty_shootout');
  const injured = facts.find((f) => f.kind === 'injury');
  const clean = facts.find((f) => f.kind === 'clean_sheet');
  const multi = facts.find((f) => f.kind === 'multi_goal_lead');
  const sentences: string[] = [];

  if (homeScore === awayScore) {
    if (opening?.player_name) {
      sentences.push(`${club(opening.side, homeName)} opened the scoring${opening.minute ? ` ${ordinalMinute(opening.minute)}` : ''} through ${opening.player_name}, but the match finished ${homeScore}–${awayScore}.`);
    } else {
      sentences.push(`${homeName} and ${awayName} shared a ${homeScore}–${awayScore} draw.`);
    }
  } else {
    const winnerName = homeScore > awayScore ? homeName : awayName;
    if (opening?.player_name && opening.minute && opening.minute <= 45 && (homeScore + awayScore) >= 2) {
      sentences.push(`${club(opening.side, winnerName)} opened the scoring ${ordinalMinute(opening.minute)} through ${opening.player_name}.`);
    }
    if (comeback) {
      const trail = comeback.kind === 'two_goal_comeback' ? 'from two goals down' : 'from behind';
      if (late?.player_name) {
        sentences.push(`${club(comeback.side, winnerName)} recovered ${trail}, with ${late.player_name} scoring the winner ${ordinalMinute(late.minute)}.`);
      } else {
        sentences.push(`${club(comeback.side, winnerName)} recovered ${trail} to win ${homeScore}–${awayScore}.`);
      }
    } else if (late?.player_name) {
      sentences.push(`${late.player_name} scored the winner ${ordinalMinute(late.minute)} as ${winnerName} won ${homeScore}–${awayScore}.`);
    } else if (winner?.player_name) {
      sentences.push(`${winnerName} won ${homeScore}–${awayScore}, with ${winner.player_name} scoring the winner.`);
    } else if (scoringEvents(events).length === 0) {
      sentences.push(`${winnerName} won ${homeScore}–${awayScore}.`);
    } else {
      sentences.push(`${winnerName} won ${homeScore}–${awayScore}.`);
    }
    if (multi) {
      sentences.push(`${club(multi.side, winnerName)} finished with a three-goal margin.`);
    }
  }

  if (hat?.player_name) sentences.push(`${hat.player_name} completed a hat-trick.`);
  if (red?.player_name) {
    const when = red.minute ? ` ${ordinalMinute(red.minute)}` : '';
    sentences.push(`${club(red.side, homeName)} were reduced to ten men${when} after ${red.player_name} was sent off.`);
  }
  if (injured?.player_name) {
    const when = injured.minute ? ` ${ordinalMinute(injured.minute)}` : '';
    sentences.push(`${injured.player_name} went off injured${when}.`);
  }
  if (clean && homeScore !== awayScore) {
    sentences.push(`${club(clean.side)} kept a clean sheet.`);
  }
  if (extra) sentences.push('The tie required extra time.');
  if (pens) sentences.push('The match was decided on penalties.');

  if (sentences.length === 0) {
    return `${homeName} ${homeScore}–${awayScore} ${awayName}.`;
  }
  return sentences.slice(0, 4).join(' ');
}

export function scorerLines(events: MatchEventItem[], side: 'home' | 'away'): string[] {
  const order: string[] = [];
  const buckets = new Map<string, { name: string; clocks: string[] }>();
  for (const event of scoringEvents(events).filter((item) => scoringSide(item) === side)) {
    const name = eventPlayerLabel(event);
    const key = eventPlayerId(event) || name;
    const clock = eventClock(event);
    const marked = event.type === 'own_goal' ? `${clock} (og)` : event.type === 'penalty' ? `${clock} (pen)` : clock;
    if (!buckets.has(key)) {
      order.push(key);
      buckets.set(key, { name, clocks: [] });
    }
    buckets.get(key)!.clocks.push(marked);
  }
  return order.map((key) => {
    const row = buckets.get(key)!;
    return `${row.name} ${row.clocks.join(', ')}`;
  });
}

export function otherResultsFor(fixture: Fixture | null | undefined, pool: Fixture[]): Fixture[] {
  if (!fixture) return pool.filter((row) => row.status === 'finished');
  return pool.filter((row) => (
    row.id !== fixture.id
    && row.status === 'finished'
    && row.matchweek === fixture.matchweek
    && (row.competition === fixture.competition || (row.stage && row.stage === fixture.stage))
  ));
}
