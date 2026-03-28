'use client';

import React, { useMemo } from 'react';
import styled, { useTheme } from 'styled-components';
import type { DefaultTheme } from 'styled-components';
import type { FlamebearerProfile } from '@/types/profiling';
import { labelStyleForBackground } from '@/components/profiling/flamebearerLabelColor';
import { barBackgroundForFrame } from '@/components/profiling/flamebearerBarPalette';
import { profilerSafe } from '@/components/profiling/profilerSafeTheme';

/** Pyroscope single format: 4 ints per bar — offset, total, self, nameIndex */
const J_STEP = 4;
const J_NAME = 3;

const Wrap = styled.div`
  width: 100%;
  min-height: 200px;
  font-size: 11px;
  line-height: 1;
  user-select: none;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
`;

const Row = styled.div`
  position: relative;
  width: 100%;
  height: 20px;
  margin-bottom: 1px;
`;

const Bar = styled.div<{ $left: number; $width: number; $bg: string; $fg: string; $ts: string }>`
  position: absolute;
  left: ${({ $left }) => $left}%;
  width: ${({ $width }) => Math.max($width, 0.05)}%;
  height: 100%;
  top: 0;
  background: ${({ $bg }) => $bg};
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  border-radius: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: ${({ $fg }) => $fg};
  text-shadow: ${({ $ts }) => $ts};
  padding: 2px 4px;
  box-sizing: border-box;
  cursor: default;
  font-family: ${({ theme }) => profilerSafe(theme as DefaultTheme).fontCode};
  &:hover {
    filter: brightness(1.1) saturate(1.05);
    z-index: 1;
  }
`;

const Hint = styled.div`
  margin-bottom: 8px;
  font-size: 0.75rem;
  line-height: 1.3;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fgMuted};
`;

export function FlamebearerIcicle({ profile }: { profile: FlamebearerProfile }) {
  const theme = useTheme() as DefaultTheme;
  const { numTicks, names, levels } = profile.flamebearer;
  const fmt = profile.metadata?.format || 'single';

  const rows = useMemo(() => {
    if (!levels?.length || numTicks <= 0 || fmt !== 'single') {
      return [];
    }
    return levels.map((row, depth) => {
      const bars: { left: number; width: number; label: string; key: string }[] = [];
      for (let t = 0; t + J_STEP - 1 < row.length; t += J_STEP) {
        const offset = row[t];
        const total = row[t + 1];
        const nameIdx = row[t + J_NAME];
        const label = names[nameIdx] ?? `?(${nameIdx})`;
        const left = (offset / numTicks) * 100;
        const width = (total / numTicks) * 100;
        bars.push({
          left,
          width,
          label,
          key: `${depth}-${t}-${nameIdx}`,
        });
      }
      return { depth, bars };
    });
  }, [levels, names, numTicks, fmt]);

  if (fmt !== 'single') {
    return (
      <Wrap>
        <Hint>Flame graph preview supports format &quot;single&quot; only (got {String(fmt)}).</Hint>
      </Wrap>
    );
  }

  if (rows.length === 0) {
    return (
      <Wrap>
        <Hint>No levels to render (numTicks={numTicks}).</Hint>
      </Wrap>
    );
  }

  return (
    <Wrap>
      <Hint>Stack depth (top → bottom). Labels show when the bar is wide enough.</Hint>
      {rows.map(({ depth, bars }) => (
        <Row key={depth}>
          {bars.map((b) => {
            const bg = barBackgroundForFrame(theme, b.label, depth);
            const { color, textShadow } = labelStyleForBackground(bg, theme);
            return (
              <Bar
                key={b.key}
                title={b.label}
                $left={b.left}
                $width={b.width}
                $bg={bg}
                $fg={color}
                $ts={textShadow}
              >
                {b.width > 6 ? b.label : ''}
              </Bar>
            );
          })}
        </Row>
      ))}
    </Wrap>
  );
}
