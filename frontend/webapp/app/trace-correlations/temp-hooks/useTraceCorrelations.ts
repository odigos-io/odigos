import { useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { GET_TRACE_CORRELATIONS } from '@/graphql';

// TODO: move this to the ui-kit to work with OdigosApiContext

export type TraceCorrelationsAttribute = {
  key: string;
  value: string;
};

export type TraceCorrelationsOutputSeries = {
  attributes: TraceCorrelationsAttribute[];
  connectionCount: number;
  firstDetectedAt: string;
};

export type TraceCorrelationsInputGroup = {
  attributes: TraceCorrelationsAttribute[];
  outputs: TraceCorrelationsOutputSeries[];
};

export type TraceCorrelationsWorkload = {
  namespace: string;
  kind: string;
  name: string;
  containerName: string;
  telemetrySdkLanguage?: string | null;
  processRuntimeName?: string | null;
  processRuntimeVersion?: string | null;
  inputs: TraceCorrelationsInputGroup[];
};

export type WorkloadFilter = {
  namespace?: string;
  kind?: string;
  name?: string;
};

export type TraceCorrelationsTimeRange = {
  start: string;
  end: string;
};

export type TraceCorrelationsTimePreset = '5m' | '15m' | '1h' | '6h' | '24h' | 'custom';

type TraceCorrelationsRelativePreset = Exclude<TraceCorrelationsTimePreset, 'custom'>;

const PRESET_DURATIONS_MS: Record<TraceCorrelationsRelativePreset, number> = {
  '5m': 5 * 60 * 1000,
  '15m': 15 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
};

export const TRACE_CORRELATIONS_TIME_PRESETS: { id: TraceCorrelationsRelativePreset; label: string }[] = [
  { id: '5m', label: 'Last 5m' },
  { id: '15m', label: 'Last 15m' },
  { id: '1h', label: 'Last 1h' },
  { id: '6h', label: 'Last 6h' },
  { id: '24h', label: 'Last 24h' },
];

export function toDatetimeLocalValue(date: Date) {
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function parseDatetimeLocalValue(value: string) {
  if (!value) {
    return null;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date;
}

export function buildTraceCorrelationsTimeRange(preset: TraceCorrelationsRelativePreset, now = Date.now()): TraceCorrelationsTimeRange {
  const end = new Date(now);
  const start = new Date(now - PRESET_DURATIONS_MS[preset]);
  return {
    start: start.toISOString(),
    end: end.toISOString(),
  };
}

export function buildCustomTraceCorrelationsTimeRange(startValue: string, endValue: string): TraceCorrelationsTimeRange | null {
  const start = parseDatetimeLocalValue(startValue);
  const end = parseDatetimeLocalValue(endValue);
  if (!start || !end || end <= start) {
    return null;
  }
  return {
    start: start.toISOString(),
    end: end.toISOString(),
  };
}

export function resolveTraceCorrelationsTimeRange(options: {
  preset: TraceCorrelationsTimePreset;
  customStart: string;
  customEnd: string;
  now?: number;
}): TraceCorrelationsTimeRange | null {
  if (options.preset === 'custom') {
    return buildCustomTraceCorrelationsTimeRange(options.customStart, options.customEnd);
  }
  return buildTraceCorrelationsTimeRange(options.preset, options.now);
}

export function formatTraceCorrelationsTimeRangeLabel(timeRange: TraceCorrelationsTimeRange) {
  const start = new Date(timeRange.start);
  const end = new Date(timeRange.end);
  const timeFormatter = new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
  });
  const dateFormatter = new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
  });
  const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  });

  if (start.toDateString() === end.toDateString()) {
    return `Data for ${dateFormatter.format(start)}, ${timeFormatter.format(start)} – ${timeFormatter.format(end)}`;
  }

  return `Data for ${dateTimeFormatter.format(start)} – ${dateTimeFormatter.format(end)}`;
}

type TraceCorrelationsResponse = {
  traceCorrelations: {
    workloads: TraceCorrelationsWorkload[];
  };
};

export const TRACE_CORRELATIONS_HELM_VALUES_SNIPPET = `traceCorrelations:
  serviceIO:
    enabled: true`;

export const TRACE_CORRELATIONS_CLI_COMMANDS_SNIPPET = `odigos install --set traceCorrelations.serviceIO.enabled=true
odigos upgrade --set traceCorrelations.serviceIO.enabled=true`;

type EffectiveConfigTraceCorrelations = {
  traceCorrelations?: {
    serviceIO?: {
      enabled?: boolean | null;
    } | null;
  } | null;
};

export function isTraceCorrelationsEnabled(effectiveConfig: unknown) {
  if (!effectiveConfig || typeof effectiveConfig !== 'object') {
    return false;
  }

  const traceCorrelations = (effectiveConfig as EffectiveConfigTraceCorrelations).traceCorrelations;
  return traceCorrelations?.serviceIO?.enabled === true;
}

export function isTraceCorrelationsDisabledError(error: { message?: string } | null | undefined) {
  return Boolean(error?.message?.includes('trace correlations are not enabled'));
}

export function isTraceCorrelationsMetricsStoreUnavailableError(error: { message?: string } | null | undefined) {
  const message = error?.message?.toLowerCase() ?? '';
  if (!message) {
    return false;
  }
  if (message.includes('trace correlations metrics store is unavailable')) {
    return true;
  }

  return (
    message.includes('odigos-correlations-metrics') &&
    (message.includes('connection refused') ||
      message.includes('no such host') ||
      message.includes('dial tcp') ||
      message.includes('connect: network is unreachable'))
  );
}

export const useTraceCorrelations = (
  filter?: WorkloadFilter,
  timeRange?: TraceCorrelationsTimeRange | null,
  options?: { enabled?: boolean },
) => {
  const queryEnabled = (options?.enabled ?? true) && !!timeRange;
  const variables = useMemo(
    () => ({
      filter: filter ?? null,
      timeRange: timeRange ?? null,
    }),
    [filter, timeRange],
  );

  const { data, loading, error, refetch } = useQuery<TraceCorrelationsResponse>(GET_TRACE_CORRELATIONS, {
    variables,
    skip: !queryEnabled,
  });

  return {
    workloads: data?.traceCorrelations?.workloads ?? [],
    loading,
    error,
    refetch,
  };
};
