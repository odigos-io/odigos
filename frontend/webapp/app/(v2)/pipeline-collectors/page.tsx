'use client';

import React from 'react';
import { useCollectors, useEffectiveConfig } from '@/hooks';
import { PipelineCollectors } from '@odigos/ui-kit/containers/v2';

export default function Page() {
  const { effectiveConfig } = useEffectiveConfig();
  const { getGatewayInfo, getGatewayPods, getNodeCollectorInfo, getNodeCollectorPods, getExtendedPodInfo } = useCollectors();

  return (
    <PipelineCollectors
      minSupportedVersion={1.12}
      effectiveConfig={effectiveConfig?.odigosOwnTelemetryStore ? { odigosOwnTelemetryStore: effectiveConfig.odigosOwnTelemetryStore } : null}
      getGatewayInfo={getGatewayInfo}
      getGatewayPods={getGatewayPods}
      getNodeCollectorInfo={getNodeCollectorInfo}
      getNodeCollectorPods={getNodeCollectorPods}
      getExtendedPodInfo={getExtendedPodInfo}
    />
  );
}
