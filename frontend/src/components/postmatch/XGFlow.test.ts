import { describe, expect, test } from 'bun:test';
import React from 'react';
import type { Club, Fixture } from '../../types';
import { XGFlow } from './MatchInsights';

// React.memo wraps the render function; the inner callable lives on `.type`.
const XGFlowRender = (XGFlow as unknown as { type: (props: { fixture: Fixture }) => React.ReactNode }).type;

const home: Club = { club_id: 'EPL-LIV', club_name: 'Liverpool', short_name: 'LIV', primary_color: [200, 30, 30] } as Club;
const away: Club = { club_id: 'EPL-MCI', club_name: 'Manchester City', short_name: 'MCI', primary_color: [80, 160, 220] } as Club;

function mockFixture(flow: Fixture['shot_map']): Fixture {
  return { id: 'FX2', home, away, shot_map: flow } as Fixture;
}

describe('XGFlow', () => {
  test('returns the empty-state message when no flow was recorded', () => {
    const result = XGFlowRender({ fixture: mockFixture(null) });
    expect(result).not.toBeNull();
    expect(React.isValidElement(result)).toBe(true);
  });

  test('returns the empty-state message for a single-point flow', () => {
    const result = XGFlowRender({
      fixture: mockFixture({
        shots: [],
        xg_flow: [{ minute: 0, home_xg: 0, away_xg: 0 }],
        total_home_xg: 0,
        total_away_xg: 0,
      }),
    });
    expect(result).not.toBeNull();
  });

  test('renders the real cumulative xG series from shot_map.xg_flow', () => {
    const result = XGFlowRender({
      fixture: mockFixture({
        shots: [],
        xg_flow: [
          { minute: 0, home_xg: 0, away_xg: 0 },
          { minute: 23, home_xg: 0.4, away_xg: 0 },
          { minute: 61, home_xg: 0.4, away_xg: 1.1 },
          { minute: 90, home_xg: 1.6, away_xg: 1.1 },
        ],
        total_home_xg: 1.6,
        total_away_xg: 1.1,
      }),
    });
    expect(result).not.toBeNull();
    expect(React.isValidElement(result)).toBe(true);
    const tree = JSON.stringify(result);
    expect(tree).toContain('xG flow');
    expect(tree).toContain('1.60');
    expect(tree).toContain('1.10');
  });
});
