import type { DefaultTheme } from 'styled-components';

function hashHue(label: string, depth: number): number {
  let h = depth * 131;
  for (let i = 0; i < label.length; i++) {
    h = (h + label.charCodeAt(i) * 17) >>> 0;
  }
  return h % 360;
}

/** Fallback when `theme.v2.colors` is missing (minimal / embedded theme). */
function fallbackHsl(theme: DefaultTheme, label: string, depth: number, dark: boolean): string {
  const hue = hashHue(label, depth);
  return dark ? `hsl(${hue}, 58%, 42%)` : `hsl(${hue}, 48%, 72%)`;
}

/**
 * Flame bar fills: Odigos v2 palette when available; HSL fallbacks otherwise.
 * Labels use `flamebearerLabelColor` (contrast against fill).
 */
export function barBackgroundForFrame(theme: DefaultTheme, label: string, depth: number): string {
  const v2 = theme.v2?.colors;
  const dark = theme.darkMode !== false;

  if (!v2) {
    return fallbackHsl(theme, label, depth, dark);
  }

  const darkPalette = [
    v2.purple?.['500'],
    v2.green?.['500'],
    v2.blue?.['500'],
    v2.yellow?.['600'],
    v2.red?.['500'],
    theme.colors?.majestic_blue,
    theme.colors?.orange_og,
    theme.colors?.dark_green,
    v2.purple?.['400'],
    v2.blue?.['600'],
    v2.green?.['600'],
  ].filter(Boolean) as string[];

  const lightPalette = [
    v2.purple?.['300'],
    v2.green?.['300'],
    v2.blue?.['300'],
    v2.yellow?.['400'],
    v2.red?.['300'],
    theme.colors?.majestic_blue_soft || v2.blue?.['400'],
    theme.colors?.orange_soft || v2.orange?.['600'],
    v2.green?.['400'],
    v2.purple?.['400'],
    v2.blue?.['400'],
    v2.green?.['500'],
    v2.silver?.['400'] || v2.grey?.['400'],
  ].filter(Boolean) as string[];

  const palettes = (dark ? darkPalette : lightPalette).length ? (dark ? darkPalette : lightPalette) : [];

  if (!palettes.length) {
    return fallbackHsl(theme, label, depth, dark);
  }

  let h = depth * 131;
  for (let i = 0; i < label.length; i++) {
    h = (h + label.charCodeAt(i) * 17) >>> 0;
  }
  return palettes[h % palettes.length];
}
