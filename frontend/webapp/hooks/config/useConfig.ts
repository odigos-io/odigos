'use client';

import type { FetchedConfig } from '@odigos/ui-kit/types';
import { useApiQuery } from '@odigos/ui-kit/contexts';

export const useConfig = (): { config: FetchedConfig; isReadonly: boolean } => {
  const { data: config } = useApiQuery('GET_CONFIG');

  const isReadonly = config?.readonly || false;

  return { config, isReadonly };
};
