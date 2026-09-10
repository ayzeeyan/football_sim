import React, { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { cx, rgbCss, contrastText } from '../../lib/format';
import { getClubCrestUrl } from '../../lib/clubLogos';
import type { Club } from '../../types';

/* ---------- Layout primitives ---------- */

export const Card: React.FC<{ className?: string; children: React.ReactNode }> = ({ className, children }) => (
  <div className={cx('panel-pad', className)}>{children}</div>
);

export const PanelHeader: React.FC<{
  kicker?: string;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  right?: React.ReactNode;
}> = ({ kicker, title, subtitle, right }) => (
  <div className="flex flex-wrap items-start justify-between gap-3">
    <div className="min-w-0">
      {kicker && <p className="text-[13px] text-sage mb-1">{kicker}</p>}
      <h2 className="font-display text-[28px] leading-none font-semibold text-bone text-pretty tracking-tight">{title}</h2>
      {subtitle && <p className="text-[14px] text-sage font-normal mt-1.5 max-w-prose text-pretty leading-relaxed">{subtitle}</p>}
    </div>
    {right && <div className="flex items-center gap-2 shrink-0">{right}</div>}
  </div>
);

/* ---------- Badges / pills ---------- */

type BadgeTone = 'cyan' | 'gold' | 'green' | 'red' | 'slate' | 'purple' | 'amber';

const badgeTones: Record<BadgeTone, string> = {
  cyan: 'bg-[#8AB4C8]/10 text-[#A9CBDD] border-[#8AB4C8]/30 border',
  gold: 'bg-brass/10 text-brass border-brass/40 border',
  green: 'bg-pitchtone/10 text-[#A9CDBB] border-pitchtone/30 border',
  red: 'bg-ember/10 text-[#D89A84] border-ember/40 border',
  slate: 'bg-cardLight text-sage border-line border',
  purple: 'bg-[#8E86C8]/10 text-[#B9B3E6] border-[#8E86C8]/30 border',
  amber: 'bg-brass/10 text-brass border-brass/30 border',
};

export const Badge: React.FC<{ tone?: BadgeTone; className?: string; children: React.ReactNode }> = ({
  tone = 'slate',
  className,
  children,
}) => (
  <span className={cx('px-2 py-0.5 text-[12px] font-medium whitespace-nowrap', badgeTones[tone], className)}>
    {children}
  </span>
);

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
      'px-3.5 py-2 border text-[13px] font-semibold flex items-center gap-1.5 transition-colors',
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
  <div className="panel-pad">
    <div className="text-[13px] text-sage">{label}</div>
    <div className="font-display text-[28px] font-semibold text-bone mt-1 truncate tracking-tight" title={value}>
      {value}
    </div>
    <div className="text-[13px] mt-1 text-sage">{sub}</div>
  </div>
);

export const EmptyState: React.FC<{ message: string; className?: string }> = ({ message, className }) => (
  <div className={cx('text-center text-sage text-[13px] py-16 px-6', className)}>{message}</div>
);

export const LoadingState: React.FC<{ message?: string }> = ({ message = 'Loading…' }) => (
  <div className="p-8 text-center text-sage text-sm animate-pulse">{message}</div>
);

export const FormPips: React.FC<{ form: string[]; size?: 'sm' | 'md' }> = ({ form, size = 'md' }) => {
  const box = size === 'sm' ? 'w-5 h-5 text-[10px]' : 'w-6 h-6 text-[11px]';
  const last = form.slice(-5);
  if (last.length === 0) {
    return <span className="text-[12px] text-sage font-mono">No form yet</span>;
  }
  return (
    <span className="flex gap-1">
      {last.map((r, i) => (
        <span
          key={i}
          title={r === 'W' ? 'Win' : r === 'D' ? 'Draw' : 'Loss'}
          className={cx(
            box,
            'rounded-md flex items-center justify-center font-bold font-mono border',
            r === 'W'
              ? 'bg-pitchtone/15 text-[#A9CDBB] border-pitchtone/30'
              : r === 'D'
                ? 'bg-brass/15 text-brass border-brass/30'
                : 'bg-ember/15 text-[#D89A84] border-ember/30',
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
  <div className={cx('w-full h-1.5 bg-line/60 rounded-full overflow-hidden', className)}>
    <div
      className={cx('h-full rounded-full transition-[width] duration-300', toneClass)}
      style={{ width: `${Math.max(0, Math.min(100, pct))}%` }}
    />
  </div>
);

/* ---------- Club crest (real artwork, initials fallback) ---------- */

const CrestFallback: React.FC<{ club: Club; size: number }> = ({ club, size }) => (
  <div
    className="w-full h-full flex items-center justify-center font-bold font-mono"
    style={{
      backgroundColor: rgbCss(club.primary_color),
      color: contrastText(club.primary_color),
      fontSize: Math.max(9, size * 0.28),
    }}
  >
    {club.short_name.slice(0, 2)}
  </div>
);

export const ClubCrest: React.FC<{ club: Club | null; size?: number; className?: string }> = ({
  club,
  size = 40,
  className,
}) => {
  const [failed, setFailed] = useState(false);
  const url = getClubCrestUrl(club);
  useEffect(() => setFailed(false), [url]);

  if (!club) return <div className={cx('rounded-lg bg-cardLight shrink-0 border border-line', className)} style={{ width: size, height: size }} />;
  return (
    <div
      className={cx('rounded-lg overflow-hidden shrink-0 border border-bone/25 bg-bone', className)}
      style={{ width: size, height: size }}
      title={club.club_name}
    >
      {url && !failed ? (
        <img
          src={url}
          alt={`${club.club_name} crest`}
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

/* ---------- Modal ---------- */

export const Modal: React.FC<{
  open: boolean;
  onClose: () => void;
  children: React.ReactNode;
  maxWidth?: string;
  labelledBy?: string;
}> = ({ open, onClose, children, maxWidth = 'max-w-4xl', labelledBy = 'dialog-title' }) => {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = prev;
      window.removeEventListener('keydown', onKey);
    };
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-ink/85 p-4 sm:p-6 animate-fade-in"
      style={{ paddingTop: 'max(1rem, env(safe-area-inset-top))', paddingBottom: 'max(1rem, env(safe-area-inset-bottom))' }}
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby={labelledBy}
    >
      <div
        className={cx(
          'bg-dugout border border-line w-full shadow-raised overflow-hidden flex flex-col max-h-[min(85vh,calc(100dvh-2rem))] overscroll-contain',
          maxWidth,
        )}
      >
        {children}
      </div>
    </div>
  );
};

export const ModalHeader: React.FC<{ title: React.ReactNode; subtitle?: string; onClose: () => void }> = ({
  title,
  subtitle,
  onClose,
}) => (
  <div className="px-5 py-4 border-b border-line flex items-center justify-between bg-dugout gap-3">
    <div className="min-w-0">
      <h2 id="dialog-title" className="font-display text-lg font-semibold text-bone flex items-center gap-2 text-pretty">
        {title}
      </h2>
      {subtitle && <p className="text-[13px] text-sage font-normal mt-0.5 text-pretty">{subtitle}</p>}
    </div>
    <button
      onClick={onClose}
      aria-label="Close dialog"
      className="p-2 rounded-lg bg-cardLight hover:bg-cardHover text-sage hover:text-bone transition-colors border border-line shrink-0"
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
  <div className="mt-3 p-3 rounded-xl border border-ember/40 bg-ember/[0.08] flex flex-wrap items-center justify-between gap-3" role="alertdialog" aria-label={message}>
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
