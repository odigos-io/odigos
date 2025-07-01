'use client';

import React from 'react';
import { HEADER_HEIGHT } from '@/utils';
import { useServiceMap } from '@/hooks';
import { ServiceMap } from '@odigos/ui-kit/containers';

export default function Page() {
  const { serviceMap } = useServiceMap();

  return (
    <>
      <ServiceMap heightToRemove={HEADER_HEIGHT} serviceMap={serviceMap} />
    </>
  );
}
