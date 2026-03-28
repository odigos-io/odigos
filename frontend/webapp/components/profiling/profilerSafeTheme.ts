import type { DefaultTheme } from 'styled-components';

function parseRgbChannel(input: string): { r: number; g: number; b: number } | null {
  const s = input.trim();
  if (s.startsWith('#')) {
    const h = s.slice(1);
    if (h.length === 6 || (h.length === 8 && /^[0-9a-f]+$/i.test(h))) {
      return {
        r: parseInt(h.slice(0, 2), 16),
        g: parseInt(h.slice(2, 4), 16),
        b: parseInt(h.slice(4, 6), 16),
      };
    }
    if (h.length === 3) {
      return {
        r: parseInt(h[0] + h[0], 16),
        g: parseInt(h[1] + h[1], 16),
        b: parseInt(h[2] + h[2], 16),
      };
    }
  }
  const m = s.match(/^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i);
  if (m) return { r: +m[1], g: +m[2], b: +m[3] };
  return null;
}

function linearChannel(c: number): number {
  const x = c / 255;
  return x <= 0.03928 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4;
}

function relativeLuminance(rgb: { r: number; g: number; b: number }): number {
  return (
    0.2126 * linearChannel(rgb.r) + 0.7152 * linearChannel(rgb.g) + 0.0722 * linearChannel(rgb.b)
  );
}

/** Prefer zustand shell flag when the styled theme is partial (e.g. source drawer mount). */
export function resolveProfilerIsDark(theme: DefaultTheme | undefined, shellIsDark?: boolean): boolean {
  if (typeof shellIsDark === 'boolean') return shellIsDark;
  if (theme?.darkMode === true) return true;
  if (theme?.darkMode === false) return false;
  return true;
}

function pickReadable(
  token: string | undefined,
  isDark: boolean,
  fallbackLight: string,
  fallbackDark: string,
): string {
  if (!token) return isDark ? fallbackDark : fallbackLight;
  const rgb = parseRgbChannel(token);
  if (!rgb) return token;
  const L = relativeLuminance(rgb);
  if (isDark && L < 0.45) return fallbackDark;
  if (!isDark && L > 0.55) return fallbackLight;
  return token;
}

export interface ProfilerPalette {
  isDark: boolean;
  fg: string;
  fgMuted: string;
  fgWhite: string;
  err: string;
  border: string;
  surface: string;
  surfaceRaised: string;
  sticky: string;
  fontBody: string;
  fontCode: string;
  success: string;
  inkOnLightBar: string;
}

/**
 * Odigos theme tokens when present; readable fallbacks when missing or inconsistent
 * (e.g. `text.primary` dark while the shell is dark).
 */
export function profilerSafe(theme: DefaultTheme | undefined, shellIsDark?: boolean): ProfilerPalette {
  const isDark = resolveProfilerIsDark(theme, shellIsDark);
  const fgFallback = isDark ? '#f4f4f5' : '#18181b';
  const mutedFallback = isDark ? '#a1a1aa' : '#52525b';

  return {
    isDark,
    fg: pickReadable(theme?.text?.primary, isDark, '#18181b', '#f4f4f5'),
    fgMuted: pickReadable(theme?.text?.secondary, isDark, '#52525b', '#a1a1aa'),
    fgWhite: theme?.text?.white ?? '#ffffff',
    err: theme?.text?.error ?? '#f87171',
    border: theme?.colors?.border ?? (isDark ? '#3f3f46' : '#d4d4d8'),
    surface:
      theme?.colors?.dropdown_bg_2 ?? theme?.colors?.dropdown_bg ?? (isDark ? '#27272a' : '#fafafa'),
    surfaceRaised: theme?.colors?.dropdown_bg ?? (isDark ? '#3f3f46' : '#ffffff'),
    sticky: theme?.colors?.translucent_bg ?? theme?.colors?.dropdown_bg ?? (isDark ? '#3f3f46' : '#f4f4f5'),
    fontBody: theme?.font_family?.primary ?? 'system-ui, -apple-system, Segoe UI, sans-serif',
    fontCode: theme?.font_family?.code ?? theme?.font_family?.secondary ?? 'ui-monospace, monospace',
    success: theme?.colors?.success ?? '#22c55e',
    inkOnLightBar:
      theme?.text?.darker_grey ??
      theme?.text?.dark_grey ??
      theme?.v2?.colors?.black?.['500'] ??
      '#0f172a',
  };
}
