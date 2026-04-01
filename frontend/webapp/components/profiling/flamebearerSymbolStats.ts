import type { FlamebearerProfile } from '@/types/profiling';

const STEP = 4;
const IDX_TOTAL = 1;
const IDX_SELF = 2;
const IDX_NAME = 3;

export interface SymbolStatsRow {
  symbol: string;
  nameIndex: number;
  self: number;
  total: number;
}

export function buildSymbolStatsRows(profile: FlamebearerProfile | null | undefined): SymbolStatsRow[] {
  const fmt = profile?.metadata?.format || 'single';
  if (!profile?.flamebearer || fmt !== 'single') {
    return [];
  }
  const { levels, names, numTicks } = profile.flamebearer;
  if (!levels?.length || !names?.length || numTicks <= 0) {
    return [];
  }

  const selfSum = new Map<number, number>();
  const totalMax = new Map<number, number>();

  for (const row of levels) {
    for (let t = 0; t + STEP - 1 < row.length; t += STEP) {
      const nameIdx = row[t + IDX_NAME];
      if (nameIdx < 0 || nameIdx >= names.length) continue;
      const self = row[t + IDX_SELF] ?? 0;
      const total = row[t + IDX_TOTAL] ?? 0;
      selfSum.set(nameIdx, (selfSum.get(nameIdx) ?? 0) + self);
      const prev = totalMax.get(nameIdx) ?? 0;
      if (total > prev) totalMax.set(nameIdx, total);
    }
  }

  const rows: SymbolStatsRow[] = [];
  for (const [idx, self] of selfSum) {
    rows.push({
      nameIndex: idx,
      symbol: names[idx] ?? `?(${idx})`,
      self,
      total: totalMax.get(idx) ?? 0,
    });
  }

  rows.sort((a, b) => b.self - a.self);
  return rows;
}

export function formatSampleCount(n: number): string {
  if (!Number.isFinite(n) || n < 0) return '0';
  if (n < 1000) return n.toLocaleString();
  if (n < 1_000_000) return `${(n / 1000).toFixed(1)}K`;
  if (n < 1_000_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  return `${(n / 1_000_000_000).toFixed(2)}B`;
}
