import React, { useCallback, useState } from 'react';
import type { TransferRecordsData } from '../../../types';
import { fetchTransfers, fetchTransferRecords, advanceMarket, resetSeason} from '../../../services/api';
import { Zap, CheckCircle, Newspaper, Handshake, ListChecks, Heart, Hourglass, FlagOff, Landmark, Sparkle, Trophy } from 'lucide-react';
import { soundManager } from '../../../audio/webAudio';
import { useAsyncData } from '../../../hooks/useAsyncData';
import { formatMillions, cx, stripEmojis } from '../../../lib/format';
import { getClubCrestUrlByShort } from '../../../lib/clubLogos';
import { focusTabAt, nextTabIndex } from '../../../lib/rovingTabindex';
import { Badge, Card, ConfirmBar, EmptyState, ErrorState, LoadingState, PanelHeader, PrimaryButton, ProgressBar } from '../../ui/ui';
import { usePlayerSheet } from '../../clubs/PlayerSheet';
import { canStartNextSeason, type TransfersSubTab, type TransfersTabProps } from './helpers';
import { NegotiationCard } from './NegotiationCard';


export const TransfersTab: React.FC<TransfersTabProps> = ({ onShowToast }) => {
  const { data, loading, error, reload } = useAsyncData(fetchTransfers);
  const { openPlayer } = usePlayerSheet();
  const [subTab, setSubTab] = useState<TransfersSubTab>('negotiations');
  const [advancing, setAdvancing] = useState(false);
  const [endingWindow, setEndingWindow] = useState(false);
  const [confirmEnd, setConfirmEnd] = useState(false);
  const [records, setRecords] = useState<TransferRecordsData | null>(null);
  const [recordsLoading, setRecordsLoading] = useState(false);

  const loadRecords = useCallback(() => {
    if (records) return;
    setRecordsLoading(true);
    fetchTransferRecords()
      .then((res) => setRecords({ ...res, top_signings: res.top_signings ?? [] }))
      .finally(() => setRecordsLoading(false));
  }, [records]);

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
      if (res.status !== 'success') throw new Error(res.message);
      onShowToast(stripEmojis(res.message));
      reload();
    } catch {
      onShowToast('Could not start the new season. Try again.');
    } finally {
      setEndingWindow(false);
    }
  };

  if (loading) return <Card><LoadingState message="Connecting to European transfer market network…" /></Card>;
  if (error || !data) return <Card><ErrorState message={error || 'The transfer market returned no data.'} onRetry={reload} /></Card>;

  const totalSpend = data.completed_transfers.reduce((acc, c) => acc + c.fee_eur, 0);
  const windowOpen = data.is_window_open;
  const windowWeek = data.window_week ?? data.window_day;
  const maxWeeks = data.max_window_weeks ?? 12;
  const startNextSeason = canStartNextSeason(data);
  const isDeadlineDay = !!data.deadline_day || (windowOpen && windowWeek === maxWeeks);
  const handleSubTabKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    const tablist = event.currentTarget.closest('[role="tablist"]');
    if (!tablist) return;
    const tabs = Array.from(tablist.querySelectorAll<HTMLButtonElement>('[role="tab"]'));
    const current = tabs.indexOf(event.currentTarget);
    const next = current < 0 ? null : nextTabIndex(event.key, current, tabs.length);
    if (next == null) return;
    event.preventDefault();
    focusTabAt(tablist, next);
  };

  return (
    <div className="space-y-4">
      {/* Deadline day stays prominent without competing with the rest of the market. */}
      {isDeadlineDay && (
        <div className="relative overflow-hidden border border-ember/35 bg-gradient-to-r from-[#251b16] via-[#14231b] to-[#18251d] p-4 sm:p-5 shadow-raised">
          <div className="flex flex-wrap items-center justify-between gap-3 relative z-10">
            <div className="flex items-center gap-3">
              <span className="h-2.5 w-2.5 shrink-0 rounded-full bg-ember shadow-[0_0_12px_rgba(220,38,38,0.55)]" aria-hidden="true" />
              <div>
                <span className="text-[10px] font-bold uppercase tracking-[0.18em] text-[#E6A28F]">
                  Transfer deadline day
                </span>
                <h3 className="font-display text-[18px] font-bold text-bone sm:text-[21px]">
                  Final Hours of the European Transfer Window
                </h3>
                <p className="mt-1 text-[12px] text-sage">
                  Clubs are rushing last-minute bids and hijack offers before the midnight market shutdown.
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <span className="border border-ember/35 bg-black/20 px-3 py-1.5 text-bone font-mono font-semibold text-[12px]">
                Final Week ({windowWeek}/{maxWeeks})
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Main Transfer Centre Header Card */}
      <Card>
        <PanelHeader
          kicker={
            windowOpen
              ? `Window open · Week ${windowWeek} of ${maxWeeks}`
              : 'Window closed'
          }
          title={`Transfer Centre · ${data.window_name}`}
          subtitle={
            windowOpen
              ? `${data.active_negotiations.length} active negotiations · ${data.completed_transfers.length} completed · ${formatMillions(totalSpend)} total spend across 96 clubs.`
              : startNextSeason
                ? 'The summer market has concluded. Review permanent squad moves, then start the new season when ready.'
                : 'Market closed. AI scouts and clubs prepare future shortlists until the window reopens.'
          }
          right={
            windowOpen ? (
              <PrimaryButton tone="cyan" onClick={handleAdvance} disabled={advancing}>
                <Zap size={15} aria-hidden="true" /> {advancing ? 'Advancing…' : isDeadlineDay ? 'Process Deadline Midnight' : 'Advance One Week'}
              </PrimaryButton>
            ) : startNextSeason ? (
              <PrimaryButton tone="brass" onClick={() => setConfirmEnd(true)} disabled={endingWindow}>
                <FlagOff size={15} aria-hidden="true" /> {endingWindow ? 'Starting…' : 'Start New Campaign'}
              </PrimaryButton>
            ) : (
              <PrimaryButton tone="cyan" onClick={handleAdvance} disabled>
                <Zap size={15} /> Window closed
              </PrimaryButton>
            )
          }
        />

        {confirmEnd && startNextSeason && (
          <ConfirmBar
            message={`Start the next season after the completed ${maxWeeks}-week summer window? Current squads and all permanent transfers will be preserved.`}
            confirmLabel="Start New Campaign"
            busyLabel="Starting…"
            busy={endingWindow}
            onConfirm={() => {
              setConfirmEnd(false);
              void handleEndWindow();
            }}
            onCancel={() => setConfirmEnd(false)}
          />
        )}

        {/* Market Overview 3-Card Summary Strip (Panel 4) */}
        <div className="mt-5 grid grid-cols-1 gap-3.5 md:grid-cols-3">
          {/* Card 1: Window Status */}
          <div className="console-card space-y-2 p-4">
            <div className="flex items-center justify-between text-[11px] font-mono uppercase text-sage">
              <span className="font-bold text-brass">{data.window_name}</span>
              <span>{windowOpen ? 'Market Open' : 'Market Closed'}</span>
            </div>
            <div className="flex items-baseline justify-between">
              <span className="font-display text-[22px] font-bold text-bone">
                Week {windowWeek} <span className="text-[14px] text-sage font-normal">of {maxWeeks}</span>
              </span>
              <span className="text-[11.5px] font-mono text-sage">
                {Math.round((windowWeek / maxWeeks) * 100)}%
              </span>
            </div>
            <ProgressBar pct={(windowWeek / maxWeeks) * 100} toneClass="bg-brass" />
          </div>

          {/* Card 2: Market Economics */}
          <div className="console-card space-y-2 p-4">
            <div className="flex items-center justify-between text-[11px] font-mono uppercase text-sage">
              <span className="font-bold text-brass">Market Volume</span>
              <span>96 Clubs</span>
            </div>
            <div className="flex items-baseline justify-between">
              <span className="font-display text-[22px] font-bold text-bone">
                {formatMillions(totalSpend)}
              </span>
              <span className="text-[11.5px] font-mono text-sage">
                {data.completed_transfers.length} deals
              </span>
            </div>
            <div className="flex items-center justify-between text-[11.5px] text-sage pt-1 border-t border-[#214332]/60">
              <span>Active Negotiations</span>
              <strong className="text-bone font-mono">{data.active_negotiations.length} live</strong>
            </div>
          </div>

          {/* Card 3: Latest Deals Ticker */}
          <div className="console-card space-y-1.5 p-4">
            <div className="flex items-center justify-between text-[11px] font-mono uppercase text-sage pb-1 border-b border-[#214332]">
              <span className="font-bold text-brass">Latest Completed Deals</span>
              <Sparkle size={11} className="text-brass" />
            </div>
            {data.completed_transfers.length > 0 ? (
              <div className="space-y-1 pt-0.5">
                {data.completed_transfers.slice(-2).reverse().map((c, i) => (
                  <div key={i} className="flex items-center justify-between text-[12px]">
                    <span className="truncate text-bone font-medium max-w-[140px]">{c.player_name}</span>
                    <span className="font-mono font-bold text-[#A9CDBB] shrink-0">{c.formatted_fee}</span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-[12px] text-sage/70 py-2">No completed signings yet this window.</p>
            )}
          </div>
        </div>

        {/* Transfer Centre Sub-Navigation Bar */}
        <div className="section-tabs -mx-4 sm:-mx-5 -mb-4 sm:-mb-5 mt-5 px-2" role="tablist" aria-label="Transfer centre sections">
          <button
            type="button"
            id="transfer-tab-negotiations"
            role="tab"
            aria-selected={subTab === 'negotiations'}
            aria-controls="transfer-panel-negotiations"
            tabIndex={subTab === 'negotiations' ? 0 : -1}
            onKeyDown={handleSubTabKeyDown}
            onClick={() => { soundManager.playClick(); setSubTab('negotiations'); }}
            className={cx(
              'section-tab flex items-center gap-2 whitespace-nowrap',
              subTab === 'negotiations' ? 'section-tab-active' : '',
            )}
          >
            <Handshake size={14} />
            <span>Active Deals</span>
            <span className={cx('px-1.5 py-0.2 rounded text-[11px] font-mono', subTab === 'negotiations' ? 'bg-ink/20 text-ink' : 'bg-ink/40 text-sage')}>
              {data.active_negotiations.length}
            </span>
          </button>

          <button
            type="button"
            id="transfer-tab-feed"
            role="tab"
            aria-selected={subTab === 'feed'}
            aria-controls="transfer-panel-feed"
            tabIndex={subTab === 'feed' ? 0 : -1}
            onKeyDown={handleSubTabKeyDown}
            onClick={() => { soundManager.playClick(); setSubTab('feed'); }}
            className={cx(
              'section-tab flex items-center gap-2 whitespace-nowrap',
              subTab === 'feed' ? 'section-tab-active' : '',
            )}
          >
            <Newspaper size={14} />
            <span>Market Wire</span>
            <span className={cx('px-1.5 py-0.2 rounded text-[11px] font-mono', subTab === 'feed' ? 'bg-ink/20 text-ink' : 'bg-ink/40 text-sage')}>
              {data.transfer_feed.length}
            </span>
          </button>

          <button
            type="button"
            id="transfer-tab-warchests"
            role="tab"
            aria-selected={subTab === 'warchests'}
            aria-controls="transfer-panel-warchests"
            tabIndex={subTab === 'warchests' ? 0 : -1}
            onKeyDown={handleSubTabKeyDown}
            onClick={() => { soundManager.playClick(); setSubTab('warchests'); }}
            className={cx(
              'section-tab flex items-center gap-2 whitespace-nowrap',
              subTab === 'warchests' ? 'section-tab-active' : '',
            )}
          >
            <Landmark size={14} />
            <span>Club Warchests</span>
            <span className={cx('px-1.5 py-0.2 rounded text-[11px] font-mono', subTab === 'warchests' ? 'bg-ink/20 text-ink' : 'bg-ink/40 text-sage')}>
              {data.warchests.length}
            </span>
          </button>

          <button
            type="button"
            id="transfer-tab-history"
            role="tab"
            aria-selected={subTab === 'history'}
            aria-controls="transfer-panel-history"
            tabIndex={subTab === 'history' ? 0 : -1}
            onKeyDown={handleSubTabKeyDown}
            onClick={() => { soundManager.playClick(); setSubTab('history'); }}
            className={cx(
              'section-tab flex items-center gap-2 whitespace-nowrap',
              subTab === 'history' ? 'section-tab-active' : '',
            )}
          >
            <ListChecks size={14} />
            <span>Completed & Expiring</span>
            <span className={cx('px-1.5 py-0.2 rounded text-[11px] font-mono', subTab === 'history' ? 'bg-ink/20 text-ink' : 'bg-ink/40 text-sage')}>
              {data.completed_transfers.length}
            </span>
          </button>

          <button
            type="button"
            id="transfer-tab-records"
            role="tab"
            aria-selected={subTab === 'records'}
            aria-controls="transfer-panel-records"
            tabIndex={subTab === 'records' ? 0 : -1}
            onKeyDown={handleSubTabKeyDown}
            onClick={() => { soundManager.playClick(); setSubTab('records'); loadRecords(); }}
            className={cx(
              'section-tab flex items-center gap-2 whitespace-nowrap',
              subTab === 'records' ? 'section-tab-active' : '',
            )}
          >
            <Trophy size={14} />
            <span>All-Time Records</span>
          </button>
        </div>
      </Card>

      {/* SUBTAB 1: ACTIVE NEGOTIATIONS */}
      {subTab === 'negotiations' && (
        <section id="transfer-panel-negotiations" role="tabpanel" aria-labelledby="transfer-tab-negotiations" tabIndex={0} className="space-y-4">
          <Card>
            <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
              <div>
                <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                  <Handshake size={17} className="text-brass" /> Active Market Negotiations
                </h3>
                <p className="text-[12.5px] text-sage mt-0.5">
                  5-stage negotiation tracker. Selling clubs demand market value while rival clubs attempt hijacks.
                </p>
              </div>
              <Badge tone="slate">{data.active_negotiations.length} in progress</Badge>
            </div>

            {data.active_negotiations.length === 0 ? (
              <EmptyState message={windowOpen ? 'No active negotiations. Advance one week to trigger AI transfer inquiries.' : 'No active negotiations while the window is shut.'} />
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3.5">
                {data.active_negotiations.map((neg) => (
                  <NegotiationCard key={neg.negotiation_id} neg={neg} onPlayerClick={openPlayer} />
                ))}
              </div>
            )}
          </Card>
        </section>
      )}

      {/* SUBTAB 2: MARKET WIRE & FEED */}
      {subTab === 'feed' && (
        <section id="transfer-panel-feed" role="tabpanel" aria-labelledby="transfer-tab-feed" tabIndex={0}>
        <Card>
          <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
            <div>
              <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                <Newspaper size={17} className="text-brass" /> Live Transfer Wire & Rumour Mill
              </h3>
              <p className="text-[12.5px] text-sage mt-0.5">
                Breaking journalism, verified Here-We-Go exclusives, and high-heat bidding alerts.
              </p>
            </div>
            <Badge tone="slate">{data.transfer_feed.length} stories</Badge>
          </div>

          {data.transfer_feed.length === 0 ? (
            <EmptyState message="The wire is currently quiet. Advance the market week to read breaking scoops." />
          ) : (
            <div className="space-y-2.5 max-h-[640px] overflow-y-auto pr-1">
              {data.transfer_feed.map((item, idx) => {
                const isHwg = item.category === 'HERE_WE_GO';
                const isHijack = item.category === 'HIJACK';
                return (
                  <div
                    key={`${item.timestamp}-${idx}`}
                    className={cx(
                      'p-3.5 rounded-xl border text-[13px] leading-relaxed transition-colors',
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
                              : item.is_wonderkid
                                ? 'bg-brass/15 text-brass border border-brass/40'
                                : 'bg-cardLight text-sage border border-line',
                        )}
                      >
                        {item.category.replace(/_/g, ' ')}
                      </span>
                    </div>
                    <div className="font-normal text-[13.5px]">{stripEmojis(item.headline)}</div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>
        </section>
      )}

      {/* SUBTAB 3: CLUB FINANCES & WARCHESTS */}
      {subTab === 'warchests' && (
        <section id="transfer-panel-warchests" role="tabpanel" aria-labelledby="transfer-tab-warchests" tabIndex={0}>
        <Card>
          <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
            <div>
              <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                <Landmark size={17} className="text-brass" /> Club Transfer Warchests & Spending Limits
              </h3>
              <p className="text-[12.5px] text-sage mt-0.5">
                Current club budgets governed by transfer balances, wage caps, and financial fairness rules.
              </p>
            </div>
            <Badge tone="gold">Week {windowWeek}/{maxWeeks}</Badge>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2.5">
            {data.warchests.map((w) => (
              <div key={w.club_short} className="p-3 bg-ink/40 rounded-xl border border-line flex flex-col justify-between" title={`${w.club_name} · Manager: ${w.manager_name}`}>
                <div>
                  <p className="font-mono font-bold text-[13px] text-bone">{w.club_short}</p>
                  <p className="text-[11px] text-sage truncate mt-0.5">{w.club_name}</p>
                </div>
                <div className="mt-2.5 pt-2 border-t border-line/60">
                  <span className="text-[10px] font-mono text-sage uppercase tracking-wider block">Budget</span>
                  <p className="font-mono font-bold text-[14px] text-brass mt-0.5">{w.formatted_budget}</p>
                  <p className="text-[10.5px] text-sage truncate mt-0.5">{w.manager_name}</p>
                </div>
              </div>
            ))}
          </div>
        </Card>
        </section>
      )}

      {/* SUBTAB 4: COMPLETED TRANSFERS & EXPIRING DEALS */}
      {subTab === 'history' && (
        <div id="transfer-panel-history" role="tabpanel" aria-labelledby="transfer-tab-history" tabIndex={0} className="grid grid-cols-1 items-start gap-5 lg:grid-cols-12">
          {/* Completed Signings */}
          <Card className="lg:col-span-7">
            <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
              <div>
                <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                  <ListChecks size={17} className="text-pitchtone" /> Confirmed Permanent Signings
                </h3>
                <p className="text-[12.5px] text-sage mt-0.5">
                  Executed contracts that updated club rosters and balances.
                </p>
              </div>
              <Badge tone="slate">{data.completed_transfers.length} signings</Badge>
            </div>

            {data.completed_transfers.length === 0 ? (
              <EmptyState message="No completed permanent transfers yet in this window." className="py-8" />
            ) : (
              <div className="space-y-2 max-h-[520px] overflow-y-auto pr-1">
                {data.completed_transfers.map((t, idx) => (
                  <div key={`${t.player_name}-${idx}`} className="p-3 bg-ink/40 rounded-xl border border-line flex items-center justify-between gap-2.5">
                    <div className="flex items-center gap-3 min-w-0">
                      {getClubCrestUrlByShort(t.buyer_short) ? (
                        <img src={getClubCrestUrlByShort(t.buyer_short)!} alt={`${t.buyer_name} crest`} loading="lazy" draggable={false} className="w-8 h-8 rounded-full bg-bone object-contain p-[3px] border border-bone/25 shrink-0" />
                      ) : (
                        <CheckCircle className="text-pitchtone shrink-0" size={18} />
                      )}
                      <div className="min-w-0">
                        <div className="font-semibold text-bone text-[13px] truncate">
                          {t.player_name} <span className="text-sage font-mono font-normal">({t.player_pos}, {t.player_ovr})</span>
                        </div>
                        <div className="text-[12px] text-sage flex items-center gap-1.5 mt-0.5">
                          <span>{t.seller_name}</span>
                          <span aria-hidden className="text-sage/60">→</span>
                          <strong className="text-bone/90">{t.buyer_name}</strong>
                          <span className="text-line">•</span>
                          <span className="font-mono text-[11px]">Wk {t.matchweek}</span>
                        </div>
                      </div>
                    </div>
                    <div className="font-mono font-bold text-[#A9CDBB] text-[14px] shrink-0">{t.formatted_fee}</div>
                  </div>
                ))}
              </div>
            )}
          </Card>

          {/* Expiring Contracts */}
          <Card className="lg:col-span-5">
            <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
              <div>
                <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                  <Hourglass size={17} className="text-brass" /> Expiring Deals
                </h3>
                <p className="text-[12.5px] text-sage mt-0.5">
                  Players entering the final 12 months of contract.
                </p>
              </div>
              <Badge tone="gold">{data.expiring_contracts.length} expiring</Badge>
            </div>

            {data.expiring_contracts.length === 0 ? (
              <EmptyState message="No high-profile contracts running down." className="py-8" />
            ) : (
              <div className="space-y-2 max-h-[520px] overflow-y-auto pr-1">
                {data.expiring_contracts.map((p) => (
                  <button
                    key={p.player_id}
                    type="button"
                    onClick={() => {
                      soundManager.playClick();
                      openPlayer(p.player_id);
                    }}
                    className="w-full text-left p-3 bg-ink/40 hover:bg-cardHover rounded-xl border border-line flex items-center justify-between gap-2.5 transition-colors"
                  >
                    <div className="min-w-0">
                      <div className="font-semibold text-bone text-[13px] truncate">
                        {p.full_name} <span className="text-sage font-mono font-normal">({p.position}, {p.ovr})</span>
                      </div>
                      <div className="text-[12px] text-sage mt-0.5">{p.club_name} · {p.formatted_wage}</div>
                    </div>
                    <span
                      className="inline-flex items-center gap-1 font-mono text-[11px] text-sage shrink-0"
                      title={`Loyalty ${p.loyalty}/100`}
                    >
                      <Heart size={12} className={p.loyalty >= 70 ? 'text-brass' : 'text-ember'} fill="currentColor" />
                      {p.loyalty}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </Card>
        </div>
      )}

      {/* SUBTAB 5: ALL-TIME RECORDS */}
      {subTab === 'records' && (
        <div id="transfer-panel-records" role="tabpanel" aria-labelledby="transfer-tab-records" tabIndex={0} className="grid grid-cols-1 items-start gap-5 lg:grid-cols-12">
          <Card className="lg:col-span-7">
            <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
              <div>
                <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                  <Trophy size={17} className="text-brass" /> Record Signings
                </h3>
                <p className="text-[12.5px] text-sage mt-0.5">
                  The biggest permanent fees ever paid in this career.
                </p>
              </div>
              {records && <Badge tone="gold">{records.total_transfers_count} transfers all-time</Badge>}
            </div>

            {recordsLoading ? (
              <LoadingState message="Opening the record book…" />
            ) : !records || records.top_signings.length === 0 ? (
              <EmptyState message="No permanent transfers have been completed yet. Records appear after the first window." className="py-8" />
            ) : (
              <div className="space-y-2 max-h-[520px] overflow-y-auto pr-1">
                {records.top_signings.map((t, idx) => (
                  <div key={`${t.player_id}-${idx}`} className="p-3 bg-ink/40 rounded-xl border border-line flex items-center justify-between gap-2.5">
                    <div className="flex items-center gap-3 min-w-0">
                      <span className="font-mono font-bold text-[12px] text-brass w-5 shrink-0 text-right">{idx + 1}.</span>
                      {getClubCrestUrlByShort(t.buyer_short) ? (
                        <img src={getClubCrestUrlByShort(t.buyer_short)!} alt={`${t.buyer_name} crest`} loading="lazy" draggable={false} className="w-8 h-8 rounded-full bg-bone object-contain p-[3px] border border-bone/25 shrink-0" />
                      ) : (
                        <CheckCircle className="text-pitchtone shrink-0" size={18} />
                      )}
                      <div className="min-w-0">
                        <button
                          type="button"
                          onClick={() => { soundManager.playClick(); openPlayer(t.player_id); }}
                          className="font-semibold text-bone text-[13px] truncate hover:text-brass transition-colors text-left"
                        >
                          {t.player_name}
                          <span className="text-sage font-mono font-normal"> ({t.player_pos}, {t.player_ovr})</span>
                          {t.is_wonderkid && <span className="ml-1.5 px-1 py-0.2 rounded bg-brass/20 text-brass text-[10px] font-mono font-bold">PRODIGY</span>}
                        </button>
                        <div className="text-[12px] text-sage flex items-center gap-1.5 mt-0.5">
                          <span>{t.seller_name}</span>
                          <span aria-hidden className="text-sage/60">→</span>
                          <strong className="text-bone/90">{t.buyer_name}</strong>
                        </div>
                      </div>
                    </div>
                    <div className="font-mono font-bold text-[#A9CDBB] text-[14px] shrink-0">{t.formatted_fee}</div>
                  </div>
                ))}
              </div>
            )}
          </Card>

          <Card className="lg:col-span-5">
            <div className="flex items-center justify-between pb-3 border-b border-line mb-4">
              <div>
                <h3 className="font-display text-[17px] font-semibold text-bone flex items-center gap-2">
                  <Landmark size={17} className="text-pitchtone" /> Net Spend Board
                </h3>
                <p className="text-[12.5px] text-sage mt-0.5">
                  Career balance sheet: fees paid minus fees received.
                </p>
              </div>
            </div>

            {recordsLoading ? (
              <LoadingState message="Balancing the books…" />
            ) : !records || Object.keys(records.net_spend).length === 0 ? (
              <EmptyState message="Net spend appears once the first transfer completes." className="py-8" />
            ) : (
              <div className="space-y-1.5 max-h-[520px] overflow-y-auto pr-1">
                {Object.values(records.net_spend)
                  .filter((row) => row.spent > 0 || row.received > 0)
                  .sort((a, b) => b.net - a.net)
                  .slice(0, 20)
                  .map((row) => {
                    const maxNet = Math.max(...Object.values(records.net_spend).map((r) => Math.abs(r.net)), 1);
                    return (
                      <div key={row.club_id} className="p-2.5 bg-ink/40 rounded-xl border border-line">
                        <div className="flex items-center justify-between gap-2.5">
                          <span className="flex min-w-0 items-center gap-2">
                            {getClubCrestUrlByShort(row.short_name) && (
                              <img src={getClubCrestUrlByShort(row.short_name)!} alt={`${row.club_name} crest`} loading="lazy" draggable={false} className="w-6 h-6 rounded-full bg-bone object-contain p-[2px] border border-bone/25 shrink-0" />
                            )}
                            <span className="truncate text-[13px] font-semibold text-bone">{row.club_name}</span>
                          </span>
                          <span className={cx('font-mono font-bold text-[13px] shrink-0', row.net > 0 ? 'text-ember' : 'text-pitchtone')}>
                            {row.net > 0 ? '−' : '+'}{formatMillions(Math.abs(row.net))}
                          </span>
                        </div>
                        <div className="mt-2 h-1 rounded-full bg-white/[0.06] overflow-hidden">
                          <div
                            className={cx('h-full rounded-full', row.net > 0 ? 'bg-ember/80' : 'bg-pitchtone/80')}
                            style={{ width: `${(Math.abs(row.net) / maxNet) * 100}%` }}
                          />
                        </div>
                        <p className="mt-1 font-mono text-[10px] text-sage">
                          Spent {formatMillions(row.spent)} · Received {formatMillions(row.received)}
                        </p>
                      </div>
                    );
                  })}
              </div>
            )}
          </Card>
        </div>
      )}
    </div>
  );
};
