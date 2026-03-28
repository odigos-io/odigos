'use client';

import styled from 'styled-components';
import type { DefaultTheme } from 'styled-components';
import { FlexColumn } from '@odigos/ui-kit/components';
import { profilerSafe } from '@/components/profiling/profilerSafeTheme';
import { TABLE_MAX_WIDTH } from '@/utils';

/** Shared Odigos-themed primitives for `/profiling` and other full-page profiler contexts. */
export const ProfilerPagePanel = styled(FlexColumn)`
  max-width: ${TABLE_MAX_WIDTH};
  width: 100%;
  gap: 16px;
  padding: 16px 0;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  font-family: ${({ theme }) => profilerSafe(theme as DefaultTheme).fontBody};
`;

export const ProfilerTitle = styled.h1`
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
`;

export const ProfilerMuted = styled.p`
  font-size: 0.8125rem;
  margin: 0;
  line-height: 1.4;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fgMuted};
  code {
    font-family: ${({ theme }) => profilerSafe(theme as DefaultTheme).fontCode};
    font-size: 0.92em;
    color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
    background: ${({ theme }) => profilerSafe(theme as DefaultTheme).surfaceRaised};
    padding: 1px 6px;
    border-radius: 4px;
    border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  }
`;

export const ProfilerHelp = styled.p`
  font-size: 0.875rem;
  margin: 0;
  line-height: 1.45;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fgMuted};
  strong {
    color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
    font-weight: 600;
  }
  code {
    font-family: ${({ theme }) => profilerSafe(theme as DefaultTheme).fontCode};
    color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
    background: ${({ theme }) => profilerSafe(theme as DefaultTheme).surfaceRaised};
    padding: 1px 6px;
    border-radius: 4px;
    border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  }
`;

export const ProfilerErrorLine = styled.p`
  font-size: 0.875rem;
  margin: 0;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).err};
  line-height: 1.4;
`;

export const ProfilerStatsLine = styled.p`
  font-size: 0.875rem;
  margin: 0;
  line-height: 1.45;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
`;

export const ProfilerFormRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: flex-end;
`;

export const ProfilerField = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 140px;
  font-size: 0.8125rem;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fgMuted};
`;

export const ProfilerInput = styled.input`
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  background: ${({ theme }) => profilerSafe(theme as DefaultTheme).surfaceRaised};
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  font-size: 13px;
  outline: none;
  &::placeholder {
    color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fgMuted};
  }
`;

export const ProfilerSelect = styled.select`
  padding: 8px 10px;
  border-radius: 8px;
  min-width: 140px;
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  background: ${({ theme }) => profilerSafe(theme as DefaultTheme).surfaceRaised};
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  font-size: 13px;
`;
