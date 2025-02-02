import { SignalUppercase } from '@/utils/constants';
import { LogsIcon, MetricsIcon, TracesIcon, Types } from '@odigos/ui-components';

export const getMonitorIcon = (type: string) => {
  const LOGOS: Record<SignalUppercase, Types.SVG> = {
    ['LOGS']: LogsIcon,
    ['METRICS']: MetricsIcon,
    ['TRACES']: TracesIcon,
  };

  return LOGOS[type.toUpperCase() as SignalUppercase];
};
