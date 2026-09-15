'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';

export default function Page() {
  const router = useRouter();

  useEffect(() => {
    router.replace(ROUTES.OVERVIEW);
  }, [router]);

  return null;
}
