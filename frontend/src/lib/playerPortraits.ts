import type { Player } from '../types';

/**
 * Known photographic portrait asset mappings.
 * When real assets are added to the repository or CDN, map player_id -> URL here.
 * Any unmapped player will use an honest, deterministic player-ID-based fallback.
 */
export const PLAYER_PORTRAIT_URLS: Record<string, string> = {};

/**
 * Returns the asset URL for a player portrait if one exists.
 * Returns null if no real asset exists—never returns fake metadata or randomized photos.
 */
export function getPlayerPortraitUrl(
  playerOrId: Pick<Player, 'player_id'> | string | null | undefined
): string | null {
  if (!playerOrId) return null;
  const id = typeof playerOrId === 'string' ? playerOrId : playerOrId.player_id;
  if (!id) return null;
  return PLAYER_PORTRAIT_URLS[id] ?? null;
}

/**
 * Deterministic palette of dark sports tones with high-contrast text for player initials fallback.
 */
export const FALLBACK_PALETTES = [
  { bg: '#1E293B', text: '#E2E8F0', border: '#334155' }, // Slate
  { bg: '#14291F', text: '#A9CDBB', border: '#2D4A3E' }, // Deep Pitch Green
  { bg: '#2A1F1D', text: '#E5C2B2', border: '#4D3630' }, // Warm Clay
  { bg: '#1E1E38', text: '#C7C6EB', border: '#373563' }, // Indigo Night
  { bg: '#2C2214', text: '#E5C992', border: '#4E3E26' }, // Brass Amber
  { bg: '#13282C', text: '#A3D3DC', border: '#264B52' }, // Dark Teal
  { bg: '#2D1B28', text: '#E2BACD', border: '#4E3146' }, // Deep Plum
  { bg: '#21261B', text: '#C4D4B5', border: '#3C4731' }, // Forest Olive
] as const;

/**
 * Deterministically hashes a player ID into a stable positive integer.
 */
export function hashPlayerId(playerId: string): number {
  let hash = 0;
  for (let i = 0; i < playerId.length; i++) {
    hash = (hash << 5) - hash + playerId.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash);
}

/**
 * Returns a stable, deterministic color scheme derived purely from the player ID.
 */
export function getPlayerFallbackColors(playerId: string): { bg: string; text: string; border: string } {
  if (!playerId) {
    return FALLBACK_PALETTES[0];
  }
  const idx = hashPlayerId(playerId) % FALLBACK_PALETTES.length;
  return FALLBACK_PALETTES[idx];
}

/**
 * Extracts clean, honest initials from a player's full name.
 * "Venjamin Valerio" -> "VV"
 * "Pedri" -> "PE"
 * "P01412" -> "P"
 */
export function getPlayerInitials(fullName: string | null | undefined): string {
  if (!fullName) return '?';
  const parts = fullName.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}
