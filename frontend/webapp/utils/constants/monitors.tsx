import { SETUP } from '@/utils/constants';

export type SignalUppercase = 'TRACES' | 'METRICS' | 'LOGS';
export type SignalLowercase = 'traces' | 'metrics' | 'logs';

export type MonitoringOption = {
  id: SignalLowercase;
  value: string;
};

export const MONITORS_OPTIONS: MonitoringOption[] = [
  {
    id: 'logs',
    value: SETUP.MONITORS.LOGS,
  },
  {
    id: 'metrics',
    value: SETUP.MONITORS.METRICS,
  },
  {
    id: 'traces',
    value: SETUP.MONITORS.TRACES,
  },
];
