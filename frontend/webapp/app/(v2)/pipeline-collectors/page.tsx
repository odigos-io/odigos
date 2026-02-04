'use client';

import React from 'react';
import { useCollectors } from '@/hooks';
import { PipelineCollectors } from '@odigos/ui-kit/containers/v2';

export default function Page() {
  const { getGatewayInfo, getGatewayPods, getNodeCollectorInfo, getNodeCollectorPods, getExtendedPodInfo } = useCollectors();

  return (
    <PipelineCollectors
      minSupportedVersion={1.12}
      tableRowsMaxHeight={'calc(100vh - 350px)'}
      getGatewayInfo={getGatewayInfo}
      getGatewayPods={getGatewayPods}
      getNodeCollectorInfo={getNodeCollectorInfo}
      getNodeCollectorPods={getNodeCollectorPods}
      getExtendedPodInfo={getExtendedPodInfo}
    />
  );
}
