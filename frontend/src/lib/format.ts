/** Shared string / class / number helpers. */

export function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(' ');
}

export function rgb(color: [number, number, number] | number[] | undefined, fallback = '59,130,246'): string {
  if (!color || color.length < 3) return fallback;
  return `${color[0]},${color[1]},${color[2]}`;
}

export function rgbCss(color: [number, number, number] | number[] | undefined, fallback = '#3b82f6'): string {
  if (!color || color.length < 3) return fallback;
  return `rgb(${color[0]},${color[1]},${color[2]})`;
}

/** White or black text depending on background luminance. */
export function contrastText(color: [number, number, number] | number[] | undefined): string {
  if (!color || color.length < 3) return '#fff';
  const sum = color[0] + color[1] + color[2];
  return sum > 400 ? '#000' : '#fff';
}

export function formatMillions(valueEur: number): string {
  return `€${(valueEur / 1_000_000).toFixed(1)}M`;
}

export function formatGd(gd: number): string {
  return gd > 0 ? `+${gd}` : `${gd}`;
}

/** 178 cm (5'10") */
export function formatHeight(cm: number): string {
  const totalIn = cm / 2.54;
  let feet = Math.floor(totalIn / 12);
  let inches = Math.round(totalIn % 12);
  if (inches === 12) {
    feet += 1;
    inches = 0;
  }
  const metric = Math.abs(cm - Math.round(cm)) < 0.05 ? `${Math.round(cm)}` : cm.toFixed(1);
  return `${metric} cm (${feet}'${inches}")`;
}

export function loyaltyLabel(loyalty: number): string {
  if (loyalty >= 85) return 'one-club servant';
  if (loyalty >= 70) return 'settled';
  if (loyalty >= 55) return 'content';
  if (loyalty >= 40) return 'restless';
  return 'angling for a move';
}

/** Strips emoji/pictograph ranges from server-fed text so the UI stays word-led. */
export function stripEmojis(text: string): string {
  return text
    .replace(/[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{FE0F}\u{200D}]/gu, '')
    .replace(/\s{2,}/g, ' ')
    .trim();
}
