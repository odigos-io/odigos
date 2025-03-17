'use client';

import React from 'react';
import { TABLE_MAX_HEIGHT, TABLE_MAX_WIDTH } from '@/utils';
import { InstrumentationRuleTable } from '@odigos/ui-kit/containers';

export default function Page() {
  return <InstrumentationRuleTable maxHeight={TABLE_MAX_HEIGHT} maxWidth={TABLE_MAX_WIDTH} />;
}
