'use client';

import React from 'react';
import { useServiceMap } from '@/hooks';
import { ServiceMap } from '@odigos/ui-kit/containers';

export default function Page() {
  const { serviceMap, refetch } = useServiceMap();

  return <ServiceMap serviceMap={serviceMap} onRefresh={() => refetch()} />;
}
