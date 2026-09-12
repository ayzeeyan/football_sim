import React from 'react';
import { ScrollText, Sparkles } from 'lucide-react';
import type { SeasonAwards } from '../../types';
import { getClubCrestUrlByShort } from '../../lib/clubLogos';
import { Modal, ModalHeader } from '../ui/ui';

interface AwardsModalProps {
  open: boolean;
  data: SeasonAwards | null;
  onClose: () => void;
  onWatchCeremony: () => void;
}

const HonourCrest: React.FC<{ shortName?: string | null; name: string; size?: number }> = ({ shortName, name, size = 52 }) => {
  const url = getClubCrestUrlByShort(shortName);
  if (!url) return null;
  return (
    <img
      src={url}
      alt={`${name} crest`}
      width={size}
      height={size}
      loading="lazy"
      draggable={false}
      className="rounded-xl bg-bone object-contain p-1.5 border border-bone/25 shrink-0"
      style={{ width: size, height: size }}
    />
  );
};

export const AwardsModal: React.FC<AwardsModalProps> = ({ open, data, onClose, onWatchCeremony }) => (
  <Modal open={open} onClose={onClose} maxWidth="max-w-xl">
    <ModalHeader
      title={
        <span className="flex items-center gap-2">
          <ScrollText size={19} className="text-brass" />
          <span>Season honours, {data?.season_name || '2026–27'}</span>
        </span>
      }
      subtitle="Decided across the Super League season and the UCL campaign."
      onClose={onClose}
    />
    <div className="p-6 space-y-3 font-mono text-xs">
      <div className="p-4 bg-cardLight border border-brass/40 rounded-xl flex justify-between items-center gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <HonourCrest shortName={data?.super_league_champion?.short_name} name={data?.super_league_champion?.club_name ?? 'champions'} />
          <div className="min-w-0">
            <div className="text-brass font-semibold text-[11px] uppercase tracking-[0.14em]">Super League champions</div>
            <div className="font-display text-lg font-semibold text-bone mt-1">{data?.super_league_champion?.club_name || 'Season in progress'}</div>
          </div>
        </div>
        <div className="font-display text-2xl font-semibold text-brass shrink-0">{data?.super_league_champion?.pts || 0} pts</div>
      </div>

      <div className="p-4 bg-cardLight border border-line rounded-xl flex justify-between items-center gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <HonourCrest shortName={data?.ucl_champion?.short_name} name={data?.ucl_champion?.club_name ?? 'winners'} size={44} />
          <div className="min-w-0">
            <div className="text-sage font-semibold text-[11px] uppercase tracking-[0.14em]">UCL winners</div>
            <div className="font-display text-lg font-semibold text-bone mt-1">{data?.ucl_champion?.club_name || 'Knockouts in progress'}</div>
          </div>
        </div>
        <div className="text-bone/70 font-semibold text-[13px] shrink-0">Champions</div>
      </div>

      <div className="p-4 bg-cardLight border border-brass/40 rounded-xl flex justify-between items-center gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <HonourCrest shortName={data?.player_of_the_season?.short_name} name={data?.player_of_the_season?.full_name ?? 'player'} />
          <div className="min-w-0">
            <div className="text-brass font-semibold text-[11px] uppercase tracking-[0.14em]">Player of the season · open to all</div>
            <div className="font-display text-lg font-semibold text-bone mt-1">{data?.player_of_the_season?.full_name || 'Season in progress'}</div>
            <div className="text-sage text-[11px] mt-0.5">
              {data?.player_of_the_season ? `${data.player_of_the_season.club_name} · ${data.player_of_the_season.goals} goals · ${data.player_of_the_season.assists} assists` : 'Decided when the season ends'}
            </div>
          </div>
        </div>
        <div className="font-display text-2xl font-semibold text-brass shrink-0">{data?.player_of_the_season?.ovr || 0} <span className="text-sm">OVR</span></div>
      </div>

      <div className="p-4 bg-cardLight border border-line rounded-xl flex justify-between items-center gap-3">
        <div>
          <div className="text-sage font-semibold text-[11px] uppercase tracking-[0.14em]">Golden boy · best U-14</div>
          <div className="font-display text-lg font-semibold text-bone mt-1">{data?.golden_boy?.full_name || 'Under review'}</div>
        </div>
        <div className="text-bone font-semibold text-sm">{data?.golden_boy?.ovr ? `${data.golden_boy.ovr} OVR` : '—'}</div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="p-4 bg-cardLight border border-line rounded-xl">
          <div className="text-sage font-semibold text-[11px] uppercase tracking-[0.14em]">Golden boot</div>
          <div className="text-bone font-semibold text-sm mt-1.5">{data?.top_scorer?.full_name || 'No goals yet'}</div>
          <div className="text-brass font-semibold mt-0.5">{data?.top_scorer?.goals || 0} goals</div>
        </div>
        <div className="p-4 bg-cardLight border border-line rounded-xl">
          <div className="text-sage font-semibold text-[11px] uppercase tracking-[0.14em]">Playmaker award</div>
          <div className="text-bone font-semibold text-sm mt-1.5">{data?.top_assister?.full_name || 'No assists yet'}</div>
          <div className="text-[#A9CBDD] font-semibold mt-0.5">{data?.top_assister?.assists || 0} assists</div>
        </div>
      </div>

      <div className="pt-2 text-center flex items-center justify-center gap-2">
        <button
          onClick={onWatchCeremony}
          className="px-6 py-2 bg-brass hover:bg-[#D4AF4D] text-ink rounded-xl text-[13px] font-bold transition-colors inline-flex items-center gap-1.5"
        >
          <Sparkles size={14} /> Watch the ceremony
        </button>
        <button
          onClick={onClose}
          className="px-6 py-2 bg-cardLight hover:bg-cardHover text-bone border border-line rounded-xl text-[13px] font-semibold transition-colors"
        >
          Close honours
        </button>
      </div>
    </div>
  </Modal>
);
