'use client';

import React from 'react';
import styled from 'styled-components';
import type { DefaultTheme } from 'styled-components';
import { FlexRow } from '@odigos/ui-kit/components';
import { profilerSafe } from '@/components/profiling/profilerSafeTheme';
import type { ProfilerViewMode } from '@/components/profiling/profilerViewMode';

const Group = styled(FlexRow)`
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
`;

const ModeButton = styled.button.attrs({ type: 'button' })<{ $active: boolean }>`
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  font-family: ${({ theme }) => profilerSafe(theme as DefaultTheme).fontBody};
  cursor: pointer;
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  background: ${({ theme, $active }) =>
    $active ? profilerSafe(theme as DefaultTheme).surfaceRaised : 'transparent'};
  transition: background 0.12s ease;
  &:hover {
    background: ${({ theme, $active }) =>
      $active
        ? profilerSafe(theme as DefaultTheme).surfaceRaised
        : profilerSafe(theme as DefaultTheme).surface};
  }
  &:focus-visible {
    outline: 2px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
    outline-offset: 2px;
  }
`;

const LABELS: Record<ProfilerViewMode, string> = {
  table: 'Top table',
  flame: 'Flame graph',
  both: 'Both',
};

export function ProfilerViewModeToggle({
  value,
  onChange,
  'aria-label': ariaLabel = 'Profile view layout',
}: {
  value: ProfilerViewMode;
  onChange: (mode: ProfilerViewMode) => void;
  'aria-label'?: string;
}) {
  const modes = ['table', 'flame', 'both'] as const;
  return (
    <Group role="group" aria-label={ariaLabel}>
      {modes.map((m) => (
        <ModeButton key={m} $active={value === m} onClick={() => onChange(m)}>
          {LABELS[m]}
        </ModeButton>
      ))}
    </Group>
  );
}
