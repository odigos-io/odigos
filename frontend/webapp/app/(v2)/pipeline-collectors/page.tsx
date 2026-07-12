'use client';

import React from 'react';
import { PipelineCollectors } from '@odigos/ui-kit/containers';

export default function Page() {
  return <PipelineCollectors minSupportedVersion={1.12} />;
}
