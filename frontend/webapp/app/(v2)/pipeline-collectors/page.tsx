'use client';

import React from 'react';
import { PlatformType } from '@odigos/ui-kit/types';
import { PipelineCollectors } from '@odigos/ui-kit/containers';

export default function Page() {
  return (
    <PipelineCollectors
      minSupportedVersion={{
        [PlatformType.K8s]: 1.12,
      }}
    />
  );
}
