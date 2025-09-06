'use client';

import React from 'react';
import { useServiceMap } from '@/hooks';
import { ServiceMap } from '@odigos/ui-kit/containers';
import { HEADER_HEIGHT, NO_MENU_GAP_HEIGHT } from '@/utils';

export default function Page() {
  const { serviceMap } = useServiceMap();

  return (
    <>
      <ServiceMap heightToRemove={HEADER_HEIGHT + NO_MENU_GAP_HEIGHT} serviceMap={serviceMap} />
    </>
  );
}
