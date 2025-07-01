'use client';

import React from 'react';
import { useServiceMap } from '@/hooks';
import { ServiceMap } from '@odigos/ui-kit/containers';
import { HEADER_HEIGHT, MENU_BAR_HEIGHT } from '@/utils';

export default function Page() {
  const { serviceMap } = useServiceMap();

  return (
    <>
      <ServiceMap heightToRemove={HEADER_HEIGHT + MENU_BAR_HEIGHT} serviceMap={serviceMap} />
    </>
  );
}
