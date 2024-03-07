'use client';
import { useState } from 'react';
import { KeyvalNotification } from '@/design.system';

export function useNotification() {
  const [data, show] = useState<any>(false);

  function Notification() {
    return data && <KeyvalNotification {...data} onClose={() => show(false)} />;
  }

  return { show, Notification };
}
