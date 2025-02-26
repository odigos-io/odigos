'use client';

import React from 'react';
import { useActionCRUD } from '@/hooks';
import { ActionTable } from '@odigos/ui-containers';

export default function Page() {
  const { actions } = useActionCRUD();

  return <ActionTable actions={actions} maxHeight='calc(100vh - 220px)' maxWidth='calc(100vw - 70px)' />;
}
