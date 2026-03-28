'use client';

import React, { useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import styled, { ThemeProvider, useTheme } from 'styled-components';
import type { DefaultTheme } from 'styled-components';
import { useQuery } from '@apollo/client';
import { useDarkMode } from '@odigos/ui-kit/store';
import { Button } from '@odigos/ui-kit/components';
import { ProfilingFlamegraph } from '@/components/profiling/ProfilingFlamegraph';
import { profilerSafe } from '@/components/profiling/profilerSafeTheme';
import {
  ProfilerErrorLine,
  ProfilerField,
  ProfilerFormRow,
  ProfilerHelp,
  ProfilerInput,
  ProfilerMuted,
  ProfilerPagePanel,
  ProfilerSelect,
  ProfilerStatsLine,
  ProfilerTitle,
} from '@/components/profiling/ProfilerPrimitives';
import { ProfilerViewModeToggle } from '@/components/profiling/ProfilerViewModeToggle';
import type { ProfilerViewMode } from '@/components/profiling/profilerViewMode';
import {
  PROFILING_SLOTS_QUERY,
  type ProfilingSlotsDebug,
  useProfilingAutoRefresh,
  useProfilingHTTP,
} from '@/hooks/profiling';

const KIND_OPTIONS = ['Deployment', 'StatefulSet', 'DaemonSet', 'CronJob', 'Job'] as const;

const SymbolSearch = styled(ProfilerInput)`
  max-width: 280px;
`;

const DiagPanel = styled.aside`
  margin-top: 4px;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid ${({ theme }) => profilerSafe(theme as DefaultTheme).border};
  font-size: 0.8125rem;
  background: ${({ theme }) => profilerSafe(theme as DefaultTheme).surface};
`;

function ProfilingPageInner() {
  const searchParams = useSearchParams();
  const initialNs = searchParams.get('namespace') || searchParams.get('ns') || '';
  const initialKind = searchParams.get('kind') || 'Deployment';
  const initialName = searchParams.get('name') || '';
  const showProfilingDiag = searchParams.get('debug') === '1';

  const { data: slotsDiag, loading: slotsLoading } = useQuery<{ profilingSlots: ProfilingSlotsDebug }>(
    PROFILING_SLOTS_QUERY,
    {
      skip: !showProfilingDiag,
      pollInterval: showProfilingDiag ? 8000 : undefined,
      fetchPolicy: 'network-only',
    },
  );

  const [namespace, setNamespace] = React.useState(initialNs);
  const [kind, setKind] = React.useState(initialKind);
  const [name, setName] = React.useState(initialName);
  const [viewMode, setViewMode] = useState<ProfilerViewMode>('both');
  const [symbolSearch, setSymbolSearch] = useState('');

  const { loading, error, profile, lastSourceKey, enableAndLoad, load, clear } = useProfilingHTTP();

  const canSubmit = useMemo(() => !!(namespace.trim() && kind.trim() && name.trim()), [namespace, kind, name]);

  useProfilingAutoRefresh(load, namespace.trim(), kind.trim(), name.trim(), profile, { enabled: canSubmit });

  const onEnableAndLoad = async () => {
    if (!canSubmit) return;
    await enableAndLoad(namespace.trim(), kind.trim(), name.trim());
  };

  const onRefresh = async () => {
    if (!canSubmit) return;
    await load(namespace.trim(), kind.trim(), name.trim());
  };

  const ticks = profile?.flamebearer?.numTicks ?? 0;
  const emptyProfile = !profile || ticks === 0;

  return (
    <ProfilerPagePanel>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <ProfilerTitle>Continuous profiling</ProfilerTitle>
        <ProfilerHelp title="Data is kept only for workloads with an active slot (Enable here or open the Source drawer Profiler tab). Re-enable after a UI pod restart.">
          Loads aggregated CPU profile data for a workload. Use the real Kubernetes workload name (e.g.{' '}
          <code>frontend</code>), not the InstrumentationConfig resource name.
        </ProfilerHelp>
        <ProfilerMuted style={{ margin: 0 }}>
          Add <code>?debug=1</code> for slot diagnostics (polls GraphQL <code>profilingSlots</code>).
        </ProfilerMuted>
      </div>

      <ProfilerFormRow>
        <ProfilerField>
          <span>Namespace</span>
          <ProfilerInput
            value={namespace}
            onChange={(e) => setNamespace(e.target.value)}
            placeholder="e.g. online-boutique"
            autoComplete="off"
          />
        </ProfilerField>
        <ProfilerField>
          <span>Kind</span>
          <ProfilerSelect value={kind} onChange={(e) => setKind(e.target.value)} aria-label="Workload kind">
            {KIND_OPTIONS.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </ProfilerSelect>
        </ProfilerField>
        <ProfilerField>
          <span>Workload name</span>
          <ProfilerInput
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. productcatalogservice"
            autoComplete="off"
          />
        </ProfilerField>
        <Button variant="primary" type="button" disabled={!canSubmit || loading} onClick={onEnableAndLoad}>
          {loading ? 'Loading…' : 'Enable & load profile'}
        </Button>
        <Button variant="secondary" type="button" disabled={!canSubmit || loading} onClick={onRefresh}>
          Refresh
        </Button>
        <Button variant="tertiary" type="button" onClick={clear}>
          Clear
        </Button>
      </ProfilerFormRow>

      {showProfilingDiag && (
        <DiagPanel aria-label="Profiling slot diagnostics">
          <ProfilerTitle style={{ fontSize: '0.95rem', marginBottom: 8 }}>Diagnostics (debug)</ProfilerTitle>
          {slotsLoading && !slotsDiag && (
            <ProfilerMuted style={{ margin: 0 }}>Loading profilingSlots…</ProfilerMuted>
          )}
          {slotsDiag?.profilingSlots && (
            <>
          <ProfilerMuted style={{ margin: '0 0 8px' }}>
            <code>profilingSlots</code> — your workload should appear under active keys before data buffers.
          </ProfilerMuted>
          <ProfilerStatsLine style={{ margin: '4px 0' }}>
            Memory: {slotsDiag.profilingSlots.totalBytesUsed} / {slotsDiag.profilingSlots.maxTotalBytesBudget} bytes
            budget · {slotsDiag.profilingSlots.maxSlots} slots · TTL {slotsDiag.profilingSlots.slotTtlSeconds}s · cap{' '}
            {slotsDiag.profilingSlots.slotMaxBytes} B/slot
          </ProfilerStatsLine>
          <ProfilerMuted style={{ margin: '6px 0 4px' }}>Active keys</ProfilerMuted>
          <pre style={{ margin: 0, fontSize: 12, whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            {slotsDiag.profilingSlots.activeKeys.length ? slotsDiag.profilingSlots.activeKeys.join(', ') : '(none)'}
          </pre>
          <ProfilerMuted style={{ margin: '8px 0 4px' }}>Keys with data</ProfilerMuted>
          <pre style={{ margin: 0, fontSize: 12, whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            {slotsDiag.profilingSlots.keysWithData.length ? slotsDiag.profilingSlots.keysWithData.join(', ') : '(none)'}
          </pre>
            </>
          )}
        </DiagPanel>
      )}

      {lastSourceKey && (
        <ProfilerMuted>
          Source key: <code>{lastSourceKey}</code>
        </ProfilerMuted>
      )}

      {error && <ProfilerErrorLine>{error}</ProfilerErrorLine>}

      {profile && !error && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <ProfilerStatsLine>
            Samples (ticks): {ticks.toLocaleString()} · Frames: {profile.flamebearer.names.length} ·{' '}
            {profile.metadata?.name || 'cpu'} ({profile.metadata?.units || 'samples'})
          </ProfilerStatsLine>
          {profile.metadata?.symbolsHint && <ProfilerMuted>{profile.metadata.symbolsHint}</ProfilerMuted>}
          <ProfilerFormRow style={{ alignItems: 'center' }}>
            <ProfilerField style={{ flex: 1, minWidth: 180, maxWidth: 360 }}>
              <span>Filter symbols (table)</span>
              <SymbolSearch
                value={symbolSearch}
                onChange={(e) => setSymbolSearch(e.target.value)}
                placeholder="e.g. httpx"
                autoComplete="off"
                aria-label="Filter symbol table"
              />
            </ProfilerField>
            <ProfilerViewModeToggle value={viewMode} onChange={setViewMode} aria-label="Profiling page layout" />
          </ProfilerFormRow>
        </div>
      )}

      {emptyProfile && !loading && !error && profile && (
        <ProfilerHelp>
          No samples yet — this page auto-refreshes until data appears. Keep traffic on the workload; first batches may
          lack Kubernetes labels and are dropped until the collector enriches them.
        </ProfilerHelp>
      )}

      {profile && !emptyProfile && (
        <ProfilingFlamegraph profile={profile} viewMode={viewMode} search={symbolSearch} />
      )}
    </ProfilerPagePanel>
  );
}

export default function ProfilingPage() {
  const parentTheme = useTheme() as DefaultTheme;
  const { darkMode } = useDarkMode();
  const theme = useMemo(
    () => ({
      ...(parentTheme ?? ({} as DefaultTheme)),
      darkMode,
    }),
    [parentTheme, darkMode],
  );

  return (
    <ThemeProvider theme={theme}>
      <ProfilingPageInner />
    </ThemeProvider>
  );
}
