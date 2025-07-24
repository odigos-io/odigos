import { safeJsonParse } from '@odigos/ui-kit/functions';
import type { OdigosConfig } from '@odigos/ui-kit/types';
import type { FetchedOdigosConfig, OdigosConfigInput } from '@/types';

export const mapFetchedOdigosConfig = (item: FetchedOdigosConfig): Omit<OdigosConfig, 'userInstrumentationEnvs'> => {
  return {
    ...item,
    nodeSelector: safeJsonParse(item.nodeSelector, {}),
  };
};

export const mapOdigosConfigToInput = (item: Partial<OdigosConfig>): OdigosConfigInput => {
  return {
    ...item,
    nodeSelector: JSON.stringify(item.nodeSelector || {}),
  };
};
