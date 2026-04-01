'use client';

import React, { useEffect, useMemo, useState } from 'react';
import styled, { ThemeProvider, useTheme } from 'styled-components';
import type { DefaultTheme } from 'styled-components';
import type { Source } from '@odigos/ui-kit/types';
import { useDarkMode } from '@odigos/ui-kit/store';
import { profilerSafe } from '@/components/profiling/profilerSafeTheme';
import { Button, FlexColumn, FlexRow } from '@odigos/ui-kit/components';
import { ProfilingFlamegraph } from '@/components/profiling/ProfilingFlamegraph';
import type { ProfilerViewMode } from '@/components/profiling/profilerViewMode';
import { ProfilerViewModeToggle } from '@/components/profiling/ProfilerViewModeToggle';
import type { FlamebearerProfile } from '@/types/profiling';
import {
  fetchProfilingSlotsDebug,
  type ProfilingSlotsDebug,
  useProfilingAutoRefresh,
  useProfilingHTTP,
} from '@/hooks/profiling';

function formatProfilingBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MiB`;
}

const LIVE_TOOLTIP =
  'Profiles are buffered in the UI pod only while this tab is open. After a UI restart, open the tab again. See /metrics on the UI service for odigos_ui_profiling_* counters.';

const Panel = styled(FlexColumn)`
  width: 100%;
  gap: 12px;
  padding: 4px 0 16px;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  font-family: ${({ theme }) => profilerSafe(theme as DefaultTheme).fontBody};
`;

const Muted = styled.p`
  font-size: 0.8125rem;
  margin: 0;
  line-height: 1.4;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fgMuted};
`;

const ProfileStatsLine = styled.p`
  font-size: 0.8125rem;
  margin: 0;
  line-height: 1.45;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
`;

const ErrorText = styled.p`
  font-size: 0.8125rem;
  margin: 0;
  line-height: 1.4;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).err};
`;

const Toolbar = styled(FlexRow)`
  width: 100%;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
`;

const TitleRow = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 1rem;
  font-weight: 600;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
`;

const SearchInput = styled.input`
  flex: 1;
  min-width: 140px;
  max-width: 320px;
  padding: 8px 12px;
  border-radius: 10px;
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  background: ${({ theme }) => profilerSafe(theme as DefaultTheme).surfaceRaised};
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  font-size: 13px;
  outline: none;
  &::placeholder {
    color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fgMuted};
    opacity: 0.9;
  }
`;

const ModeToolbar = styled(FlexRow)`
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
`;

const LiveBadge = styled.span`
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fgMuted};
  cursor: help;
  &::before {
    content: '';
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: ${({ theme }) => profilerSafe(theme as DefaultTheme).success};
    flex-shrink: 0;
  }
`;

const ActionsRow = styled(FlexRow)`
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  width: 100%;
`;

const KeyCode = styled.code`
  font-size: 0.85em;
  font-family: ${({ theme }) => profilerSafe(theme as DefaultTheme).fontCode};
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  background: ${({ theme }) => profilerSafe(theme as DefaultTheme).surfaceRaised};
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
`;

const DiagnosticsPre = styled.pre`
  margin: 0;
  padding: 10px;
  font-size: 11px;
  line-height: 1.4;
  border-radius: 8px;
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  background: ${({ theme }) => profilerSafe(theme as DefaultTheme).surfaceRaised};
  color: ${({ theme }) => profilerSafe(theme as DefaultTheme).fg};
  font-family: ${({ theme }) => profilerSafe(theme as DefaultTheme).fontCode};
  overflow: auto;
  max-height: 200px;
`;

function downloadProfileJson(profile: FlamebearerProfile, ns: string, workload: string) {
  const safe = workload.replace(/[^a-zA-Z0-9._-]+/g, '-');
  const blob = new Blob([JSON.stringify(profile, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `cpu-profile-${ns}-${safe}-${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

function SourceProfilerTabInner({ source }: { source: Source }) {
  const theme = useTheme() as DefaultTheme;
  const safe = profilerSafe(theme);

  const ns = source.namespace;
  const kind = String(source.kind || 'Deployment');
  const name = source.name;

  const [viewMode, setViewMode] = useState<ProfilerViewMode>('both');
  const [search, setSearch] = useState('');
  const [diagLoading, setDiagLoading] = useState(false);
  const [diagError, setDiagError] = useState<string | null>(null);
  const [diagOpen, setDiagOpen] = useState(false);
  const [diagJson, setDiagJson] = useState<string | null>(null);
  const [slotStats, setSlotStats] = useState<ProfilingSlotsDebug | null>(null);

  const { loading, error, profile, lastSourceKey, enableMeta, enableAndLoad, load } = useProfilingHTTP();

  useEffect(() => {
    let cancelled = false;
    const refresh = () => {
      void fetchProfilingSlotsDebug()
        .then((d) => {
          if (!cancelled) setSlotStats(d);
        })
        .catch(() => {});
    };
    refresh();
    const id = setInterval(refresh, 12000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, []);

  // Run enable+initial load when the source identity changes only. Including enableAndLoad in deps retriggers
  // on every callback identity change and can leave `loading` true often (drawer feels frozen).
  useEffect(() => {
    if (!ns || !name || !kind) return;
    void enableAndLoad(ns, kind, name);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- stable initial load per workload
  }, [ns, kind, name]);

  useProfilingAutoRefresh(load, ns, kind, name, profile, { enabled: !!(ns && name && kind) });

  const ticks = profile?.flamebearer?.numTicks ?? 0;
  const emptyProfile = !profile || ticks === 0;

  const loadDiagnostics = async () => {
    setDiagLoading(true);
    setDiagError(null);
    try {
      const d = await fetchProfilingSlotsDebug();
      setDiagJson(JSON.stringify(d, null, 2));
      setDiagOpen(true);
    } catch (e) {
      setDiagError(e instanceof Error ? e.message : String(e));
      setDiagJson(null);
    } finally {
      setDiagLoading(false);
    }
  };

  return (
    <Panel>
      <Toolbar>
        <TitleRow>CPU Profiling</TitleRow>
        <SearchInput
          type="search"
          placeholder="Filter symbols in table"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Filter symbols"
        />
        <ModeToolbar>
          <ProfilerViewModeToggle value={viewMode} onChange={setViewMode} />
          <LiveBadge title={LIVE_TOOLTIP} aria-label={LIVE_TOOLTIP}>
            Live
          </LiveBadge>
        </ModeToolbar>
      </Toolbar>

      <ActionsRow>
        <Button
          variant="secondary"
          disabled={loading || !profile || emptyProfile}
          onClick={() => profile && downloadProfileJson(profile, ns, name)}
        >
          Download snapshot
        </Button>
        <Button
          variant="tertiary"
          disabled={diagLoading}
          onClick={() => void loadDiagnostics()}
          style={{ color: safe.fg }}
        >
          {diagLoading ? 'Loading diagnostics…' : 'Slot diagnostics'}
        </Button>
        <Button variant="secondary" disabled={loading} onClick={() => void load(ns, kind, name)}>
          {loading ? 'Loading…' : 'Refresh'}
        </Button>
      </ActionsRow>

      <FlexColumn style={{ gap: 8 }}>
        <Muted>
          On-demand CPU profile for this workload. Bar colors use the Odigos palette; flame labels pick light or dark text
          for contrast.
        </Muted>
        {lastSourceKey && (
          <Muted>
            Source key: <KeyCode>{lastSourceKey}</KeyCode>
          </Muted>
        )}
        {enableMeta && (
          <Muted>
            Profiling slots in use: {enableMeta.activeSlots} / {enableMeta.maxSlots} (in-memory; oldest evicted when
            full).
          </Muted>
        )}
        {slotStats && slotStats.maxTotalBytesBudget > 0 && (
          <Muted>
            Profile buffer: {formatProfilingBytes(slotStats.totalBytesUsed)} /{' '}
            {formatProfilingBytes(slotStats.maxTotalBytesBudget)} across all active slots · per-slot cap{' '}
            {formatProfilingBytes(slotStats.slotMaxBytes)} · TTL {slotStats.slotTtlSeconds}s
          </Muted>
        )}
      </FlexColumn>

      {diagError && <ErrorText>{diagError}</ErrorText>}
      {diagOpen && diagJson && (
        <FlexColumn style={{ gap: 6 }}>
          <Muted style={{ margin: 0 }}>Active keys vs keys with buffered data (UI backend):</Muted>
          <DiagnosticsPre>{diagJson}</DiagnosticsPre>
        </FlexColumn>
      )}

      {error && <ErrorText>{error}</ErrorText>}

      {profile && !error && ticks > 0 && (
        <ProfileStatsLine>
          Total samples in view {ticks.toLocaleString()} · Frames: {profile.flamebearer.names.length} ·{' '}
          {profile.metadata?.name || 'cpu'} ({profile.metadata?.units || 'samples'})
          {profile.metadata?.symbolsHint ? ` · ${profile.metadata.symbolsHint}` : ''}
        </ProfileStatsLine>
      )}

      {profile && !error && emptyProfile && !loading && (
        <Muted>
          No usable CPU samples yet (auto-refresh runs while this tab is open). Send traffic to the workload; OTLP
          batches must include namespace/workload labels to match this source. If you see “symbols unavailable” or only 1
          frame, the collector may be sending chunks without full dictionaries — try another service (e.g.
          productcatalogservice) or use Refresh after load.
        </Muted>
      )}

      {profile && !emptyProfile && (
        <ProfilingFlamegraph profile={profile} viewMode={viewMode} search={search} />
      )}
    </Panel>
  );
}

/**
 * Merges zustand `darkMode` into styled-components theme so profiler tokens match the shell (source drawer).
 */
export function SourceProfilerTab({ source }: { source: Source }) {
  const parentTheme = useTheme() as DefaultTheme;
  const { darkMode } = useDarkMode();
  const mergedTheme = useMemo(
    () => ({
      ...(parentTheme ?? ({} as DefaultTheme)),
      darkMode,
    }),
    [parentTheme, darkMode],
  );

  return (
    <ThemeProvider theme={mergedTheme}>
      <SourceProfilerTabInner source={source} />
    </ThemeProvider>
  );
}
