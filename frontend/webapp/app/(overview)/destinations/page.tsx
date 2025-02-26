'use client';

import React from 'react';
import { useDestinationCRUD } from '@/hooks';
import { DestinationTable } from '@odigos/ui-containers';

export default function Page() {
  const { destinations } = useDestinationCRUD();

  return <DestinationTable destinations={destinations} tableMaxHeight='calc(100vh - 220px)' />;
}
