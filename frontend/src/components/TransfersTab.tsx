import React, { useState } from 'react';
import type { TransferNegotiation } from '../types';
import { fetchTransfers, advanceMarket, resetSeason } from '../services/api';
import { Zap, Flame, CheckCircle, Newspaper, Handshake, ListChecks, Heart, Hourglass, FlagOff, Landmark } from 'lucide-react';
import { soundManager } from '../audio/webAudio';
import { useAsyncData } from '../hooks/useAsyncData';
import { formatMillions, cx, stripEmojis } from '../lib/format';
import { getClubCrestUrlByShort } from '../lib/clubLogos';
import { Badge, Card, ClubDot, ConfirmBar, EmptyState, LoadingState, PanelHeader, PrimaryButton, ProgressBar } from './ui/ui';

interface TransfersTabProps {
  onShowToast: (msg: string) => void;
}

const STAGE_TONES = ['', 'bg-[#5B7FA6]', 'bg-[#C98A4B]', 'bg-[#8E86C8]', 'bg-[#8AB4C8]', 'bg-pitchtone'];

const NegotiationCard: React.FC<{ neg: TransferNegotiation }> = ({ neg }) => (
  <div
    className={cx(
      'p-4 rounded-xl border transition-colors',
      neg.is_hijacked ? 'border-ember/50 bg-ember/[0.06]' : neg.is_wonderkid ? 'border-brass/45 bg-brass/[0.06]' : 'border-line bg-ink/40',
    )}
  >
    <div className="flex items-center justify-between gap-2">
      <div className="font-semibold text-[14px] text-bone min-w-0">
        <span className="truncate">{neg.player.full_name}</span>
        <span className="text-[12px] text-sage font-mono ml-2 shrink-0">[{neg.player.ovr} · {neg.player.position}]</span>
      </div>
      <div className="text-[#A9CDBB] font-bold font-mono text-[14px] shrink-0">Bid {neg.formatted_bid}</div>
    </div>
    <div className="text-[13px] text-sage mt-1.5 flex items-center gap-2 flex-wrap">
      <span className="inline-flex items-center gap-1.5"><ClubDot club={neg.seller} size={24} />{neg.seller.club_name}</span>
      <span aria-hidden>→</span>
      <span className="text-bone/85 font-semibold inline-flex items-center gap-1.5"><ClubDot club={neg.buyer} size={24} />{neg.buyer.club_name}</span>
      {neg.is_hijacked && (
        <span className="text-[#D89A84] font-semibold text-[11px] flex items-center gap-1 ml-auto uppercase tracking-[0.06em]">
          <Flame size={12} /> Rival bid from {neg.original_buyer?.short_name}
        </span>
      )}
    </div>
    <div className="mt-3 space-y-1.5">
      <div className="flex justify-between font-mono text-[11px] font-semibold">
        <span className="text-sage uppercase tracking-[0.1em]">Stage {neg.stage_index}/5 · {neg.stage_name}</span>
        <span className="text-bone/80">{neg.progress_pct}%</span>
      </div>
      <ProgressBar pct={neg.progress_pct} toneClass={STAGE_TONES[neg.stage_index] || 'bg-brass'} />
    </div>
  </div>
);

export const TransfersTab: React.FC<TransfersTabProps> = ({ onShowToast }) => {
  const { data, loading, reload } = useAsyncData(fetchTransfers);
  const [advancing, setAdvancing] = useState(false);
  const [endingWindow, setEndingWindow] = useState(false);
  const [confirmEnd, setConfirmEnd] = useState(false);

  const handleAdvance = async () => {
    soundManager.playClick();
    setAdvancing(true);
    try {
      const res = await advanceMarket();
      if (!res.is_window_open && res.window_name.includes('opens')) {
        onShowToast(res.window_name);
      } else {
        const week = res.window_week ?? res.window_day;
        onShowToast(`Transfer week ${week} of ${res.max_window_weeks ?? 12}. Negotiations and wire updated.`);
      }
      reload();
    } catch {
      onShowToast('Could not advance the market. Try again.');
    } finally {
      setAdvancing(false);
    }
  };

  const handleEndWindow = async () => {
    soundManager.playWhistle();
    setEndingWindow(true);
    try {
      const res = await resetSeason();
      onShowToast(stripEmojis(res.message));
      reload();
    } catch {
      onShowToast('Could not start the new season. Try again.');
    } finally {
      setEndingWindow(false);
    }
  };

  if (loading || !data) return <Card><LoadingState message="Loading the transfer market…" /></Card>;

  const totalSpend = data.completed_transfers.reduce((acc, c) => acc + c.fee_eur, 0);
  const windowOpen = data.is_window_open;
  const windowWeek = data.window_week ?? data.window_day;

  return (
    <div className="space-y-4">
      <Card>
        <PanelHeader
          kicker={
            windowOpen
              ? `Window open · week ${windowWeek} of ${data.max_window_weeks || 12}`
              : 'Window closed'
          }
          title={`Transfers · ${data.window_name}`}
          subtitle={
            windowOpen
              ? `${data.active_negotiations.length} active negotiations · ${data.completed_transfers.length} completed · ${formatMillions(totalSpend)} spent. Permanent deals update club-owned warchests immediately.`
              : 'Business is done behind closed doors until the season ends. Completed permanent transfers remain part of the live world state.'
          }
          right={
            windowOpen ? (
              <div className="flex items-center gap-2">
                <PrimaryButton tone="cyan" onClick={handleAdvance} disabled={advancing}>
                  <Zap size={15} aria-hidden="true" /> {advancing ? 'Advancing…' : 'Advance One Week'}
                </PrimaryButton>
                <PrimaryButton tone="brass" onClick={() => setConfirmEnd(true)} disabled={endingWindow}>
                  <FlagOff size={15} aria-hidden="true" /> {endingWindow ? 'Starting…' : 'End Window'}
                </PrimaryButton>
              </div>
            ) : (
              <PrimaryButton tone="cyan" onClick={handleAdvance} disabled>
                <Zap size={15} /> Window closed
              </PrimaryButton>
            )
          }
        />
        {confirmEnd && (
          <ConfirmBar
            message="End the completed 12-week window and start the next season? Current squads and all permanent transfers will be preserved."
            confirmLabel="Start New Season"
            busyLabel="Starting…"
            busy={endingWindow}
            onConfirm={() => {
              setConfirmEnd(false);
              void handleEndWindow();
            }}
            onCancel={() => setConfirmEnd(false)}
          />
        )}
      </Card>

      {!windowOpen && data.expiring_contracts.length > 0 && (
        <Card>
          <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
            <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
              <Hourglass size={16} className="text-brass" /> Deals running down
            </h3>
            <Badge tone="gold">{data.expiring_contracts.length} in their final year</Badge>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            {data.expiring_contracts.map((p) => (
              <div key={p.player_id} className="p-3 bg-ink/40 rounded-xl border border-line flex items-center justify-between gap-2">
                <div className="min-w-0">
                  <div className="font-semibold text-bone text-[13px] truncate">
                    {p.full_name} <span className="text-sage font-mono font-normal">({p.position}, {p.ovr})</span>
                  </div>
                  <div className="text-[12px] text-sage">{p.club_name} · {p.formatted_wage}</div>
                </div>
                <span
                  className="inline-flex items-center gap-1 font-mono text-[11px] text-sage shrink-0"
                  title={`Loyalty ${p.loyalty}/100`}
                >
                  <Heart size={12} className={p.loyalty >= 70 ? 'text-brass' : 'text-ember'} fill="currentColor" />
                  {p.loyalty}
                </span>
              </div>
            ))}
          </div>
        </Card>
      )}

      {windowOpen && data.warchests.length > 0 && (
        <Card>
          <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
            <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
              <Landmark size={16} className="text-brass" /> Club transfer warchests
            </h3>
            <Badge tone="gold">Week {windowWeek}/{data.max_window_weeks ?? 12}</Badge>
          </div>
          <p className="text-[12px] text-sage mb-3">Current club-owned spending limits. Purchases reduce them and sales replenish them according to the backend finance rules.</p>
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-2">
            {data.warchests.map((w) => (
              <div key={w.club_short} className="p-2.5 bg-ink/40 rounded-xl border border-line" title={`${w.club_name} · managed by ${w.manager_name}`}>
                <p className="font-mono font-bold text-[12px] text-bone">{w.club_short}</p>
                <p className="font-mono font-bold text-[13px] text-brass mt-0.5">{w.formatted_budget}</p>
                <p className="text-[10.5px] text-sage truncate mt-0.5">{w.manager_name}</p>
              </div>
            ))}
          </div>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        <div className="lg:col-span-7 space-y-4">
          <Card>
            <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
              <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                <Handshake size={16} className="text-sage" /> Negotiations
              </h3>
              <Badge tone="slate">{data.active_negotiations.length} in progress</Badge>
            </div>
            <div className="space-y-3 max-h-[460px] overflow-y-auto pr-1">
              {data.active_negotiations.length === 0 && (
                <EmptyState message={windowOpen ? 'No active negotiations. Advance a week and the phones will ring.' : 'No active negotiations while the window is shut.'} />
              )}
              {data.active_negotiations.map((neg) => (
                <NegotiationCard key={neg.negotiation_id} neg={neg} />
              ))}
            </div>
          </Card>

          <Card>
            <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
              <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                <ListChecks size={16} className="text-sage" /> Completed deals
              </h3>
              <Badge tone="slate">{data.completed_transfers.length} signings</Badge>
            </div>
            <div className="space-y-2 max-h-[220px] overflow-y-auto pr-1">
              {data.completed_transfers.length === 0 && <EmptyState message="No completed permanent transfers yet this window." className="py-6" />}
              {data.completed_transfers.map((t, idx) => (
                <div key={`${t.player_name}-${idx}`} className="p-3 bg-ink/40 rounded-xl border border-line flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2.5 min-w-0">
                    {getClubCrestUrlByShort(t.buyer_short) ? (
                      <img src={getClubCrestUrlByShort(t.buyer_short)!} alt={`${t.buyer_name} crest`} loading="lazy" draggable={false} className="w-8 h-8 rounded-full bg-bone object-contain p-[3px] border border-bone/25 shrink-0" />
                    ) : (
                      <CheckCircle className="text-pitchtone shrink-0" size={16} />
                    )}
                    <div className="min-w-0">
                      <div className="font-semibold text-bone text-[13px] truncate">
                        {t.player_name} <span className="text-sage font-mono font-normal">({t.player_pos}, {t.player_ovr})</span>
                      </div>
                      <div className="text-[12px] text-sage">
                        {t.seller_name} <span aria-hidden>→</span> <strong className="text-bone/85">{t.buyer_name}</strong> · Week {t.matchweek}
                      </div>
                    </div>
                  </div>
                  <div className="font-mono font-semibold text-[#A9CDBB] text-[14px] shrink-0">{t.formatted_fee}</div>
                </div>
              ))}
            </div>
          </Card>
        </div>

        <Card className="lg:col-span-5 flex flex-col h-[740px]">
          <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
            <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
              <Newspaper size={16} className="text-sage" /> {windowOpen ? 'Transfer wire' : 'Rumor mill'}
            </h3>
            <Badge tone="slate">{data.transfer_feed.length} stories</Badge>
          </div>
          <div className="flex-1 overflow-y-auto space-y-2.5 pr-1">
            {data.transfer_feed.length === 0 && <EmptyState message="The wire is quiet. Check back after advancing the market." />}
            {data.transfer_feed.map((item, idx) => {
              const isHwg = item.category === 'HERE_WE_GO';
              const isHijack = item.category === 'HIJACK';
              return (
                <div
                  key={`${item.timestamp}-${idx}`}
                  className={cx(
                    'p-3.5 rounded-xl border text-[13px] leading-relaxed',
                    isHwg
                      ? 'bg-pitchtone/10 border-pitchtone/40 text-bone'
                      : isHijack
                        ? 'bg-ember/10 border-ember/45 text-bone'
                        : item.is_wonderkid
                          ? 'bg-brass/10 border-brass/40 text-bone'
                          : 'bg-ink/40 border-line text-bone/80',
                  )}
                >
                  <div className="flex items-center justify-between font-mono font-semibold text-[11px] mb-1.5 gap-2">
                    <span className="text-sage">{item.timestamp}</span>
                    <span
                      className={cx(
                        'px-1.5 py-0.5 rounded font-semibold uppercase tracking-[0.08em]',
                        isHwg
                          ? 'bg-pitchtone/15 text-[#A9CDBB] border border-pitchtone/35'
                          : isHijack
                            ? 'bg-ember/15 text-[#D89A84] border border-ember/40'
                            : 'bg-cardLight text-sage border border-line',
                      )}
                    >
                      {item.category.replace(/_/g, ' ')}
                    </span>
                  </div>
                  <div className="font-normal">{stripEmojis(item.headline)}</div>
                </div>
              );
            })}
          </div>
        </Card>
      </div>
    </div>
  );
};
