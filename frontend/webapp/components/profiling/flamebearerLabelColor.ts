/**
 * Label contrast for flame bars: WCAG-style luminance + Odigos `profilerSafe` fallbacks.
 */

import type { DefaultTheme } from 'styled-components';
import { profilerSafe } from '@/components/profiling/profilerSafeTheme';

function parseRgb(input: string): { r: number; g: number; b: number } | null {
  const s = input.trim();
  if (s.startsWith('#')) {
    const hex = s.slice(1);
    if (hex.length === 6) {
      return {
        r: parseInt(hex.slice(0, 2), 16),
        g: parseInt(hex.slice(2, 4), 16),
        b: parseInt(hex.slice(4, 6), 16),
      };
    }
    if (hex.length === 8) {
      return {
        r: parseInt(hex.slice(0, 2), 16),
        g: parseInt(hex.slice(2, 4), 16),
        b: parseInt(hex.slice(4, 6), 16),
      };
    }
    if (hex.length === 3) {
      return {
        r: parseInt(hex[0] + hex[0], 16),
        g: parseInt(hex[1] + hex[1], 16),
        b: parseInt(hex[2] + hex[2], 16),
      };
    }
  }
  const m = s.match(/^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i);
  if (m) {
    return { r: +m[1], g: +m[2], b: +m[3] };
  }
  const hsl = s.match(/^hsla?\(\s*([\d.]+)\s*,\s*([\d.]+)%\s*,\s*([\d.]+)%/i);
  if (hsl) {
    const h = (+hsl[1] / 360) % 1;
    const sat = +hsl[2] / 100;
    const light = +hsl[3] / 100;
    const q = light < 0.5 ? light * (1 + sat) : light + sat - light * sat;
    const p = 2 * light - q;
    const hue2rgb = (t: number) => {
      if (t < 0) t += 1;
      if (t > 1) t -= 1;
      if (t < 1 / 6) return p + (q - p) * 6 * t;
      if (t < 1 / 2) return q;
      if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
      return p;
    };
    const r = Math.round(hue2rgb(h + 1 / 3) * 255);
    const g = Math.round(hue2rgb(h) * 255);
    const b = Math.round(hue2rgb(h - 1 / 3) * 255);
    return { r, g, b };
  }
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

function contrastRatio(fg: { r: number; g: number; b: number }, bg: { r: number; g: number; b: number }): number {
  const L1 = relativeLuminance(fg) + 0.05;
  const L2 = relativeLuminance(bg) + 0.05;
  return L1 > L2 ? L1 / L2 : L2 / L1;
}

export interface LabelStyleForBar {
  color: string;
  textShadow: string;
}

const whiteRgb = { r: 255, g: 255, b: 255 };

export function labelStyleForBackground(bgCss: string, theme: DefaultTheme): LabelStyleForBar {
  const safe = profilerSafe(theme);
  const rgb = parseRgb(bgCss);
  if (!rgb) {
    return {
      color: safe.fgWhite,
      textShadow: '0 1px 2px rgba(0, 0, 0, 0.85)',
    };
  }

  const inkRgb = parseRgb(safe.inkOnLightBar) ?? { r: 15, g: 23, b: 42 };
  const useInk = contrastRatio(inkRgb, rgb) >= contrastRatio(whiteRgb, rgb);

  if (useInk) {
    return {
      color: safe.inkOnLightBar,
      textShadow: safe.isDark
        ? '0 1px 2px rgba(0, 0, 0, 0.35)'
        : '0 1px 1px rgba(255, 255, 255, 0.65)',
    };
  }
  return {
    color: safe.fgWhite,
    textShadow: '0 1px 2px rgba(0, 0, 0, 0.85)',
  };
}
