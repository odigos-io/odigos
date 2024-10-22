import { SETUP } from '@/utils/constants';
import { LogsFocusIcon, LogsIcon, MetricsFocusIcon, MetricsIcon, TraceFocusIcon, TraceIcon } from '@keyval-dev/design-system';

export type MonitoringOption = {
  title: string;
  tapped: boolean;
  icons: object;
  id: number;
};

export const MONITORING_OPTIONS = [
  {
    id: 1,
    icons: {
      notFocus: () => <LogsIcon />,
      focus: () => <LogsFocusIcon />,
    },
    title: SETUP.MONITORS.LOGS,
    type: 'logs',
    tapped: true,
  },
  {
    id: 2,
    icons: {
      notFocus: () => <MetricsIcon />,
      focus: () => <MetricsFocusIcon />,
    },
    title: SETUP.MONITORS.METRICS,
    type: 'metrics',
    tapped: true,
  },
  {
    id: 3,
    icons: {
      notFocus: () => <TraceIcon />,
      focus: () => <TraceFocusIcon />,
    },
    title: SETUP.MONITORS.TRACES,
    type: 'traces',
    tapped: true,
  },
];

export const MONITORS_OPTIONS = [
  {
    id: 'logs',
    value: 'Logs',
  },
  {
    id: 'metrics',
    value: 'Metrics',
  },
  {
    id: 'traces',
    value: 'Traces',
  },
];
