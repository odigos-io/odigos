'use client';

import React, { useCallback, useMemo } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { TelemetryWorkspace } from '@odigos/ui-kit/containers';
import type { TelemetryNavigation, TelemetrySourceRef } from '@odigos/ui-kit/types';
import { ROUTES } from '@/utils';

export const TelemetryPage = () => {
  const router = useRouter();
  const params = useSearchParams();
  const navigation = useMemo<TelemetryNavigation>(
    () => ({
      activeOnly: params.get('tab') === 'active',
      changeId: params.get('change') || undefined,
      source: params.get('source') === 'trace_surge' ? 'trace_surge' : undefined,
      sourceId: params.get('sourceId') || undefined,
    }),
    [params],
  );
  const onNavigate = useCallback(
    (next: TelemetryNavigation) => {
      const query = new URLSearchParams();
      if (next.activeOnly) query.set('tab', 'active');
      if (next.changeId) query.set('change', next.changeId);
      if (next.source) query.set('source', next.source);
      if (next.sourceId) query.set('sourceId', next.sourceId);
      router.push(`${ROUTES.TELEMETRY_ACTIVITY}${query.size ? `?${query.toString()}` : ''}`);
    },
    [router],
  );
  const onOpenSource = useCallback((ref: TelemetrySourceRef) => router.push(`${ROUTES.SAMPLING}?${new URLSearchParams({ rule: ref.id }).toString()}`), [router]);
  return <TelemetryWorkspace navigation={navigation} onNavigate={onNavigate} onOpenSource={onOpenSource} />;
};
