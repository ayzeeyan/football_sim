import React, { useEffect, useState } from 'react';
import { AlertTriangle, RefreshCw, X } from 'lucide-react';
import { cx, rgbCss, contrastText } from '../../lib/format';
import { getClubCrestUrl, getClubCrestUrlByShort } from '../../lib/clubLogos';
import { getPlayerPortraitUrl, getPlayerFallbackColors, getPlayerInitials } from '../../lib/playerPortraits';
import type { Club, Player } from '../../types';

/* ---------- Layout primitives ---------- */

export const Card: React.FC<{ className?: string; children: React.ReactNode }> = ({ className, children }) => (
  <div className={cx('panel-pad rounded-lg transition-colors duration-200', className)}>{children}</div>
);

export const PanelHeader: React.FC<{
  kicker?: string;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  right?: React.ReactNode;
}> = ({ kicker, title, subtitle, right }) => (
  <div className="panel-heading flex flex-wrap items-end justify-between gap-3 border-b border-white/[0.08] pb-3.5 mb-4">
    <div className="min-w-0">
      {kicker && <p className="mb-1.5 text-[10px] font-bold uppercase tracking-[0.18em] text-[#c5a568]">{kicker}</p>}
      <h2 className="font-display text-[clamp(1.45rem,2vw,1.75rem)] leading-[0.98] font-bold text-bone text-pretty tracking-[-0.025em]">{title}</h2>
      {subtitle && <p className="mt-1.5 max-w-2xl text-[13px] leading-relaxed text-sage text-pretty">{subtitle}</p>}
    </div>
    {right && <div className="flex items-center gap-2 shrink-0 pb-0.5">{right}</div>}
  </div>
);

/* ---------- Badges / pills ---------- */

type BadgeTone = 'cyan' | 'gold' | 'green' | 'red' | 'slate' | 'purple' | 'amber';

const badgeTones: Record<BadgeTone, string> = {
  cyan: 'bg-[#8AB4C8]/10 text-[#b6d2e0] border-[#8AB4C8]/30 border',
  gold: 'bg-[#c5a568]/10 text-[#dec58e] border-[#c5a568]/35 border',
  green: 'bg-pitchtone/10 text-[#b8d3c1] border-pitchtone/35 border',
  red: 'bg-ember/10 text-[#e8a394] border-ember/35 border',
  slate: 'bg-white/[0.035] text-[#c0c9ba] border-white/[0.12] border',
  purple: 'bg-[#8E86C8]/10 text-[#c2bce9] border-[#8E86C8]/30 border',
  amber: 'bg-[#c5a568]/10 text-[#dec58e] border-[#c5a568]/30 border',
};

export const Badge: React.FC<{ tone?: BadgeTone; className?: string; children: React.ReactNode }> = ({
  tone = 'slate',
  className,
  children,
}) => (
  <span className={cx('inline-flex items-center rounded-sm px-2 py-0.5 text-[11px] font-semibold leading-4 tracking-[0.015em] whitespace-nowrap', badgeTones[tone], className)}>
    {children}
  </span>
);

export const OvrBadge: React.FC<{ ovr: number; size?: 'sm' | 'md' | 'lg'; className?: string }> = ({
  ovr,
  size = 'md',
  className,
}) => {
  const sizeClasses = {
    sm: 'w-6 h-6 text-[11px]',
    md: 'w-8 h-8 text-[13px]',
    lg: 'w-11 h-11 text-[17px]',
  }[size];

  let toneClass = 'bg-gradient-to-b from-[#34443a] to-[#1b2a21] text-[#e5e9df] border border-white/15';
  if (ovr >= 88) {
    toneClass = 'bg-gradient-to-b from-[#d5bd82] via-[#b19251] to-[#79643b] text-[#141b14] border border-[#f1dfac]/70 shadow-md shadow-black/25';
  } else if (ovr >= 82) {
    toneClass = 'bg-gradient-to-b from-[#407258] to-[#1c422d] text-[#e0f0e4] border border-[#76a589]/45';
  } else if (ovr >= 76) {
    toneClass = 'bg-gradient-to-b from-[#355365] to-[#1b3039] text-[#e0edf0] border border-[#668a99]/45';
  }

  return (
    <div
      className={cx(
        sizeClasses,
        toneClass,
        'font-display font-bold rounded-sm flex items-center justify-center shrink-0 tracking-tight select-none shadow-[inset_0_1px_0_rgba(255,255,255,0.14)]',
        className,
      )}
      title={`Rating: ${ovr}`}
    >
      {ovr}
    </div>
  );
};

export const StatusChip: React.FC<{
  type: 'injured' | 'suspended' | 'unhappy' | 'expiring' | 'in_form' | 'prodigy' | 'fatigued';
  label: string;
  count?: number;
  onClick?: () => void;
  className?: string;
}> = ({ type, label, count, onClick, className }) => {
  const styles = {
    injured: 'bg-ember/15 text-[#FFA8A8] border-ember/40 hover:bg-ember/25',
    suspended: 'bg-amber-500/15 text-amber-300 border-amber-500/40 hover:bg-amber-500/25',
    unhappy: 'bg-purple-500/15 text-purple-300 border-purple-500/40 hover:bg-purple-500/25',
    expiring: 'bg-blue-500/15 text-blue-300 border-blue-500/40 hover:bg-blue-500/25',
    in_form: 'bg-pitchtone/20 text-[#82E4A6] border-pitchtone/50 hover:bg-pitchtone/30',
    prodigy: 'bg-brass/20 text-brass border-brass/50 hover:bg-brass/30',
    fatigued: 'bg-amber-600/15 text-amber-300 border-amber-600/40 hover:bg-amber-600/25',
  }[type];

  const content = (
    <span
      className={cx(
        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-sm text-[11px] font-semibold border transition-colors',
        styles,
        onClick && 'cursor-pointer active:scale-95',
        className,
      )}
    >
      {count != null && <span className="font-bold font-mono">[{count}]</span>}
      <span>{label}</span>
    </span>
  );

  if (onClick) {
    return (
      <button type="button" onClick={onClick} className="inline-block rounded-sm text-left focus-visible:outline-offset-2">
        {content}
      </button>
    );
  }
  return content;
};

/* ---------- Buttons ---------- */

export const PrimaryButton: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement> & { tone?: 'green' | 'gold' | 'cyan' | 'blue' | 'brass' }> = ({
  tone = 'brass',
  className,
  children,
  ...rest
}) => {
  const tones = {
    brass: 'bg-bone hover:bg-[#fff6dc] text-ink',
    gold: 'bg-bone hover:bg-[#fff6dc] text-ink',
    green: 'bg-pitchtone hover:bg-[#81b08a] text-ink',
    cyan: 'bg-[#5A8AAB] hover:bg-[#6b9bb8] text-ink',
    blue: 'bg-[#3D6B8A] hover:bg-[#4a7a9a] text-bone',
  } as const;
  return (
    <button className={cx('btn-primary', tones[tone], className)} {...rest}>
      {children}
    </button>
  );
};

export const GhostButton: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement> & { active?: boolean }> = ({
  active,
  className,
  children,
  ...rest
}) => (
  <button
    className={cx(
      'min-h-9 rounded-sm px-3.5 py-2 border text-[13px] font-semibold flex items-center gap-1.5 transition-colors disabled:cursor-not-allowed disabled:opacity-50',
      active
        ? 'bg-bone/15 border-bone/50 text-bone'
        : 'bg-cardLight hover:bg-cardHover text-bone/80 border-line',
      className,
    )}
    {...rest}
  >
    {children}
  </button>
);

/* ---------- Data display ---------- */

export const StatCard: React.FC<{ label: string; value: string; sub: string; accent?: string }> = ({
  label,
  value,
  sub,
}) => (
  <div className="panel-pad rounded-lg">
    <div className="text-[10px] font-semibold uppercase tracking-[0.14em] text-sage">{label}</div>
    <div className="mt-2 truncate font-display text-[clamp(1.65rem,3vw,2rem)] font-semibold leading-none tracking-tight text-bone" title={value}>
      {value}
    </div>
    <div className="mt-2 text-[12px] leading-relaxed text-sage">{sub}</div>
  </div>
);

export const StatMeter: React.FC<{
  label: string;
  value: number;
  max?: number;
  statusText?: string;
  toneClass?: string;
  className?: string;
}> = ({ label, value, max = 100, statusText, toneClass = 'bg-emerald-500', className }) => {
  const pct = Math.min(100, Math.max(0, (value / max) * 100));
  return (
    <div className={cx('space-y-1', className)}>
      <div className="flex items-center justify-between text-[12px]">
        <span className="text-sage font-medium">{label}</span>
        {statusText && <span className="font-semibold text-bone">{statusText}</span>}
      </div>
      <div className="w-full h-1.5 bg-[#142B20] rounded-full overflow-hidden border border-[#214332]">
        <div
          className={cx('h-full rounded-full transition-all duration-300', toneClass)}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
};

export const EmptyState: React.FC<{ message: string; className?: string }> = ({ message, className }) => (
  <div className={cx('empty-state px-6 py-14 text-center text-[13px] leading-relaxed text-sage', className)}>{message}</div>
);

export const LoadingState: React.FC<{ message?: string }> = ({ message = 'Loading…' }) => (
  <div className="flex items-center justify-center gap-3 p-8 text-center text-[13px] font-medium text-sage" role="status" aria-live="polite">
    <span className="state-spinner h-4 w-4 shrink-0 rounded-full border border-[#c5a568]/35 border-t-[#c5a568]" aria-hidden="true" />
    <span>{message}</span>
  </div>
);

export const ErrorState: React.FC<{ message?: string; onRetry?: () => void }> = ({
  message = 'This view could not be loaded.',
  onRetry,
}) => (
  <div className="error-state rounded-md border border-[#b86754]/25 bg-[#7b3025]/[0.07] p-7 text-center" role="alert">
    <AlertTriangle size={22} className="mx-auto text-[#d68f7a]" aria-hidden="true" />
    <p className="mt-3 text-[14px] font-semibold text-bone">Something interrupted the feed</p>
    <p className="mx-auto mt-1 max-w-md text-[13px] leading-relaxed text-sage">{message}</p>
    {onRetry && (
      <button type="button" onClick={onRetry} className="btn-primary mt-4 bg-cardLight text-bone hover:bg-cardHover">
        <RefreshCw size={14} aria-hidden="true" /> Retry
      </button>
    )}
  </div>
);

export const FormPips: React.FC<{ form: string[]; size?: 'sm' | 'md' }> = ({ form, size = 'md' }) => {
  const box = size === 'sm' ? 'w-5 h-5 text-[10px]' : 'w-6 h-6 text-[11px]';
  const last = form.slice(-5);
  if (last.length === 0) {
    return <span className="text-[12px] text-sage font-mono">No form yet</span>;
  }
  return (
    <span className="flex gap-1.5">
      {last.map((r, i) => (
        <span
          key={i}
          title={r === 'W' ? 'Win' : r === 'D' ? 'Draw' : 'Loss'}
          className={cx(
            box,
          'rounded-sm flex items-center justify-center font-bold font-mono shadow-sm',
            r === 'W'
              ? 'bg-[#18683E] text-[#C2F7D7] border border-[#25975A]'
              : r === 'D'
                ? 'bg-[#7A5B18] text-[#F9E8A2] border border-[#B3892B]'
                : 'bg-[#7A2424] text-[#FFA8A8] border border-[#B33B3B]',
          )}
        >
          {r}
        </span>
      ))}
    </span>
  );
};

export const ProgressBar: React.FC<{ pct: number; toneClass?: string; className?: string }> = ({
  pct,
  toneClass = 'bg-brass',
  className,
}) => (
  <div className={cx('w-full h-1.5 bg-white/[0.08] rounded-full overflow-hidden', className)}>
    <div
      className={cx('h-full rounded-full transition-[width] duration-300', toneClass)}
      style={{ width: `${Math.max(0, Math.min(100, pct))}%` }}
    />
  </div>
);

/* ---------- Club crest (real artwork, initials fallback) ---------- */

export type CrestClubInput = {
  club_id?: string;
  short_name?: string;
  club_name?: string;
  primary_color?: [number, number, number];
};

const CrestFallback: React.FC<{ club: CrestClubInput; size: number }> = ({ club, size }) => (
  <div
    className="w-full h-full flex items-center justify-center font-bold font-mono"
    style={{
      backgroundColor: club.primary_color ? rgbCss(club.primary_color) : '#1B3B2B',
      color: club.primary_color ? contrastText(club.primary_color) : '#F4ECD8',
      fontSize: Math.max(9, size * 0.28),
    }}
  >
    {(club.short_name ?? 'FC').slice(0, 2)}
  </div>
);

export const ClubCrest: React.FC<{ club: CrestClubInput | null | undefined; size?: number; className?: string }> = ({
  club,
  size = 40,
  className,
}) => {
  const [failed, setFailed] = useState(false);
  const url = club ? (club.club_id ? getClubCrestUrl({ club_id: club.club_id }) : getClubCrestUrlByShort(club.short_name)) : null;
  useEffect(() => setFailed(false), [url]);

  if (!club) return <div className={cx('rounded-lg bg-cardLight shrink-0 border border-line', className)} style={{ width: size, height: size }} />;
  return (
    <div
      className={cx('rounded-lg overflow-hidden shrink-0 border border-bone/25 bg-bone', className)}
      style={{ width: size, height: size }}
      title={club.club_name ?? club.short_name}
    >
      {url && !failed ? (
        <img
          src={url}
          alt={`${club.club_name ?? club.short_name ?? 'Club'} crest`}
          width={size}
          height={size}
          loading="lazy"
          draggable={false}
          className="w-full h-full object-contain p-[12%]"
          onError={() => setFailed(true)}
        />
      ) : (
        <CrestFallback club={club} size={size} />
      )}
    </div>
  );
};

export const ClubDot: React.FC<{ club: Club | null; size?: number; className?: string }> = ({
  club,
  size = 22,
  className,
}) => {
  const [failed, setFailed] = useState(false);
  const url = getClubCrestUrl(club);
  useEffect(() => setFailed(false), [url]);

  if (!club) return <span className={cx('rounded-full bg-cardLight border border-line shrink-0 inline-block', className)} style={{ width: size, height: size }} />;
  if (url && !failed) {
    return (
      <img
        src={url}
        alt=""
        aria-hidden
        width={size}
        height={size}
        loading="lazy"
        draggable={false}
        title={club.club_name}
        onError={() => setFailed(true)}
        className={cx('rounded-full bg-bone object-contain p-[2px] shrink-0 inline-block border border-bone/25', className)}
        style={{ width: size, height: size }}
      />
    );
  }
  return (
    <span
      className={cx('rounded-full border border-bone/30 shrink-0 inline-block', className)}
      style={{ width: size, height: size, backgroundColor: rgbCss(club.primary_color) }}
      title={club.club_name}
    />
  );
};

/* ---------- Player portrait (artwork, honest deterministic initials fallback) ---------- */

export const PlayerPortrait: React.FC<{
  player: Pick<Player, 'player_id' | 'full_name'> | null | undefined;
  size?: number;
  className?: string;
}> = ({ player, size = 40, className }) => {
  const [failed, setFailed] = useState(false);
  const url = getPlayerPortraitUrl(player?.player_id);
  useEffect(() => setFailed(false), [url]);

  if (!player) {
    return (
      <div
        className={cx('rounded-lg bg-cardLight shrink-0 border border-line', className)}
        style={{ width: size, height: size }}
      />
    );
  }

  if (url && !failed) {
    return (
      <div
        className={cx('rounded-lg overflow-hidden shrink-0 border border-bone/25 bg-bone', className)}
        style={{ width: size, height: size }}
        title={player.full_name}
      >
        <img
          src={url}
          alt={player.full_name}
          width={size}
          height={size}
          loading="lazy"
          draggable={false}
          className="w-full h-full object-cover"
          onError={() => setFailed(true)}
        />
      </div>
    );
  }

  const colors = getPlayerFallbackColors(player.player_id);
  const initials = getPlayerInitials(player.full_name);

  return (
    <div
      className={cx('rounded-lg shrink-0 flex items-center justify-center font-bold font-mono select-none border', className)}
      style={{
        width: size,
        height: size,
        backgroundColor: colors.bg,
        color: colors.text,
        borderColor: colors.border,
        fontSize: Math.max(9, size * 0.35),
      }}
      title={player.full_name}
    >
      {initials}
    </div>
  );
};

/* ---------- Controls & Nav ---------- */

export const TabPillGroup: React.FC<{
  tabs: Array<{ id: string; label: string; count?: number; icon?: React.ReactNode }>;
  activeId: string;
  onChange: (id: string) => void;
  className?: string;
}> = ({ tabs, activeId, onChange, className }) => (
  <div className={cx('section-tabs', className)} role="tablist">
    {tabs.map((tab) => {
      const active = tab.id === activeId;
      return (
        <button
          key={tab.id}
          type="button"
          onClick={() => onChange(tab.id)}
          className={cx(
            'section-tab flex items-center gap-2',
            active
              ? 'section-tab-active'
              : '',
          )}
          role="tab"
          aria-selected={active}
        >
          {tab.icon}
          <span>{tab.label}</span>
          {tab.count != null && (
            <span
              className={cx(
                'px-1.5 py-0.2 rounded text-[11px] font-mono font-bold',
                active ? 'bg-black/10 text-ink' : 'bg-white/[0.06] text-sage',
              )}
            >
              {tab.count}
            </span>
          )}
        </button>
      );
    })}
  </div>
);

/* ---------- Modal ---------- */

// Reference-counted body scroll lock so nested modals cannot restore a
// stale `overflow: hidden` and freeze the page after every dialog closes.
let bodyScrollLocks = 0;
let bodyScrollPrev = '';

function lockBodyScroll(): () => void {
  if (bodyScrollLocks === 0) {
    bodyScrollPrev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
  }
  bodyScrollLocks += 1;
  let released = false;
  return () => {
    if (released) return;
    released = true;
    bodyScrollLocks = Math.max(0, bodyScrollLocks - 1);
    if (bodyScrollLocks === 0) {
      document.body.style.overflow = bodyScrollPrev;
    }
  };
}

export const Modal: React.FC<{
  open: boolean;
  onClose: () => void;
  children: React.ReactNode;
  maxWidth?: string;
  labelledBy?: string;
  fillViewport?: boolean;
}> = ({ open, onClose, children, maxWidth = 'max-w-4xl', labelledBy = 'dialog-title', fillViewport = false }) => {
  const onCloseRef = React.useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCloseRef.current();
    };
    const release = lockBodyScroll();
    window.addEventListener('keydown', onKey);
    return () => {
      release();
      window.removeEventListener('keydown', onKey);
    };
  }, [open]);

  if (!open) return null;
  return (
    <div
      className="app-modal-backdrop fixed inset-0 z-50 flex items-center justify-center bg-ink/85 p-2 sm:p-3 animate-fade-in"
      style={{ paddingTop: 'max(0.5rem, env(safe-area-inset-top))', paddingBottom: 'max(0.5rem, env(safe-area-inset-bottom))' }}
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby={labelledBy}
    >
      <div
        className={cx(
          'app-modal min-h-0 bg-dugout border border-line w-full shadow-raised overflow-hidden flex flex-col max-h-[min(94vh,calc(100dvh-1rem))] overscroll-contain',
          fillViewport && 'h-[min(94vh,calc(100dvh-1rem))]',
          maxWidth,
        )}
      >
        {children}
      </div>
    </div>
  );
};

export const ModalHeader: React.FC<{ title: React.ReactNode; subtitle?: string; onClose: () => void }> = ({
  title, subtitle, onClose,
}) => (
  <div className="app-modal-header shrink-0 px-5 py-4 border-b border-line flex items-center justify-between bg-dugout gap-3">
    <div className="min-w-0">
      <h2 id="dialog-title" className="font-display text-lg font-semibold text-bone flex items-center gap-2 text-pretty">
        {title}
      </h2>
      {subtitle && <p className="text-[13px] text-sage font-normal mt-0.5 text-pretty">{subtitle}</p>}
    </div>
    <button
      onClick={onClose}
      aria-label="Close dialog"
      className="rounded-sm border border-white/10 bg-white/[0.035] p-2 text-sage transition-colors hover:border-white/20 hover:bg-white/[0.07] hover:text-bone shrink-0"
    >
      <X size={18} aria-hidden="true" />
    </button>
  </div>
);

export const ConfirmBar: React.FC<{
  message: string;
  confirmLabel: string;
  busyLabel?: string;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}> = ({ message, confirmLabel, busyLabel = 'Working…', busy, onConfirm, onCancel }) => (
  <div className="mt-3 p-3 rounded-md border border-ember/35 bg-ember/[0.07] flex flex-wrap items-center justify-between gap-3" role="alertdialog" aria-label={message}>
    <p className="text-[13px] text-bone/90 min-w-0 text-pretty">{message}</p>
    <div className="flex items-center gap-2 shrink-0">
      <GhostButton onClick={onCancel} disabled={busy}>
        Cancel
      </GhostButton>
      <PrimaryButton tone="brass" onClick={onConfirm} disabled={busy}>
        {busy ? busyLabel : confirmLabel}
      </PrimaryButton>
    </div>
  </div>
);
