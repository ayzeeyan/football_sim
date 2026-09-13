/** League table qualification bands for the Top Five. Last three go down. */

export type QualBand = 'ucl' | 'el' | 'ecl' | 'rel' | null;

export function qualificationBand(league: string, pos: number, clubs: number): QualBand {
  if (pos < 1 || clubs < 8) return null;
  if (pos >= clubs - 2) return 'rel';
  if (league === 'Ligue 1') {
    if (pos <= 3) return 'ucl';
    if (pos === 4) return 'el';
    if (pos === 5) return 'ecl';
    return null;
  }
  if (pos <= 4) return 'ucl';
  if (pos === 5) return 'el';
  if (pos === 6) return 'ecl';
  return null;
}

export function qualificationBarClass(band: QualBand): string {
  switch (band) {
    case 'ucl':
      return 'bg-brass';
    case 'el':
      return 'bg-[#8AB4C8]';
    case 'ecl':
      return 'bg-pitchtone';
    case 'rel':
      return 'bg-ember';
    default:
      return 'bg-sage/30';
  }
}

export function qualificationLabel(band: QualBand): string {
  switch (band) {
    case 'ucl':
      return 'Champions League';
    case 'el':
      return 'Europa League';
    case 'ecl':
      return 'Conference League';
    case 'rel':
      return 'Relegation';
    default:
      return '';
  }
}
