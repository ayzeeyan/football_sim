import type { CommentaryItem, MatchEventItem, MatchTickPayload } from '../types';

export type ConnectionStatus = 'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'RECONNECTING';
export type WsCommand = Record<string, unknown> & { action: string };

const MAX_BACKOFF_MS = 8000;
const BASE_BACKOFF_MS = 800;

/**
 * Robust WebSocket client for the 60fps match engine.
 * - Queues commands sent while disconnected and replays them flat
 *   as `{ action, ...payload }` (fixes the legacy `{action, payload}` bug).
 * - Exponential backoff with jitter for reconnects.
 */
/** Slim per-actor update shipped on delta ticks (no player blob). */
export interface DeltaCoord {
  x: number;
  y: number;
  player_id: string;
  sent_off?: boolean;
}

/** Ordinary-tick update applied onto the last full snapshot. */
export interface DeltaTick {
  tick_type: 'delta';
  seq: number;
  state: MatchTickPayload['state'];
  minute: number;
  speed: number;
  phase: string;
  active_third: MatchTickPayload['active_third'];
  home_score: number;
  away_score: number;
  home_shots: number;
  away_shots: number;
  home_shots_on_target: number;
  away_shots_on_target: number;
  home_corners: number;
  away_corners: number;
  home_possession_pct: number;
  away_possession_pct: number;
  possession_momentum: number;
  ball: MatchTickPayload['ball'];
  goal_banner: string | null;
  league_fixture: MatchTickPayload['league_fixture'];
  pass_trail: MatchTickPayload['pass_trail'];
  home_coords: DeltaCoord[];
  away_coords: DeltaCoord[];
  last_event: MatchEventItem | null;
  event_count: number;
  last_commentary: CommentaryItem | null;
  commentary_count: number;
  home_tactical_stance?: MatchTickPayload['home_tactical_stance'];
  away_tactical_stance?: MatchTickPayload['away_tactical_stance'];
  latest_tactical_shift?: MatchTickPayload['latest_tactical_shift'];
  weather?: string;
}

const DELTA_SCALARS = [
  'state',
  'minute',
  'speed',
  'phase',
  'active_third',
  'home_score',
  'away_score',
  'home_shots',
  'away_shots',
  'home_shots_on_target',
  'away_shots_on_target',
  'home_corners',
  'away_corners',
  'home_possession_pct',
  'away_possession_pct',
  'possession_momentum',
  'ball',
  'goal_banner',
  'league_fixture',
  'pass_trail',
  'home_tactical_stance',
  'away_tactical_stance',
  'latest_tactical_shift',
  'weather',
] as const;

export class MatchWebSocketService {
  private ws: WebSocket | null = null;
  private onTick: ((data: MatchTickPayload) => void) | null = null;
  private reconnectTimer: number | null = null;
  private intentionalDisconnect = false;
  private queue: WsCommand[] = [];
  private listeners = new Set<(s: ConnectionStatus) => void>();
  private status: ConnectionStatus = 'DISCONNECTED';
  private attempts = 0;
  private snapshot: MatchTickPayload | null = null;
  private lastResubscribe = 0;

  addStatusListener(cb: (s: ConnectionStatus) => void): () => void {
    this.listeners.add(cb);
    cb(this.status);
    return () => {
      this.listeners.delete(cb);
    };
  }

  private setStatus(s: ConnectionStatus) {
    this.status = s;
    this.listeners.forEach((fn) => fn(s));
  }

  private wsUrl(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}/ws/match`;
  }

  connect(onTick: (data: MatchTickPayload) => void) {
    this.onTick = onTick;
    this.intentionalDisconnect = false;
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) return;

    this.setStatus(this.attempts > 0 ? 'RECONNECTING' : 'CONNECTING');
    try {
      const ws = new WebSocket(this.wsUrl());
      this.ws = ws;

      ws.onopen = () => {
        const wasReconnect = this.attempts > 0;
        this.attempts = 0;
        this.setStatus('CONNECTED');
        if (this.reconnectTimer) {
          clearTimeout(this.reconnectTimer);
          this.reconnectTimer = null;
        }
        // Replay queued commands in order, preserving flat shape.
        while (this.queue.length > 0) {
          const cmd = this.queue.shift();
          if (cmd && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(cmd));
        }
        if (wasReconnect) {
          // Identity may have changed while away: drop the stale snapshot and
          // ask for a fresh full one.
          this.snapshot = null;
          this.sendCommand('resubscribe');
        }
      };

      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data) as MatchTickPayload & Partial<DeltaTick>;
          if (msg && (msg as { tick_type?: string }).tick_type === 'delta') {
            const merged = this.applyDelta(msg as unknown as DeltaTick);
            if (merged) this.onTick?.(merged);
            return;
          }
          this.snapshot = msg;
          this.onTick?.(msg);
        } catch {
          /* ignore malformed ticks */
        }
      };

      ws.onclose = () => {
        if (this.intentionalDisconnect) {
          this.setStatus('DISCONNECTED');
          return;
        }
        this.scheduleReconnect();
      };

      ws.onerror = () => {
        try {
          ws.close();
        } catch {
          /* noop */
        }
      };
    } catch {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    this.setStatus('RECONNECTING');
    if (this.reconnectTimer) return;
    this.attempts += 1;
    const backoff = Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** Math.min(this.attempts, 4));
    const jitter = Math.random() * 300;
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      if (this.onTick && !this.intentionalDisconnect) this.connect(this.onTick);
    }, backoff + jitter);
  }

  private requestResubscribe() {
    // Throttled so a burst of unusable deltas can't spam the server.
    const now = Date.now();
    if (now - this.lastResubscribe < 2000) return;
    this.lastResubscribe = now;
    this.sendCommand('resubscribe');
  }

  /**
   * Applies a delta tick onto the last full snapshot. Returns null when there
   * is no snapshot yet (late join) — the caller waits for a full snapshot
   * instead of painting empty players.
   */
  private applyDelta(d: DeltaTick): MatchTickPayload | null {
    if (!this.snapshot) {
      this.requestResubscribe();
      return null;
    }
    const base = this.snapshot;
    const merged: MatchTickPayload = { ...base };
    for (const key of DELTA_SCALARS) {
      const value = (d as unknown as Record<string, unknown>)[key];
      if (value !== undefined) {
        (merged as unknown as Record<string, unknown>)[key] = value;
      }
    }
    merged.home_coords = this.mergeCoords(base.home_coords, d.home_coords);
    merged.away_coords = this.mergeCoords(base.away_coords, d.away_coords);
    // Append-only feeds: deltas carry the newest item plus its total count.
    if (d.last_commentary && typeof d.commentary_count === 'number') {
      const cur = base.commentary ?? [];
      if (cur.length < d.commentary_count) {
        merged.commentary = [...cur, d.last_commentary].slice(-300);
      }
    }
    if (d.last_event && typeof d.event_count === 'number') {
      const cur = base.match_events ?? [];
      if (cur.length < d.event_count) {
        merged.match_events = [...cur, d.last_event];
      }
    }
    merged.seq = d.seq;
    this.snapshot = merged;
    return merged;
  }

  private mergeCoords(
    base: MatchTickPayload['home_coords'],
    slim: DeltaCoord[] | undefined,
  ): MatchTickPayload['home_coords'] {
    if (!slim || slim.length !== base.length) {
      // Shape changed under us (e.g. red card trimmed the XI and the full
      // hasn't landed yet): keep the snapshot and heal on the next full.
      this.requestResubscribe();
      return base;
    }
    let healed = false;
    const out = base.map((actor, i) => {
      const s = slim[i];
      if (!s) return actor;
      if (s.player_id && actor.player && s.player_id !== actor.player.player_id) healed = true;
      return { ...actor, x: s.x, y: s.y, sent_off: s.sent_off };
    });
    if (healed) this.requestResubscribe();
    return out;
  }

  sendCommand(action: string, payload: Record<string, unknown> = {}) {
    const cmd: WsCommand = { action, ...payload };
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(cmd));
      return;
    }
    this.queue.push(cmd);
    // Bound the queue so a long disconnect can't grow memory unbounded.
    if (this.queue.length > 50) this.queue.shift();
    if (!this.ws || this.ws.readyState === WebSocket.CLOSED) {
      if (this.onTick) this.connect(this.onTick);
    }
  }

  disconnect() {
    this.intentionalDisconnect = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    try {
      this.ws?.close();
    } catch {
      /* noop */
    }
    this.ws = null;
    this.setStatus('DISCONNECTED');
  }
}

export const matchWs = new MatchWebSocketService();
