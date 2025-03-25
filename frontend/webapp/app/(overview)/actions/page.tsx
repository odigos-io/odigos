'use client';

import React from 'react';
import { ActionTable } from '@odigos/ui-kit/containers';
import { TABLE_MAX_HEIGHT, TABLE_MAX_WIDTH } from '@/utils';

export default function Page() {
  return <ActionTable maxHeight={TABLE_MAX_HEIGHT} maxWidth={TABLE_MAX_WIDTH} />;
}
