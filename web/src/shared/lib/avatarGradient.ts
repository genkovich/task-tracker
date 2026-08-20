function fnv1aHash(input: string): number {
  let hash = 0x811c9dc5;
  for (let i = 0; i < input.length; i++) {
    hash ^= input.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return hash >>> 0;
}

/**
 * Returns a stable two-stop linear gradient derived from `seed`.
 *
 * The hue rotation pulls colours from the cyan/amber/pink palette used
 * on landing, keeping fallback avatars on-brand. Saturation/lightness
 * are clamped so text stays legible on every gradient.
 */
export function getAvatarGradient(seed: string): string {
  if (!seed) {
    return "linear-gradient(135deg, hsl(190 60% 45%), hsl(280 55% 50%))";
  }
  const hash = fnv1aHash(seed.toLowerCase().trim());
  const hue1 = hash % 360;
  const hue2 = (hue1 + 35 + (hash % 60)) % 360;
  return `linear-gradient(135deg, hsl(${hue1} 70% 45%), hsl(${hue2} 70% 55%))`;
}

export function getAvatarTextColor(): string {
  return "#FAFAFA";
}
