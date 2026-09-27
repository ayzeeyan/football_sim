import { describe, expect, test } from 'bun:test';
import React from 'react';
import type { Club, Fixture } from '../../types';
import { PressConference } from './PressConference';

const home: Club = { club_id: 'LAL-RMA', club_name: 'Real Madrid', short_name: 'RMA' } as Club;
const away: Club = { club_id: 'LAL-BAR', club_name: 'Barcelona', short_name: 'BAR' } as Club;

function mockFixture(press: Fixture['press_conference']): Fixture {
  return { id: 'FX1', press_conference: press } as Fixture;
}

describe('PressConference', () => {
  test('returns null when the fixture has no press conference', () => {
    const result = PressConference({ fixture: mockFixture(null), home, away });
    expect(result).toBeNull();
  });

  test('returns null when the conference is fully empty', () => {
    const result = PressConference({
      fixture: mockFixture({ headline: '', home_quote: '', away_quote: '' }),
      home,
      away,
    });
    expect(result).toBeNull();
  });

  test('renders quotes and headline when the backend supplied them', () => {
    const result = PressConference({
      fixture: mockFixture({
        headline: 'Madrid edge Clasico thriller',
        home_quote: 'The boys were magnificent.',
        away_quote: 'We deserved more from the game.',
        home_manager: 'Carlo Ancelotti',
        away_manager: 'Hansi Flick',
        narrative: 'A late winner settled a tense contest.',
      }),
      home,
      away,
    });
    expect(result).not.toBeNull();
    expect(React.isValidElement(result)).toBe(true);
  });

  test('renders with only a headline (no quotes)', () => {
    const result = PressConference({
      fixture: mockFixture({ headline: 'Honours even in the capital' }),
      home,
      away,
    });
    expect(result).not.toBeNull();
  });
});
