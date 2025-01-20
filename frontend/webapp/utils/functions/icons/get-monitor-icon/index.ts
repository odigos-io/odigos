import { SignalUppercase } from '@/utils/constants';
import { LogsIcon, MetricsIcon, SVG, TracesIcon } from '@/assets';

export const getMonitorIcon = (type: string) => {
  const LOGOS: Record<SignalUppercase, SVG> = {
    ['LOGS']: LogsIcon,
    ['METRICS']: MetricsIcon,
    ['TRACES']: TracesIcon,
  };

  return LOGOS[type.toUpperCase() as SignalUppercase];
};
