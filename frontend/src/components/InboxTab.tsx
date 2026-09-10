import React, { useCallback, useEffect, useMemo, useState } from 'react';
import type { InboxItem } from '../types';
import { fetchInbox, fetchFixture, markInboxRead } from '../services/api';
import { soundManager } from '../audio/webAudio';
import { cx } from '../lib/format';
import { Card, EmptyState, LoadingState, PanelHeader } from './ui/ui';
import { usePlayerSheet } from './PlayerSheet';
import { MatchDetailModal } from './MatchDetailModal';
import type { Fixture } from '../types';

const CATEGORIES: Array<{ id: 'all' | InboxItem['category']; label: string }> = [
  { id: 'all', label: 'All' },
  { id: 'match', label: 'Matches' },
  { id: 'cup', label: 'Cups' },
  { id: 'wonderkid', label: 'Wonderkids' },
  { id: 'honour', label: 'Honours' },
  { id: 'race', label: 'Table' },
  { id: 'transfer', label: 'Transfers' },
  { id: 'dugout', label: 'Dugout' },
  { id: 'injury', label: 'Injuries' },
  { id: 'youth', label: 'Academy' },
];

const TONE: Record<InboxItem['category'], string> = {
  match: 'text-bone',
  cup: 'text-[#A9CBDD]',
  wonderkid: 'text-brass',
  honour: 'text-brass',
  race: 'text-[#A9CDBB]',
  transfer: 'text-bone',
  system: 'text-sage',
  injury: 'text-[#D89A84]',
  dugout: 'text-[#B9B3E6]',
  youth: 'text-[#A9CDBB]',
};

function categoryLabel(c: InboxItem['category']): string {
  switch (c) {
    case 'match':
      return 'Match';
    case 'cup':
      return 'Cup';
    case 'wonderkid':
      return 'Wonderkid';
    case 'honour':
      return 'Honour';
    case 'race':
      return 'Table';
    case 'transfer':
      return 'Transfer';
    case 'injury':
      return 'Injury';
    case 'dugout':
      return 'Dugout';
    case 'youth':
      return 'Academy';
    default:
      return 'Season';
  }
}

interface InboxTabProps {
  careerKey?: number;
  onUnread?: (n: number) => void;
}

// Thread ids group the season memory: school and board exam-term letters
// (F4), transfers, and cups each read as one thread.
export type InboxThreadId = 'school' | 'board' | 'transfer' | 'cup' | 'match' | 'wonderkid' | 'table' | 'other';

export function threadOf(item: InboxItem): InboxThreadId {
  const head = `${item.headline} ${item.body}`.toLowerCase();
  if (item.category === 'transfer') return 'transfer';
  if (item.category === 'cup') return 'cup';
  if (item.category === 'match') return 'match';
  if (item.category === 'race') return 'table';
  if (item.category === 'wonderkid' || item.category === 'honour' || item.category === 'injury' || item.category === 'dugout') return 'wonderkid';
  if (head.includes('board letter')) return 'board';
  if (
    item.category === 'youth' ||
    item.category === 'system' ||
    head.includes('school letter') ||
    head.includes('exam') ||
    head.includes('stays in school') ||
    head.includes('leaves school')
  ) {
    return 'school';
  }
  return 'other';
}

const THREAD_LABEL: Record<InboxThreadId, string> = {
  school: 'School',
  board: 'Board',
  transfer: 'Transfers',
  cup: 'Cups',
  match: 'Matches',
  wonderkid: 'Wonderkids',
  table: 'Table',
  other: 'Season',
};

const THREAD_ORDER: InboxThreadId[] = ['school', 'board', 'transfer', 'cup', 'match', 'wonderkid', 'table', 'other'];

export function threadInbox(list: InboxItem[]): Array<{ id: InboxThreadId; label: string; items: InboxItem[] }> {
  const buckets = new Map<InboxThreadId, InboxItem[]>();
  for (const item of list) {
    const id = threadOf(item);
    const bucket = buckets.get(id);
    if (bucket) bucket.push(item);
    else buckets.set(id, [item]);
  }
  return THREAD_ORDER.filter((id) => buckets.has(id)).map((id) => ({
    id,
    label: THREAD_LABEL[id],
    items: (buckets.get(id) ?? []).slice().sort((a, b) => b.matchweek - a.matchweek),
  }));
}

export const InboxTab: React.FC<InboxTabProps> = ({ careerKey = 0, onUnread }) => {
  const { openPlayer } = usePlayerSheet();
  const [items, setItems] = useState<InboxItem[]>([]);
  const [unread, setUnread] = useState(0);
  const [season, setSeason] = useState('2026-27');
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<(typeof CATEGORIES)[number]['id']>('all');
  const [openFixture, setOpenFixture] = useState<Fixture | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const load = useCallback(async () => {
    const feed = await fetchInbox(100);
    setItems(feed.items);
    setUnread(feed.unread);
    setSeason(feed.season_name);
    onUnread?.(feed.unread);
  }, [onUnread]);

  useEffect(() => {
    load().finally(() => setLoading(false));
  }, [load, careerKey]);

  const visible = useMemo(
    () => (filter === 'all' ? items : items.filter((i) => i.category === filter)),
    [items, filter],
  );

  // Thread grouping (F9): school, board, transfer and cup letters read as
  // threads. No new letter DB — pure client grouping over the feed.
  const threads = useMemo(() => threadInbox(visible), [visible]);

  const handleOpen = async (item: InboxItem) => {
    soundManager.playClick();
    setNotice(null);
    if (item.unread) {
      const res = await markInboxRead(item.id);
      setUnread(res.unread);
      onUnread?.(res.unread);
      setItems((prev) => prev.map((i) => (i.id === item.id ? { ...i, unread: false } : i)));
    }
    // Jump to the fixture sheet when the route exists (decided games only),
    // else to the kid sheet when a player route exists. Broken targets get a
    // notice instead of a silent no-op.
    if (item.fixture_id) {
      const fx = await fetchFixture(item.fixture_id);
      if (fx && fx.status === 'finished') {
        setOpenFixture(fx);
        return;
      }
      if (fx) {
        setNotice(`“${item.headline}” is still to play (MW ${fx.matchweek}). The sheet opens once it is decided.`);
        return;
      }
      if (item.player_id) {
        openPlayer(item.player_id);
        return;
      }
      setNotice(`“${item.headline}” has no sheet yet — the tie may have moved weeks.`);
      return;
    }
    if (item.player_id) {
      openPlayer(item.player_id);
    }
  };

  const handleReadAll = async () => {
    soundManager.playClick();
    const res = await markInboxRead(undefined, true);
    setUnread(res.unread);
    onUnread?.(res.unread);
    setItems((prev) => prev.map((i) => ({ ...i, unread: false })));
  };

  if (loading) return <Card><LoadingState message="Loading the paper…" /></Card>;

  return (
    <div className="space-y-4">
      <Card>
        <PanelHeader
          kicker={`${season} · season memory`}
          title="Inbox"
          subtitle="Title races, cup nights, wonderkid breakouts and big transfers — the year as it happened, not only the window."
          right={
            unread > 0 ? (
              <button
                onClick={() => void handleReadAll()}
                className="px-3 py-1.5 text-[13px] font-semibold text-sage hover:text-bone border border-line"
              >
                Mark all read
              </button>
            ) : null
          }
        />
        <div className="flex gap-1.5 overflow-x-auto mt-4" role="tablist" aria-label="Inbox filter">
          {CATEGORIES.map((c) => (
            <button
              key={c.id}
              onClick={() => {
                soundManager.playClick();
                setFilter(c.id);
              }}
              aria-pressed={filter === c.id}
              className={cx(
                'px-3.5 py-2 text-[13px] font-semibold whitespace-nowrap border',
                filter === c.id ? 'bg-bone text-ink border-bone' : 'bg-cardLight text-sage hover:text-bone border-line',
              )}
            >
              {c.label}
            </button>
          ))}
        </div>
      </Card>

      {notice && (
        <div className="px-4 py-2.5 border border-line bg-cardLight text-[13px] text-bone" role="status">
          {notice}
        </div>
      )}

      <div className="panel-tight">
        {visible.length === 0 ? (
          <EmptyState message="The paper is quiet. Play a matchweek and the season starts writing itself." />
        ) : (
          threads.map((thread) => (
            <section key={thread.id} aria-label={`${thread.label} thread`}>
              <div className="px-5 pt-4 pb-1 flex items-center justify-between">
                <p className="text-[11px] font-mono uppercase tracking-[0.12em] text-brass">{thread.label}</p>
                <span className="text-[11px] font-mono text-sage">{thread.items.length}</span>
              </div>
              <ul className="divide-y divide-line/70">
                {thread.items.map((item) => (
                  <li key={item.id}>
                    <button
                      type="button"
                      onClick={() => void handleOpen(item)}
                      className={cx(
                        'w-full text-left px-5 py-4 hover:bg-cardLight/60 transition-colors',
                        item.unread && 'bg-brass/[0.04]',
                      )}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <p className="text-[12px] font-mono text-sage">
                            {item.unread && <span className="inline-block w-1.5 h-1.5 rounded-full bg-brass mr-2 align-middle" />}
                            {categoryLabel(item.category)} · MW {item.matchweek} · {item.season_name}
                          </p>
                          <p className={cx('font-semibold text-[16px] mt-1 leading-snug', TONE[item.category])}>
                            {item.headline}
                          </p>
                          {item.body && <p className="text-[13px] text-sage mt-1 leading-relaxed">{item.body}</p>}
                        </div>
                        {(item.player_id || item.fixture_id) && (
                          <span className="text-[11px] font-mono text-sage shrink-0 mt-1">Open</span>
                        )}
                      </div>
                    </button>
                  </li>
                ))}
              </ul>
            </section>
          ))
        )}
      </div>

      <MatchDetailModal fixture={openFixture} onClose={() => setOpenFixture(null)} />
    </div>
  );
};
