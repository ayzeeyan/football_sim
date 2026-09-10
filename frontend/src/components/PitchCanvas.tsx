import React, { useEffect, useRef } from 'react';
import type { Club, MatchTickPayload } from '../types';
import { cx, rgbCss } from '../lib/format';

interface PitchCanvasProps {
  matchData: MatchTickPayload | null;
  homeClub: Club | null;
  awayClub: Club | null;
  className?: string;
}

interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  color: string;
  life: number;
  maxLife: number;
  radius: number;
}

type PlayerDot = MatchTickPayload['home_coords'][number];

const HOME_FALLBACK = '#C47A3A';
const AWAY_FALLBACK = '#3D6B8A';

const CHALK = 'rgba(213, 228, 194, 0.92)';
const BRASS = '#F3E6C4';
const STEEL = '#5A8AAB';

function drawTurf(ctx: CanvasRenderingContext2D, w: number, h: number) {
  const stripeW = 48;
  const numStripes = Math.ceil(w / stripeW) + 1;
  for (let i = 0; i < numStripes; i++) {
    ctx.fillStyle = i % 2 === 0 ? '#1A4A32' : '#16422C';
    ctx.fillRect(i * stripeW, 0, stripeW, h);
  }

  const lamp = ctx.createRadialGradient(w / 2, -h * 0.1, 20, w / 2, h * 0.2, w * 0.7);
  lamp.addColorStop(0, 'rgba(243, 230, 196, 0.12)');
  lamp.addColorStop(1, 'rgba(8, 20, 14, 0.35)');
  ctx.fillStyle = lamp;
  ctx.fillRect(0, 0, w, h);
}

function drawMarkings(ctx: CanvasRenderingContext2D, w: number, h: number) {
  const cx = w / 2;
  const cy = h / 2;

  ctx.lineWidth = 2.5;
  ctx.strokeStyle = CHALK;
  ctx.strokeRect(2, 2, w - 4, h - 4);

  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(cx, 2);
  ctx.lineTo(cx, h - 2);
  ctx.stroke();

  ctx.beginPath();
  ctx.arc(cx, cy, 60, 0, Math.PI * 2);
  ctx.stroke();
  ctx.beginPath();
  ctx.arc(cx, cy, 3.5, 0, Math.PI * 2);
  ctx.fillStyle = '#EAE4D6';
  ctx.fill();

  const boxW = 118;
  const boxH = 224;
  const sixW = 44;
  const sixH = 112;
  const boxY = (h - boxH) / 2;
  const sixY = (h - sixH) / 2;

  ctx.strokeRect(2, boxY, boxW, boxH);
  ctx.strokeRect(2, sixY, sixW, sixH);
  ctx.beginPath();
  ctx.arc(80, cy, 3, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.arc(80, cy, 48, -Math.PI / 2.6, Math.PI / 2.6);
  ctx.stroke();

  ctx.strokeRect(w - boxW - 2, boxY, boxW, boxH);
  ctx.strokeRect(w - sixW - 2, sixY, sixW, sixH);
  ctx.beginPath();
  ctx.arc(w - 80, cy, 3, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.arc(w - 80, cy, 48, Math.PI - Math.PI / 2.6, Math.PI + Math.PI / 2.6);
  ctx.stroke();

  drawGoals(ctx, w, h);
  drawCorners(ctx, w, h);
}

function drawGoals(ctx: CanvasRenderingContext2D, w: number, h: number) {
  const goalH = 84;
  const goalD = 18;
  const goalY = (h - goalH) / 2;
  const drawNet = (x: number) => {
    ctx.fillStyle = 'rgba(8, 11, 9, 0.75)';
    ctx.fillRect(x, goalY, goalD, goalH);
    ctx.lineWidth = 1;
    ctx.strokeStyle = 'rgba(234, 228, 214, 0.28)';
    for (let ny = goalY; ny <= goalY + goalH; ny += 7) {
      ctx.beginPath();
      ctx.moveTo(x, ny);
      ctx.lineTo(x + goalD, ny);
      ctx.stroke();
    }
    ctx.strokeStyle = '#EAE4D6';
    ctx.lineWidth = 2.5;
    ctx.strokeRect(x, goalY, goalD, goalH);
  };
  drawNet(0);
  drawNet(w - goalD);
}

function drawCorners(ctx: CanvasRenderingContext2D, w: number, h: number) {
  ctx.strokeStyle = CHALK;
  ctx.lineWidth = 2;
  ctx.beginPath(); ctx.arc(2, 2, 14, 0, Math.PI / 2); ctx.stroke();
  ctx.beginPath(); ctx.arc(w - 2, 2, 14, Math.PI / 2, Math.PI); ctx.stroke();
  ctx.beginPath(); ctx.arc(2, h - 2, 14, -Math.PI / 2, 0); ctx.stroke();
  ctx.beginPath(); ctx.arc(w - 2, h - 2, 14, Math.PI, Math.PI * 1.5); ctx.stroke();

  const flags: Array<[number, number]> = [[4, 4], [w - 4, 4], [4, h - 4], [w - 4, h - 4]];
  flags.forEach(([fx, fy]) => {
    ctx.strokeStyle = '#EAE4D6';
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(fx, fy);
    ctx.lineTo(fx, fy - 9);
    ctx.stroke();
    ctx.fillStyle = '#BE5A38';
    ctx.beginPath();
    ctx.moveTo(fx, fy - 9);
    ctx.lineTo(fx + (fx < 50 ? 7 : -7), fy - 6);
    ctx.lineTo(fx, fy - 3);
    ctx.fill();
  });
}

function drawThirdHighlight(ctx: CanvasRenderingContext2D, w: number, h: number, third: MatchTickPayload['active_third']) {  const thirdW = w / 3;
  ctx.fillStyle = 'rgba(199, 162, 58, 0.07)';
  if (third === 'DEFENSIVE') ctx.fillRect(2, 2, thirdW - 2, h - 4);
  else if (third === 'MIDFIELD') ctx.fillRect(thirdW, 2, thirdW, h - 4);
  else if (third === 'ATTACKING') ctx.fillRect(2 * thirdW, 2, thirdW - 2, h - 4);

  ctx.font = '600 9px "IBM Plex Mono", monospace';
  ctx.fillStyle = 'rgba(154, 167, 157, 0.85)';
  ctx.fillText('DEFENSIVE THIRD', 22, 20);
  ctx.fillText('MIDFIELD', thirdW + 24, 20);
  ctx.fillText('ATTACKING THIRD', 2 * thirdW + 24, 20);
}

// --- Weather overlays (F3, canvas only — no sim-math or WS changes) ---------
// Cheap loops: rain streaks, snow dots over a darker turf wash, wind wobble
// streaks plus a sinusoidal nudge applied to the ball at draw time.
interface RainDrop { x: number; y: number; v: number; len: number }
interface SnowFlake { x: number; y: number; v: number; r: number; drift: number }

function drawRain(ctx: CanvasRenderingContext2D, w: number, h: number, drops: RainDrop[]) {
  ctx.strokeStyle = 'rgba(174, 194, 214, 0.38)';
  ctx.lineWidth = 1.2;
  ctx.beginPath();
  for (const d of drops) {
    ctx.moveTo(d.x * w, d.y * h);
    ctx.lineTo((d.x - 0.008) * w, (d.y + d.len) * h);
  }
  ctx.stroke();
}

function drawSnow(ctx: CanvasRenderingContext2D, w: number, h: number, flakes: SnowFlake[]) {
  ctx.fillStyle = 'rgba(235, 240, 245, 0.75)';
  for (const f of flakes) {
    ctx.beginPath();
    ctx.arc(f.x * w, f.y * h, f.r, 0, Math.PI * 2);
    ctx.fill();
  }
}

function drawWindStreaks(ctx: CanvasRenderingContext2D, w: number, h: number, t: number) {
  ctx.strokeStyle = 'rgba(200, 214, 200, 0.22)';
  ctx.lineWidth = 1.5;
  ctx.beginPath();
  for (let i = 0; i < 4; i++) {
    const y = h * (0.2 + i * 0.2);
    const off = ((t * 0.12 + i * 0.27) % 1) * w;
    ctx.moveTo(off - 90, y);
    ctx.quadraticCurveTo(off - 45, y - 6, off, y);
  }
  ctx.stroke();
}

/**
 * High-performance pitch renderer.
 * The rAF loop is mounted ONCE — live tick data flows through refs so
 * 60fps WebSocket updates never tear down / recreate the canvas loop
 * (the legacy version re-subscribed on every tick).
 */
export const PitchCanvas: React.FC<PitchCanvasProps> = ({ matchData, homeClub, awayClub, className }) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const matchRef = useRef(matchData);
  const homeRef = useRef(homeClub);
  const awayRef = useRef(awayClub);
  const particles = useRef<Particle[]>([]);
  const lastBanner = useRef<string | null>(null);
  const rain = useRef<RainDrop[]>([]);
  const snow = useRef<SnowFlake[]>([]);
  const windT = useRef(0);

  // Keep refs fresh without restarting the loop.
  matchRef.current = matchData;
  homeRef.current = homeClub;
  awayRef.current = awayClub;

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let raf = 0;

    const drawPlayer = (
      w: number,
      h: number,
      px: number,
      py: number,
      player: PlayerDot['player'],
      pCol: string,
      sCol: string,
      ballX: number,
      ballY: number,
      sentOff?: boolean,
    ) => {
      const sx = px * w;
      const sy = py * h;
      const isCarrier = !sentOff && Math.abs(px - ballX) < 0.04 && Math.abs(py - ballY) < 0.06;
      ctx.globalAlpha = sentOff ? 0.28 : 1;

      ctx.fillStyle = 'rgba(5, 8, 6, 0.65)';
      ctx.beginPath();
      ctx.ellipse(sx, sy + 3.5, 12, 5.5, 0, 0, Math.PI * 2);
      ctx.fill();

      if (isCarrier) {
        const pulse = 17 + Math.sin(Date.now() * 0.008) * 3;
        ctx.strokeStyle = BRASS;
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.arc(sx, sy, pulse, 0, Math.PI * 2);
        ctx.stroke();
      }

      if (player.universe_wonderkid) {
        const pulse = 19 + Math.sin(Date.now() * 0.005) * 2.5;
        const rot = Date.now() * 0.002;
        ctx.fillStyle = 'rgba(243, 230, 196, 0.22)';
        ctx.beginPath();
        ctx.arc(sx, sy, pulse + 2, 0, Math.PI * 2);
        ctx.fill();
        ctx.strokeStyle = BRASS;
        ctx.lineWidth = 1.8;
        ctx.beginPath();
        ctx.arc(sx, sy, pulse, rot, rot + Math.PI * 1.5);
        ctx.stroke();

        ctx.fillStyle = BRASS;
        ctx.beginPath();
        ctx.arc(sx + 10, sy - 10, 3, 0, Math.PI * 2);
        ctx.fill();
      }

      // Category ring: read the line at a glance (keeper brass, defence
      // steel, midfield sage, attack bone).
      const lineCol =
        player.category === 'GK' ? BRASS
        : player.category === 'DEF' ? STEEL
        : player.category === 'MID' ? '#9AA79D'
        : '#EAE4D6';
      ctx.strokeStyle = lineCol;
      ctx.lineWidth = 2.6;
      ctx.beginPath();
      ctx.arc(sx, sy, 14.5, 0, Math.PI * 2);
      ctx.stroke();

      ctx.fillStyle = pCol;
      ctx.beginPath();
      ctx.arc(sx, sy, 13, 0, Math.PI * 2);
      ctx.fill();
      ctx.strokeStyle = sCol;
      ctx.lineWidth = 2;
      ctx.stroke();

      // Full position tag, large enough to read mid-flow
      ctx.font = '700 10.5px "IBM Plex Mono", monospace';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillStyle = player.universe_wonderkid ? BRASS : '#EAE4D6';
      ctx.fillText(player.position.slice(0, 3), sx, sy + 0.5);
      if (sentOff) {
        ctx.strokeStyle = '#BE5A38';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(sx - 8, sy - 8);
        ctx.lineTo(sx + 8, sy + 8);
        ctx.stroke();
      }
      ctx.globalAlpha = 1;
    };

    const render = () => {
      const dpr = window.devicePixelRatio || 1;
      const rect = canvas.getBoundingClientRect();
      if (rect.width === 0 || rect.height === 0) {
        raf = requestAnimationFrame(render);
        return;
      }
      if (canvas.width !== Math.round(rect.width * dpr) || canvas.height !== Math.round(rect.height * dpr)) {
        canvas.width = Math.round(rect.width * dpr);
        canvas.height = Math.round(rect.height * dpr);
      }

      const match = matchRef.current;
      const home = homeRef.current;
      const away = awayRef.current;

      ctx.save();
      ctx.scale(dpr, dpr);
      const w = rect.width;
      const h = rect.height;

      drawTurf(ctx, w, h);
      drawMarkings(ctx, w, h);

      const weather = (matchRef.current?.weather ?? '').toLowerCase();
      const isRain = weather === 'rain';
      const isSnow = weather === 'snow';
      const isWind = weather === 'wind';

      if (isSnow) {
        // Darker snow turf wash under the markings.
        ctx.fillStyle = 'rgba(24, 34, 52, 0.38)';
        ctx.fillRect(0, 0, w, h);
        if (snow.current.length === 0) {
          for (let i = 0; i < 55; i++) {
            snow.current.push({ x: Math.random(), y: Math.random(), v: 0.05 + Math.random() * 0.09, r: 1 + Math.random() * 1.8, drift: Math.random() * Math.PI * 2 });
          }
        }
      } else {
        snow.current = [];
      }
      if (isRain && rain.current.length === 0) {
        for (let i = 0; i < 70; i++) {
          rain.current.push({ x: Math.random(), y: Math.random(), v: 0.55 + Math.random() * 0.5, len: 0.02 + Math.random() * 0.02 });
        }
      } else if (!isRain) {
        rain.current = [];
      }
      if (isWind || isSnow) windT.current += 0.016;
      const windWobble = isWind ? Math.sin(windT.current * 6) * 4 : 0;

      if (match) {
        drawThirdHighlight(ctx, w, h, match.active_third);

        if (match.pass_trail) {
          const trail = match.pass_trail;
          const fx = trail.from[0] * w;
          const fy = trail.from[1] * h;
          const tx = trail.to[0] * w;
          const ty = trail.to[1] * h;
          ctx.strokeStyle = trail.is_shot ? BRASS : STEEL;
          ctx.lineWidth = trail.is_shot ? 3.5 : 2.2;
          ctx.beginPath();
          ctx.moveTo(fx, fy);
          ctx.lineTo(tx, ty);
          ctx.stroke();
          ctx.fillStyle = trail.is_shot ? BRASS : STEEL;
          ctx.beginPath();
          ctx.arc(tx, ty, trail.is_shot ? 4 : 3, 0, Math.PI * 2);
          ctx.fill();
        }

        const homeCol = home ? rgbCss(home.primary_color, HOME_FALLBACK) : HOME_FALLBACK;
        const homeSec = home ? rgbCss(home.secondary_color, '#EAE4D6') : '#EAE4D6';
        const awayCol = away ? rgbCss(away.primary_color, AWAY_FALLBACK) : AWAY_FALLBACK;
        const awaySec = away ? rgbCss(away.secondary_color, '#EAE4D6') : '#EAE4D6';

        match.home_coords.forEach((item) => drawPlayer(w, h, item.x, item.y, item.player, homeCol, homeSec, match.ball.x, match.ball.y, item.sent_off));
        match.away_coords.forEach((item) => drawPlayer(w, h, item.x, item.y, item.player, awayCol, awaySec, match.ball.x, match.ball.y, item.sent_off));

        const bx = match.ball.x * w + windWobble;
        const by = match.ball.y * h;
        const bh = match.ball.height || 0;
        const shadowW = Math.max(4, 10 - bh * 0.18);
        const shadowH = Math.max(2, 5.5 - bh * 0.09);
        const shadowAlpha = Math.max(0.12, 0.65 - bh * 0.02);
        ctx.fillStyle = `rgba(5, 8, 6, ${shadowAlpha})`;
        ctx.beginPath();
        ctx.ellipse(bx, by + 2.5, shadowW, shadowH, 0, 0, Math.PI * 2);
        ctx.fill();

        const ballElevY = by - bh;
        const ballRadius = Math.max(6, 7.5 + bh * 0.08);
        ctx.fillStyle = '#F3E6C4';
        ctx.beginPath();
        ctx.arc(bx, ballElevY, ballRadius, 0, Math.PI * 2);
        ctx.fill();
        ctx.strokeStyle = '#263026';
        ctx.lineWidth = 1;
        ctx.stroke();
        ctx.fillStyle = '#263026';
        ctx.beginPath();
        ctx.arc(bx, ballElevY, Math.max(1, ballRadius - 4), 0, Math.PI * 2);
        ctx.fill();
        ctx.fillStyle = '#F5F1E6';
        ctx.beginPath();
        ctx.arc(bx - 2, ballElevY - 2, 1.2, 0, Math.PI * 2);
        ctx.fill();

        // Weather overlays paint above play, below banners/legend.
        if (isRain) {
          for (const d of rain.current) {
            d.y += d.v * 0.016;
            if (d.y > 1.02) { d.y = -0.02; d.x = Math.random(); }
          }
          drawRain(ctx, w, h, rain.current);
        }
        if (isSnow) {
          for (const f of snow.current) {
            f.y += f.v * 0.016;
            f.x += Math.sin(windT.current * 2 + f.drift) * 0.0004;
            if (f.y > 1.02) { f.y = -0.02; f.x = Math.random(); }
          }
          drawSnow(ctx, w, h, snow.current);
        }
        if (isWind) drawWindStreaks(ctx, w, h, windT.current);

        if (match.goal_banner && match.goal_banner !== lastBanner.current) {
          lastBanner.current = match.goal_banner;
          const targetX = match.ball.x > 0.5 ? w - 15 : 15;
          const targetY = match.ball.y * h;
          for (let i = 0; i < 50; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = 50 + Math.random() * 200;
            particles.current.push({
              x: targetX,
              y: targetY,
              vx: Math.cos(angle) * speed,
              vy: Math.sin(angle) * speed - 40,
              color: Math.random() > 0.5 ? BRASS : STEEL,
              life: 1.0,
              maxLife: 1.0,
              radius: 2.5 + Math.random() * 2.5,
            });
          }
        }
        if (!match.goal_banner) lastBanner.current = null;
      }

      const next: Particle[] = [];
      particles.current.forEach((p) => {
        p.x += p.vx * 0.016;
        p.y += p.vy * 0.016;
        p.vy += 90 * 0.016;
        p.life -= 0.016;
        if (p.life > 0) {
          ctx.fillStyle = p.color;
          ctx.globalAlpha = p.life / p.maxLife;
          ctx.beginPath();
          ctx.arc(p.x, p.y, p.radius, 0, Math.PI * 2);
          ctx.fill();
          ctx.globalAlpha = 1.0;
          next.push(p);
        }
      });
      particles.current = next;

      const legH = 28;
      const legY = h - legH;
      ctx.fillStyle = 'rgba(18, 38, 28, 0.94)';
      ctx.fillRect(0, legY, w, legH);
      ctx.strokeStyle = 'rgba(213, 228, 194, 0.2)';
      ctx.lineWidth = 1;
      ctx.strokeRect(0, legY, w, legH);
      ctx.font = '600 10px "Inter", sans-serif';
      ctx.textAlign = 'left';
      ctx.textBaseline = 'middle';

      const homeLabel = home?.short_name ?? 'Home';
      const awayLabel = away?.short_name ?? 'Away';
      let lx = 20;
      ctx.fillStyle = home ? rgbCss(home.primary_color, HOME_FALLBACK) : HOME_FALLBACK;
      ctx.beginPath(); ctx.arc(lx, legY + 14, 5, 0, Math.PI * 2); ctx.fill();
      ctx.fillStyle = '#F3E6C4';
      ctx.fillText(`${homeLabel} XI`, lx + 10, legY + 14);
      lx += 85;
      ctx.fillStyle = away ? rgbCss(away.primary_color, AWAY_FALLBACK) : AWAY_FALLBACK;
      ctx.beginPath(); ctx.arc(lx, legY + 14, 5, 0, Math.PI * 2); ctx.fill();
      ctx.fillStyle = '#F3E6C4';
      ctx.fillText(`${awayLabel} XI`, lx + 10, legY + 14);
      lx += 85;
      ctx.fillStyle = '#9AA79D';
      ctx.fillText('Keeper lamp, defence steel, midfield sage, attack chalk', lx, legY + 14);

      ctx.restore();
      raf = requestAnimationFrame(render);
    };

    raf = requestAnimationFrame(render);
    return () => cancelAnimationFrame(raf);
  }, []);

  return (
    <div className={cx('relative w-full h-[420px] sm:h-[520px] lg:h-[580px] overflow-hidden bg-ink', className)}>
      <canvas
        ref={canvasRef}
        className="w-full h-full block"
        role="img"
        aria-label={`Live tactical radar. ${homeClub?.club_name ?? 'Home'} against ${awayClub?.club_name ?? 'Away'}.`}
      />
      {matchData?.latest_tactical_shift && (
        <div className="absolute top-4 left-1/2 -translate-x-1/2 py-2 px-5 bg-card/95 backdrop-blur-md border border-brass/50 rounded-full shadow-raised text-center flex items-center gap-3 animate-fade-in z-20 pointer-events-none max-w-[90%]">
          <span className="w-2.5 h-2.5 rounded-full bg-brass animate-ping shrink-0" />
          <span className="font-mono text-[11px] font-bold text-brass uppercase tracking-wider shrink-0">
            {matchData.latest_tactical_shift.label}
          </span>
          <span className="text-[12px] text-bone truncate">
            {matchData.latest_tactical_shift.text}
          </span>
        </div>
      )}
      {matchData?.goal_banner && (
        <div className="absolute inset-x-10 top-1/2 -translate-y-1/2 py-5 px-8 bg-ink/95 border border-brass/60 rounded-2xl shadow-raised text-center animate-fade-in">
          <div className="font-display text-brass font-semibold text-[34px] tracking-wide">
            Goal
          </div>
          <div className="text-bone font-medium text-sm mt-1">{matchData.goal_banner}</div>
        </div>
      )}
    </div>
  );
};
