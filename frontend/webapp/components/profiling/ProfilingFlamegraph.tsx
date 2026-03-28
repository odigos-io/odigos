'use client';

import React, { useMemo } from 'react';
import styled from 'styled-components';
import type { DefaultTheme } from 'styled-components';
import type { FlamebearerProfile } from '@/types/profiling';
import type { ProfilerViewMode } from '@/components/profiling/profilerViewMode';
import { profilerSafe } from '@/components/profiling/profilerSafeTheme';
import { buildSymbolStatsRows } from '@/components/profiling/flamebearerSymbolStats';
import { FlamebearerIcicle } from '@/components/profiling/FlamebearerIcicle';
import { ProfilingSymbolTable } from '@/components/profiling/ProfilingSymbolTable';

const GraphWrap = styled.div`
  width: 100%;
  min-height: 200px;
  overflow: auto;
  border-radius: 12px;
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  background: ${({ theme }) => {
    const t = theme as DefaultTheme;
    const s = profilerSafe(t);
    if (s.isDark)
      return t.colors?.dropdown_bg_2 ?? t.colors?.dropdown_bg ?? s.surface;
    return t.colors?.translucent_bg ?? t.colors?.dropdown_bg_2 ?? t.colors?.dropdown_bg ?? s.surfaceRaised;
  }};
  padding: 8px;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
`;

const ContentSplit = styled.div`
  display: flex;
  flex-direction: row;
  gap: 16px;
  width: 100%;
  align-items: flex-start;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  @media (max-width: 960px) {
    flex-direction: column;
  }
`;

const TableColumn = styled.div`
  flex: 0 0 42%;
  min-width: 260px;
  max-width: 100%;
`;

const FlameColumn = styled.div`
  flex: 1;
  min-width: 0;
  width: 100%;
`;

/**
 * In-app flame graph + symbol table (same React as Odigos; avoids @pyroscope/flamegraph’s bundled React 18).
 */
export function ProfilingFlamegraph({
  profile,
  viewMode = 'both',
  search = '',
}: {
  profile: FlamebearerProfile;
  viewMode?: ProfilerViewMode;
  search?: string;
}) {
  const symbolRows = useMemo(() => buildSymbolStatsRows(profile), [profile]);

  if (!profile?.flamebearer?.names?.length) {
    return null;
  }

  if (viewMode === 'table') {
    return <ProfilingSymbolTable rows={symbolRows} search={search} />;
  }

  if (viewMode === 'flame') {
    return (
      <GraphWrap>
        <FlamebearerIcicle profile={profile} />
      </GraphWrap>
    );
  }

  return (
    <ContentSplit>
      <TableColumn>
        <ProfilingSymbolTable rows={symbolRows} search={search} />
      </TableColumn>
      <FlameColumn>
        <GraphWrap>
          <FlamebearerIcicle profile={profile} />
        </GraphWrap>
      </FlameColumn>
    </ContentSplit>
  );
}
