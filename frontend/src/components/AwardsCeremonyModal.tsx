import React, { useEffect, useState } from 'react';
import { Sparkles, Trophy, ChevronRight, RotateCcw } from 'lucide-react';
import type { AwardsCategory, AwardsCeremony, BallonDorRankItem, TeamOfTheSeason, ManagerOfTheYear } from '../types';
import { fetchAwardsCeremony } from '../services/api';
import { soundManager } from '../audio/webAudio';
import { cx } from '../lib/format';
import { getClubCrestUrlByShort } from '../lib/clubLogos';
import { Modal, ModalHeader, LoadingState, EmptyState } from './ui/ui';

interface AwardsCeremonyModalProps {
  open: boolean;
  onClose: () => void;
}

const NomineeCard: React.FC<{
  name: string;
  club: string;
  short: string;
  stats: string;
  ovr: number;
  isWinner: boolean;
  dimmed: boolean;
  index: number;
}> = ({ name, club, short, stats, ovr, isWinner, dimmed, index }) => {
  const url = getClubCrestUrlByShort(short);
  return (
    <div
      className={cx(
        'p-4 rounded-xl border flex items-center gap-3 animate-fade-in transition-opacity',
        isWinner ? 'border-brass bg-brass/[0.1] shadow-raised' : 'border-line bg-ink/50',
        dimmed && !isWinner && 'opacity-45',
      )}
      style={{ animationDelay: `${index * 120}ms` }}
    >
      {url ? (
        <img src={url} alt={`${club} crest`} loading="lazy" draggable={false} className="w-11 h-11 rounded-lg bg-bone object-contain p-1 border border-bone/25 shrink-0" />
      ) : (
        <span className="w-11 h-11 rounded-lg bg-cardLight border border-line shrink-0" />
      )}
      <div className="min-w-0 flex-1">
        <p className={cx('font-semibold truncate', isWinner ? 'text-brass text-[15px]' : 'text-bone text-[14px]')}>{name}</p>
        <p className="text-[12px] text-sage truncate">{club}</p>
        <p className="font-mono text-[11.5px] text-bone/75 mt-0.5">{stats}</p>
      </div>
      <span className={cx('font-mono font-bold text-[15px] shrink-0', isWinner ? 'text-brass' : 'text-sage')}>{ovr}</span>
    </div>
  );
};

const CategoryStage: React.FC<{
  category: AwardsCategory;
  index: number;
  total: number;
  onNext: () => void;
  isLast: boolean;
}> = ({ category, index, total, onNext, isLast }) => {
  const [revealed, setRevealed] = useState(false);

  useEffect(() => {
    setRevealed(false);
  }, [category.key]);

  const explicitWinnerId = category.winner_id ?? category.winner?.player_id;
  const nomineeIsWinner = (playerId: string | undefined): boolean => {
    if (!explicitWinnerId || !playerId) return false;
    return explicitWinnerId === playerId;
  };

  return (
    <div>
      <p className="eyebrow text-center">Award {index + 1} of {total}</p>
      <h3 className="font-display text-[26px] font-semibold text-bone text-center mt-1">{category.title}</h3>
      <p className="text-center text-[13px] text-sage mt-1">{category.blurb}</p>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5 mt-5">
        {category.nominees.map((n, i) => (
          <NomineeCard
            key={n.player_id ?? `${n.full_name}-${i}`}
            name={n.full_name}
            club={n.club_name}
            short={n.short_name}
            stats={n.stats_line}
            ovr={n.ovr}
            isWinner={revealed && nomineeIsWinner(n.player_id)}
            dimmed={revealed}
            index={i}
          />
        ))}
      </div>

      <div className="mt-5 text-center">
        {!revealed ? (
          <button
            onClick={() => {
              soundManager.playWhistle();
              setRevealed(true);
            }}
            className="px-8 py-3 rounded-xl bg-brass hover:bg-[#D4AF4D] text-ink text-[14px] font-bold transition-colors inline-flex items-center gap-2"
          >
            <Sparkles size={16} /> Reveal the winner
          </button>
        ) : (
          <div className="space-y-3 animate-fade-in">
            <p className="font-display text-[19px] text-brass font-semibold flex items-center justify-center gap-2">
              <Trophy size={18} /> {category.winner?.full_name} takes it
            </p>
            <button
              onClick={() => {
                soundManager.playClick();
                onNext();
              }}
              className="px-8 py-3 rounded-xl bg-brass hover:bg-[#D4AF4D] text-ink text-[14px] font-bold transition-colors inline-flex items-center gap-2"
            >
              {isLast ? 'Advance to Ballon d\'Or Gala' : 'Next award'} <ChevronRight size={16} />
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

const BallonDorPodiumView: React.FC<{ rankings: BallonDorRankItem[]; onNext: () => void }> = ({ rankings, onNext }) => {
  const top1 = rankings[0];
  const top2 = rankings[1];
  const top3 = rankings[2];
  const rest = rankings.slice(3, 10);

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="text-center">
        <span className="px-3 py-1 rounded-full bg-brass/15 border border-brass/40 text-brass text-[11px] font-mono font-bold uppercase tracking-wider">Global Football Honors</span>
        <h3 className="font-display text-[26px] font-semibold text-bone mt-2 flex items-center justify-center gap-2">
          <Trophy size={24} className="text-brass" /> Ballon d'Or Gala
        </h3>
        <p className="text-[13px] text-sage mt-1">Continental jury ranking based on performance, production and team achievement.</p>
      </div>

      <div className="grid grid-cols-3 gap-2 sm:gap-4 items-end pt-4 pb-2">
        {top2 && (
          <div className="flex flex-col items-center">
            <div className="mb-2 text-center">
              <span className="text-xl">🥈</span>
              <p className="font-bold text-[14px] text-bone truncate max-w-[120px]">{top2.full_name}</p>
              <p className="text-[11px] text-sage font-mono">{top2.club_short} · {top2.ovr} OVR</p>
              <p className="text-[11px] text-bone/80 font-mono mt-0.5">{top2.goals}G · {top2.assists}A</p>
            </div>
            <div className="w-full h-32 rounded-t-xl bg-gradient-to-t from-ink/80 to-slate-400/20 border-t-2 border-x border-slate-300/40 flex flex-col items-center justify-center p-2">
              <span className="font-mono text-[10px] text-slate-300 font-bold uppercase">2nd Place</span>
              <span className="font-mono font-bold text-[16px] text-bone mt-1">{top2.score} pts</span>
            </div>
          </div>
        )}

        {top1 && (
          <div className="flex flex-col items-center">
            <div className="mb-2 text-center animate-bounce"><span className="text-3xl">🥇</span></div>
            <div className="mb-2 text-center">
              <span className="inline-block px-2.5 py-0.5 rounded-full bg-brass/20 text-brass border border-brass/50 text-[10px] font-mono font-bold uppercase tracking-wider mb-1">Winner</span>
              <p className="font-bold text-[16px] text-brass truncate max-w-[140px]">{top1.full_name}</p>
              <p className="text-[12px] text-bone/85 font-mono">{top1.club_name} ({top1.club_short})</p>
              <p className="text-[12px] text-brass font-mono font-semibold mt-0.5">{top1.goals}G · {top1.assists}A · {top1.ovr} OVR</p>
            </div>
            <div className="w-full h-44 rounded-t-xl bg-gradient-to-t from-brass/25 to-brass/10 border-t-4 border-x border-brass flex flex-col items-center justify-center p-2 shadow-[0_-4px_20px_rgba(199,162,58,0.25)]">
              <Trophy size={28} className="text-brass animate-pulse" />
              <span className="font-mono text-[11px] text-brass font-bold uppercase mt-1">1st Place</span>
              <span className="font-mono font-bold text-[20px] text-bone mt-0.5">{top1.score} pts</span>
            </div>
          </div>
        )}

        {top3 && (
          <div className="flex flex-col items-center">
            <div className="mb-2 text-center">
              <span className="text-xl">🥉</span>
              <p className="font-bold text-[14px] text-bone truncate max-w-[120px]">{top3.full_name}</p>
              <p className="text-[11px] text-sage font-mono">{top3.club_short} · {top3.ovr} OVR</p>
              <p className="text-[11px] text-bone/80 font-mono mt-0.5">{top3.goals}G · {top3.assists}A</p>
            </div>
            <div className="w-full h-24 rounded-t-xl bg-gradient-to-t from-ink/80 to-amber-700/20 border-t-2 border-x border-amber-600/40 flex flex-col items-center justify-center p-2">
              <span className="font-mono text-[10px] text-amber-500 font-bold uppercase">3rd Place</span>
              <span className="font-mono font-bold text-[14px] text-bone mt-1">{top3.score} pts</span>
            </div>
          </div>
        )}
      </div>

      {rest.length > 0 && (
        <div className="rounded-xl border border-line bg-ink/40 p-3.5 space-y-2">
          <p className="eyebrow">Top 10 Finalists</p>
          <div className="divide-y divide-line/60">
            {rest.map((item) => (
              <div key={item.player_id} className="py-2 flex items-center justify-between text-[13px]">
                <div className="flex items-center gap-2.5">
                  <span className="w-6 font-mono text-sage text-[12px] font-bold">#{item.rank}</span>
                  <div>
                    <span className="font-semibold text-bone">{item.full_name}</span>
                    <span className="text-[11px] text-sage font-mono ml-2">({item.club_short} · {item.position})</span>
                    {item.is_wonderkid && <span className="ml-2 px-1.5 py-0.5 rounded text-[10px] font-mono bg-brass/15 text-brass font-bold">U-14</span>}
                  </div>
                </div>
                <div className="flex items-center gap-3 font-mono text-[12px]">
                  <span className="text-sage">{item.goals}G · {item.assists}A</span>
                  <span className="text-brass font-bold">{item.score} pts</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="text-center pt-2">
        <button onClick={onNext} className="px-8 py-3 rounded-xl bg-brass hover:bg-[#D4AF4D] text-ink text-[14px] font-bold transition-colors inline-flex items-center gap-2">
          View Team of the Season <ChevronRight size={16} />
        </button>
      </div>
    </div>
  );
};

const TeamOfTheSeasonView: React.FC<{ tots: TeamOfTheSeason; onNext: () => void }> = ({ tots, onNext }) => {
  const formation = tots.formation || '4-3-3';
  const lines = [
    { label: 'Attacking Trio', players: [tots.fwd1, tots.fwd2, tots.fwd3].filter(Boolean) },
    { label: 'Midfield Engine', players: [tots.mid1, tots.mid2, tots.mid3].filter(Boolean) },
    { label: 'Defensive Line', players: [tots.lb, tots.cb1, tots.cb2, tots.rb].filter(Boolean) },
    { label: 'Goalkeeper', players: [tots.gk].filter(Boolean) },
  ];

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="text-center">
        <span className="px-3 py-1 rounded-full bg-brass/15 border border-brass/40 text-brass text-[11px] font-mono font-bold uppercase tracking-wider">Super League All-Stars</span>
        <h3 className="font-display text-[26px] font-semibold text-bone mt-2">Team of the Season</h3>
        <p className="text-[13px] text-sage mt-1">Official {formation} Continental Best XI based on seasonal ratings and impact.</p>
      </div>

      <div className="space-y-4">
        {lines.map((line) => (
          <div key={line.label} className="space-y-2">
            <p className="eyebrow">{line.label}</p>
            <div className={cx('grid gap-2.5', line.players.length === 1 ? 'grid-cols-1 max-w-sm mx-auto' : line.players.length === 4 ? 'grid-cols-2 sm:grid-cols-4' : 'grid-cols-1 sm:grid-cols-3')}>
              {line.players.map((card) => {
                const url = getClubCrestUrlByShort(card.club_short);
                return (
                  <div key={card.player_id} className="p-3.5 rounded-xl border border-brass/35 bg-brass/[0.05] hover:border-brass/70 transition-all flex flex-col justify-between">
                    <div className="flex items-center justify-between gap-2">
                      <span className="px-1.5 py-0.5 rounded bg-cardLight text-sage font-mono text-[10px] font-bold">{card.role || card.position}</span>
                      <span className="font-mono font-bold text-brass text-[15px]">{card.ovr} OVR</span>
                    </div>
                    <div className="flex items-center gap-2.5 my-2">
                      {url ? <img src={url} alt="" className="w-8 h-8 rounded-md bg-bone p-0.5 object-contain shrink-0" /> : <span className="w-8 h-8 rounded-md bg-cardLight shrink-0" />}
                      <div className="min-w-0">
                        <p className="font-semibold text-bone text-[13px] truncate">{card.full_name}</p>
                        <p className="text-[11px] text-sage truncate font-mono">{card.club_short}</p>
                      </div>
                    </div>
                    <div className="text-[11px] font-mono text-sage pt-2 border-t border-line/40 flex justify-between">
                      <span>{card.goals} G · {card.assists} A</span>
                      {card.is_wonderkid && <span className="text-brass font-bold">PRODIGY</span>}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>

      <div className="text-center pt-2">
        <button onClick={onNext} className="px-8 py-3 rounded-xl bg-brass hover:bg-[#D4AF4D] text-ink text-[14px] font-bold transition-colors inline-flex items-center gap-2">
          View Manager of the Year <ChevronRight size={16} />
        </button>
      </div>
    </div>
  );
};

const ManagerOfTheYearView: React.FC<{ manager: ManagerOfTheYear; onClose: () => void }> = ({ manager, onClose }) => {
  const url = getClubCrestUrlByShort(manager.short_name);
  return (
    <div className="space-y-6 animate-fade-in">
      <div className="text-center">
        <span className="px-3 py-1 rounded-full bg-brass/15 border border-brass/40 text-brass text-[11px] font-mono font-bold uppercase tracking-wider">Tactical Excellence Award</span>
        <h3 className="font-display text-[26px] font-semibold text-bone mt-2">Manager of the Year</h3>
        <p className="text-[13px] text-sage mt-1">Honouring the manager whose results most exceeded the club's modeled expectations.</p>
      </div>

      <div className="p-6 rounded-2xl border-2 border-brass/60 bg-gradient-to-br from-cardLight via-ink to-cardLight shadow-2xl space-y-5">
        <div className="flex flex-col sm:flex-row items-center sm:items-start gap-4 text-center sm:text-left">
          {url ? <img src={url} alt="" className="w-16 h-16 rounded-xl bg-bone p-1 object-contain border border-brass/50 shadow-md shrink-0" /> : <div className="w-16 h-16 rounded-xl bg-cardLight border border-brass/50 shrink-0" />}
          <div className="flex-1">
            <span className="px-2 py-0.5 rounded bg-brass text-ink font-mono font-bold text-[11px] uppercase tracking-wider">{manager.club_name}</span>
            <h4 className="font-display font-bold text-[24px] text-bone mt-1.5">{manager.name || manager.manager_name}</h4>
            <p className="text-[13px] text-brass font-mono mt-0.5">{manager.tactic} · {manager.style}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-center">
          <div className="p-3 rounded-xl bg-ink/60 border border-line"><p className="text-[10px] text-sage uppercase font-mono">League Finish</p><p className="font-mono font-bold text-[18px] text-brass mt-1">#{manager.actual_finish}</p></div>
          <div className="p-3 rounded-xl bg-ink/60 border border-line"><p className="text-[10px] text-sage uppercase font-mono">Expectation</p><p className="font-mono font-bold text-[18px] text-bone mt-1">#{manager.expected_finish}</p></div>
          <div className="p-3 rounded-xl bg-pitchtone/10 border border-pitchtone/40"><p className="text-[10px] text-pitchtone uppercase font-mono">Overperformed</p><p className="font-mono font-bold text-[18px] text-pitchtone mt-1">+{manager.outperformed_places} places</p></div>
          <div className="p-3 rounded-xl bg-ink/60 border border-line"><p className="text-[10px] text-sage uppercase font-mono">Trophies Lifted</p><p className="font-mono font-bold text-[18px] text-bone mt-1">{manager.trophies_won}</p></div>
        </div>

        {manager.accolade && <blockquote className="p-4 rounded-xl bg-ink/70 border-l-4 border-brass text-[14px] text-bone/90 italic leading-relaxed">“{manager.accolade}”</blockquote>}
      </div>

      <div className="text-center pt-2">
        <button onClick={onClose} className="px-8 py-3 rounded-xl bg-cardLight hover:bg-cardHover text-bone border border-line text-[14px] font-bold transition-colors inline-flex items-center gap-2">
          <RotateCcw size={15} /> Back to the season
        </button>
      </div>
    </div>
  );
};

export const AwardsCeremonyModal: React.FC<AwardsCeremonyModalProps> = ({ open, onClose }) => {
  const [data, setData] = useState<AwardsCeremony | null>(null);
  const [loading, setLoading] = useState(false);
  const [stage, setStage] = useState<'intro' | 'categories' | 'ballondor' | 'tots' | 'manager'>('intro');
  const [catIndex, setCatIndex] = useState(0);

  useEffect(() => {
    if (!open) return;
    setStage('intro');
    setCatIndex(0);
    setData(null);
    setLoading(true);
    fetchAwardsCeremony().then(setData).finally(() => setLoading(false));
  }, [open]);

  const categories = data?.categories ?? [];
  const ballonDorList = data?.ballon_dor ?? [];
  const tots = data?.team_of_the_season ?? null;
  const moty = data?.manager_of_the_year ?? null;

  return (
    <Modal open={open} onClose={onClose} maxWidth="max-w-4xl">
      <ModalHeader
        title={<span className="flex items-center gap-2"><Trophy size={19} className="text-brass" /><span>Awards night{data ? `, ${data.season_name}` : ''}</span></span>}
        subtitle="The continental gala of prestige and legacy."
        onClose={onClose}
      />
      <div className="p-6">
        {loading && <LoadingState message="Sealing the envelopes…" />}
        {!loading && !data && <EmptyState message="The votes are still being counted. Finish the season first." />}

        {!loading && data && stage === 'intro' && (
          <div className="text-center py-8 animate-fade-in max-w-lg mx-auto">
            <p className="eyebrow">End of {data.season_name}</p>
            <h3 className="font-display text-[30px] font-semibold text-bone mt-2">Continental Honors Gala</h3>
            <p className="text-[13px] text-sage mt-2 leading-relaxed">Finalists are presented independently of the sealed winner selection. Reveal the envelopes, then continue to the Ballon d'Or ranking, Team of the Season and Manager of the Year.</p>
            <button onClick={() => { soundManager.playWhistle(); setStage('categories'); }} className="mt-6 px-10 py-3.5 rounded-xl bg-brass hover:bg-[#D4AF4D] text-ink text-[15px] font-bold transition-colors inline-flex items-center gap-2 shadow-lg">
              <Sparkles size={17} /> Begin the ceremony
            </button>
          </div>
        )}

        {!loading && data && stage !== 'intro' && (
          <div className="space-y-6">
            <div className="flex items-center justify-center gap-1.5 p-1 bg-ink/60 border border-line rounded-xl overflow-x-auto">
              <button type="button" onClick={() => setStage('categories')} className={cx('px-3 py-1.5 rounded-lg text-[12px] font-semibold transition-colors shrink-0', stage === 'categories' ? 'bg-brass text-ink' : 'text-sage hover:text-bone')}>Award Envelopes ({catIndex + 1}/{categories.length || 4})</button>
              <button type="button" onClick={() => setStage('ballondor')} className={cx('px-3 py-1.5 rounded-lg text-[12px] font-semibold transition-colors shrink-0', stage === 'ballondor' ? 'bg-brass text-ink' : 'text-sage hover:text-bone')}>Ballon d'Or 🥇</button>
              <button type="button" onClick={() => setStage('tots')} className={cx('px-3 py-1.5 rounded-lg text-[12px] font-semibold transition-colors shrink-0', stage === 'tots' ? 'bg-brass text-ink' : 'text-sage hover:text-bone')}>Team of the Season</button>
              <button type="button" onClick={() => setStage('manager')} className={cx('px-3 py-1.5 rounded-lg text-[12px] font-semibold transition-colors shrink-0', stage === 'manager' ? 'bg-brass text-ink' : 'text-sage hover:text-bone')}>Manager of the Year</button>
            </div>

            {stage === 'categories' && categories[catIndex] && (
              <CategoryStage
                key={categories[catIndex].key}
                category={categories[catIndex]}
                index={catIndex}
                total={categories.length}
                onNext={() => {
                  if (catIndex < categories.length - 1) setCatIndex((i) => i + 1);
                  else setStage('ballondor');
                }}
                isLast={catIndex === categories.length - 1}
              />
            )}
            {stage === 'ballondor' && <BallonDorPodiumView rankings={ballonDorList} onNext={() => setStage('tots')} />}
            {stage === 'tots' && tots && <TeamOfTheSeasonView tots={tots} onNext={() => setStage('manager')} />}
            {stage === 'manager' && moty && <ManagerOfTheYearView manager={moty} onClose={onClose} />}
          </div>
        )}
      </div>
    </Modal>
  );
};
