'use client';

import React from 'react';
import { useTraces } from '@/hooks';
import { Text } from '@odigos/ui-kit/components';

export default function Page() {
  const { traces } = useTraces({ serviceName: 'frontend' });
  console.log('traces', traces);

  return <Text>{JSON.stringify(traces, null, 2)}</Text>;
}
