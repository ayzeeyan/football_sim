export function milestoneTone(badge: string): string {
  if (badge === 'gold') return 'border-brass/45 bg-brass/[0.07] text-bone';
  if (badge === 'cyan') return 'border-[#8AB4C8]/35 bg-[#8AB4C8]/[0.07] text-bone';
  if (badge === 'green') return 'border-pitchtone/35 bg-pitchtone/[0.08] text-bone';
  return 'border-[#8E86C8]/35 bg-[#8E86C8]/[0.08] text-bone';
}
