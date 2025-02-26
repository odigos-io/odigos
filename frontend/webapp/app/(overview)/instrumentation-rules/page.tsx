'use client';

import React from 'react';
import { useInstrumentationRuleCRUD } from '@/hooks';
import { InstrumentationRuleTable } from '@odigos/ui-containers';

export default function Page() {
  const { instrumentationRules } = useInstrumentationRuleCRUD();

  return <InstrumentationRuleTable instrumentationRules={instrumentationRules} tableMaxHeight='calc(100vh - 220px)' />;
}
