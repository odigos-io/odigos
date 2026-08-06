import { useCallback, useEffect, useMemo, useState } from 'react';
import { useUpdateLocalUiConfig } from './useUpdateLocalUiConfig';

// TODO: move this to the ui-kit to work with OdigosApiContext

export type TraceCorrelationsSettings = {
  inputSpanAttributes: string[];
  outputSpanAttributes: string[];
  metricsFlushInterval: string;
};

const DEFAULT_METRICS_FLUSH_INTERVAL = '60s';

type EffectiveConfigServiceIO = {
  inputSpanAttributes?: string[] | null;
  outputSpanAttributes?: string[] | null;
  metricsFlushInterval?: string | null;
};

export function getTraceCorrelationsSettings(effectiveConfig: unknown): TraceCorrelationsSettings {
  if (!effectiveConfig || typeof effectiveConfig !== 'object') {
    return {
      inputSpanAttributes: [],
      outputSpanAttributes: [],
      metricsFlushInterval: DEFAULT_METRICS_FLUSH_INTERVAL,
    };
  }

  const serviceIO = (effectiveConfig as { traceCorrelations?: { serviceIO?: EffectiveConfigServiceIO | null } | null })
    .traceCorrelations?.serviceIO;

  return {
    inputSpanAttributes: serviceIO?.inputSpanAttributes ?? [],
    outputSpanAttributes: serviceIO?.outputSpanAttributes ?? [],
    metricsFlushInterval: serviceIO?.metricsFlushInterval || DEFAULT_METRICS_FLUSH_INTERVAL,
  };
}

export function isValidMetricsFlushInterval(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return false;
  }
  return /^\d+(ms|s|m|h)$/.test(trimmed);
}

function normalizeAttributeList(values: string[]) {
  const seen = new Set<string>();
  const normalized: string[] = [];
  for (const value of values) {
    const trimmed = value.trim();
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    normalized.push(trimmed);
  }
  return normalized;
}

export function settingsAreEqual(a: TraceCorrelationsSettings, b: TraceCorrelationsSettings) {
  return (
    a.metricsFlushInterval === b.metricsFlushInterval &&
    a.inputSpanAttributes.join('\n') === b.inputSpanAttributes.join('\n') &&
    a.outputSpanAttributes.join('\n') === b.outputSpanAttributes.join('\n')
  );
}

export const useTraceCorrelationsSettings = (effectiveConfig: unknown, refetchEffectiveConfig: () => Promise<unknown>) => {
  const appliedSettings = useMemo(() => getTraceCorrelationsSettings(effectiveConfig), [effectiveConfig]);
  const [draft, setDraft] = useState<TraceCorrelationsSettings>(appliedSettings);
  const { updateLocalUiConfig, loading } = useUpdateLocalUiConfig();

  useEffect(() => {
    setDraft(appliedSettings);
  }, [appliedSettings]);

  const isDirty = useMemo(() => !settingsAreEqual(draft, appliedSettings), [draft, appliedSettings]);
  const flushIntervalInvalid = !isValidMetricsFlushInterval(draft.metricsFlushInterval);

  const saveSettings = useCallback(async () => {
    if (flushIntervalInvalid) {
      return false;
    }

    await updateLocalUiConfig({
      traceCorrelations: {
        serviceIO: {
          inputSpanAttributes: normalizeAttributeList(draft.inputSpanAttributes),
          outputSpanAttributes: normalizeAttributeList(draft.outputSpanAttributes),
          metricsFlushInterval: draft.metricsFlushInterval.trim(),
        },
      },
    } as Parameters<typeof updateLocalUiConfig>[0]);

    await refetchEffectiveConfig();
    return true;
  }, [draft, flushIntervalInvalid, refetchEffectiveConfig, updateLocalUiConfig]);

  const resetDraft = useCallback(() => {
    setDraft(appliedSettings);
  }, [appliedSettings]);

  return {
    draft,
    setDraft,
    appliedSettings,
    isDirty,
    flushIntervalInvalid,
    saveSettings,
    resetDraft,
    saving: loading,
  };
};
