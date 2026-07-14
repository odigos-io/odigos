'use client';

import React from 'react';
import { ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { Onboarding } from '@odigos/ui-kit/containers';

export default function Page() {
  const router = useRouter();
  return <Onboarding onDone={() => router.push(ROUTES.OVERVIEW)} />;
}
