'use client';

import React, { Suspense } from 'react';
import { SamplingRules } from '@odigos/ui-kit/containers';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTelemetryApi } from '@odigos/ui-kit/contexts';
import { useConfig } from '@/hooks';
import { ROUTES } from '@/utils';

function SamplingPage() {
  const api = useTelemetryApi();
  const router = useRouter();
  const params = useSearchParams();
  const { config } = useConfig();
  return (
    <SamplingRules
      traceSurgesEnabled={!!config?.traceSurgesEnabled}
      requestedRuleId={params.get('rule') || undefined}
      onOpenActivity={api ? (ruleId) => router.push(`${ROUTES.TELEMETRY_ACTIVITY}${ruleId ? `?source=trace_surge&sourceId=${encodeURIComponent(ruleId)}` : ''}`) : undefined}
    />
  );
}

export default function Page() {
  return (
    <Suspense fallback={null}>
      <SamplingPage />
    </Suspense>
  );
}
