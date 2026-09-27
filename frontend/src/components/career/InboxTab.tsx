import React, { useCallback, useEffect, useMemo, useState } from 'react';
import type { InboxItem } from '../../types';
import { fetchInbox, fetchFixture, markInboxRead, replyInbox } from '../../services/api';
import { soundManager } from '../../audio/webAudio';
import { cx } from '../../lib/format';
import { Card, EmptyState, LoadingState, PanelHeader } from '../ui/ui';
import { usePlayerSheet } from '../clubs/PlayerSheet';
import { PostMatchModal } from '../postmatch/PostMatchBroadcast';
import type { Fixture } from '../../types';

type InboxFilter = 'all' | 'transfers' | 'players' | 'managers' | 'competitions' | 'awards' | 'world';

const CATEGORIES: Array<{ id: InboxFilter; label: string }> = [
  { id: 'all', label: 'All' },
  { id: 'transfers', label: 'Transfers' },
  { id: 'players', label: 'Players' },
  { id: 'managers', label: 'Managers' },
  { id: 'competitions', label: 'Competitions' },
  { id: 'awards', label: 'Awards' },
  { id: 'world', label: 'World' },
];

function inFilter(item: InboxItem, filter: InboxFilter): boolean {
  if (filter === 'all') return true;
  if (filter === 'transfers') return item.category === 'transfer';
  if (filter === 'players') return ['wonderkid', 'injury', 'youth'].includes(item.category);
  if (filter === 'managers') return item.category === 'dugout';
  if (filter === 'competitions') return ['cup', 'race', 'match'].includes(item.category);
  if (filter === 'awards') return ['honour', 'nxgn'].includes(item.category);
  return item.category === 'system';
}

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
  nxgn: 'text-brass',
  milestone: 'text-brass',
  manager: 'text-[#B9B3E6]',
  watch: 'text-[#A9CBDD]',
  club: 'text-bone',
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
    case 'nxgn':
      return 'NXGN';
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
  if (item.category === 'wonderkid' || item.category === 'honour' || item.category === 'injury' || item.category === 'dugout' || item.category === 'nxgn') return 'wonderkid';
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
  const [filter, setFilter] = useState<InboxFilter>('all');
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
    () => items.filter((item) => inFilter(item, filter)),
    [items, filter],
  );

  // Thread grouping (F9): school, board, transfer and cup letters read as
  // threads. No new letter DB — pure client grouping over the feed.
  const threads = useMemo(() => threadInbox(visible), [visible]);

  const handleOpen = async (item: InboxItem) => {
    soundManager.playClick();
    setNotice(null);
    if (item.unread) {
      try {
        const res = await markInboxRead(item.id);
        setUnread(res.unread);
        onUnread?.(res.unread);
        setItems((prev) => prev.map((i) => (i.id === item.id ? { ...i, unread: false } : i)));
      } catch {
        setNotice('Could not reach the paper. The unread badge is unchanged — try again.');
      }
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

  const handleReply = async (item: InboxItem, choiceId: string) => {
    soundManager.playClick();
    const res = await replyInbox(item.id, choiceId);
    setNotice(res.message ?? (res.status === 'success' ? 'Reply sent.' : 'Could not reply.'));
    await load();
  };

  const handleReadAll = async () => {
    soundManager.playClick();
    try {
      const res = await markInboxRead(undefined, true);
      setUnread(res.unread);
      onUnread?.(res.unread);
      setItems((prev) => prev.map((i) => ({ ...i, unread: false })));
    } catch {
      setNotice('Could not reach the paper. Nothing was marked read — try again.');
    }
  };

  if (loading) return <Card><LoadingState message="Loading the paper…" /></Card>;

  return (
    <div className="space-y-4">
      <Card>
        <PanelHeader
          kicker={`${season} · world feed`}
          title="News"
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
        <div className="section-tabs -mx-4 sm:-mx-5 -mb-4 sm:-mb-5 mt-4 px-2" role="tablist" aria-label="News filter">
          {CATEGORIES.map((c) => (
            <button
              key={c.id}
              onClick={() => {
                soundManager.playClick();
                setFilter(c.id);
              }}
              role="tab"
              aria-selected={filter === c.id}
              className={cx(
                'section-tab whitespace-nowrap',
                filter === c.id
                  ? 'section-tab-active'
                  : '',
              )}
            >
              {c.label}
            </button>
          ))}
        </div>
      </Card>

      {notice && (
        <div className="px-4 py-2.5 rounded-xl border border-line bg-cardLight text-[13px] text-bone" role="status">
          {notice}
        </div>
      )}

      <div className="console-card overflow-hidden">
        {visible.length === 0 ? (
          <EmptyState message="The paper is quiet. Play a matchweek and the season starts writing itself." />
        ) : (
          threads.map((thread) => (
            <section key={thread.id} aria-label={`${thread.label} thread`}>
              <div className="px-5 pt-4 pb-2 flex items-center justify-between border-b border-line/40 bg-cardLight/30">
                <p className="text-[11px] font-mono uppercase tracking-[0.14em] text-brass font-bold">{thread.label}</p>
                <span className="text-[11px] font-mono text-sage">{thread.items.length} item{thread.items.length === 1 ? '' : 's'}</span>
              </div>
              <ul className="divide-y divide-line/70">
                {thread.items.map((item) => (
                  <li key={item.id} className="transition-colors">
                    <button
                      type="button"
                      onClick={() => void handleOpen(item)}
                      className={cx(
                        'w-full text-left px-5 py-4 hover:bg-cardLight/70 transition-colors',
                        item.unread && 'bg-brass/[0.04]',
                      )}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            {item.unread && <span className="w-2 h-2 rounded-full bg-brass shadow-[0_0_8px_rgba(229,193,88,0.8)] shrink-0" />}
                            <span className="text-[11px] font-mono uppercase tracking-wider px-2 py-0.5 rounded bg-cardLight border border-line text-sage font-medium">
                              {categoryLabel(item.category)}
                            </span>
                            <span className="text-[12px] font-mono text-sage">
                              MW {item.matchweek} · {item.season_name}
                            </span>
                          </div>
                          <p className={cx('font-semibold text-[15.5px] mt-1.5 leading-snug', TONE[item.category])}>
                            {item.headline}
                          </p>
                          {item.body && <p className="text-[13px] text-sage mt-1 leading-relaxed">{item.body}</p>}
                        </div>
                        {(item.player_id || item.fixture_id) && (
                          <span className="text-[11.5px] font-mono text-brass border border-brass/40 bg-brass/10 px-2.5 py-1 rounded-lg shrink-0 mt-1 hover:bg-brass/20 transition-colors">
                            Open &rarr;
                          </span>
                        )}
                      </div>
                    </button>
                    {item.choices && item.choices.length > 0 && !item.resolved && (
                      <div className="px-5 pb-3.5 flex flex-wrap gap-2">
                        {item.choices.map((choice) => (
                          <button
                            key={choice.id}
                            type="button"
                            onClick={() => void handleReply(item, choice.id)}
                            className="px-3 py-1.5 rounded-lg border border-brass/50 bg-brass/10 hover:bg-brass/25 text-brass font-semibold text-[12.5px] transition-colors"
                          >
                            {choice.label}
                          </button>
                        ))}
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          ))
        )}
      </div>

      <PostMatchModal fixture={openFixture} onClose={() => setOpenFixture(null)} />
    </div>
  );
};
