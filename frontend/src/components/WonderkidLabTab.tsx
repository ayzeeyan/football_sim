import React, { useCallback, useEffect, useState } from 'react';
import type { GrowthMilestoneItem, ProdigyData, ProdigyTimelineResponse } from '../types';
import { fetchProdigies, fetchGrowthMilestones, setProdigyPositionPath, setProdigySchoolTrack, fetchProdigyTimeline } from '../services/api';
import { Dna, Zap, TrendingUp, ClipboardCheck, Award, LayoutGrid, Eye, Sparkle, Microscope, CheckCircle2, Lock, Ruler, Scale, Calendar, Trophy, ChevronRight } from 'lucide-react';
import { soundManager } from '../audio/webAudio';
import { cx, stripEmojis, formatHeight } from '../lib/format';
import { getClubCrestUrlByShort } from '../lib/clubLogos';
import { Badge, Card, EmptyState, LoadingState, PanelHeader, ProgressBar } from './ui/ui';
import { ProdigyRadar } from './ProdigyRadar';
import { usePlayerSheet } from './PlayerSheet';

interface WonderkidLabTabProps {
  onShowToast: (msg: string) => void;
}

function milestoneTone(badge: string): string {
  if (badge === 'gold') return 'border-brass/45 bg-brass/[0.07] text-bone';
  if (badge === 'cyan') return 'border-[#8AB4C8]/35 bg-[#8AB4C8]/[0.07] text-bone';
  if (badge === 'green') return 'border-pitchtone/35 bg-pitchtone/[0.08] text-bone';
  return 'border-[#8E86C8]/35 bg-[#8E86C8]/[0.08] text-bone';
}

export const WonderkidLabTab: React.FC<WonderkidLabTabProps> = ({ onShowToast }) => {
  const { openPlayer } = usePlayerSheet();
  const [prodigies, setProdigies] = useState<ProdigyData[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [milestones, setMilestones] = useState<GrowthMilestoneItem[]>([]);
  const [viewMode, setViewMode] = useState<'deepdive' | 'comparison'>('deepdive');
  const [loading, setLoading] = useState(true);
  const [timelineData, setTimelineData] = useState<ProdigyTimelineResponse | null>(null);
  const [loadingTimeline, setLoadingTimeline] = useState(false);

  const selected = prodigies.find((p) => p.player_id === selectedId) ?? prodigies[0] ?? null;

  const loadData = useCallback(async () => {
    const [prod, miles] = await Promise.all([fetchProdigies(), fetchGrowthMilestones()]);
    setProdigies(prod);
    setMilestones(miles);
    setSelectedId((prev) => {
      if (prev && prod.some((p) => p.player_id === prev)) return prev;
      return prod[0]?.player_id ?? null;
    });
  }, []);

  useEffect(() => {
    loadData().finally(() => setLoading(false));
  }, [loadData]);

  useEffect(() => {
    if (!selected?.player_id) return;
    let active = true;
    setLoadingTimeline(true);
    fetchProdigyTimeline(selected.player_id)
      .then((res) => {
        if (active) setTimelineData(res);
      })
      .catch(() => {
        if (active) setTimelineData(null);
      })
      .finally(() => {
        if (active) setLoadingTimeline(false);
      });
    return () => {
      active = false;
    };
  }, [selected?.player_id]);

  if (loading) return <Card><LoadingState message="Loading the incubator…" /></Card>;

  const energy = selected?.training_energy ?? 3;
  const maxEnergy = selected?.max_training_energy ?? 3;

  return (
    <div className="space-y-5">
      <Card>
        <PanelHeader
          kicker="U-14 development · 12 outfield prodigies, age 14"
          title="Wonderkids"
          subtitle="School calendars, growth, and a second position after adult height. Minutes still do most of the developing."
          right={
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1 bg-ink/60 p-1 rounded-xl border border-line" role="group" aria-label="Lab view">
                <button
                  onClick={() => { soundManager.playClick(); setViewMode('deepdive'); }}
                  aria-pressed={viewMode === 'deepdive'}
                  className={cx('px-3 py-1.5 rounded-lg text-[13px] font-semibold transition-colors flex items-center gap-1.5', viewMode === 'deepdive' ? 'bg-brass text-ink' : 'text-sage hover:text-bone')}
                >
                  <Eye size={13} /> Deep dive
                </button>
                <button
                  onClick={() => { soundManager.playClick(); setViewMode('comparison'); }}
                  aria-pressed={viewMode === 'comparison'}
                  className={cx('px-3 py-1.5 rounded-lg text-[13px] font-semibold transition-colors flex items-center gap-1.5', viewMode === 'comparison' ? 'bg-brass text-ink' : 'text-sage hover:text-bone')}
                >
                  <LayoutGrid size={13} /> Compare all
                </button>
              </div>
              <Badge tone="gold" className="flex items-center gap-1.5 px-3 py-2">
                <Zap size={13} /> {energy} / {maxEnergy} sessions
              </Badge>
            </div>
          }
        />
      </Card>

      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 lg:grid-cols-12 gap-2" role="listbox" aria-label="Prodigy selector">
        {prodigies.map((p) => {
          const isSel = selected?.player_id === p.player_id;
          return (
            <button
              key={p.player_id}
              onClick={() => { soundManager.playClick(); setSelectedId(p.player_id); }}
              aria-selected={isSel}
              className={cx(
                'p-2.5 rounded-xl border text-center transition-colors flex flex-col items-center justify-between',
                isSel ? 'bg-brass/10 border-brass/55' : 'bg-cardBg border-line hover:border-sage/50',
              )}
            >
              <div className="w-9 h-9 rounded-lg border border-bone/25 bg-bone overflow-hidden mb-1.5 shrink-0">
                {getClubCrestUrlByShort(p.club_short) ? (
                  <img src={getClubCrestUrlByShort(p.club_short)!} alt={`${p.club_short} crest`} loading="lazy" draggable={false} className="w-full h-full object-contain p-[10%]" />
                ) : (
                  <span className="w-full h-full flex items-center justify-center text-[10px] font-bold font-mono text-ink">{p.club_short}</span>
                )}
              </div>
              <div className="font-semibold text-[12px] text-bone truncate w-full">{p.full_name.split(' ')[0]}</div>
              <div className="flex items-center gap-1.5 mt-0.5">
                <span className="text-[11px] font-mono font-bold text-brass">{p.ovr}</span>
                <span className="text-[10px] text-sage font-mono">{p.position}</span>
              </div>
            </button>
          );
        })}
      </div>

      {viewMode === 'deepdive' && selected && (
        <div className="space-y-5">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
          <Card className="lg:col-span-5 space-y-5">
            <div className="flex items-center justify-between border-b border-line pb-4 gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <Badge tone="gold"><span className="flex items-center gap-1"><Sparkle size={10} /> U-14 prodigy</span></Badge>
                  <Badge tone="slate">
                    {selected.position}
                    {selected.secondary_position ? ` / ${selected.secondary_position}` : ''}
                  </Badge>
                  <span className="text-[12px] text-sage font-medium">
                    Age {selected.age} · {selected.height_display || formatHeight(selected.current_height_cm)} · {selected.puberty_stage}
                  </span>
                </div>
                <h3 className="font-display text-[26px] font-semibold text-bone mt-1.5 truncate">
                  <button type="button" onClick={() => openPlayer(selected.player_id)} className="hover:text-brass text-left">
                    {selected.full_name}
                  </button>
                </h3>
                <div className="text-[13px] text-sage font-normal mt-0.5">
                  {selected.club_name} · <strong className="text-bone/85 font-mono font-semibold">{selected.formatted_value}</strong>
                </div>
                <div className="grid grid-cols-2 gap-2 mt-3">
                  <div className="px-3 py-2 rounded-lg bg-brass/[0.08] border border-brass/40">
                    <p className="eyebrow !text-brass">All-time</p>
                    <p className="font-mono font-bold text-[15px] text-bone mt-0.5">{selected.career_goals} G · {selected.career_assists} A</p>
                  </div>
                  <div className="px-3 py-2 rounded-lg bg-ink/40 border border-line">
                    <p className="eyebrow">Best season</p>
                    <p className="font-mono font-bold text-[15px] text-bone mt-0.5">
                      {selected.best_season ? `${selected.best_goals} G · ${selected.best_season}` : 'Yet to come'}
                    </p>
                  </div>
                </div>
              </div>
              <div className="text-right shrink-0">
                <div className="score-display text-[34px] text-bone">{selected.ovr}</div>
                <div className="text-[11px] font-semibold font-mono text-brass uppercase tracking-[0.1em]">Pot {selected.potential}</div>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex justify-between font-mono text-[12px] font-semibold">
                <span className="text-sage flex items-center gap-1.5"><TrendingUp size={14} /> Skill XP</span>
                <span className="text-bone/80">{selected.accumulated_xp} / {selected.level_xp_target} ({selected.xp_pct}%)</span>
              </div>
              <ProgressBar pct={selected.xp_pct} toneClass="bg-brass" className="h-1.5" />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="bg-ink/40 border border-line rounded-xl p-4">
                <div className="eyebrow">Height</div>
                <div className="score-display text-[22px] text-bone mt-1 leading-tight">
                  {selected.height_display || formatHeight(selected.current_height_cm)}
                </div>
                <div className="mt-1.5 font-mono text-[12px]">
                  <span className="text-brass font-semibold">+{selected.height_gain_cm} cm</span>
                  <span className="text-sage"> from {formatHeight(selected.baseline_height_cm)}</span>
                </div>
                <div className="text-[12px] text-sage mt-1">
                  {selected.still_growing === false
                    ? `Adult height at ${selected.adult_height_age ?? selected.age}. Attributes still climb toward ${selected.potential} pot.`
                    : `Taller until about ${selected.adult_height_age ?? 19}. Attributes keep rising after that.`}
                </div>
              </div>
              <div className="bg-ink/40 border border-line rounded-xl p-4">
                <div className="eyebrow">Frame</div>
                <div className="score-display text-[24px] text-bone mt-1">{selected.current_weight_kg} <span className="text-[13px] text-sage font-sans">kg</span></div>
                <div className="mt-1.5 font-mono text-[12px]">
                  <span className="text-[#A9CDBB] font-semibold">+{selected.weight_gain_kg} kg</span>
                  <span className="text-sage"> · base {selected.baseline_weight_kg}</span>
                </div>
                <div className="text-[12px] text-sage mt-1">Lifts shielding and stamina.</div>
              </div>
            </div>

            <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-2.5">
              <p className="eyebrow">School</p>
              <p className="text-[13px] text-bone leading-relaxed">{selected.education_label || '—'}</p>
              {selected.age < 16 && (
                <p className="text-[12px] text-sage">
                  {selected.school_want_label || 'He has a preference.'} At 16 he chooses for himself.
                </p>
              )}
              <div>
                <p className="text-[11px] font-mono uppercase tracking-[0.1em] text-sage mb-1.5">School track</p>
                <div className="flex flex-wrap gap-1.5" role="radiogroup" aria-label="School track">
                  {([
                    ['stay', 'Stay-in-school'],
                    ['football_first', 'Football-first'],
                    ['club_forced', 'Club-forced'],
                  ] as const).map(([value, label]) => {
                    const active = (selected.school_track || 'stay') === value;
                    return (
                      <button
                        key={value}
                        type="button"
                        role="radio"
                        aria-checked={active}
                        onClick={async () => {
                          if (active) return;
                          soundManager.playClick();
                          const res = await setProdigySchoolTrack(selected.player_id, value);
                          onShowToast(stripEmojis(res.status === 'success' ? `School track: ${label}.` : (res.message || 'Could not set school track.')));
                          void loadData();
                        }}
                        className={cx(
                          'px-3 py-1.5 rounded-lg text-[12px] font-semibold border transition-colors',
                          active ? 'bg-brass text-ink border-brass' : 'bg-cardBg text-sage hover:text-bone border-line',
                        )}
                      >
                        {label}
                      </button>
                    );
                  })}
                </div>
                <p className="text-[12px] text-sage mt-1.5">{selected.school_track_label || 'Stay-in-school · sits exam weeks'}</p>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div className="bg-ink/40 border border-brass/35 rounded-xl p-4 space-y-1.5">
                <div className="flex items-center justify-between">
                  <p className="eyebrow !text-brass">Autonomous Mindset</p>
                  <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded bg-brass/15 text-brass">
                    {selected.personality_badge || 'Dedicated Pro'}
                  </span>
                </div>
                <p className="text-[13px] font-semibold text-bone">{selected.personality_title || 'The Dedicated Professional'}</p>
                <p className="text-[12px] text-sage leading-relaxed">
                  {selected.personality_desc || 'Relentless work ethic, daily deliberate practice, and uncompromising standards.'}
                </p>
              </div>

              <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-1.5">
                <p className="eyebrow">Senior Mentorship</p>
                {selected.mentor_name ? (
                  <>
                    <p className="text-[13px] font-semibold text-bone flex items-center justify-between">
                      <span>{selected.mentor_name}</span>
                      <span className="font-mono text-xs text-brass font-bold">{selected.mentor_ovr} OVR</span>
                    </p>
                    <p className="text-[12px] text-sage leading-relaxed">
                      Shares dressing room leadership and tactical wisdom, imparting natural growth buffs to his apprentice.
                    </p>
                    <div className="flex flex-wrap gap-1.5 pt-1">
                      <span className="inline-flex items-center text-[11px] font-mono px-2 py-0.5 rounded-md bg-brass/15 text-brass border border-brass/30 font-semibold">
                        +{Math.min(25, Math.max(5, Math.round(((selected.mentor_ovr || 75) - 70) * 1)))}% Match XP Buff
                      </span>
                      <span className="inline-flex items-center text-[11px] font-mono px-2 py-0.5 rounded-md bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 font-semibold">
                        Composure Drills
                      </span>
                    </div>
                  </>
                ) : (
                  <p className="text-[12px] text-sage">Paired with club senior leaders during first-team matchdays.</p>
                )}
              </div>
            </div>

            <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-2.5">
              <p className="eyebrow">Position path</p>
              {selected.secondary_position ? (
                <p className="text-[13px] text-bone">Learned {selected.secondary_position}. Still listed as {selected.position}.</p>
              ) : selected.still_growing !== false ? (
                <p className="text-[13px] text-sage">A second position opens after adult height ({selected.adult_height_age ?? 19}).</p>
              ) : (
                <>
                  {selected.position_path ? (
                    <div>
                      <p className="text-[13px] text-bone">Learning {selected.position_path}</p>
                      <ProgressBar pct={selected.position_xp ?? 0} toneClass="bg-brass" className="h-1.5 mt-2" />
                      <p className="font-mono text-[11px] text-sage mt-1">{selected.position_xp ?? 0} / 100</p>
                    </div>
                  ) : (
                    <div className="flex flex-wrap gap-2">
                      {(selected.position_options ?? []).map((pos) => (
                        <button
                          key={pos}
                          type="button"
                          onClick={async () => {
                            soundManager.playClick();
                            const res = await setProdigyPositionPath(selected.player_id, pos);
                            onShowToast(stripEmojis(res.message || `Learning ${pos}.`));
                            void loadData();
                          }}
                          className="px-3 py-2 text-[12px] font-semibold border border-line bg-cardLight hover:bg-cardHover text-bone"
                        >
                          Learn {pos}
                        </button>
                      ))}
                      {(selected.position_options ?? []).length === 0 && (
                        <p className="text-[13px] text-sage">No path from this position.</p>
                      )}
                    </div>
                  )}
                </>
              )}
            </div>

            <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-2.5">
              <div className="eyebrow">Derived attributes</div>
              <div className="grid grid-cols-2 gap-2 font-mono text-[12px]">
                {(
                  [
                    ['Aerial reach', selected.attributes.aerial_reach],
                    ['Heading', selected.attributes.heading_power],
                    ['Strength', selected.attributes.strength],
                    ['Shielding', selected.attributes.shielding],
                    ['Press resistance', selected.attributes.press_resistance],
                    ['Stamina', selected.attributes.stamina],
                    ['Composure', selected.attributes.composure],
                  ] as const
                ).map(([label, val]) => (
                  <div key={label} className="flex justify-between px-2.5 py-2 bg-cardBg rounded-lg border border-line">
                    <span className="text-sage">{label}</span>
                    <span className="font-bold text-bone">{val}</span>
                  </div>
                ))}
              </div>
            </div>
          </Card>

          <Card className="lg:col-span-4 flex flex-col justify-between">
            <ProdigyRadar attributes={selected.attributes} ovr={selected.ovr} />
            <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-2.5 mt-4">
              <div className="flex items-center justify-between">
                <div className="text-[13px] font-semibold text-bone flex items-center gap-1.5">
                  <ClipboardCheck size={15} className="text-brass" /><span>Managed by club staff</span>
                </div>
                <span className="text-[12px] font-mono text-sage font-semibold">{energy}/{maxEnergy} sessions a week</span>
              </div>
              <p className="text-[13px] text-sage leading-relaxed">
                Club staff run occasional sessions, rotating hypertrophy, technical, and tactical work.
                Minutes in matches do most of the developing. They stop getting taller at their adult
                height age, then keep improving toward potential.
              </p>
            </div>
          </Card>

          <Card className="lg:col-span-3 flex flex-col h-[650px]">
            <div className="flex items-center justify-between pb-3 border-b border-line mb-3">
              <h4 className="font-display text-[16px] font-semibold text-bone flex items-center gap-1.5">
                <Award size={15} className="text-brass" /><span>Milestones</span>
              </h4>
              <Badge tone="slate">Live</Badge>
            </div>
            <div className="flex-1 overflow-y-auto space-y-2.5 pr-1">
              {milestones.length === 0 && <EmptyState message="Milestones appear as players develop in matchweeks and training." />}
              {milestones.map((m, idx) => (
                <div key={`${m.timestamp}-${idx}`} className={cx('p-3 rounded-xl border text-[13px] leading-relaxed', milestoneTone(m.badge_color))}>
                  <div className="flex justify-between items-center font-mono text-[11px] font-semibold mb-1 opacity-80 gap-2">
                    <span className="truncate">{m.player_name}</span>
                    <span className="shrink-0">{m.timestamp}</span>
                  </div>
                  <div>{stripEmojis(m.description)}</div>
                </div>
              ))}
            </div>
          </Card>
          </div>

          <Card className="space-y-6">
            <PanelHeader
              kicker="Dynasty Progression · Biometrics & Milestones"
              title="Career Growth Timeline"
              subtitle={`Tracking ${selected.full_name}'s physiological development, seasonal ratings curve, and milestone achievements.`}
              right={
                <div className="flex items-center gap-2">
                  <Badge tone="gold">Age {selected.age}</Badge>
                  <Badge tone="slate">OVR {selected.ovr} / Pot {selected.potential}</Badge>
                </div>
              }
            />

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Ruler size={16} className="text-brass" />
                    <span className="font-semibold text-[13px] text-bone">Height Trajectory</span>
                  </div>
                  <span className="text-[12px] font-mono text-brass font-semibold">
                    +{timelineData?.height_gain_cm ?? selected.height_gain_cm} cm gained
                  </span>
                </div>

                <div className="grid grid-cols-3 gap-2 text-center text-[12px] font-mono">
                  <div className="p-2 rounded bg-cardBg border border-line">
                    <div className="text-[10px] text-sage uppercase">Starting (U-14)</div>
                    <div className="font-bold text-bone mt-0.5">{timelineData?.baseline_height_cm ?? selected.baseline_height_cm} cm</div>
                  </div>
                  <div className="p-2 rounded bg-brass/10 border border-brass/40">
                    <div className="text-[10px] text-brass uppercase">Current (Age {selected.age})</div>
                    <div className="font-bold text-brass mt-0.5">{timelineData?.current_height_cm ?? selected.current_height_cm} cm</div>
                  </div>
                  <div className="p-2 rounded bg-cardBg border border-line">
                    <div className="text-[10px] text-sage uppercase">Adult Peak</div>
                    <div className="font-bold text-bone mt-0.5">
                      {selected.still_growing === false
                        ? `${timelineData?.current_height_cm ?? selected.current_height_cm} cm`
                        : `~${Math.max(timelineData?.current_height_cm ?? selected.current_height_cm, (timelineData?.baseline_height_cm ?? selected.baseline_height_cm) + 16)} cm`}
                    </div>
                  </div>
                </div>

                <div className="space-y-1">
                  <div className="flex justify-between text-[11px] font-mono text-sage">
                    <span>Maturity progress</span>
                    <span>
                      {Math.min(
                        100,
                        Math.round(
                          (((timelineData?.current_height_cm ?? selected.current_height_cm) - (timelineData?.baseline_height_cm ?? selected.baseline_height_cm)) /
                            Math.max(1, ((timelineData?.baseline_height_cm ?? selected.baseline_height_cm) + 16) - (timelineData?.baseline_height_cm ?? selected.baseline_height_cm))) *
                            100,
                        ),
                      )}%
                    </span>
                  </div>
                  <ProgressBar
                    pct={Math.min(
                      100,
                      Math.max(
                        10,
                        Math.round(
                          (((timelineData?.current_height_cm ?? selected.current_height_cm) - (timelineData?.baseline_height_cm ?? selected.baseline_height_cm)) /
                            Math.max(1, ((timelineData?.baseline_height_cm ?? selected.baseline_height_cm) + 16) - (timelineData?.baseline_height_cm ?? selected.baseline_height_cm))) *
                            100,
                        ),
                      ),
                    )}
                    toneClass="bg-brass"
                    className="h-2"
                  />
                </div>
              </div>

              <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Scale size={16} className="text-[#A9CDBB]" />
                    <span className="font-semibold text-[13px] text-bone">Frame & Mass Index</span>
                  </div>
                  <span className="text-[12px] font-mono text-[#A9CDBB] font-semibold">
                    +{timelineData?.weight_gain_kg ?? selected.weight_gain_kg} kg muscle/frame
                  </span>
                </div>

                <div className="grid grid-cols-3 gap-2 text-center text-[12px] font-mono">
                  <div className="p-2 rounded bg-cardBg border border-line">
                    <div className="text-[10px] text-sage uppercase">Base (Age 14)</div>
                    <div className="font-bold text-bone mt-0.5">{timelineData?.baseline_weight_kg ?? selected.baseline_weight_kg} kg</div>
                  </div>
                  <div className="p-2 rounded bg-pitchtone/15 border border-pitchtone/40">
                    <div className="text-[10px] text-[#A9CDBB] uppercase">Current (Age {selected.age})</div>
                    <div className="font-bold text-[#A9CDBB] mt-0.5">{timelineData?.current_weight_kg ?? selected.current_weight_kg} kg</div>
                  </div>
                  <div className="p-2 rounded bg-cardBg border border-line">
                    <div className="text-[10px] text-sage uppercase">Peak Target</div>
                    <div className="font-bold text-bone mt-0.5">
                      ~{Math.max(timelineData?.current_weight_kg ?? selected.current_weight_kg, (timelineData?.baseline_weight_kg ?? selected.baseline_weight_kg) + 18)} kg
                    </div>
                  </div>
                </div>

                <div className="space-y-1">
                  <div className="flex justify-between text-[11px] font-mono text-sage">
                    <span>Hypertrophy progress</span>
                    <span>
                      {Math.min(
                        100,
                        Math.round(
                          (((timelineData?.current_weight_kg ?? selected.current_weight_kg) - (timelineData?.baseline_weight_kg ?? selected.baseline_weight_kg)) /
                            Math.max(1, 18)) *
                            100,
                        ),
                      )}%
                    </span>
                  </div>
                  <ProgressBar
                    pct={Math.min(
                      100,
                      Math.max(
                        10,
                        Math.round(
                          (((timelineData?.current_weight_kg ?? selected.current_weight_kg) - (timelineData?.baseline_weight_kg ?? selected.baseline_weight_kg)) /
                            Math.max(1, 18)) *
                            100,
                        ),
                      ),
                    )}
                    toneClass="bg-pitchtone"
                    className="h-2"
                  />
                </div>
              </div>
            </div>

            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="eyebrow flex items-center gap-1.5">
                  <Calendar size={13} className="text-brass" /> Seasonal Evolution & OVR Trajectory
                </div>
                <span className="text-[11px] font-mono text-sage">Recorded at season rollover</span>
              </div>

              {loadingTimeline ? (
                <div className="py-8 text-center text-sage text-[13px] font-mono">Loading timeline history...</div>
              ) : (
                <div className="flex items-stretch gap-3 overflow-x-auto pb-2 scrollbar-thin">
                  {(timelineData?.progression_history && timelineData.progression_history.length > 0
                    ? timelineData.progression_history
                    : selected.progression_history && selected.progression_history.length > 0
                      ? selected.progression_history
                      : [
                          {
                            season: '2026-27',
                            age: selected.age,
                            ovr: selected.ovr,
                            height_cm: selected.current_height_cm,
                            weight_kg: selected.current_weight_kg,
                            goals: selected.goals,
                            assists: selected.assists,
                            appearances: selected.appearances,
                            club_short: selected.club_short,
                            mentor_name: selected.mentor_name || 'Senior Leader',
                          },
                        ]
                  ).map((entry, idx, arr) => (
                    <React.Fragment key={`${entry.season}-${entry.age}-${idx}`}>
                      <div className="min-w-[210px] p-3.5 rounded-xl border border-line bg-ink/40 flex flex-col justify-between space-y-3 shrink-0 hover:border-brass/40 transition-colors">
                        <div className="flex items-center justify-between">
                          <span className="font-mono font-bold text-[13px] text-brass">{entry.season}</span>
                          <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded bg-cardLight text-sage">
                            Age {entry.age}
                          </span>
                        </div>

                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-1.5">
                            <span className="w-5 h-5 rounded-full border border-bone/20 bg-bone overflow-hidden inline-flex items-center justify-center shrink-0">
                              {getClubCrestUrlByShort(entry.club_short) ? (
                                <img src={getClubCrestUrlByShort(entry.club_short)!} alt="" className="w-full h-full object-contain p-[1px]" />
                              ) : (
                                <span className="text-[9px] font-bold text-ink">{entry.club_short}</span>
                              )}
                            </span>
                            <span className="font-mono text-[12px] font-semibold text-bone">{entry.club_short}</span>
                          </div>
                          <div className="flex items-center gap-1">
                            <span className="text-[10px] text-sage font-mono uppercase">OVR</span>
                            <span className="font-mono font-bold text-[16px] text-bone">{entry.ovr}</span>
                          </div>
                        </div>

                        <div className="p-2 rounded-lg bg-cardBg border border-line text-[11px] font-mono space-y-1">
                          <div className="flex justify-between text-sage">
                            <span>Form</span>
                            <span className="text-bone font-semibold">{entry.goals}G · {entry.assists}A ({entry.appearances} apps)</span>
                          </div>
                          <div className="flex justify-between text-sage">
                            <span>Physique</span>
                            <span className="text-bone">{formatHeight(entry.height_cm)} · {entry.weight_kg}kg</span>
                          </div>
                        </div>

                        {entry.mentor_name ? (
                          <div className="text-[11px] text-sage truncate">
                            <span className="text-brass/80 font-medium">Mentor:</span> {entry.mentor_name}
                          </div>
                        ) : (
                          <div className="text-[11px] text-sage/60 italic">Academy scholar</div>
                        )}
                      </div>

                      {idx < arr.length - 1 && (
                        <div className="flex items-center justify-center text-sage/40 shrink-0">
                          <ChevronRight size={18} />
                        </div>
                      )}
                    </React.Fragment>
                  ))}
                </div>
              )}
            </div>

            <div className="space-y-3 pt-2 border-t border-line">
              <div className="flex items-center justify-between">
                <div className="eyebrow flex items-center gap-1.5">
                  <Trophy size={13} className="text-brass" /> Career Milestone Achievements
                </div>
                <span className="text-[11px] font-mono text-sage">
                  {(timelineData?.milestones ?? []).filter((m) => m.unlocked).length} / {(timelineData?.milestones ?? []).length || 8} Unlocked
                </span>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                {(timelineData?.milestones && timelineData.milestones.length > 0
                  ? timelineData.milestones
                  : [
                      {
                        id: 'first_contract',
                        title: 'First Professional Contract',
                        description: 'Signed official youth forms with senior team at age 14.',
                        unlocked: true,
                        badge: 'gold',
                      },
                      {
                        id: 'growth_spurt',
                        title: 'Puberty Growth Spurt',
                        description: `Grew +${selected.height_gain_cm} cm and added frame.`,
                        unlocked: selected.height_gain_cm >= 1.0,
                        badge: 'cyan',
                      },
                      {
                        id: 'senior_debut',
                        title: 'Senior Debut',
                        description: 'Earned first team match minutes in the European Super League.',
                        unlocked: selected.career_goals > 0 || selected.appearances > 0,
                        badge: 'green',
                      },
                      {
                        id: 'first_goal',
                        title: 'First European Goal',
                        description: 'Slotted the ball home against elite continental competition.',
                        unlocked: (selected.career_goals + selected.goals) >= 1,
                        badge: 'gold',
                      },
                      {
                        id: 'youth_cap',
                        title: 'Youth International Cap',
                        description: 'Earned national youth honors after surpassing 78 OVR threshold.',
                        unlocked: selected.ovr >= 78,
                        badge: 'purple',
                      },
                      {
                        id: 'elite_playmaker',
                        title: 'Elite Playmaker',
                        description: 'Created 5+ career assists orchestrating offensive attacks.',
                        unlocked: (selected.career_assists + selected.assists) >= 5,
                        badge: 'cyan',
                      },
                      {
                        id: 'century_mark',
                        title: 'Trophy & Goal Contender',
                        description: 'Reached 15+ combined career goals and assists.',
                        unlocked: (selected.career_goals + selected.goals + selected.career_assists + selected.assists) >= 15,
                        badge: 'gold',
                      },
                      {
                        id: 'ballon_dor_contender',
                        title: 'Ballon d\'Or Candidate',
                        description: 'Surpassed 85 OVR ascending to world-class elite status.',
                        unlocked: selected.ovr >= 85,
                        badge: 'gold',
                      },
                    ]
                ).map((m) => {
                  const unlocked = m.unlocked;
                  return (
                    <div
                      key={m.id}
                      className={cx(
                        'p-3.5 rounded-xl border flex flex-col justify-between transition-colors',
                        unlocked
                          ? milestoneTone(m.badge || 'gold')
                          : 'border-line/60 bg-cardBg/40 opacity-60 text-sage',
                      )}
                    >
                      <div className="space-y-1.5">
                        <div className="flex items-center justify-between">
                          <span className="text-[10px] font-mono uppercase tracking-wider font-semibold">
                            {unlocked ? 'Achievement' : 'Locked'}
                          </span>
                          {unlocked ? (
                            <CheckCircle2 size={15} className="text-brass shrink-0" />
                          ) : (
                            <Lock size={13} className="text-sage/60 shrink-0" />
                          )}
                        </div>
                        <div className="font-semibold text-[13px] text-bone leading-snug">{m.title}</div>
                        <div className="text-[12px] text-sage leading-relaxed">{m.description}</div>
                      </div>
                      <div className="mt-3 pt-2 border-t border-line/40 flex items-center justify-between text-[11px] font-mono">
                        <span className="text-sage">Status:</span>
                        <span className={unlocked ? 'text-brass font-bold' : 'text-sage/60'}>
                          {unlocked ? 'Unlocked' : 'In Progress'}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </Card>
        </div>
      )}

      {viewMode === 'comparison' && (
        <div className="panel-tight overflow-hidden">
          <div className="px-4 py-3.5 bg-cardLight/60 border-b border-line flex items-center justify-between flex-wrap gap-2">
            <h3 className="font-display text-[16px] font-semibold text-bone">All 12 prodigies, compared</h3>
            <span className="text-[12px] font-mono text-sage">Age 14 · Attackers only</span>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-[13px]">
              <thead className="table-head">
                <tr>
                  {['Prodigy', 'Club', 'Pos', 'Age', 'OVR', 'Pot', 'Height', 'Frame', 'Value', 'Goals', 'Assists', 'All-time', 'Best'].map((h) => (
                    <th key={h} className="py-3 px-3">{h}</th>
                  ))}
                  <th className="py-3 px-4 text-center">Open</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line/70">
                {prodigies.map((p) => (
                  <tr key={p.player_id} className="hover:bg-cardLight/60 transition-colors">
                    <td className="py-3 px-3 font-semibold text-bone">
                      <span className="flex items-center gap-2">
                        <span className="w-6 h-6 rounded-full border border-bone/25 bg-bone overflow-hidden shrink-0 inline-block">
                          {getClubCrestUrlByShort(p.club_short) ? (
                            <img src={getClubCrestUrlByShort(p.club_short)!} alt="" aria-hidden loading="lazy" draggable={false} className="w-full h-full object-contain p-[2px]" />
                          ) : null}
                        </span>
                        <span>{p.full_name}</span>
                      </span>
                    </td>
                    <td className="py-3 px-3 font-mono text-[#A9CBDD] font-semibold">{p.club_short}</td>
                    <td className="py-3 px-3 font-mono text-bone/75 font-semibold">{p.position}</td>
                    <td className="py-3 px-3 font-mono text-sage">{p.age}</td>
                    <td className="py-3 px-3 font-bold font-mono text-[14px] text-brass">{p.ovr}</td>
                    <td className="py-3 px-3 font-mono text-bone/75 font-semibold">{p.potential}</td>
                    <td className="py-3 px-3 font-mono text-[12px]"><span className="text-bone">{p.height_display || formatHeight(p.current_height_cm)}</span> <span className="text-brass font-semibold">(+{p.height_gain_cm})</span></td>
                    <td className="py-3 px-3 font-mono text-[12px]"><span className="text-bone">{p.current_weight_kg}kg</span> <span className="text-[#A9CDBB] font-semibold">(+{p.weight_gain_kg})</span></td>
                    <td className="py-3 px-3 font-mono text-[#A9CDBB] font-semibold">{p.formatted_value}</td>
                    <td className="py-3 px-3 text-bone font-semibold font-mono">{p.goals}</td>
                    <td className="py-3 px-3 text-sage font-mono">{p.assists}</td>
                    <td className="py-3 px-3 font-mono text-[12px] whitespace-nowrap"><span className="text-bone font-semibold">{p.career_goals} G</span> <span className="text-sage">· {p.career_assists} A</span></td>
                    <td className="py-3 px-3 font-mono text-[12px] whitespace-nowrap">
                      {p.best_season ? (
                        <span><span className="text-brass font-semibold">{p.best_goals} G</span> <span className="text-sage">({p.best_season})</span></span>
                      ) : (
                        <span className="text-sage/60">—</span>
                      )}
                    </td>
                    <td className="py-3 px-4 text-center">
                      <button
                        onClick={() => { soundManager.playClick(); setSelectedId(p.player_id); setViewMode('deepdive'); }}
                        className="px-2.5 py-1.5 bg-cardLight hover:bg-cardHover text-bone border border-line rounded-lg text-[12px] font-semibold transition-colors inline-flex items-center gap-1"
                      >
                        <Microscope size={12} /> Inspect
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <p className="flex items-center gap-2 text-[12px] text-sage font-mono">
        <Dna size={13} /> Growth compounds: minutes, training cycles, and milestones all feed potential.
      </p>
    </div>
  );
};
